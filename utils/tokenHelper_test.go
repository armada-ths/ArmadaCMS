package utils

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestValidateJWTSecretReturnsErrorWhenMissing(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "")

	if err := ValidateJWTSecret(); err == nil {
		t.Fatal("expected ValidateJWTSecret to fail when jwtsecret_laganda is missing")
	}
}

func TestValidateJWTSecretAcceptsConfiguredSecret(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "super-secret-value")

	if err := ValidateJWTSecret(); err != nil {
		t.Fatalf("expected ValidateJWTSecret to succeed, got error: %v", err)
	}
}

func TestGenerateAccessTokenReadsSecretAfterEnvIsSet(t *testing.T) {
	t.Setenv("jwtsecret_laganda", "super-secret-value")

	token, err := GenerateAccessToken(123, []string{"admin"}, []string{"*"})
	if err != nil {
		t.Fatalf("expected GenerateAccessToken to succeed, got error: %v", err)
	}

	if token == "" {
		t.Fatal("expected GenerateAccessToken to return a token")
	}
}

func TestHashRefreshTokenDoesNotRetainCredential(t *testing.T) {
	token := "high-entropy-refresh-token"
	hashed := HashRefreshToken(token)

	if hashed == token || len(hashed) != 64 {
		t.Fatalf("unexpected refresh-token digest %q", hashed)
	}
	if hashed != HashRefreshToken(token) {
		t.Fatal("expected refresh-token digest to be deterministic")
	}
}

func TestVerifyAccessTokenRejectsNonHS256Token(t *testing.T) {
	secret := "super-secret-value"
	t.Setenv("jwtsecret_laganda", secret)
	claims := accessClaims{
		UserID: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute)),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS512, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign HS512 token: %v", err)
	}

	if _, err := VerifyAccessToken(token); err == nil {
		t.Fatal("expected VerifyAccessToken to reject HS512")
	}
}
