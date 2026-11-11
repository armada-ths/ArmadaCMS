package main

import (
	"ArmadaCMS/main/utils"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

func TestLoungeOrdersRequireAuthentication(t *testing.T) {
	t.Setenv("LOUNGE_DISPATCH_AUDIENCE", "https://cms.example.test/api/v1/internal/lounge-orders/dispatch")
	t.Setenv("LOUNGE_DISPATCH_SERVICE_ACCOUNT", "scheduler@example.test")
	t.Setenv("LOUNGE_ORDER_SUBMISSION_SECRET", "test-live")
	t.Setenv("LOUNGE_ORDER_TEST_SUBMISSION_SECRET", "test-preview")
	handler := CreateMuxClient()
	for _, path := range []string{
		"/api/v1/lounge-orders", "/api/v1/lounge-orders/1",
		"/api/v1/lounge-orders/totals", "/api/v1/lounge-orders/filter-options",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s leaked without authentication: %d", path, response.Code)
		}
	}
	for _, path := range []string{"/api/v1/lounge-orders/submit", "/api/v1/internal/lounge-orders/dispatch", "/api/v1/lounge-orders/1/retry"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, nil))
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("%s accepted without authentication: %d", path, response.Code)
		}
	}
}

func TestLoungePermissionsAreIndependent(t *testing.T) {
	t.Setenv("jwtsecret_laganda", strings.Repeat("s", 32))
	handler := CreateMuxClient()
	for _, test := range []struct {
		permission, method, path string
	}{
		{"lounge-catalogue.*", http.MethodGet, "/api/v1/lounge-orders"},
		{"lounge-catalogue.*", http.MethodGet, "/api/v1/lounge-orders/1"},
		{"lounge-catalogue.*", http.MethodGet, "/api/v1/lounge-orders/totals"},
		{"lounge-catalogue.*", http.MethodGet, "/api/v1/lounge-orders/filter-options"},
		{"lounge-catalogue.*", http.MethodPut, "/api/v1/lounge-orders/1"},
		{"lounge-catalogue.*", http.MethodPost, "/api/v1/lounge-orders/1/retry"},
		{"lounge-orders.*", http.MethodPost, "/api/v1/lounge-items"},
		{"lounge-orders.*", http.MethodPut, "/api/v1/lounge-items/1"},
		{"lounge-orders.*", http.MethodDelete, "/api/v1/lounge-categories/1"},
		{"lounge-orders.*", http.MethodPut, "/api/v1/lounge-catalogue"},
		{"lounge-orders.view", http.MethodPut, "/api/v1/lounge-orders/1"},
		{"lounge-orders.view", http.MethodPost, "/api/v1/lounge-orders/1/retry"},
	} {
		t.Run(test.permission+" "+test.method+" "+test.path, func(t *testing.T) {
			token, err := utils.GenerateAccessToken(1, nil, []string{test.permission})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(test.method, test.path, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("expected forbidden, got %d", response.Code)
			}
		})
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
	for _, header := range []string{"Content-Range", "X-Armada-Error-Code"} {
		exposed := false
		for _, value := range strings.Split(response.Header().Get("Access-Control-Expose-Headers"), ",") {
			exposed = exposed || strings.EqualFold(strings.TrimSpace(value), header)
		}
		if !exposed {
			t.Fatalf("response header %s is not exposed to browser clients", header)
		}
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

func TestHealthIdentifiesServingRevisionWithoutDatabase(t *testing.T) {
	t.Setenv("K_REVISION", "armadacms-release-test")
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	response := httptest.NewRecorder()
	CreateMuxClient().ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Body.String() != "OK" {
		t.Fatalf("unexpected liveness response: %d %q", response.Code, response.Body.String())
	}
	for header, expected := range map[string]string{
		"X-Armada-Commit":   buildCommit,
		"X-Armada-Revision": "armadacms-release-test",
		"Cache-Control":     "no-store",
	} {
		if got := response.Header().Get(header); got != expected {
			t.Fatalf("%s = %q, want %q", header, got, expected)
		}
	}
}
