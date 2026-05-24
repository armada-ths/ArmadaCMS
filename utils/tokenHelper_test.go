package utils

import "testing"

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
