package main

import (
	"ArmadaCMS/main/utils"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTranslationDispatcherRejectsCMSUserToken(t *testing.T) {
	t.Setenv("TRANSLATION_DISPATCH_AUDIENCE", "https://cms.example/api/v1/internal/translations/dispatch")
	t.Setenv("TRANSLATION_DISPATCH_SERVICE_ACCOUNT", "scheduler@project.iam.gserviceaccount.com")
	t.Setenv("jwtsecret_laganda", "dispatcher-test-signing-secret-value")
	token, err := utils.GenerateAccessToken(1, nil, []string{"*"})
	if err != nil {
		t.Fatal(err)
	}
	router := CreateMuxClient()
	for _, header := range []string{"", "Bearer " + token} {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/internal/translations/dispatch", nil)
		req.Header.Set("Authorization", header)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, req)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("dispatcher accepted CMS identity: %d", response.Code)
		}
	}
}
