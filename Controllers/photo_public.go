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
	"gorm.io/gorm"
)

var errPhotoLimitReached = errors.New("photo limit reached")

func photoEventFromToken(w http.ResponseWriter, r *http.Request) (*models.PhotoEvent, bool) {
	id, version, err := utils.ParsePhotoEventToken(mux.Vars(r)["token"])
	if err != nil {
		writePhotoPublicError(w, "invalid_event_link", "Invalid event link", http.StatusNotFound)
		return nil, false
	}
	var event models.PhotoEvent
	if err := db.DB.First(&event, id).Error; err != nil || event.DeletionRequestedAt != nil || event.TokenVersion != version {
		writePhotoPublicError(w, "event_is_unavailable", "Event is unavailable", http.StatusNotFound)
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
	var recorded int64
	if err := db.DB.Model(&models.EventPhoto{}).Where("event_id = ? AND guest_hash = ?", event.ID, hash).Count(&recorded).Error; err != nil {
		return 0, err
	}
	var usage models.PhotoGuestUpload
	result := db.DB.Where("event_id = ? AND guest_hash = ?", event.ID, hash).Limit(1).Find(&usage)
	if result.Error != nil {
		return 0, result.Error
	}
	if result.RowsAffected == 0 {
		return recorded, nil
	}
	return max(recorded, usage.UploadCount), nil
}

// reservePhotoUpload atomically consumes one lifetime slot and registers the
// photo. A rejected or later deleted photo does not return its slot.
func reservePhotoUpload(event *models.PhotoEvent, photo *models.EventPhoto) (int64, error) {
	var count int64
	err := db.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Raw(`
			insert into public.photo_guest_uploads as quota (event_id, guest_hash, upload_count)
			select ?, ?, count(*) + 1
			from public.event_photos
			where event_id = ? and guest_hash = ?
			having count(*) < ?
			on conflict (event_id, guest_hash) do update
			set upload_count = quota.upload_count + 1
			where quota.upload_count < ?
			returning upload_count`,
			event.ID, photo.GuestHash, event.ID, photo.GuestHash,
			event.MaxPhotosPerGuest, event.MaxPhotosPerGuest,
		).Scan(&count)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errPhotoLimitReached
		}
		return tx.Create(photo).Error
	})
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
			writePhotoPublicError(w, "invalid_guest_id", "Invalid guest ID", http.StatusBadRequest)
			return
		}
		remaining = max(0, remaining-int(count))
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"name": event.Name, "description": event.Description,
		"nameSv": event.NameSv, "descriptionSv": event.DescriptionSv,
		"uploads_open_at": event.UploadsOpenAt, "uploads_close_at": event.UploadsCloseAt,
		"gallery_close_at": event.GalleryCloseAt,
		"uploads_open":     !now.Before(event.UploadsOpenAt) && now.Before(event.UploadsCloseAt),
		"gallery_open":     now.Before(event.GalleryCloseAt), "remaining": remaining,
	})
}

func PhotoEventUpload(w http.ResponseWriter, r *http.Request) {
	event, ok := photoEventFromToken(w, r)
	if !ok {
		return
	}
	now := time.Now()
	if now.Before(event.UploadsOpenAt) || !now.Before(event.UploadsCloseAt) {
		writePhotoPublicError(w, "uploads_are_closed", "Uploads are closed", http.StatusForbidden)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, utils.MaxPhotoOriginalBytes+1024*1024)
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		writePhotoPublicError(w, "upload_too_large", "Upload too large", http.StatusRequestEntityTooLarge)
		return
	}
	if r.FormValue("privacy_confirmed") != "true" {
		writePhotoPublicError(w, "privacy_confirmation_required", "Privacy confirmation required", http.StatusBadRequest)
		return
	}
	guestID := r.FormValue("guest_id")
	count, err := photoGuestQuota(event, guestID)
	if err != nil {
		writePhotoPublicError(w, "invalid_guest_id", "Invalid guest ID", http.StatusBadRequest)
		return
	}
	if count >= int64(event.MaxPhotosPerGuest) {
		writePhotoPublicError(w, "photo_limit_reached", "Photo limit reached", http.StatusTooManyRequests)
		return
	}
	if err := utils.VerifyPhotoRecaptcha(r.Context(), r.FormValue("recaptcha_token")); err != nil {
		writePhotoPublicError(w, "verification_failed", "Verification failed", http.StatusForbidden)
		return
	}
	file, header, err := r.FormFile("photo")
	if err != nil {
		writePhotoPublicError(w, "a_photo_is_required", "A photo is required", http.StatusBadRequest)
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
		writePhotoPublicError(w, "unsupported_photo_format", "Unsupported photo format", http.StatusUnsupportedMediaType)
		return
	}
	format = detected
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		writePhotoPublicError(w, "invalid_photo", "Invalid photo", http.StatusUnprocessableEntity)
		return
	}
	jpeg, width, height, err := utils.ProcessPhoto(file, header.Size)
	if err != nil {
		writePhotoPublicError(w, "invalid_photo", "Invalid photo", http.StatusUnprocessableEntity)
		return
	}
	key := fmt.Sprintf("events/%d/%d.jpg", event.ID, time.Now().UnixNano())
	if err := utils.UploadPrivatePhoto(r.Context(), key, bytes.NewReader(jpeg), "image/jpeg", false); err != nil {
		writePhotoPublicError(w, "storage_unavailable", "Storage unavailable", http.StatusServiceUnavailable)
		return
	}
	var thumbnailKey *string
	if thumbnail, err := utils.StorePhotoThumbnail(r.Context(), key, jpeg); err != nil {
		log.Printf("photo_thumbnail result=failed event_id=%d: %v", event.ID, err)
	} else {
		thumbnailKey = &thumbnail
	}
	cleanup := func() {
		_ = utils.DeletePhotoVariants(r.Context(), models.EventPhoto{ObjectKey: &key, ThumbnailObjectKey: thumbnailKey})
	}
	hash, err := utils.HashPhotoGuest(event.ID, guestID)
	if err != nil {
		cleanup()
		writePhotoPublicError(w, "could_not_register_photo", "Could not register photo", http.StatusServiceUnavailable)
		return
	}
	photo := models.EventPhoto{EventID: event.ID, ObjectKey: &key, ThumbnailObjectKey: thumbnailKey, Status: "pending", AIReviewStatus: "not_scanned", GuestHash: hash, ByteSize: int64(len(jpeg)), Width: width, Height: height, UploadedAt: time.Now()}
	registeredCount, err := reservePhotoUpload(event, &photo)
	if err != nil {
		cleanup()
		if errors.Is(err, errPhotoLimitReached) {
			writePhotoPublicError(w, "photo_limit_reached", "Photo limit reached", http.StatusTooManyRequests)
			return
		}
		writePhotoPublicError(w, "could_not_register_photo", "Could not register photo", http.StatusServiceUnavailable)
		return
	}
	photoStatus := "pending"
	if event.AutoApproveSafePhotos {
		photoStatus = assessUploadedPhoto(r.Context(), photo, jpeg)
	}
	result = photoStatus
	var pending int64
	if db.DB.Model(&models.EventPhoto{}).Where("event_id = ? AND status = 'pending'", event.ID).Count(&pending).Error == nil {
		log.Printf("photo_moderation_queue pending=%d", pending)
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": photo.ID, "status": photoStatus, "remaining": max(0, event.MaxPhotosPerGuest-int(registeredCount))})
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
		writePhotoPublicError(w, "gallery_is_closed", "Gallery is closed", http.StatusForbidden)
		return
	}
	limit := 48
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 48 {
			writePhotoPublicError(w, "invalid_limit", "Invalid limit", http.StatusBadRequest)
			return
		}
		limit = parsed
	}
	query := db.DB.Where("event_id = ? AND status = 'approved' AND object_key IS NOT NULL", event.ID)
	if raw := r.URL.Query().Get("cursor"); raw != "" {
		decoded, err := base64.RawURLEncoding.DecodeString(raw)
		var cursor galleryCursor
		if err != nil || json.Unmarshal(decoded, &cursor) != nil || cursor.ID == 0 || cursor.ModeratedAt.IsZero() {
			writePhotoPublicError(w, "invalid_cursor", "Invalid cursor", http.StatusBadRequest)
			return
		}
		query = query.Where("(moderated_at, id) < (?, ?)", cursor.ModeratedAt, cursor.ID)
	}
	var photos []models.EventPhoto
	if err := query.Order("moderated_at DESC, id DESC").Limit(limit + 1).Find(&photos).Error; err != nil {
		writePhotoPublicError(w, "gallery_unavailable", "Gallery unavailable", http.StatusServiceUnavailable)
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
			writePhotoPublicError(w, "gallery_unavailable", "Gallery unavailable", http.StatusServiceUnavailable)
			return
		}
		thumbnail, err := utils.SignPhotoThumbnail(r.Context(), photo, time.Hour)
		if err != nil {
			writePhotoPublicError(w, "gallery_unavailable", "Gallery unavailable", http.StatusServiceUnavailable)
			return
		}
		items = append(items, map[string]any{"id": photo.ID, "url": url, "thumbnail_url": thumbnail, "urls_expires_at": time.Now().Add(time.Hour), "width": photo.Width, "height": photo.Height, "approved_at": photo.ModeratedAt})
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
		writePhotoPublicError(w, "gallery_is_closed", "Gallery is closed", http.StatusForbidden)
		return
	}
	var input struct {
		IDs []uint64 `json:"ids"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&input) != nil || len(input.IDs) < 1 || len(input.IDs) > 100 {
		writePhotoPublicError(w, "invalid_photo_ids", "Invalid photo IDs", http.StatusBadRequest)
		return
	}
	var photos []models.EventPhoto
	if err := db.DB.Where("event_id = ? AND status = 'approved' AND id IN ? AND object_key IS NOT NULL", event.ID, input.IDs).Find(&photos).Error; err != nil {
		writePhotoPublicError(w, "gallery_unavailable", "Gallery unavailable", http.StatusServiceUnavailable)
		return
	}
	urls := make(map[uint64]string, len(photos))
	thumbnails := make(map[uint64]string, len(photos))
	for _, photo := range photos {
		if photo.ObjectKey == nil {
			continue
		}
		url, err := utils.SignPrivatePhoto(r.Context(), *photo.ObjectKey, false, time.Hour)
		if err != nil {
			writePhotoPublicError(w, "gallery_unavailable", "Gallery unavailable", http.StatusServiceUnavailable)
			return
		}
		urls[photo.ID] = url
		thumbnail, err := utils.SignPhotoThumbnail(r.Context(), photo, time.Hour)
		if err != nil {
			writePhotoPublicError(w, "gallery_unavailable", "Gallery unavailable", http.StatusServiceUnavailable)
			return
		}
		thumbnails[photo.ID] = thumbnail
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{"urls": urls, "thumbnail_urls": thumbnails, "urls_expires_at": time.Now().Add(time.Hour)})
}
