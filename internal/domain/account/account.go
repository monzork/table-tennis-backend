package account

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvalidGoogleSub = errors.New("google sub cannot be empty")
	ErrInvalidEmail     = errors.New("email cannot be empty")
)

// Account is a parent/guardian login backed by Google OAuth. It is
// deliberately separate from admin.Admin (which is username/password based)
// and from player.Player (which has no login of its own).
type Account struct {
	ID         string
	GoogleSub  string
	Email      string
	Name       string
	PictureURL string
	CreatedAt  time.Time
}

// NewAccount creates a new guardian account tied to a Google identity.
func NewAccount(id, googleSub, email, name, pictureURL string) (*Account, error) {
	if googleSub == "" {
		return nil, ErrInvalidGoogleSub
	}
	if email == "" {
		return nil, ErrInvalidEmail
	}
	return &Account{
		ID:         id,
		GoogleSub:  googleSub,
		Email:      email,
		Name:       name,
		PictureURL: pictureURL,
	}, nil
}

// Repository persists and retrieves Account records.
type Repository interface {
	GetByID(ctx context.Context, id string) (*Account, error)
	GetByIDs(ctx context.Context, ids []string) ([]*Account, error)
	GetByGoogleSub(ctx context.Context, sub string) (*Account, error)
	GetByEmail(ctx context.Context, email string) (*Account, error)
	Save(ctx context.Context, a *Account) error // upsert
}
