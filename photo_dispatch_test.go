package main

import (
	"ArmadaCMS/main/utils"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPhotoDispatcherRejectsCMSUserToken(t *testing.T) {
	t.Setenv("PHOTO_DISPATCH_AUDIENCE", "https://cms.example/api/v1/internal/photos/dispatch")
	t.Setenv("PHOTO_DISPATCH_SERVICE_ACCOUNT", "scheduler@project.iam.gserviceaccount.com")
	t.Setenv("jwtsecret_laganda", "photo-dispatch-test-signing-secret-value")
	token, err := utils.GenerateAccessToken(1, nil, []string{"*"})
	if err != nil {
		t.Fatal(err)
	}
	router := CreateMuxClient()
	for _, header := range []string{"", "Bearer " + token} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/photos/dispatch", nil)
		req.Header.Set("Authorization", header)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("dispatcher accepted CMS identity: %d", response.Code)
		}
	}
}
