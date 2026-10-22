package main

import (
	controllers "ArmadaCMS/main/Controllers"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestPhotoThumbnailIntegration(t *testing.T) {
	dsn := os.Getenv("PHOTO_RETENTION_TEST_DSN")
	if dsn == "" {
		t.Skip("requires disposable photo_retention_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Path != "/photo_retention_test" || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "host.docker.internal") {
		t.Fatal("requires disposable local test database")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previous := db.DB
	db.DB = database
	t.Cleanup(func() { db.DB = previous; sqlDB, _ := database.DB(); _ = sqlDB.Close() })
	var source bytes.Buffer
	_ = jpeg.Encode(&source, image.NewRGBA(image.Rect(0, 0, 1200, 800)), &jpeg.Options{Quality: 88})
	var mu sync.Mutex
	objects := map[string][]byte{}
	blockStarted, release := make(chan struct{}), make(chan struct{})
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" && strings.Contains(r.URL.Path, "concurrent.jpg.thumb") {
			close(blockStarted)
			<-release
		}
		mu.Lock()
		defer mu.Unlock()
		switch r.Method {
		case "GET":
			if strings.Contains(r.URL.Path, "broken.jpg") {
				w.WriteHeader(403)
				return
			}
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(source.Bytes())
		case "PUT":
			data, _ := io.ReadAll(r.Body)
			objects[r.URL.Path] = data
			w.WriteHeader(200)
		case "DELETE":
			delete(objects, r.URL.Path)
			w.WriteHeader(204)
		default:
			w.WriteHeader(405)
		}
	}))
	defer storage.Close()
	t.Setenv("S3_ENDPOINT", storage.URL)
	t.Setenv("PHOTO_S3_PRESIGN_ENDPOINT", storage.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "thumbnail-test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "thumbnail-test")
	t.Setenv("S3_REGION", "local")
	t.Setenv("S3_BUCKET", "public-files")
	t.Setenv("PHOTO_S3_BUCKET", "test-photos")
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("s", 48))
	now := time.Now().UTC()
	makeEvent := func() models.PhotoEvent {
		event := models.PhotoEvent{Name: "Thumbnail test", MaxPhotosPerGuest: 25, UploadsOpenAt: now.Add(-time.Hour), UploadsCloseAt: now.Add(time.Hour), GalleryCloseAt: now.Add(48 * time.Hour)}
		if err := database.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
		return event
	}
	event, other := makeEvent(), makeEvent()
	t.Cleanup(func() {
		database.Where("event_id IN ?", []uint64{event.ID, other.ID}).Delete(&models.EventPhoto{})
		database.Where("id IN ?", []uint64{event.ID, other.ID}).Delete(&models.PhotoEvent{})
	})
	makePhoto := func(eventID uint64, status, key string) models.EventPhoto {
		photo := models.EventPhoto{EventID: eventID, Status: status, ObjectKey: &key, GuestHash: "test", ByteSize: int64(source.Len()), Width: 1200, Height: 800, UploadedAt: now, ModeratedAt: &now}
		if status == "rejected" {
			deadline := now.Add(time.Hour)
			photo.RejectedDeleteAt = &deadline
		}
		if err := database.Create(&photo).Error; err != nil {
			t.Fatal(err)
		}
		return photo
	}
	approved := makePhoto(event.ID, "approved", "approved.jpg")
	pending := makePhoto(event.ID, "pending", "pending.jpg")
	rejected := makePhoto(event.ID, "rejected", "rejected.jpg")
	_ = makePhoto(event.ID, "pending", "broken.jpg")
	foreign := makePhoto(other.ID, "approved", "foreign.jpg")
	if err := backfillPhotoThumbnails(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, photo := range []*models.EventPhoto{&approved, &pending, &rejected, &foreign} {
		if err := database.First(photo, photo.ID).Error; err != nil || photo.ThumbnailObjectKey == nil {
			t.Fatalf("backfill failed: %v %+v", err, photo)
		}
	}
	mu.Lock()
	count := len(objects)
	mu.Unlock()
	if count != 4 {
		t.Fatalf("expected 4 derivatives, got %d", count)
	}
	if err := backfillPhotoThumbnails(context.Background()); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	count = len(objects)
	mu.Unlock()
	if count != 4 {
		t.Fatal("backfill not idempotent")
	}

	token, err := utils.PhotoEventToken(event.ID, event.TokenVersion)
	if err != nil {
		t.Fatal(err)
	}
	request := mux.SetURLVars(httptest.NewRequest("GET", "/gallery", nil), map[string]string{"token": token})
	response := httptest.NewRecorder()
	controllers.PhotoEventGallery(response, request)
	if response.Code != 200 {
		t.Fatalf("gallery %d: %s", response.Code, response.Body.String())
	}
	var gallery struct {
		Items []struct {
			URL       string    `json:"url"`
			Thumbnail string    `json:"thumbnail_url"`
			Expires   time.Time `json:"urls_expires_at"`
		}
	}
	if err := json.Unmarshal(response.Body.Bytes(), &gallery); err != nil {
		t.Fatal(err)
	}
	if len(gallery.Items) != 1 || !strings.Contains(gallery.Items[0].Thumbnail, ".thumb-v1.jpg") || strings.Contains(gallery.Items[0].URL, ".thumb-v1.jpg") || gallery.Items[0].Expires.Before(now) {
		t.Fatalf("bad gallery %s", response.Body.String())
	}
	body := fmt.Sprintf(`{"event_id":%d,"ids":[%d,%d]}`, event.ID, approved.ID, foreign.ID)
	request = httptest.NewRequest("POST", "/eventphotos/refresh", strings.NewReader(body))
	response = httptest.NewRecorder()
	controllers.RefreshEventPhotoURLs(response, request)
	var links struct {
		URLs map[string]string `json:"thumbnail_urls"`
	}
	_ = json.Unmarshal(response.Body.Bytes(), &links)
	if response.Code != 200 || len(links.URLs) != 1 || links.URLs[fmt.Sprint(approved.ID)] == "" {
		t.Fatalf("event scope: %s", response.Body.String())
	}
	response = httptest.NewRecorder()
	auth.RequirePermission("eventphotos.view", controllers.RefreshEventPhotoURLs)(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatal("missing permission accepted")
	}

	// Backfill must finish under its row lock before permanent deletion proceeds.
	concurrent := makePhoto(event.ID, "pending", "concurrent.jpg")
	user := models.User{Username: fmt.Sprintf("thumb-%d", now.UnixNano()), Password: "unused", Name: "Test"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		database.Where("actor_user_id = ?", user.ID).Delete(&models.AuditLog{})
		database.Delete(&user)
	})
	backfilled := make(chan error, 1)
	go func() { backfilled <- backfillPhotoThumbnails(context.Background()) }()
	select {
	case <-blockStarted:
	case <-time.After(5 * time.Second):
		t.Fatal("backfill did not upload")
	}
	deleted := make(chan int, 1)
	go func() {
		r := auth.WithUserID(mux.SetURLVars(httptest.NewRequest("POST", "/moderate", strings.NewReader(`{"action":"delete"}`)), map[string]string{"id": fmt.Sprint(concurrent.ID)}), int(user.ID))
		w := httptest.NewRecorder()
		controllers.ModerateEventPhoto(w, r)
		deleted <- w.Code
	}()
	select {
	case <-deleted:
		t.Fatal("deletion bypassed thumbnail row lock")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	if err := <-backfilled; err != nil {
		t.Fatal(err)
	}
	if code := <-deleted; code != http.StatusNoContent {
		t.Fatalf("delete HTTP %d", code)
	}
	mu.Lock()
	_, exists := objects["/test-photos/concurrent.jpg.thumb-v1.jpg"]
	mu.Unlock()
	if exists {
		t.Fatal("thumbnail survived concurrent deletion")
	}

	// Retention removes derivatives as well as the full-size object.
	database.Model(&rejected).Update("rejected_delete_at", now.Add(-time.Hour))
	if err := cleanupRejectedPhotos(context.Background()); err != nil {
		t.Fatal(err)
	}
	rejected = models.EventPhoto{ID: rejected.ID}
	if err := database.First(&rejected, rejected.ID).Error; err != nil {
		t.Fatal(err)
	}
	if rejected.ObjectKey != nil || rejected.ThumbnailObjectKey != nil {
		t.Fatal("retention left object references")
	}
}
