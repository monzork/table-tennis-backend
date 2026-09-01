package leaderboard_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/leaderboard"
)

type fakePreviousEloRepo struct {
	snapshots map[string]int16
	err       error
}

func (f *fakePreviousEloRepo) GetPreviousEloSnapshots(ctx context.Context, rankType string) (map[string]int16, error) {
	return f.snapshots, f.err
}

func TestGetRankMovementUseCase_Execute(t *testing.T) {
	repo := &fakePreviousEloRepo{snapshots: map[string]int16{"p1": 1200}}
	uc := leaderboard.NewGetRankMovementUseCase(repo)

	got, err := uc.Execute(context.Background(), "singles")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["p1"] != 1200 {
		t.Errorf("expected p1's snapshot to be 1200, got %v", got)
	}
}

func TestGetRankMovementUseCase_Execute_PropagatesError(t *testing.T) {
	repo := &fakePreviousEloRepo{err: errors.New("boom")}
	uc := leaderboard.NewGetRankMovementUseCase(repo)

	if _, err := uc.Execute(context.Background(), "doubles"); err == nil {
		t.Fatal("expected error to propagate from the repository")
	}
}
