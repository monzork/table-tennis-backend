package match

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/player"
	tournament "table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
)

type CreateMatchUseCase struct {
	matchRepo      tournament.MatchRepository
	playerRepo     player.Repository
	tournamentRepo tournament.Repository
}

func NewCreateMatchUseCase(
	matchRepo tournament.MatchRepository,
	players player.Repository,
	tournaments tournament.Repository,
) *CreateMatchUseCase {
	return &CreateMatchUseCase{
		matchRepo:      matchRepo,
		playerRepo:     players,
		tournamentRepo: tournaments,
	}
}

func (uc *CreateMatchUseCase) Execute(ctx context.Context, tournamentID string, matchType string, teamAPlayerIDs, teamBPlayerIDs []string, opts ...string) (*tournament.Match, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, errors.New("tournament not found")
	}

	isTeamBased := matchType == "doubles" || matchType == "teams"

	// For team-based matches, resolve team IDs to their players
	teamPlayersMap := make(map[string][]*player.Player)
	if isTeamBased {
		for _, team := range t.Teams {
			teamPlayersMap[team.ID] = team.Players
		}
	}

	var teamA []*player.Player
	for _, id := range teamAPlayerIDs {
		if isTeamBased {
			if players, ok := teamPlayersMap[id]; ok && len(players) > 0 {
				teamA = append(teamA, players...)
			} else {
				return nil, errors.New("team A not found in tournament")
			}
		} else {
			p, err := uc.playerRepo.GetById(ctx, id)
			if err != nil {
				return nil, errors.New("team A player not found")
			}
			teamA = append(teamA, p)
		}
	}

	var teamB []*player.Player
	for _, id := range teamBPlayerIDs {
		if isTeamBased {
			if players, ok := teamPlayersMap[id]; ok && len(players) > 0 {
				teamB = append(teamB, players...)
			} else {
				return nil, errors.New("team B not found in tournament")
			}
		} else {
			p, err := uc.playerRepo.GetById(ctx, id)
			if err != nil {
				return nil, errors.New("team B player not found")
			}
			teamB = append(teamB, p)
		}
	}

	if matchType == "" {
		matchType = "singles"
	}

	stage := "group"
	if len(opts) > 0 && opts[0] != "" {
		stage = opts[0]
	}

	m := &tournament.Match{
		ID:           uuid.NewString(),
		TournamentID: tournamentID,
		MatchType:    matchType,
		TeamA:        teamA,
		TeamB:        teamB,
		Status:       "in_progress",
		Sets:         []tournament.MatchSet{},
		Stage:        stage,
	}

	// Add match to tournament
	t.AddMatch(*m)

	// Save match via repository
	if err := uc.matchRepo.Save(ctx, m); err != nil {
		return nil, err
	}

	return m, nil
}
