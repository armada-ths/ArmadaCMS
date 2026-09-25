package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func photoEventFromToken(w http.ResponseWriter, r *http.Request) (*models.PhotoEvent, bool) {
	id, version, err := utils.ParsePhotoEventToken(mux.Vars(r)["token"])
	if err != nil {
		http.Error(w, "Invalid event link", http.StatusNotFound)
		return nil, false
	}
	var event models.PhotoEvent
	if err := db.DB.First(&event, id).Error; err != nil || !event.Active || event.TokenVersion != version || !time.Now().Before(event.DeleteAfter) {
		http.Error(w, "Event is unavailable", http.StatusNotFound)
		return nil, false
	}
	return &event, true
}

func photoGuestQuota(event *models.PhotoEvent, guestID string) (int64, error) {
	if len(guestID) < 16 || len(guestID) > 128 {
		return 0, errors.New("invalid guest ID")
	}
	hash, err := utils.HashPhotoGuest(event.ID, guestID)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.DB.Model(&models.EventPhoto{}).Where("event_id = ? AND guest_hash = ? AND status <> 'rejected'", event.ID, hash).Count(&count).Error
	return count, err
}

func PhotoEventAccess(w http.ResponseWriter, r *http.Request) {
	event, ok := photoEventFromToken(w, r)
	if !ok {
		return
	}
	now := time.Now()
	remaining := event.MaxPhotosPerGuest
	if guestID := r.URL.Query().Get("guest_id"); guestID != "" {
		count, err := photoGuestQuota(event, guestID)
		if err != nil {
			http.Error(w, "Invalid guest ID", http.StatusBadRequest)
			return
		}
		remaining = max(0, remaining-int(count))
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name": event.Name, "description": event.Description,
		"uploads_open_at": event.UploadsOpenAt, "uploads_close_at": event.UploadsCloseAt,
		"gallery_close_at": event.GalleryCloseAt, "privacy_url": event.PrivacyURL,
		"uploads_open": !now.Before(event.UploadsOpenAt) && now.Before(event.UploadsCloseAt),
		"gallery_open": now.Before(event.GalleryCloseAt), "remaining": remaining,
	})
}

func PhotoEventUpload(w http.ResponseWriter, r *http.Request) {
	event, ok := photoEventFromToken(w, r)
	if !ok {
		return
	}
	now := time.Now()
	if now.Before(event.UploadsOpenAt) || !now.Before(event.UploadsCloseAt) {
		http.Error(w, "Uploads are closed", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxPhotoOriginalBytes+1024*1024)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		http.Error(w, "Upload too large", http.StatusRequestEntityTooLarge)
		return
	}
	if r.FormValue("privacy_confirmed") != "true" {
		http.Error(w, "Privacy confirmation required", http.StatusBadRequest)
		return
	}
	guestID := r.FormValue("guest_id")
	count, err := photoGuestQuota(event, guestID)
	if err != nil {
		http.Error(w, "Invalid guest ID", http.StatusBadRequest)
		return
	}
	if count >= int64(event.MaxPhotosPerGuest) {
		http.Error(w, "Photo limit reached", http.StatusTooManyRequests)
		return
	}
	if err := utils.VerifyPhotoRecaptcha(r.Context(), r.FormValue("recaptcha_token")); err != nil {
		http.Error(w, "Verification failed", http.StatusForbidden)
		return
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, "A photo is required", http.StatusBadRequest)
		return
	}
	defer file.Close()
	started := time.Now()
	result := "rejected"
	format := "unknown"
	defer func() {
		log.Printf("photo_upload result=%s format=%s duration_ms=%d", result, format, time.Since(started).Milliseconds())
	}()
	signature := make([]byte, 512)
	n, _ := file.Read(signature)
	detected, err := utils.DetectPhotoFormat(signature[:n])
	if err != nil {
		http.Error(w, "Only JPEG photos are supported", http.StatusUnsupportedMediaType)
		return
	}
	format = detected
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		http.Error(w, "Invalid photo", http.StatusUnprocessableEntity)
		return
	}
	jpeg, width, height, err := utils.ProcessPhoto(file, header.Size)
	if err != nil {
		http.Error(w, "Invalid photo", http.StatusUnprocessableEntity)
		return
	}
	key := fmt.Sprintf("events/%d/%d.jpg", event.ID, time.Now().UnixNano())
	if err := utils.UploadPrivatePhoto(r.Context(), key, bytes.NewReader(jpeg), "image/jpeg", false); err != nil {
		http.Error(w, "Storage unavailable", http.StatusServiceUnavailable)
		return
	}
	hash, _ := utils.HashPhotoGuest(event.ID, guestID)
	photo := models.EventPhoto{EventID: event.ID, ObjectKey: &key, Status: "pending", GuestHash: hash, ByteSize: int64(len(jpeg)), Width: width, Height: height, UploadedAt: time.Now()}
	if err := db.DB.Create(&photo).Error; err != nil {
		_ = utils.DeletePrivatePhoto(r.Context(), key, false)
		http.Error(w, "Could not register photo", http.StatusServiceUnavailable)
		return
	}
	result = "pending"
	var pending int64
	if db.DB.Model(&models.EventPhoto{}).Where("event_id = ? AND status = 'pending'", event.ID).Count(&pending).Error == nil {
		log.Printf("photo_moderation_queue pending=%d", pending)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": photo.ID, "status": "pending", "remaining": max(0, event.MaxPhotosPerGuest-int(count)-1)})
}

type galleryCursor struct {
	ModeratedAt time.Time `json:"t"`
	ID          uint64    `json:"i"`
}

func PhotoEventGallery(w http.ResponseWriter, r *http.Request) {
	event, ok := photoEventFromToken(w, r)
	if !ok {
		return
	}
	if !time.Now().Before(event.GalleryCloseAt) {
		http.Error(w, "Gallery is closed", http.StatusForbidden)
		return
	}
	limit := 48
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 48 {
			http.Error(w, "Invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	query := db.DB.Where("event_id = ? AND status = 'approved' AND object_key IS NOT NULL", event.ID)
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		var cursor galleryCursor
		if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.ID == 0 || cursor.ModeratedAt.IsZero() {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		query = query.Where("(moderated_at, id) < (?, ?)", cursor.ModeratedAt, cursor.ID)
	}
	var photos []models.EventPhoto
	if err := query.Order("moderated_at DESC, id DESC").Limit(limit + 1).Find(&photos).Error; err != nil {
		http.Error(w, "Gallery unavailable", http.StatusServiceUnavailable)
		return
	}
	hasMore := len(photos) > limit
	if hasMore {
		photos = photos[:limit]
	}
	items := make([]map[string]any, 0, len(photos))
	for _, photo := range photos {
		if photo.ObjectKey == nil {
			continue
		}
		url, err := utils.SignPrivatePhoto(r.Context(), *photo.ObjectKey, false, time.Hour)
		if err != nil {
			http.Error(w, "Gallery unavailable", http.StatusServiceUnavailable)
			return
		}
		items = append(items, map[string]any{"id": photo.ID, "url": url, "width": photo.Width, "height": photo.Height, "approved_at": photo.ModeratedAt})
	}
	var next string
	if hasMore && len(photos) > 0 {
		last := photos[len(photos)-1]
		if last.ModeratedAt != nil {
			raw, _ := json.Marshal(galleryCursor{*last.ModeratedAt, last.ID})
			next = base64.RawURLEncoding.EncodeToString(raw)
		}
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"items": items, "next_cursor": next})
}

// PhotoEventRefreshURLs renews links for photos already present in a guest's gallery.
// It does not discover newly approved photos; that remains a manual refresh.
func PhotoEventRefreshURLs(w http.ResponseWriter, r *http.Request) {
	event, ok := photoEventFromToken(w, r)
	if !ok {
		return
	}
	if !time.Now().Before(event.GalleryCloseAt) {
		http.Error(w, "Gallery is closed", http.StatusForbidden)
		return
	}
	var input struct {
		IDs []uint64 `json:"ids"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil || len(input.IDs) < 1 || len(input.IDs) > 100 {
		http.Error(w, "Invalid photo IDs", http.StatusBadRequest)
		return
	}
	var photos []models.EventPhoto
	if err := db.DB.Where("event_id = ? AND status = 'approved' AND id IN ? AND object_key IS NOT NULL", event.ID, input.IDs).Find(&photos).Error; err != nil {
		http.Error(w, "Gallery unavailable", http.StatusServiceUnavailable)
		return
	}
	urls := make(map[uint64]string, len(photos))
	for _, photo := range photos {
		if photo.ObjectKey == nil {
			continue
		}
		url, err := utils.SignPrivatePhoto(r.Context(), *photo.ObjectKey, false, time.Hour)
		if err != nil {
			http.Error(w, "Gallery unavailable", http.StatusServiceUnavailable)
			return
		}
		urls[photo.ID] = url
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"urls": urls})
}
