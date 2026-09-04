package bun

import (
	"context"

	"table-tennis-backend/internal/domain/inactivity"

	"github.com/uptrace/bun"
)

// InactivitySettingsModel is a single row (id = "default") holding the
// admin-configurable inactivity-decay parameters. The elo floor itself
// isn't one of them -- it's computed per player from their own rating, see
// inactivity.BandFloor.
type InactivitySettingsModel struct {
	bun.BaseModel `bun:"table:inactivity_settings"`

	ID                  string `bun:"id,pk"`
	TournamentThreshold int16  `bun:"tournament_threshold,notnull,default:4"`
	EloPenalty          int16  `bun:"elo_penalty,notnull,default:50"`
}

const inactivitySettingsRowID = "default"

type InactivitySettingsRepository struct {
	db *bun.DB
}

func NewInactivitySettingsRepository(db *bun.DB) *InactivitySettingsRepository {
	return &InactivitySettingsRepository{db: db}
}

// Get returns the single settings row, seeding it with defaults (matching
// migration 073) if it's somehow missing.
func (r *InactivitySettingsRepository) Get(ctx context.Context) (*inactivity.Settings, error) {
	model := new(InactivitySettingsModel)
	err := ExtractDB(ctx, r.db).NewSelect().Model(model).Where("id = ?", inactivitySettingsRowID).Scan(ctx)
	if err != nil {
		return &inactivity.Settings{TournamentThreshold: 4, EloPenalty: 50}, nil
	}
	return &inactivity.Settings{
		TournamentThreshold: int(model.TournamentThreshold),
		EloPenalty:          int(model.EloPenalty),
	}, nil
}

func (r *InactivitySettingsRepository) Update(ctx context.Context, s *inactivity.Settings) error {
	model := &InactivitySettingsModel{
		ID:                  inactivitySettingsRowID,
		TournamentThreshold: int16(s.TournamentThreshold),
		EloPenalty:          int16(s.EloPenalty),
	}
	_, err := ExtractDB(ctx, r.db).NewInsert().Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("tournament_threshold = EXCLUDED.tournament_threshold, elo_penalty = EXCLUDED.elo_penalty").
		Exec(ctx)
	return err
}
