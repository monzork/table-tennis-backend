package match

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"table-tennis-backend/internal/domain/event"
)

type UpdateMatchScoreUseCase struct {
	matchRepo      event.MatchRepository
	tournamentRepo event.Repository
}

func NewUpdateMatchScoreUseCase(matchRepo event.MatchRepository, tournamentRepo event.Repository) *UpdateMatchScoreUseCase {
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
func ParseSetScores(raw []string) ([]event.MatchSet, error) {
	var sets []event.MatchSet
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
		sets = append(sets, event.MatchSet{Number: i + 1, ScoreA: a, ScoreB: b})
	}
	return sets, nil
}

func (uc *UpdateMatchScoreUseCase) Execute(
	ctx context.Context,
	matchIDStr string,
	rawScores []string,
	tournamentIDStr string,
	stage string,
	isForfeit bool,
) error {
	sets, err := ParseSetScores(rawScores)
	if err != nil {
		return err
	}

	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	if t.Status == "finished" {
		return fmt.Errorf("cannot update score of a finished event")
	}

	stageRule := t.GetEffectiveStageRule(stage)

	return uc.matchRepo.UpdateScore(ctx, matchIDStr, sets, stageRule, isForfeit)
}

// ExecuteWalkover records a match as won by walkover: forfeitingSide ("A"
// or "B") defaulted, so the other side wins. The winning sets are built
// here from the stage rule (PointsToWin-0 for just enough sets to clinch
// the best-of), so it works for any match -- including a team matchup
// whose rubbers were never created because a team has no players yet.
// Like any forfeit, no sets are persisted and no Elo changes.
func (uc *UpdateMatchScoreUseCase) ExecuteWalkover(
	ctx context.Context,
	matchIDStr string,
	tournamentIDStr string,
	stage string,
	forfeitingSide string,
) error {
	if forfeitingSide != "A" && forfeitingSide != "B" {
		return fmt.Errorf("invalid forfeiting side %q: cannot record walkover", forfeitingSide)
	}

	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	if t.Status == "finished" {
		return fmt.Errorf("cannot update score of a finished event")
	}

	stageRule := t.GetEffectiveStageRule(stage)
	points := stageRule.PointsToWin
	if points <= 0 {
		points = 11
	}
	if stageRule.PointsMargin > points {
		points = stageRule.PointsMargin
	}
	setsToWin := stageRule.BestOf/2 + 1

	sets := make([]event.MatchSet, setsToWin)
	for i := range sets {
		sets[i] = event.MatchSet{Number: i + 1, ScoreA: points}
		if forfeitingSide == "A" {
			sets[i] = event.MatchSet{Number: i + 1, ScoreB: points}
		}
	}

	return uc.matchRepo.UpdateScore(ctx, matchIDStr, sets, stageRule, true)
}

// ExecuteDoubleForfeit records a match as a no-contest: both sides
// defaulted, so it finishes with no winner, no Elo, and no bracket
// advancement.
func (uc *UpdateMatchScoreUseCase) ExecuteDoubleForfeit(
	ctx context.Context,
	matchIDStr string,
	tournamentIDStr string,
) error {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return fmt.Errorf("event not found: %w", err)
	}

	if t.Status == "finished" {
		return fmt.Errorf("cannot update score of a finished event")
	}

	return uc.matchRepo.DoubleForfeit(ctx, matchIDStr)
}
