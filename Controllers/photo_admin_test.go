package controllers

import (
	"ArmadaCMS/main/models"
	"testing"
	"time"
)

func TestPhotoEventActivationRequiresPrivacyAndValidTimes(t *testing.T) {
	now := time.Now()
	event := models.PhotoEvent{
		Name: "Banquet", Slug: "banquet-2026", MaxPhotosPerGuest: 25,
		UploadsOpenAt: now, UploadsCloseAt: now.Add(time.Hour),
		GalleryCloseAt: now.Add(24 * time.Hour), DeleteAfter: now.Add(48 * time.Hour),
	}
	if err := validatePhotoEvent(event); err != nil {
		t.Fatal(err)
	}
	event.Active = true
	if err := validatePhotoEvent(event); err == nil {
		t.Fatal("active event without privacy information accepted")
	}
	event.PrivacyURL = "https://armada.nu/privacy"
	if err := validatePhotoEvent(event); err != nil {
		t.Fatal(err)
	}
	event.UploadsCloseAt = event.UploadsOpenAt
	if err := validatePhotoEvent(event); err == nil {
		t.Fatal("invalid time window accepted")
	}
}
