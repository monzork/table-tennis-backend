package admin

import (
	"errors"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Admin struct {
	ID           uuid.UUID
	Username     string
	PasswordHash string
}

// NewAdmin creates a new admin entity, hashing the plain text password
func NewAdmin(username, plainPassword string) (*Admin, error) {
	if username == "" || plainPassword == "" {
		return nil, errors.New("username and password cannot be empty")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	return &Admin{
		ID:           uuid.New(),
		Username:     username,
		PasswordHash: string(hash),
	}, nil
}

// CheckPassword verifies if the given plain text password matches the hash
func (a *Admin) CheckPassword(plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(plainPassword))
	return err == nil
}
