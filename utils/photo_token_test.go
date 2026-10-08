package utils

import (
	"errors"
	"strings"
	"testing"
)

func TestPhotoEventTokenRoundTripAndTamper(t *testing.T) {
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("s", 48))
	token, err := PhotoEventToken(42, 3)
	if err != nil {
		t.Fatal(err)
	}
	id, version, err := ParsePhotoEventToken(token)
	if err != nil || id != 42 || version != 3 {
		t.Fatalf("roundtrip: %d %d %v", id, version, err)
	}
	if _, _, err := ParsePhotoEventToken(token + "x"); !errors.Is(err, ErrInvalidPhotoToken) {
		t.Fatalf("tamper should fail: %v", err)
	}
	other, err := PhotoEventToken(42, 4)
	if err != nil || other == token {
		t.Fatal("rotation must change the link")
	}
}

func TestPhotoGuestHashIsEventScoped(t *testing.T) {
	t.Setenv("PHOTO_TOKEN_SECRET", strings.Repeat("s", 48))
	a, err := HashPhotoGuest(1, "device")
	if err != nil {
		t.Fatal(err)
	}
	b, err := HashPhotoGuest(2, "device")
	if err != nil || a == b || strings.Contains(a, "device") {
		t.Fatal("guest hashes must be opaque and event-scoped")
	}
}

func TestPhotoSecretIsRequired(t *testing.T) {
	t.Setenv("PHOTO_TOKEN_SECRET", "")
	if _, err := PhotoEventToken(1, 1); err == nil {
		t.Fatal("empty secret must fail closed")
	}
}
