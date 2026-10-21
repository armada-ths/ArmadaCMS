package main

import (
	controllers "ArmadaCMS/main/Controllers"
	"ArmadaCMS/main/auth"
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Run against a disposable local DB with the migrated public schema, never app data.
func TestPhotoRejectionRetentionIntegration(t *testing.T) {
	dsn := os.Getenv("PHOTO_RETENTION_TEST_DSN")
	if dsn == "" {
		t.Skip("set PHOTO_RETENTION_TEST_DSN to a local photo_retention_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Path != "/photo_retention_test" ||
		(parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "host.docker.internal") {
		t.Fatal("retention tests require the disposable local photo_retention_test database")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := db.DB
	db.DB = database
	t.Cleanup(func() { db.DB = previousDB; sqlDB, _ := database.DB(); _ = sqlDB.Close() })
	var deletes atomic.Int32
	var failStorage atomic.Bool
	storage := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected storage method %s", r.Method)
			w.WriteHeader(405)
			return
		}
		if failStorage.Load() {
			w.WriteHeader(403)
			return
		}
		deletes.Add(1)
		w.WriteHeader(204)
	}))
	defer storage.Close()
	t.Setenv("S3_ENDPOINT", storage.URL)
	t.Setenv("AWS_ACCESS_KEY_ID", "retention-test")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "retention-test")
	t.Setenv("S3_REGION", "local")
	t.Setenv("S3_BUCKET", "public-files")
	t.Setenv("PHOTO_S3_BUCKET", "retention-test-photos")

	user := models.User{Username: fmt.Sprintf("retention-%d", time.Now().UnixNano()), Password: "unused", Name: "Retention test"}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	event := models.PhotoEvent{Name: "Retention test", MaxPhotosPerGuest: 25, UploadsOpenAt: now.Add(-time.Hour), UploadsCloseAt: now.Add(time.Hour), GalleryCloseAt: now.Add(48 * time.Hour)}
	if err := database.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	makePhoto := func(status, key string) models.EventPhoto {
		photo := models.EventPhoto{EventID: event.ID, Status: status, ObjectKey: &key, GuestHash: "test-guest", ByteSize: 1, Width: 1, Height: 1, UploadedAt: now}
		if err := database.Create(&photo).Error; err != nil {
			t.Fatal(err)
		}
		return photo
	}
	moderate := func(photo models.EventPhoto, action string) int {
		request := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/v1/eventphotos/%d", photo.ID), strings.NewReader(fmt.Sprintf(`{"action":%q}`, action)))
		request = mux.SetURLVars(auth.WithUserID(request, int(user.ID)), map[string]string{"id": fmt.Sprint(photo.ID)})
		response := httptest.NewRecorder()
		controllers.ModerateEventPhoto(response, request)
		return response.Code
	}
	reload := func(photo *models.EventPhoto) {
		id := photo.ID
		*photo = models.EventPhoto{}
		if err := database.First(photo, id).Error; err != nil {
			t.Fatal(err)
		}
	}
	photo := makePhoto("approved", "retention/approved.jpg")
	if code := moderate(photo, "reject"); code != 200 {
		t.Fatalf("reject: HTTP %d", code)
	}
	reload(&photo)
	if photo.ObjectKey == nil || !photo.CanRestore(time.Now()) || deletes.Load() != 0 {
		t.Fatal("rejection deleted the image or prevented restore")
	}
	if err := cleanupRejectedPhotos(context.Background()); err != nil {
		t.Fatal(err)
	}
	if deletes.Load() != 0 {
		t.Fatal("cleanup deleted a photo before expiration")
	}
	if code := moderate(photo, "approve"); code != 200 {
		t.Fatalf("restore: HTTP %d", code)
	}
	reload(&photo)
	if photo.Status != "approved" || photo.RejectedDeleteAt != nil {
		t.Fatal("restore did not cancel scheduled deletion")
	}
	if code := moderate(photo, "approve"); code != 409 {
		t.Fatalf("reapprove: HTTP %d", code)
	}
	if code := moderate(photo, "reject"); code != 200 {
		t.Fatalf("reject again: HTTP %d", code)
	}
	expired := now.Add(-time.Second)
	if err := database.Model(&photo).Update("rejected_delete_at", expired).Error; err != nil {
		t.Fatal(err)
	}
	if code := moderate(photo, "approve"); code != 409 {
		t.Fatalf("expired restore: HTTP %d", code)
	}
	failStorage.Store(true)
	if err := cleanupRejectedPhotos(context.Background()); err == nil {
		t.Fatal("storage failure must be reported")
	}
	reload(&photo)
	if photo.ObjectKey == nil {
		t.Fatal("storage failure cleared the retryable object key")
	}
	failStorage.Store(false)
	var cleanupWorkers sync.WaitGroup
	cleanupResults := make(chan error, 2)
	for range 2 {
		cleanupWorkers.Add(1)
		go func() { defer cleanupWorkers.Done(); cleanupResults <- cleanupRejectedPhotos(context.Background()) }()
	}
	cleanupWorkers.Wait()
	close(cleanupResults)
	for err := range cleanupResults {
		if err != nil {
			t.Fatal(err)
		}
	}
	reload(&photo)
	if photo.ObjectKey != nil || deletes.Load() != 2 {
		t.Fatal("expired photo was not permanently deleted")
	}
	if code := moderate(photo, "approve"); code != 409 {
		t.Fatalf("deleted restore: HTTP %d", code)
	}
	if err := cleanupRejectedPhotos(context.Background()); err != nil {
		t.Fatal(err)
	}
	if deletes.Load() != 2 {
		t.Fatal("repeated cleanup was not idempotent")
	}
	var auditCount int64
	database.Model(&models.AuditLog{}).Where("resource_type = 'eventphotos' AND resource_id = ?", fmt.Sprint(photo.ID)).Count(&auditCount)
	if auditCount != 3 {
		t.Fatalf("expected three audited moderation changes, got %d", auditCount)
	}
}
