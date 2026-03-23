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

func (uc *CreateMatchUseCase) Execute(ctx context.Context, tournamentID uuid.UUID, matchType string, teamAPlayerIDs, teamBPlayerIDs []uuid.UUID) (*tournament.Match, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, errors.New("tournament not found")
	}

	var teamA []*player.Player
	for _, id := range teamAPlayerIDs {
		p, err := uc.playerRepo.GetById(ctx, id)
		if err != nil {
			return nil, errors.New("team A player not found")
		}
		teamA = append(teamA, p)
	}

	var teamB []*player.Player
	for _, id := range teamBPlayerIDs {
		p, err := uc.playerRepo.GetById(ctx, id)
		if err != nil {
			return nil, errors.New("team B player not found")
		}
		teamB = append(teamB, p)
	}

	if matchType == "" {
		matchType = "singles"
	}

	m := &tournament.Match{
		ID:           uuid.New(),
		TournamentID: tournamentID,
		MatchType:    matchType,
		TeamA:        teamA,
		TeamB:        teamB,
		Status:       "in_progress",
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
