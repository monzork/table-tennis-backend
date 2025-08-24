package players

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

func (s *Service) RegisterPlayers(ctx context.Context, name, sex, country, city, birthdate string, elo *int16) (*Players, error) {

	if elo == nil {
		init_elo := int16(1000)
		elo = &init_elo
	}

	p := &Players{
		ID:         uuid.New(),
		Name:       name,
		Sex:        sex,
		Country:    country,
		City:       city,
		Birthdate:  birthdate,
		Elo:        elo,
		Created_at: time.Now().UTC(),
		Updated_at: time.Now().UTC(),
	}

	if err := s.repo.Create(ctx, p); err != nil {
		return nil, err
	}

	return p, nil
}

func (s *Service) GetAllPlayers(ctx context.Context) (*[]Players, error) {
	players, err := s.repo.GetAll(ctx)
	return players, err
}

func (s *Service) UpdatePlayers(ctx context.Context, id uuid.UUID, updates map[string]any) (*Players, error) {
	return s.repo.Update(ctx, id, updates)
}

func (s *Service) DeletePlayers(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}
