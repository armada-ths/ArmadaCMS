package utils

import "testing"

func TestHashPasswordProducesVerifiableHash(t *testing.T) {
	password := "correct horse battery staple"

	hash := HashPassword(password)
	if hash == "" {
		t.Fatal("expected HashPassword to return a non-empty hash")
	}
	if hash == password {
		t.Fatal("expected HashPassword to not return the plaintext password")
	}

	if err := CheckPasswordHash(password, hash); err != nil {
		t.Fatalf("expected CheckPasswordHash to succeed for matching password, got error: %v", err)
	}
}

func TestHashPasswordProducesDifferentHashesForSameInput(t *testing.T) {
	password := "same-password"

	hash1 := HashPassword(password)
	hash2 := HashPassword(password)

	if hash1 == hash2 {
		t.Fatal("expected bcrypt to produce different hashes (salts) for the same password")
	}
}

func TestCheckPasswordHashRejectsWrongPassword(t *testing.T) {
	hash := HashPassword("right-password")

	if err := CheckPasswordHash("wrong-password", hash); err == nil {
		t.Fatal("expected CheckPasswordHash to return an error for a wrong password")
	}
}
