package auth

import (
	"ArmadaCMS/main/utils"
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
