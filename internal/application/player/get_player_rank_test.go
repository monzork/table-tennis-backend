package player_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/player"
	playerDomain "table-tennis-backend/internal/domain/player"
)

type fakeRankRepo struct {
	playerDomain.Repository
	players []*playerDomain.Player
	err     error
}

func (f *fakeRankRepo) GetAllSingles(ctx context.Context) ([]*playerDomain.Player, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.players, nil
}

func (f *fakeRankRepo) GetAllDoubles(ctx context.Context) ([]*playerDomain.Player, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.players, nil
}

func TestGetPlayerRankUseCase_Execute(t *testing.T) {
	players := []*playerDomain.Player{
		{ID: "p1"},
		{ID: "p2", Inactive: true},
		{ID: "p3"},
		{ID: "p4"},
	}

	t.Run("excludes inactive players from position and total", func(t *testing.T) {
		repo := &fakeRankRepo{players: players}
		uc := player.NewGetPlayerRankUseCase(repo)

		rank, total, err := uc.Execute(context.Background(), "p3", "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if total != 3 {
			t.Errorf("expected total 3 (inactive p2 excluded), got %d", total)
		}
		if rank != 2 {
			t.Errorf("expected rank 2 among active players (p1, p3, p4), got %d", rank)
		}
	})

	t.Run("inactive player has no rank of its own", func(t *testing.T) {
		repo := &fakeRankRepo{players: players}
		uc := player.NewGetPlayerRankUseCase(repo)

		rank, total, err := uc.Execute(context.Background(), "p2", "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rank != 0 {
			t.Errorf("expected rank 0 for inactive player, got %d", rank)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
	})

	t.Run("player not found returns zero rank", func(t *testing.T) {
		repo := &fakeRankRepo{players: players}
		uc := player.NewGetPlayerRankUseCase(repo)

		rank, total, err := uc.Execute(context.Background(), "missing", "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rank != 0 {
			t.Errorf("expected rank 0 for unknown player, got %d", rank)
		}
		if total != 3 {
			t.Errorf("expected total 3, got %d", total)
		}
	})

	t.Run("doubles rank type reads doubles roster", func(t *testing.T) {
		repo := &fakeRankRepo{players: players}
		uc := player.NewGetPlayerRankUseCase(repo)

		rank, total, err := uc.Execute(context.Background(), "p4", "doubles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if rank != 3 || total != 3 {
			t.Errorf("expected rank 3 of 3, got rank=%d total=%d", rank, total)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		repo := &fakeRankRepo{err: errors.New("boom")}
		uc := player.NewGetPlayerRankUseCase(repo)

		if _, _, err := uc.Execute(context.Background(), "p1", "singles"); err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}
