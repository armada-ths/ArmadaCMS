package controllers

import (
	"ArmadaCMS/main/models"
	"testing"
)

func TestPhotoTranslationPersistence(t *testing.T) {
	testTranslationPersistence(t, []any{&models.PhotoEvent{}})
}
