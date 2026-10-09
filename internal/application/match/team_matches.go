package match

import (
	"context"
	"errors"
	"fmt"

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
		EventID:       teamA.EventID,
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

	// Corbillon needs only the two singles players (A, B); the optional 3rd/4th
	// entries name the doubles pair, who may be any squad players (ITTF TM1 sheet).
	minSquad := 3
	if teamFormat == "corbillon" {
		minSquad = 2
	}
	if len(squadA) < minSquad || len(squadB) < minSquad {
		return fmt.Errorf("both squads must have at least %d players selected", minSquad)
	}

	at := func(squad []string, i int) string {
		if i < len(squad) {
			return squad[i]
		}
		return ""
	}
	p1A, p2A, p3A := at(squadA, 0), at(squadA, 1), at(squadA, 2)
	p1B, p2B, p3B := at(squadB, 0), at(squadB, 1), at(squadB, 2)
	// doubles pair (corbillon): defaults to the singles players
	d1A, d2A, d1B, d2B := p1A, p2A, p1B, p2B
	if at(squadA, 2) != "" && at(squadA, 3) != "" {
		d1A, d2A = squadA[2], squadA[3]
	}
	if at(squadB, 2) != "" && at(squadB, 3) != "" {
		d1B, d2B = squadB[2], squadB[3]
	}
	if teamFormat == "corbillon" && (d1A == d2A || d1B == d2B) {
		return errors.New("doubles pair must be two different players")
	}

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
		} else if teamFormat == "corbillon" {
			switch sub.RoundNumber {
			case 1:
				teamAP1, teamBP1 = p1A, p1B
			case 2:
				teamAP1, teamBP1 = p2A, p2B
			case 3:
				teamAP1, teamAP2 = d1A, d2A
				teamBP1, teamBP2 = d1B, d2B
			case 4:
				teamAP1, teamBP1 = p1A, p2B
			case 5:
				teamAP1, teamBP1 = p2A, p1B
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
