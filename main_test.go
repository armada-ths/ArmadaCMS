package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalPhotosCORS(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	for _, origin := range []string{"http://localhost:8001", "http://127.0.0.1:8001"} {
		if !isAllowedCORSOrigin(origin) {
			t.Fatalf("local Photos origin is not allowed: %s", origin)
		}
	}
	if isAllowedCORSOrigin("https://untrusted.example:8001") {
		t.Fatal("an unrelated origin must not be allowed")
	}
}

func TestHandleCORSAllowsConfiguredOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://preview.example.com")
	handler := HandleCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	request.Header.Set("Origin", "https://preview.example.com")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "https://preview.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Fatalf("Access-Control-Allow-Credentials = %q", got)
	}
}

func TestHandleCORSDoesNotReflectUnknownOrigin(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	handler := HandleCORS(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	request := httptest.NewRequest(http.MethodGet, "/api/v1/events", nil)
	request.Header.Set("Origin", "https://attacker.example")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)

	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unknown origin was reflected: %q", got)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("credentials were allowed for an unknown origin: %q", got)
	}
}

func TestAdminAssetExistsOnlyServesFilesInsideRoot(t *testing.T) {
	parentDir := t.TempDir()
	assetDir := filepath.Join(parentDir, "assets")
	if err := os.Mkdir(assetDir, 0o755); err != nil {
		t.Fatalf("create asset directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetDir, "app.js"), []byte("asset"), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}
	if err := os.WriteFile(filepath.Join(parentDir, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("write outside file: %v", err)
	}

	files := http.Dir(assetDir)
	if !adminAssetExists(files, "app.js") {
		t.Fatal("expected in-root asset to exist")
	}
	for _, path := range []string{"../secret.txt", `..\secret.txt`, "/../secret.txt"} {
		if adminAssetExists(files, path) {
			t.Fatalf("outside path %q was accepted", path)
		}
	}
}
