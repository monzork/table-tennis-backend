package match

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/player"
	tournament "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

type MatchRepository interface {
	Save(ctx context.Context, m *tournament.Match) error
}

type CreateMatchUseCase struct {
	matchRepo      MatchRepository
	playerRepo     bun.PlayerRepository
	tournamentRepo bun.TournamentRepository
}

func NewCreateMatchUseCase(
	matchRepo MatchRepository,
	players bun.PlayerRepository,
	tournaments bun.TournamentRepository,
) *CreateMatchUseCase {
	return &CreateMatchUseCase{
		matchRepo:      matchRepo,
		playerRepo:     players,
		tournamentRepo: tournaments,
	}
}

func (uc *CreateMatchUseCase) Execute(ctx context.Context, tournamentID, playerAID, playerBID uuid.UUID) (*tournament.Match, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, errors.New("tournament not found")
	}

	pA, ok := uc.playerRepo.GetById(ctx, playerAID)
	if ok != nil {
		return nil, errors.New("player A not found")
	}

	pB, ok := uc.playerRepo.GetById(ctx, playerBID)
	if ok != nil {
		return nil, errors.New("player B not found")
	}

	m := &tournament.Match{
		ID:           uuid.New(),
		Players:      []*player.Player{pA, pB},
		Status:       "in_progress",
		TournamentID: tournamentID,
		Sets:         []tournament.MatchSet{},
	}

	// Add match to tournament
	t.AddMatch(*m)

	// Save match via repository
	if err := uc.matchRepo.Save(ctx, m); err != nil {
		return nil, err
	}

	return m, nil
}
