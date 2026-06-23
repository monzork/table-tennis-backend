package admin

import (
	"errors"
)

type Admin struct {
	ID           string
	Username     string
	PasswordHash string
}

// NewAdmin creates a new admin entity with a pre-hashed password
func NewAdmin(id, username, passwordHash string) (*Admin, error) {
	if username == "" || passwordHash == "" {
		return nil, errors.New("username and password hash cannot be empty")
	}

	return &Admin{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
	}, nil
}
