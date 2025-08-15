package user

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) RegisterUser(ctx context.Context, username, password string) (*User, error) {
	u := &User{
		ID:         uuid.New(),
		Username:   username,
		Password:   password,
		Created_at: time.Now(),
	}

	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}
