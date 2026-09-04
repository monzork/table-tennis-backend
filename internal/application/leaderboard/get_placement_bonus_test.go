package leaderboard_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/leaderboard"
)

type fakePlacementBonusRepo struct {
	bonuses map[string]float64
	err     error
}

func (f *fakePlacementBonusRepo) GetLatestTournamentPlacementBonuses(ctx context.Context, rankType string) (map[string]float64, error) {
	return f.bonuses, f.err
}

func TestGetPlacementBonusUseCase_Execute(t *testing.T) {
	repo := &fakePlacementBonusRepo{bonuses: map[string]float64{"champ": 64}}
	uc := leaderboard.NewGetPlacementBonusUseCase(repo)

	got, err := uc.Execute(context.Background(), "singles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["champ"] != 64 {
		t.Errorf("expected champ's bonus to be 64, got %v", got)
	}
}

func TestGetPlacementBonusUseCase_Execute_PropagatesError(t *testing.T) {
	repo := &fakePlacementBonusRepo{err: errors.New("boom")}
	uc := leaderboard.NewGetPlacementBonusUseCase(repo)

	if _, err := uc.Execute(context.Background(), "doubles"); err == nil {
		t.Fatal("expected error to propagate from the repository")
	}
}
