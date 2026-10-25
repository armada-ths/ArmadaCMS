package controllers

import (
	"ArmadaCMS/main/models"
	"testing"
	"time"
)

func TestRejectedPhotoGracePeriod(t *testing.T) {
	now := time.Now().UTC()
	key := "private/photo.jpg"
	for _, status := range []string{"pending", "approved"} {
		photo := models.EventPhoto{Status: status, ObjectKey: &key}
		if !applyPhotoModeration(&photo, "reject", now) || photo.ObjectKey == nil || !photo.RejectedDeleteAt.Equal(now.Add(24*time.Hour)) {
			t.Fatalf("rejection must preserve %s photo for 24 hours: %+v", status, photo)
		}
		if applyPhotoModeration(&photo, "reject", now.Add(time.Hour)) {
			t.Fatal("repeated rejection must not extend the deadline")
		}
		if !applyPhotoModeration(&photo, "approve", now.Add(23*time.Hour)) || photo.Status != "approved" || photo.RejectedDeleteAt != nil {
			t.Fatal("approve must restore an existing photo and cancel deletion")
		}
		if applyPhotoModeration(&photo, "approve", now.Add(23*time.Hour)) {
			t.Fatal("an approved photo must not be approved twice")
		}
	}
	deadline := now.Add(24 * time.Hour)
	for _, when := range []time.Time{deadline, deadline.Add(time.Second)} {
		photo := models.EventPhoto{Status: "rejected", ObjectKey: &key, RejectedDeleteAt: &deadline}
		if applyPhotoModeration(&photo, "approve", when) {
			t.Fatal("restore must fail at or after expiration")
		}
	}
	deleted := models.EventPhoto{Status: "rejected", RejectedDeleteAt: &deadline}
	if applyPhotoModeration(&deleted, "approve", now) {
		t.Fatal("already deleted photos cannot be restored")
	}
}
