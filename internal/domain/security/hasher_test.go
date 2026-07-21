package security_test

import (
	"errors"
	"testing"

	"table-tennis-backend/internal/domain/security"
)

type mockHasher struct{}

func (m *mockHasher) Hash(password string) (string, error) {
	if password == "" {
		return "", errors.New("empty password")
	}
	return "hashed_" + password, nil
}

func (m *mockHasher) Compare(hash, password string) error {
	if hash != "hashed_"+password {
		return errors.New("mismatch")
	}
	return nil
}

func TestPasswordHasherInterface(t *testing.T) {
	var hasher security.PasswordHasher = &mockHasher{}

	h, err := hasher.Hash("secret")
	if err != nil {
		t.Fatalf("unexpected error hashing: %v", err)
	}
	if h != "hashed_secret" {
		t.Errorf("expected 'hashed_secret', got '%s'", h)
	}

	if err := hasher.Compare(h, "secret"); err != nil {
		t.Errorf("expected Compare to succeed, got %v", err)
	}

	if err := hasher.Compare(h, "wrong_pass"); err == nil {
		t.Error("expected Compare to fail for wrong password, got nil")
	}

	if _, err := hasher.Hash(""); err == nil {
		t.Error("expected Hash to fail for empty password, got nil")
	}
}
