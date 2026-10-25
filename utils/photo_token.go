package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

var ErrInvalidPhotoToken = errors.New("invalid photo event token")

func photoSecret() ([]byte, error) {
	secret := []byte(os.Getenv("PHOTO_TOKEN_SECRET"))
	if len(secret) < 32 {
		return nil, errors.New("PHOTO_TOKEN_SECRET must contain at least 32 bytes")
	}
	return secret, nil
}

func PhotoEventToken(eventID uint64, version int) (string, error) {
	secret, err := photoSecret()
	if err != nil {
		return "", err
	}
	payload := fmt.Sprintf("%d:%d", eventID, version)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func ParsePhotoEventToken(token string) (uint64, int, error) {
	secret, err := photoSecret()
	if err != nil {
		return 0, 0, err
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, 0, ErrInvalidPhotoToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, 0, ErrInvalidPhotoToken
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, 0, ErrInvalidPhotoToken
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return 0, 0, ErrInvalidPhotoToken
	}
	fields := strings.Split(string(payload), ":")
	if len(fields) != 2 {
		return 0, 0, ErrInvalidPhotoToken
	}
	id, idErr := strconv.ParseUint(fields[0], 10, 64)
	version, versionErr := strconv.Atoi(fields[1])
	if idErr != nil || versionErr != nil || id == 0 || version < 1 {
		return 0, 0, ErrInvalidPhotoToken
	}
	return id, version, nil
}

func HashPhotoGuest(eventID uint64, guestID string) (string, error) {
	secret, err := photoSecret()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secret)
	_, _ = fmt.Fprintf(mac, "guest:%d:%s", eventID, guestID)
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
