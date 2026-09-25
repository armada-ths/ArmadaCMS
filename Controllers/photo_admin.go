package controllers

import (
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/skip2/go-qrcode"
	"gorm.io/gorm"
)

var photoSlug = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validatePhotoEvent(event models.PhotoEvent) error {
	if event.Name == "" || !photoSlug.MatchString(event.Slug) || event.MaxPhotosPerGuest < 1 || event.MaxPhotosPerGuest > 25 ||
		!event.UploadsOpenAt.Before(event.UploadsCloseAt) || event.UploadsCloseAt.After(event.GalleryCloseAt) || !event.GalleryCloseAt.Before(event.DeleteAfter) {
		return fmt.Errorf("invalid event fields")
	}
	if event.Active {
		privacy, err := url.Parse(event.PrivacyURL)
		if err != nil || privacy.Scheme != "https" || privacy.Host == "" {
			return fmt.Errorf("an HTTPS privacy URL is required")
		}
	}
	return nil
}

func photoAdminJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func ListPhotoEvents(w http.ResponseWriter, r *http.Request) {
	var events []models.PhotoEvent
	if err := db.DB.Order("created_at DESC").Find(&events).Error; err != nil {
		http.Error(w, "List failed", 500)
		return
	}
	w.Header().Set("Content-Range", fmt.Sprintf("photoevents 0-%d/%d", max(0, len(events)-1), len(events)))
	photoAdminJSON(w, events)
}

func GetPhotoEvent(w http.ResponseWriter, r *http.Request) {
	var event models.PhotoEvent
	if db.DB.First(&event, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	photoAdminJSON(w, event)
}

func CreatePhotoEvent(w http.ResponseWriter, r *http.Request) {
	var event models.PhotoEvent
	if json.NewDecoder(r.Body).Decode(&event) != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	event.ID = 0
	event.TokenVersion = 1
	if event.MaxPhotosPerGuest == 0 {
		event.MaxPhotosPerGuest = 25
	}
	if err := validatePhotoEvent(event); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := createWithAudit(r, "photoevents", &event, func(tx *gorm.DB) error { return tx.Create(&event).Error }, nil); err != nil {
		http.Error(w, "Create failed", 500)
		return
	}
	w.WriteHeader(http.StatusCreated)
	photoAdminJSON(w, event)
}

func UpdatePhotoEvent(w http.ResponseWriter, r *http.Request) {
	var before models.PhotoEvent
	if db.DB.First(&before, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	var input models.PhotoEvent
	if json.NewDecoder(r.Body).Decode(&input) != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	input.ID, input.TokenVersion, input.CreatedAt = before.ID, before.TokenVersion, before.CreatedAt
	if err := validatePhotoEvent(input); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	if err := updateWithAudit(r, "photoevents", before.ID, before, &input, func(tx *gorm.DB) error {
		return tx.Model(&before).Updates(map[string]any{"name": input.Name, "slug": input.Slug, "description": input.Description, "uploads_open_at": input.UploadsOpenAt, "uploads_close_at": input.UploadsCloseAt, "gallery_close_at": input.GalleryCloseAt, "delete_after": input.DeleteAfter, "active": input.Active, "privacy_url": input.PrivacyURL, "max_photos_per_guest": input.MaxPhotosPerGuest}).Error
	}, func(tx *gorm.DB) error { return tx.First(&input, before.ID).Error }); err != nil {
		http.Error(w, "Update failed", 500)
		return
	}
	photoAdminJSON(w, input)
}

func RotatePhotoEventToken(w http.ResponseWriter, r *http.Request) {
	var before models.PhotoEvent
	if db.DB.First(&before, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	after := before
	after.TokenVersion++
	if err := updateWithAudit(r, "photoevents", before.ID, before, &after, func(tx *gorm.DB) error { return tx.Model(&before).Update("token_version", after.TokenVersion).Error }, nil); err != nil {
		http.Error(w, "Rotation failed", 500)
		return
	}
	photoEventLink(w, after)
}

func PhotoEventLink(w http.ResponseWriter, r *http.Request) {
	var event models.PhotoEvent
	if db.DB.First(&event, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	photoEventLink(w, event)
}

func photoEventLink(w http.ResponseWriter, event models.PhotoEvent) {
	token, err := utils.PhotoEventToken(event.ID, event.TokenVersion)
	if err != nil {
		http.Error(w, "Link configuration missing", 500)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	photoAdminJSON(w, map[string]string{"url": "https://photos.armada.nu/e/" + token})
}

func PhotoEventQR(w http.ResponseWriter, r *http.Request) {
	var event models.PhotoEvent
	if db.DB.First(&event, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	token, err := utils.PhotoEventToken(event.ID, event.TokenVersion)
	if err != nil {
		http.Error(w, "QR configuration missing", 500)
		return
	}
	link := "https://photos.armada.nu/e/" + token
	qr, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		http.Error(w, "QR generation failed", 500)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if r.URL.Query().Get("format") == "svg" {
		bitmap := qr.Bitmap()
		var svg strings.Builder
		size := len(bitmap) + 8
		fmt.Fprintf(&svg, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" shape-rendering="crispEdges"><rect width="100%%" height="100%%" fill="white"/>`, size, size)
		for y, row := range bitmap {
			for x, on := range row {
				if on {
					fmt.Fprintf(&svg, `<rect x="%d" y="%d" width="1" height="1" fill="black"/>`, x+4, y+4)
				}
			}
		}
		svg.WriteString("</svg>")
		w.Header().Set("Content-Type", "image/svg+xml")
		w.Header().Set("Content-Disposition", `attachment; filename="armada-event-qr.svg"`)
		_, _ = w.Write([]byte(svg.String()))
		return
	}
	png, err := qr.PNG(768)
	if err != nil {
		http.Error(w, "QR generation failed", 500)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", `attachment; filename="armada-event-qr.png"`)
	_, _ = w.Write(png)
}

func ListEventPhotos(w http.ResponseWriter, r *http.Request) {
	eventID, err := strconv.ParseUint(r.URL.Query().Get("event_id"), 10, 64)
	if err != nil {
		http.Error(w, "event_id required", 400)
		return
	}
	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}
	if status != "pending" && status != "approved" && status != "rejected" {
		http.Error(w, "Invalid status", 400)
		return
	}
	page := 0
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 0 || parsed > 14 {
			http.Error(w, "Invalid page", 400)
			return
		}
		page = parsed
	}
	var photos []models.EventPhoto
	if err := db.DB.Where("event_id = ? AND status = ?", eventID, status).Order("uploaded_at DESC, id DESC").Limit(200).Offset(page * 200).Find(&photos).Error; err != nil {
		http.Error(w, "List failed", 500)
		return
	}
	items := make([]map[string]any, 0, len(photos))
	for _, photo := range photos {
		item := map[string]any{"id": photo.ID, "event_id": photo.EventID, "status": photo.Status, "uploaded_at": photo.UploadedAt, "width": photo.Width, "height": photo.Height}
		if photo.ObjectKey != nil {
			signed, err := utils.SignPrivatePhoto(r.Context(), *photo.ObjectKey, false, 15*time.Minute)
			if err != nil {
				http.Error(w, "Image unavailable", http.StatusServiceUnavailable)
				return
			}
			item["thumbnail_url"] = signed
		}
		items = append(items, item)
	}
	photoAdminJSON(w, items)
}

func ModerateEventPhoto(w http.ResponseWriter, r *http.Request) {
	var photo models.EventPhoto
	if db.DB.First(&photo, mux.Vars(r)["id"]).Error != nil {
		http.NotFound(w, r)
		return
	}
	var input struct {
		Action string `json:"action"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil || (input.Action != "approve" && input.Action != "reject" && input.Action != "delete") {
		http.Error(w, "Invalid action", 400)
		return
	}
	if input.Action == "approve" && (photo.Status != "pending" || photo.ObjectKey == nil) {
		http.Error(w, "Cannot approve", http.StatusConflict)
		return
	}
	if input.Action != "approve" && photo.ObjectKey != nil {
		if err := utils.DeletePrivatePhoto(r.Context(), *photo.ObjectKey, false); err != nil {
			http.Error(w, "Object delete failed", http.StatusServiceUnavailable)
			return
		}
	}
	if input.Action == "delete" {
		writeDeleteResponseWithAudit[models.EventPhoto](w, r, "eventphotos", mux.Vars(r)["id"], "Photo not found", nil, nil)
		return
	}
	before := photo
	now := time.Now()
	userID, _ := auth.GetUserIDFromContext(r)
	moderator := uint(userID)
	photo.ModeratedAt, photo.ModeratedBy = &now, &moderator
	if input.Action == "approve" {
		photo.Status = "approved"
	} else {
		photo.Status = "rejected"
		photo.ObjectKey = nil
	}
	if err := updateWithAudit(r, "eventphotos", photo.ID, before, &photo, func(tx *gorm.DB) error {
		return tx.Model(&models.EventPhoto{}).Where("id = ?", photo.ID).Updates(map[string]any{"status": photo.Status, "object_key": photo.ObjectKey, "moderated_at": now, "moderated_by": moderator}).Error
	}, nil); err != nil {
		http.Error(w, "Moderation failed", 500)
		return
	}
	photoAdminJSON(w, photo)
}

func ModerateEventPhotosBatch(w http.ResponseWriter, r *http.Request) {
	var input struct {
		IDs    []uint64 `json:"ids"`
		Action string   `json:"action"`
	}
	if json.NewDecoder(r.Body).Decode(&input) != nil || len(input.IDs) < 1 || len(input.IDs) > 100 || (input.Action != "approve" && input.Action != "reject" && input.Action != "delete") {
		http.Error(w, "Invalid batch", 400)
		return
	}
	succeeded, failed := 0, 0
	for _, id := range input.IDs {
		payload, _ := json.Marshal(map[string]string{"action": input.Action})
		child := mux.SetURLVars(r.Clone(r.Context()), map[string]string{"id": fmt.Sprint(id)})
		child.Body = io.NopCloser(bytes.NewReader(payload))
		recorder := httptest.NewRecorder()
		ModerateEventPhoto(recorder, child)
		if recorder.Code >= 200 && recorder.Code < 300 {
			succeeded++
		} else {
			failed++
		}
	}
	photoAdminJSON(w, map[string]int{"succeeded": succeeded, "failed": failed})
}

func DeletePhotoEvent(w http.ResponseWriter, r *http.Request) {
	// Retention cleanup owns object deletion; prevent unsafe cascade while objects still exist.
	var count int64
	db.DB.Model(&models.EventPhoto{}).Where("event_id = ?", mux.Vars(r)["id"]).Count(&count)
	var exportCount int64
	db.DB.Model(&models.PhotoExport{}).Where("event_id = ?", mux.Vars(r)["id"]).Count(&exportCount)
	if count > 0 || exportCount > 0 {
		http.Error(w, "Delete event photos and exports first", http.StatusConflict)
		return
	}
	writeDeleteResponseWithAudit[models.PhotoEvent](w, r, "photoevents", mux.Vars(r)["id"], "Event not found", nil, nil)
}
