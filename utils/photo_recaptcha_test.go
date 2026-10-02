package utils

import (
	"context"
	"testing"
)

func TestPhotoRecaptchaLocalBypass(t *testing.T) {
	t.Setenv("PHOTO_RECAPTCHA_BYPASS_LOCAL", "true")
	t.Setenv("K_SERVICE", "")
	t.Setenv("RECAPTCHA_PROJECT_ID", "")
	t.Setenv("PHOTO_RECAPTCHA_SITE_KEY", "")
	if err := VerifyPhotoRecaptcha(context.Background(), ""); err != nil {
		t.Fatalf("local bypass should accept an upload without reCAPTCHA: %v", err)
	}
}

func TestPhotoRecaptchaBypassDisabledOutsideLocalDevelopment(t *testing.T) {
	t.Setenv("RECAPTCHA_PROJECT_ID", "")
	t.Setenv("PHOTO_RECAPTCHA_SITE_KEY", "")
	for _, test := range []struct {
		name    string
		bypass  string
		service string
	}{
		{name: "bypass disabled", bypass: "false", service: ""},
		{name: "Cloud Run", bypass: "true", service: "armadacms"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("PHOTO_RECAPTCHA_BYPASS_LOCAL", test.bypass)
			t.Setenv("K_SERVICE", test.service)
			if err := VerifyPhotoRecaptcha(context.Background(), ""); err == nil {
				t.Fatal("reCAPTCHA must fail closed outside local development")
			}
		})
	}
}
