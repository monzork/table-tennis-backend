package match

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/event"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
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
// If they do not exist, it generates them according to the teamFormat.
// Note: This still relies heavily on bun internally since the save mechanism
// for batches isn't on the domain repo yet, but this extracts the logic.
func (uc *TeamMatchOrchestratorUseCase) EnsureTeamSubMatches(ctx context.Context, matchID string, teamA, teamB *event.Team, teamFormat string, stage string, db *bun.DB) error {
	parentUUID, err := uuid.Parse(matchID)
	if err != nil {
		return err
	}

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

	if teamFormat == "" {
		teamFormat = "olympic"
	}

	tUUID, _ := uuid.Parse(teamA.TournamentID)

	for order := 1; order <= 5; order++ {
		subID := uuid.New()
		matchType := "singles"
		if teamFormat == "olympic" && order == 1 {
			matchType = "doubles"
		}

		p1ID, _ := uuid.Parse(teamA.Players[0].ID)
		p2ID, _ := uuid.Parse(teamB.Players[0].ID)

		// For now, we still insert raw models here to bulk insert since `matchRepo.Save` only handles one Match.
		// Future step: Add `SaveBatch([]*event.Match)` to `event.MatchRepository`.
		subModel := map[string]interface{}{
			"id":                 subID,
			"event_id":           tUUID,
			"match_type":         matchType,
			"team_a_player_1_id": p1ID,
			"team_b_player_1_id": p2ID,
			"status":             "scheduled",
			"stage":              stage,
			"round_number":       order,
			"team_match_id":      parentUUID,
		}

		_, err := db.NewInsert().Model(&subModel).TableExpr("matches").Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// UpdateTeamSquads assigns specific players to the sub-matches of a team match.
func (uc *TeamMatchOrchestratorUseCase) UpdateTeamSquads(ctx context.Context, parentMatchID string, squadA, squadB []string, teamFormat string, stage string, db *bun.DB) error {
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

		uAP1, _ := uuid.Parse(teamAP1)
		uBP1, _ := uuid.Parse(teamBP1)

		var teamAP2Ptr, teamBP2Ptr *uuid.UUID
		if teamAP2 != "" {
			u, _ := uuid.Parse(teamAP2)
			teamAP2Ptr = &u
		}
		if teamBP2 != "" {
			u, _ := uuid.Parse(teamBP2)
			teamBP2Ptr = &u
		}

		// Still relying on raw bun here for bulk update efficiency
		subUUID, _ := uuid.Parse(sub.ID)
		_, err := db.NewUpdate().Table("matches").
			Set("team_a_player_1_id = ?", uAP1).
			Set("team_a_player_2_id = ?", teamAP2Ptr).
			Set("team_b_player_1_id = ?", uBP1).
			Set("team_b_player_2_id = ?", teamBP2Ptr).
			Where("id = ?", subUUID).
			Exec(ctx)

		if err != nil {
			return err
		}
	}

	return nil
}
