package bun

import (
	"context"
	"table-tennis-backend/internal/domain/player"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PlayerRepository struct {
	db *bun.DB
}

func NewPlayerRepository(db *bun.DB) *PlayerRepository {
	return &PlayerRepository{db: db}
}

func (r *PlayerRepository) Save(ctx context.Context, p *player.Player) error {
	model := &PlayerModel{
		ID:         p.ID,
		FirstName:  p.FirstName,
		LastName:   p.LastName,
		Birthdate:  p.Birthdate,
		Gender:     p.Gender,
		SinglesElo: p.SinglesElo,
		DoublesElo: p.DoublesElo,
		Country:    p.Country,
	}

	_, err := r.db.NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("first_name = EXCLUDED.first_name, last_name = EXCLUDED.last_name, gender = EXCLUDED.gender, singles_elo = EXCLUDED.singles_elo, doubles_elo = EXCLUDED.doubles_elo, country = EXCLUDED.country").
		Exec(ctx)

	return err
}

func (r *PlayerRepository) GetAllSingles(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := r.db.NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) GetAllDoubles(ctx context.Context) ([]*player.Player, error) {
	var models []PlayerModel
	err := r.db.NewSelect().Model(&models).OrderBy("doubles_elo", bun.OrderDesc).Scan(ctx)

	if err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

// Ensure the old GetAll still works for backward compatibility with other uses
func (r *PlayerRepository) GetAll(ctx context.Context) ([]*player.Player, error) {
	return r.GetAllSingles(ctx)
}

// GetSinglesByGender returns singles ranking filtered by gender ("M" or "F")
func (r *PlayerRepository) GetSinglesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.db.NewSelect().Model(&models).OrderBy("singles_elo", bun.OrderDesc)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

// GetDoublesByGender returns doubles ranking filtered by gender ("M", "F", or "" for mixed/all)
func (r *PlayerRepository) GetDoublesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var models []PlayerModel
	q := r.db.NewSelect().Model(&models).OrderBy("doubles_elo", bun.OrderDesc)
	if gender != "" {
		q = q.Where("gender = ?", gender)
	}
	if err := q.Scan(ctx); err != nil {
		return nil, err
	}
	return r.mapModelsToDomain(models), nil
}

func (r *PlayerRepository) mapModelsToDomain(models []PlayerModel) []*player.Player {
	players := make([]*player.Player, len(models))
	for i, m := range models {
		players[i] = &player.Player{
			ID:         m.ID,
			FirstName:  m.FirstName,
			LastName:   m.LastName,
			Birthdate:  m.Birthdate,
			Gender:     m.Gender,
			SinglesElo: m.SinglesElo,
			DoublesElo: m.DoublesElo,
			Country:    m.Country,
		}
	}
	return players
}

func (r *PlayerRepository) GetById(ctx context.Context, id uuid.UUID) (*player.Player, error) {
	var model PlayerModel
	err := r.db.NewSelect().Model(&model).Where("id = ?", id).Scan(ctx)

	if err != nil {
		return nil, err
	}

	return &player.Player{
		ID:         model.ID,
		FirstName:  model.FirstName,
		LastName:   model.LastName,
		Birthdate:  model.Birthdate,
		Gender:     model.Gender,
		SinglesElo: model.SinglesElo,
		DoublesElo: model.DoublesElo,
		Country:    model.Country,
	}, nil
}

func (r *PlayerRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.NewDelete().Model((*PlayerModel)(nil)).Where("id = ?", id).Exec(ctx)
	return err
}
