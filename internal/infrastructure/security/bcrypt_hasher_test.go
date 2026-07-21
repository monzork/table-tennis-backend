package security_test

import (
	"testing"

	"table-tennis-backend/internal/infrastructure/security"
)

func TestBcryptHasher_HashAndCompare(t *testing.T) {
	hasher := security.NewBcryptHasher()
	if hasher == nil {
		t.Fatal("expected NewBcryptHasher to return non-nil instance")
	}

	password := "SecretPassword123!"

	hashed, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("unexpected error hashing password: %v", err)
	}

	if hashed == "" || hashed == password {
		t.Errorf("expected hashed password to be non-empty and different from plain text")
	}

	// Compare matching password
	err = hasher.Compare(hashed, password)
	if err != nil {
		t.Errorf("expected Compare to succeed with matching password, got: %v", err)
	}

	// Compare wrong password
	err = hasher.Compare(hashed, "WrongPassword123!")
	if err == nil {
		t.Errorf("expected Compare to fail with incorrect password")
	}
}

func TestBcryptHasher_Hash_Error(t *testing.T) {
	hasher := security.NewBcryptHasher()
	
	// bcrypt has a maximum password length of 72 bytes.
	// Providing a longer password will return an error.
	longPassword := ""
	for i := 0; i < 80; i++ {
		longPassword += "a"
	}

	_, err := hasher.Hash(longPassword)
	if err == nil {
		t.Fatalf("expected error when hashing password > 72 bytes, got nil")
	}
}
