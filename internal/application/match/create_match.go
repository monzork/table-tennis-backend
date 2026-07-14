package match

import (
	"context"
	"errors"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	event "table-tennis-backend/internal/domain/event"
)

type CreateMatchUseCase struct {
	matchRepo      event.MatchRepository
	playerRepo     player.Repository
	tournamentRepo event.Repository
	divisionRepo   divisionDomain.Repository
}

func NewCreateMatchUseCase(
	matchRepo event.MatchRepository,
	players player.Repository,
	events event.Repository,
	divisions divisionDomain.Repository,
) *CreateMatchUseCase {
	return &CreateMatchUseCase{
		matchRepo:      matchRepo,
		playerRepo:     players,
		tournamentRepo: events,
		divisionRepo:   divisions,
	}
}

func (uc *CreateMatchUseCase) Execute(ctx context.Context, tournamentID string, matchType string, teamAPlayerIDs, teamBPlayerIDs []string, opts ...string) (*event.Match, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return nil, errors.New("event not found")
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
	if isTeamBased {
		for _, id := range teamAPlayerIDs {
			if players, ok := teamPlayersMap[id]; ok && len(players) > 0 {
				teamA = append(teamA, players...)
			} else {
				return nil, errors.New("team A not found in event")
			}
		}
	} else if len(teamAPlayerIDs) > 0 {
		ps, err := uc.playerRepo.GetByIDs(ctx, teamAPlayerIDs)
		if err != nil {
			return nil, errors.New("team A player not found")
		}
		teamA = append(teamA, ps...)
	}

	var teamB []*player.Player
	if isTeamBased {
		for _, id := range teamBPlayerIDs {
			if players, ok := teamPlayersMap[id]; ok && len(players) > 0 {
				teamB = append(teamB, players...)
			} else {
				return nil, errors.New("team B not found in event")
			}
		}
	} else if len(teamBPlayerIDs) > 0 {
		ps, err := uc.playerRepo.GetByIDs(ctx, teamBPlayerIDs)
		if err != nil {
			return nil, errors.New("team B player not found")
		}
		teamB = append(teamB, ps...)
	}

	if matchType == "" {
		matchType = "singles"
	}

	stage := "group"
	if len(opts) > 0 && opts[0] != "" {
		stage = opts[0]
	}

	// Determine division ID for this match based on player Elo
	divisionID := uc.determinePlayerDivision(ctx, t, teamA, teamB, matchType)

	m := &event.Match{
		ID:           idgen.Generate(),
		TournamentID: tournamentID,
		MatchType:    matchType,
		TeamA:        teamA,
		TeamB:        teamB,
		Status:       "in_progress",
		Sets:         []event.MatchSet{},
		Stage:        stage,
		DivisionID:   divisionID,
	}

	// Add match to event
	t.AddMatch(*m)

	// Save match via repository
	if err := uc.matchRepo.Save(ctx, m); err != nil {
		return nil, err
	}

	return m, nil
}

// determinePlayerDivision finds which division a match belongs to based on player Elo ratings.
func (uc *CreateMatchUseCase) determinePlayerDivision(ctx context.Context, t *event.Event, teamA, teamB []*player.Player, matchType string) string {
	if len(t.DivisionRules) == 0 || len(teamA) == 0 {
		return ""
	}

	// Get all divisions to check Elo ranges
	divisions, err := uc.divisionRepo.GetAll(ctx)
	if err != nil || len(divisions) == 0 {
		return ""
	}

	// Use first player's Elo for division lookup
	playerElo := teamA[0].SinglesElo
	if matchType == "doubles" {
		playerElo = teamA[0].DoublesElo
	}

	// Find matching division by Elo range
	for _, div := range divisions {
		if div.ContainsElo(playerElo) {
			return div.ID
		}
	}

	return ""
}
