package storage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/monzork/table-tennis-backend/internal/domain/players"
	"github.com/uptrace/bun"
)

type SQLitePlayersRepository struct {
	db *bun.DB
}

func NewSQLitePlayersRepository(db *bun.DB) *SQLitePlayersRepository {
	return &SQLitePlayersRepository{db: db}
}

func (r *SQLitePlayersRepository) Create(ctx context.Context, p *players.Players) error {
	// Ensure the ID is set
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	// Set timestamps
	now := time.Now().UTC()
	if p.Created_at.IsZero() {
		p.Created_at = now
	}
	p.Updated_at = now

	_, err := r.db.NewInsert().Model(p).Exec(ctx)
	return err
}

func (r *SQLitePlayersRepository) GetAll(ctx context.Context) (*[]players.Players, error) {
	players := &[]players.Players{}

	err := r.db.NewSelect().Model(players).Scan(ctx)
	return players, err
}

func (r *SQLitePlayersRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*players.Players, error) {
	query := r.db.NewUpdate().Model(&players.Players{}).Where("id = ?", id)

	// Armamos Set dinámicamente
	for k, v := range updates {
		query = query.Set(k+" = ?", v)
	}

	_, err := query.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// Traemos el jugador actualizado
	var updatedPlayer players.Players
	err = r.db.NewSelect().
		Model(&updatedPlayer).
		Where("id = ?", id).
		Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &updatedPlayer, nil
}
