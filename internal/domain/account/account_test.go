package account_test

import (
	"errors"
	"testing"

	"table-tennis-backend/internal/domain/account"
)

func TestNewAccount(t *testing.T) {
	t.Run("valid account", func(t *testing.T) {
		a, err := account.NewAccount("id1", "sub1", "a@b.com", "Alice", "http://pic")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.ID != "id1" || a.GoogleSub != "sub1" || a.Email != "a@b.com" || a.Name != "Alice" || a.PictureURL != "http://pic" {
			t.Fatalf("unexpected account: %+v", a)
		}
	})

	t.Run("empty google sub", func(t *testing.T) {
		_, err := account.NewAccount("id1", "", "a@b.com", "Alice", "")
		if !errors.Is(err, account.ErrInvalidGoogleSub) {
			t.Fatalf("expected ErrInvalidGoogleSub, got %v", err)
		}
	})

	t.Run("empty email", func(t *testing.T) {
		_, err := account.NewAccount("id1", "sub1", "", "Alice", "")
		if !errors.Is(err, account.ErrInvalidEmail) {
			t.Fatalf("expected ErrInvalidEmail, got %v", err)
		}
	})
}
