package controllers

import (
	"ArmadaCMS/main/models"
	"context"
	"errors"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net/url"
	"os"
	"testing"
	"time"
)

func TestPhotoWorkerDispatchQueueCheck(t *testing.T) {
	dsn := os.Getenv("PHOTO_DISPATCH_TEST_DSN")
	if dsn == "" {
		t.Skip("set PHOTO_DISPATCH_TEST_DSN to a disposable local photo_dispatch_test database")
	}
	parsed, err := url.Parse(dsn)
	if err != nil || parsed.Path != "/photo_dispatch_test" || (parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1") {
		t.Fatal("requires local photo_dispatch_test")
	}
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	pool, _ := database.DB()
	t.Cleanup(func() { _ = pool.Close() })
	if err = database.AutoMigrate(&models.PhotoEvent{}, &models.PhotoExport{}, &models.EventPhoto{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 2, 10, 0, 0, 0, time.UTC)
	past, future, stale := now.Add(-time.Minute), now.Add(time.Minute), now.Add(-3*time.Hour)
	original, thumbnail := "private-original.jpg", "private-thumbnail.jpg"
	var event models.PhotoEvent
	createExport := func(status string, start, expires *time.Time) {
		if err := database.Create(&models.PhotoExport{EventID: event.ID, Status: status, StartedAt: start, ExpiresAt: expires, CreatedAt: past}).Error; err != nil {
			t.Fatal(err)
		}
	}
	createPhoto := func(status string, key, thumb *string, deleteAt *time.Time) {
		if err := database.Create(&models.EventPhoto{EventID: event.ID, Status: status, ObjectKey: key, ThumbnailObjectKey: thumb, RejectedDeleteAt: deleteAt, UploadedAt: past}).Error; err != nil {
			t.Fatal(err)
		}
	}
	deleting := func(completed bool) {
		update := map[string]any{"deletion_requested_at": past}
		if completed {
			update["deletion_completed_at"] = past
		}
		if err := database.Model(&event).Updates(update).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, sample := range []struct {
		name    string
		prepare func()
		want    bool
	}{
		{"empty", func() {}, false},
		{"queued export", func() { createExport("queued", nil, nil) }, true},
		{"active export", func() { createExport("running", &past, nil) }, false},
		{"stale running export", func() { createExport("running", &stale, nil) }, true},
		{"failed export", func() { createExport("failed", &stale, &past) }, false},
		{"unexpired export", func() { createExport("completed", &past, &future) }, false},
		{"expired export", func() { createExport("completed", &stale, &past) }, true},
		{"rejected photo waiting retention", func() { createPhoto("rejected", &original, &thumbnail, &future) }, false},
		{"rejected photo due", func() { createPhoto("rejected", &original, &thumbnail, &past) }, true},
		{"already deleted original", func() { createPhoto("rejected", nil, nil, &past) }, false},
		{"missing thumbnail", func() { createPhoto("approved", &original, nil, nil) }, true},
		{"existing thumbnail", func() { createPhoto("approved", &original, &thumbnail, nil) }, false},
		{"no original to thumbnail", func() { createPhoto("approved", nil, nil, nil) }, false},
		{"deleted event thumbnail", func() { deleting(true); createPhoto("approved", &original, nil, nil) }, false},
		{"event pending deletion", func() { deleting(false) }, true},
		{"deletion blocked by active export", func() { deleting(false); createExport("running", &past, nil) }, false},
		{"queued export for deleted event", func() { deleting(true); createExport("queued", nil, nil) }, false},
	} {
		t.Run(sample.name, func(t *testing.T) {
			if err := database.Exec("TRUNCATE event_photos,photo_exports,photo_events CASCADE").Error; err != nil {
				t.Fatal(err)
			}
			event = models.PhotoEvent{Name: "Dispatch fixture", UploadsOpenAt: past, UploadsCloseAt: future, GalleryCloseAt: future}
			if err := database.Create(&event).Error; err != nil {
				t.Fatal(err)
			}
			sample.prepare()
			calls := 0
			started, err := dispatchPhotoWorker(context.Background(), database, now, func(context.Context) error { calls++; return nil })
			wantCalls := 0
			if sample.want {
				wantCalls = 1
			}
			if err != nil || started != sample.want || calls != wantCalls {
				t.Fatalf("started=%t calls=%d err=%v", started, calls, err)
			}
		})
	}
	// Failed starts leave ready work untouched for the next recovery tick.
	database.Exec("TRUNCATE event_photos,photo_exports,photo_events CASCADE")
	event = models.PhotoEvent{Name: "Retry fixture", UploadsOpenAt: past, UploadsCloseAt: future, GalleryCloseAt: future}
	if err = database.Create(&event).Error; err != nil {
		t.Fatal(err)
	}
	createExport("queued", nil, nil)
	started, err := dispatchPhotoWorker(context.Background(), database, now, func(context.Context) error { return errors.New("launch unavailable") })
	if started || err == nil {
		t.Fatal("failed launch accepted")
	}
	var queued int64
	database.Model(&models.PhotoExport{}).Where("status='queued'").Count(&queued)
	if queued != 1 {
		t.Fatal("dispatch changed export queue")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	started, err = dispatchPhotoWorker(cancelled, database, now, func(context.Context) error { t.Fatal("storage failure launched worker"); return nil })
	if started || err == nil {
		t.Fatal("query failure did not fail closed")
	}
}
