package admin_test

import (
	"testing"

	"table-tennis-backend/internal/domain/admin"
)

func TestNewAdmin_Success(t *testing.T) {
	a, err := admin.NewAdmin("admin-1", "john_doe", "hashed_secret")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if a.ID != "admin-1" {
		t.Errorf("expected ID 'admin-1', got '%s'", a.ID)
	}
	if a.Username != "john_doe" {
		t.Errorf("expected Username 'john_doe', got '%s'", a.Username)
	}
	if a.PasswordHash != "hashed_secret" {
		t.Errorf("expected PasswordHash 'hashed_secret', got '%s'", a.PasswordHash)
	}
}

func TestNewAdmin_EmptyUsername(t *testing.T) {
	_, err := admin.NewAdmin("admin-1", "", "hashed_secret")
	if err == nil {
		t.Fatal("expected error for empty username, got nil")
	}
	expectedErrMsg := "username and password hash cannot be empty"
	if err.Error() != expectedErrMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestNewAdmin_EmptyPasswordHash(t *testing.T) {
	_, err := admin.NewAdmin("admin-1", "john_doe", "")
	if err == nil {
		t.Fatal("expected error for empty password hash, got nil")
	}
	expectedErrMsg := "username and password hash cannot be empty"
	if err.Error() != expectedErrMsg {
		t.Errorf("expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestNewAdmin_BothEmpty(t *testing.T) {
	_, err := admin.NewAdmin("admin-1", "", "")
	if err == nil {
		t.Fatal("expected error for empty username and password hash, got nil")
	}
}
