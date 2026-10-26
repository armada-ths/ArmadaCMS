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
	t.Setenv("PHOTO_APP_BASE_URL", "")
	w := httptest.NewRecorder()
	photoEventLink(w, models.PhotoEvent{ID: 3, TokenVersion: 2}, "en")
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	token := strings.TrimPrefix(body.URL, "https://photos.armada.nu/en/e/")
	id, version, err := utils.ParsePhotoEventToken(token)
	if err != nil || id != 3 || version != 2 {
		t.Fatalf("unexpected event token: id=%d version=%d err=%v", id, version, err)
	}
}

func TestPhotoEventLinkUsesStagingBaseURL(t *testing.T) {
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("s", 48))
	t.Setenv("PHOTO_APP_BASE_URL", "https://staging.armada.nu/photos/")
	w := httptest.NewRecorder()
	photoEventLink(w, models.PhotoEvent{ID: 3, TokenVersion: 2}, "en")
	if w.Code != http.StatusOK {
		t.Fatalf("unexpected response: %d %s", w.Code, w.Body.String())
	}
	var body struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	const prefix = "https://staging.armada.nu/photos/en/e/"
	if !strings.HasPrefix(body.URL, prefix) {
		t.Fatalf("unexpected staging URL: %q", body.URL)
	}
	id, version, err := utils.ParsePhotoEventToken(strings.TrimPrefix(body.URL, prefix))
	if err != nil || id != 3 || version != 2 {
		t.Fatalf("unexpected event token: id=%d version=%d err=%v", id, version, err)
	}
	if got := photoEventURL("test-token", "en"); got != prefix+"test-token" {
		t.Fatalf("QR URL source differs from event link: %q", got)
	}
}

func TestPhotoPreviewRefreshRejectsInvalidIDs(t *testing.T) {
	bodies := []string{`{}`, `{"event_id":1,"ids":[]}`, `{"event_id":0,"ids":[1]}`, `{"event_id":1,"ids":` + "[" + strings.Repeat("1,", 100) + "1]}"}
	for _, body := range bodies {
		response := httptest.NewRecorder()
		RefreshEventPhotoURLs(response, httptest.NewRequest(http.MethodPost, "/eventphotos/refresh", strings.NewReader(body)))
		if response.Code != http.StatusBadRequest {
			t.Fatalf("invalid request accepted: %d", response.Code)
		}
	}
}

func TestPhotoEventURLsCarrySelectedLocale(t *testing.T) {
	t.Setenv("PHOTO_APP_BASE_URL", "https://photos.armada.nu")
	for input, want := range map[string]string{"sv": "sv", "sv-SE": "sv", "en": "en", "fr": "en", "": "en"} {
		if got := photoEventURL("event.token", input); got != "https://photos.armada.nu/"+want+"/e/event.token" {
			t.Fatalf("locale %q: %s", input, got)
		}
	}
}
