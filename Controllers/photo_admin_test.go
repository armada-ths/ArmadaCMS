package controllers

import (
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPhotoEventValidatesTimes(t *testing.T) {
	now := time.Now()
	event := models.PhotoEvent{
		Name: "Banquet", MaxPhotosPerGuest: 25,
		UploadsOpenAt: now, UploadsCloseAt: now.Add(time.Hour),
		GalleryCloseAt: now.Add(24 * time.Hour),
	}
	if err := validatePhotoEvent(event); err != nil {
		t.Fatal(err)
	}
	event.UploadsCloseAt = event.UploadsOpenAt
	if err := validatePhotoEvent(event); err == nil || err.Error() != "uploads must close after they open" {
		t.Fatalf("expected upload window error, got %v", err)
	}
	event.UploadsCloseAt = now.Add(time.Hour)
	event.GalleryCloseAt = now.Add(30 * time.Minute)
	if err := validatePhotoEvent(event); err == nil || err.Error() != "gallery must close no earlier than uploads close" {
		t.Fatalf("expected gallery window error, got %v", err)
	}
}

func TestPhotoEventBadRequestIsReadableByReactAdmin(t *testing.T) {
	w := httptest.NewRecorder()
	photoEventBadRequest(w, "uploads must close after they open")
	if w.Code != http.StatusBadRequest || !strings.Contains(w.Body.String(), `"message":"Uploads must close after they open"`) {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
}

func TestPhotoEventLinkUsesConfiguredTokenSecret(t *testing.T) {
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("s", 48))
	w := httptest.NewRecorder()
	photoEventLink(w, models.PhotoEvent{ID: 3, TokenVersion: 2})
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(body.URL, "https://photos.armada.nu/e/")
	id, version, err := utils.ParsePhotoEventToken(token)
	if err != nil || id != 3 || version != 2 {
		t.Fatalf("unexpected event token: id=%d version=%d err=%v", id, version, err)
	}
}
