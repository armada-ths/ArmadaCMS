package auth

import (
	"cloud.google.com/go/auth/credentials/idtoken"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGoogleServiceAccountAuthentication(t *testing.T) {
	const audience = "https://cms.example/api/v1/internal/translations/dispatch"
	const email = "scheduler@project.iam.gserviceaccount.com"
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public,max-age=3600")
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]string{"alg": "RS256", "kty": "RSA", "use": "sig", "kid": "test-key", "n": base64.RawURLEncoding.EncodeToString(key.PublicKey.N.Bytes()), "e": base64.RawURLEncoding.EncodeToString(big.NewInt(int64(key.PublicKey.E)).Bytes())}}})
	}))
	defer jwks.Close()
	validator, err := idtoken.NewValidator(&idtoken.ValidatorOptions{Client: jwks.Client(), RS256CertsURL: jwks.URL})
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		name, claim       string
		value             any
		code              int
		wrongKey, missing bool
	}{
		{name: "valid", code: http.StatusNoContent},
		{name: "wrong audience", claim: "aud", value: "https://other.example", code: http.StatusUnauthorized},
		{name: "expired", claim: "exp", value: time.Now().Add(-time.Minute).Unix(), code: http.StatusUnauthorized},
		{name: "forged signature", wrongKey: true, code: http.StatusUnauthorized},
		{name: "wrong account", claim: "email", value: "other@project.iam.gserviceaccount.com", code: http.StatusForbidden},
		{name: "unverified account", claim: "email_verified", value: false, code: http.StatusForbidden},
		{name: "forged issuer", claim: "iss", value: "https://other.example", code: http.StatusForbidden},
		{name: "missing subject", claim: "sub", value: "", code: http.StatusForbidden},
		{name: "missing bearer", missing: true, code: http.StatusUnauthorized},
	} {
		t.Run(sample.name, func(t *testing.T) {
			claims := jwt.MapClaims{"iss": "https://accounts.google.com", "aud": audience, "sub": "123456789", "email": email, "email_verified": true, "iat": time.Now().Add(-time.Minute).Unix(), "exp": time.Now().Add(time.Hour).Unix()}
			if sample.claim != "" {
				claims[sample.claim] = sample.value
			}
			token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
			token.Header["kid"] = "test-key"
			signingKey := key
			if sample.wrongKey {
				signingKey = other
			}
			signed, signErr := token.SignedString(signingKey)
			if signErr != nil {
				t.Fatal(signErr)
			}
			request := httptest.NewRequest(http.MethodPost, audience, nil)
			if !sample.missing {
				request.Header.Set("Authorization", "Bearer "+signed)
			}
			called := false
			handler := RequireGoogleServiceAccount(audience, email, validator.Validate, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(http.StatusNoContent) }))
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != sample.code || called != (sample.code == http.StatusNoContent) {
				t.Fatalf("response %d, called=%t", response.Code, called)
			}
		})
	}
}

func TestGoogleServiceAccountFailsClosedWithoutConfiguration(t *testing.T) {
	called := false
	validate := func(context.Context, string, string) (*idtoken.Payload, error) {
		called = true
		return nil, fmt.Errorf("unexpected validation")
	}
	handler := RequireGoogleServiceAccount("", "", validate, http.HandlerFunc(func(http.ResponseWriter, *http.Request) { t.Fatal("dispatch reached") }))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/", nil))
	if response.Code != http.StatusServiceUnavailable || called {
		t.Fatal("unconfigured dispatcher did not fail closed")
	}
}
