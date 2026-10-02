package controllers

import (
	"ArmadaCMS/main/db"
	"ArmadaCMS/main/models"
	"ArmadaCMS/main/utils"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// This integration test uses a disposable local database. It is skipped in
// ordinary unit-test runs unless PHOTO_QUOTA_TEST_DSN is explicitly provided.
func TestPhotoGuestQuotaLifetimeAndConcurrentUploads(t *testing.T) {
	dsn := os.Getenv("PHOTO_QUOTA_TEST_DSN")
	if dsn == "" {
		t.Skip("set PHOTO_QUOTA_TEST_DSN to a disposable local photo_quota_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Path != "/photo_quota_test" ||
		(parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "host.docker.internal") {
		t.Fatal("PHOTO_QUOTA_TEST_DSN must target a local photo_quota_test database")
	}
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("q", 48))

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	previousDB := db.DB
	db.DB = database
	t.Cleanup(func() { db.DB = previousDB })
	if err := database.AutoMigrate(&models.PhotoEvent{}, &models.EventPhoto{}, &models.PhotoGuestUpload{}); err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	event := models.PhotoEvent{Name: "Quota test", MaxPhotosPerGuest: 2, UploadsOpenAt: now, UploadsCloseAt: now.Add(time.Hour), GalleryCloseAt: now.Add(2 * time.Hour)}
	if err := database.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	guestID := "quota-test-guest-123"
	hash, err := utils.HashPhotoGuest(event.ID, guestID)
	if err != nil {
		t.Fatal(err)
	}
	// An existing rejected photo from before the counter migration still uses a slot.
	rejected := models.EventPhoto{EventID: event.ID, GuestHash: hash, Status: "rejected", ByteSize: 1, Width: 1, Height: 1, UploadedAt: now}
	if err := database.Create(&rejected).Error; err != nil {
		t.Fatal(err)
	}
	if count, err := photoGuestQuota(&event, guestID); err != nil || count != 1 {
		t.Fatalf("rejected photo should count: count=%d err=%v", count, err)
	}
	second := models.EventPhoto{EventID: event.ID, GuestHash: hash, Status: "pending", ByteSize: 1, Width: 1, Height: 1, UploadedAt: now}
	if count, err := reservePhotoUpload(&event, &second); err != nil || count != 2 {
		t.Fatalf("second upload should fill quota: count=%d err=%v", count, err)
	}
	if err := database.Delete(&rejected).Error; err != nil {
		t.Fatal(err)
	}
	if count, err := photoGuestQuota(&event, guestID); err != nil || count != 2 {
		t.Fatalf("permanent deletion must not return a slot: count=%d err=%v", count, err)
	}
	third := models.EventPhoto{EventID: event.ID, GuestHash: hash, Status: "pending", ByteSize: 1, Width: 1, Height: 1, UploadedAt: now}
	if _, err := reservePhotoUpload(&event, &third); !errors.Is(err, errPhotoLimitReached) {
		t.Fatalf("third upload should be rejected: %v", err)
	}

	concurrentEvent := models.PhotoEvent{Name: "Concurrent quota test", MaxPhotosPerGuest: 1, UploadsOpenAt: now, UploadsCloseAt: now.Add(time.Hour), GalleryCloseAt: now.Add(2 * time.Hour)}
	if err := database.Create(&concurrentEvent).Error; err != nil {
		t.Fatal(err)
	}
	concurrentHash, err := utils.HashPhotoGuest(concurrentEvent.ID, guestID)
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			photo := models.EventPhoto{EventID: concurrentEvent.ID, GuestHash: concurrentHash, Status: "pending", ByteSize: 1, Width: 1, Height: 1, UploadedAt: now}
			_, err := reservePhotoUpload(&concurrentEvent, &photo)
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	succeeded, limited := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, errPhotoLimitReached):
			limited++
		default:
			t.Fatalf("unexpected concurrent upload error: %v", err)
		}
	}
	if succeeded != 1 || limited != 1 {
		t.Fatalf("concurrent cap failed: succeeded=%d limited=%d", succeeded, limited)
	}
}
