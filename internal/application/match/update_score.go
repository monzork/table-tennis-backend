package match

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

type UpdateMatchScoreUseCase struct {
	matchRepo    *bun.MatchRepository
	stageRuleRepo func(ctx context.Context, tournamentID uuid.UUID, stage string) (*bun.StageRuleModel, error)
}

func NewUpdateMatchScoreUseCase(matchRepo *bun.MatchRepository) *UpdateMatchScoreUseCase {
	return &UpdateMatchScoreUseCase{matchRepo: matchRepo}
}

// SetScoreInput is one set "A-B" e.g. "11-8"
type SetScoreInput struct {
	Number int
	ScoreA int
	ScoreB int
}

// ParseSetScores parses form values like ["11-8", "11-5", "9-11"] into MatchSet slice.
func ParseSetScores(raw []string) ([]tournament.MatchSet, error) {
	var sets []tournament.MatchSet
	for i, val := range raw {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		parts := strings.Split(val, "-")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid set score format: %q (expected 'A-B')", val)
		}
		a, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return nil, err
		}
		b, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, err
		}
		sets = append(sets, tournament.MatchSet{Number: i + 1, ScoreA: a, ScoreB: b})
	}
	return sets, nil
}

func (uc *UpdateMatchScoreUseCase) Execute(
	ctx context.Context,
	matchIDStr string,
	rawScores []string,
	tournamentIDStr string,
	stage string,
) error {
	matchID, err := uuid.Parse(matchIDStr)
	if err != nil {
		return fmt.Errorf("invalid match id: %w", err)
	}
	tournamentID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		return fmt.Errorf("invalid tournament id: %w", err)
	}

	sets, err := ParseSetScores(rawScores)
	if err != nil {
		return err
	}

	// Load the stage rule for this stage
	stageRule, err := bun.GetStageRule(ctx, uc.matchRepo.DB(), tournamentID, stage)
	if err != nil {
		// Fallback to default
		stageRule = &bun.StageRuleModel{BestOf: 5, PointsToWin: 11, PointsMargin: 2, Stage: stage}
	}

	return uc.matchRepo.UpdateScore(ctx, matchID, sets, stageRule)
}
