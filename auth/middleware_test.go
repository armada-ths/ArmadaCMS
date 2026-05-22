package auth

import (
	"ArmadaCMS/main/utils"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMiddlewareAcceptsValidTokenAndSetsContext(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "middleware-test-secret")

	token, err := utils.GenerateAccessToken(42, "admin", []string{"customusers.list"})
	if err != nil {
		t.Fatalf("failed generating token: %v", err)
	}

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		uid, ok := GetUserIDFromContext(r)
		if !ok || uid != 42 {
			t.Fatalf("expected user id 42 in context, got %d (ok=%v)", uid, ok)
		}

		perms := GetPermissionsFromContext(r)
		if len(perms) != 1 || perms[0] != "customusers.list" {
			t.Fatalf("unexpected permissions in context: %#v", perms)
		}

		w.WriteHeader(http.StatusOK)
	})

	handler := Middleware(next)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/customusers", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d (body=%q)", rec.Code, rec.Body.String())
	}
}

func TestHasPermission(t *testing.T) {
	cases := []struct {
		name     string
		perms    []string
		required string
		want     bool
	}{
		{"exact match", []string{"users.list"}, "users.list", true},
		{"wildcard grants all", []string{"*"}, "anything.delete", true},
		{"missing permission", []string{"users.list"}, "users.delete", false},
		{"empty permissions", []string{}, "users.list", false},
		{"nil permissions", nil, "users.list", false},
		{"resource wildcard grants matching resource", []string{"users.*"}, "users.delete", true},
		{"resource wildcard does not grant other resource", []string{"users.*"}, "roles.delete", false},
		{"action wildcard grants matching action", []string{"*.list"}, "users.list", true},
		{"action wildcard does not grant other action", []string{"*.list"}, "users.delete", false},
		{"resource wildcard grants all actions on that resource", []string{"profiles.*"}, "profiles.create", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasPermission(tc.perms, tc.required); got != tc.want {
				t.Fatalf("HasPermission(%v, %q) = %v, want %v", tc.perms, tc.required, got, tc.want)
			}
		})
	}
}

func TestMiddlewareRejectsMissingAuthHeader(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "test-secret")

	called := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true })
	handler := Middleware(next)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if called {
		t.Fatal("expected next handler not to be called")
	}
}

func TestMiddlewareRejectsMalformedAuthHeader(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "test-secret")

	handler := Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	req.Header.Set("Authorization", "NotBearer abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for malformed Authorization header, got %d", rec.Code)
	}
}

func TestMiddlewareRejectsInvalidToken(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "test-secret")

	handler := Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/anything", nil)
	req.Header.Set("Authorization", "Bearer not-a-real-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid token, got %d", rec.Code)
	}
}

func TestRequirePermissionAllowsWhenPermissionPresent(t *testing.T) {
	called := false
	wrapped := RequirePermission("users.list", func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	req = req.WithContext(context.WithValue(req.Context(), permissionsKey, []string{"users.list"}))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !called {
		t.Fatal("expected wrapped handler to be called")
	}
}

func TestRequirePermissionRejectsWhenPermissionMissing(t *testing.T) {
	wrapped := RequirePermission("users.delete", func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called when permission is missing")
	})

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/1", nil)
	req = req.WithContext(context.WithValue(req.Context(), permissionsKey, []string{"users.list"}))
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rec.Code)
	}
}
