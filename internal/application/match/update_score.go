package match

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"table-tennis-backend/internal/domain/tournament"
)

type UpdateMatchScoreUseCase struct {
	matchRepo      tournament.MatchRepository
	tournamentRepo tournament.Repository
}

func NewUpdateMatchScoreUseCase(matchRepo tournament.MatchRepository, tournamentRepo tournament.Repository) *UpdateMatchScoreUseCase {
	return &UpdateMatchScoreUseCase{
		matchRepo:      matchRepo,
		tournamentRepo: tournamentRepo,
	}
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
	sets, err := ParseSetScores(rawScores)
	if err != nil {
		return err
	}

	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return fmt.Errorf("tournament not found: %w", err)
	}

	if t.Status == "finished" {
		return fmt.Errorf("cannot update score of a finished tournament")
	}

	// Load the stage rule for this stage from the tournament
	var stageRule tournament.StageRule
	foundRule := false
	for _, r := range t.StageRules {
		if r.Stage == stage {
			stageRule = r
			foundRule = true
			break
		}
	}
	if !foundRule {
		// Fallback to default WTT rules
		stageRule = tournament.StageRule{
			BestOf:       5,
			PointsToWin:  11,
			PointsMargin: 2,
			Stage:        stage,
		}
	}

	return uc.matchRepo.UpdateScore(ctx, matchIDStr, sets, stageRule)
}
