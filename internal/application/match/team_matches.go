package match

import (
	"context"
	"errors"

	"table-tennis-backend/internal/domain/event"
)

type TeamMatchOrchestratorUseCase struct {
	matchRepo event.MatchRepository
}

func NewTeamMatchOrchestratorUseCase(matchRepo event.MatchRepository) *TeamMatchOrchestratorUseCase {
	return &TeamMatchOrchestratorUseCase{
		matchRepo: matchRepo,
	}
}

// EnsureTeamSubMatches checks if a team match has its sub-matches created.
// If they do not exist, it generates them via the domain repository interface.
func (uc *TeamMatchOrchestratorUseCase) EnsureTeamSubMatches(ctx context.Context, matchID string, teamA, teamB *event.Team, teamFormat string, stage string) error {
	subs, err := uc.matchRepo.GetSubMatches(ctx, matchID)
	if err != nil {
		return err
	}
	if len(subs) > 0 {
		return nil // Already initialized
	}

	if teamA == nil || len(teamA.Players) == 0 || teamB == nil || len(teamB.Players) == 0 {
		return errors.New("teams must have players to create sub-matches")
	}

	var teamAIDs, teamBIDs []string
	for _, p := range teamA.Players {
		teamAIDs = append(teamAIDs, p.ID)
	}
	for _, p := range teamB.Players {
		teamBIDs = append(teamBIDs, p.ID)
	}

	return uc.matchRepo.CreateSubMatches(ctx, event.CreateSubMatchesCommand{
		ParentMatchID: matchID,
		TournamentID:  teamA.TournamentID,
		Stage:         stage,
		TeamFormat:    teamFormat,
		TeamAPlayers:  teamAIDs,
		TeamBPlayers:  teamBIDs,
	})
}

// UpdateTeamSquads assigns specific players to the sub-matches of a team match.
func (uc *TeamMatchOrchestratorUseCase) UpdateTeamSquads(ctx context.Context, parentMatchID string, squadA, squadB []string, teamFormat string, stage string) error {
	subs, err := uc.matchRepo.GetSubMatches(ctx, parentMatchID)
	if err != nil {
		return err
	}

	if len(subs) == 0 {
		return errors.New("sub-matches do not exist, please initialize them first")
	}

	if len(squadA) < 3 || len(squadB) < 3 {
		return errors.New("both squads must have at least 3 players selected")
	}

	p1A, p2A, p3A := squadA[0], squadA[1], squadA[2]
	p1B, p2B, p3B := squadB[0], squadB[1], squadB[2]

	if teamFormat == "" {
		teamFormat = "olympic"
	}

	var assignments []event.SubMatchSquadAssignment
	for _, sub := range subs {
		var teamAP1, teamAP2, teamBP1, teamBP2 string
		if teamFormat == "olympic" {
			switch sub.RoundNumber {
			case 1:
				teamAP1, teamAP2 = p1A, p2A
				teamBP1, teamBP2 = p1B, p2B
			case 2:
				teamAP1, teamBP1 = p3A, p3B
			case 3:
				teamAP1, teamBP1 = p1A, p1B
			case 4:
				teamAP1, teamBP1 = p2A, p2B
			case 5:
				teamAP1, teamBP1 = p3A, p1B
			}
		} else {
			switch sub.RoundNumber {
			case 1:
				teamAP1, teamBP1 = p1A, p1B
			case 2:
				teamAP1, teamBP1 = p2A, p2B
			case 3:
				teamAP1, teamBP1 = p3A, p3B
			case 4:
				teamAP1, teamBP1 = p1A, p2B
			case 5:
				teamAP1, teamBP1 = p2A, p1B
			}
		}

		assignments = append(assignments, event.SubMatchSquadAssignment{
			SubMatchID:     sub.ID,
			TeamAPlayer1ID: teamAP1,
			TeamAPlayer2ID: teamAP2,
			TeamBPlayer1ID: teamBP1,
			TeamBPlayer2ID: teamBP2,
		})
	}

	return uc.matchRepo.UpdateSubMatchSquads(ctx, event.UpdateSubMatchSquadsCommand{
		ParentMatchID: parentMatchID,
		Assignments:   assignments,
	})
}
