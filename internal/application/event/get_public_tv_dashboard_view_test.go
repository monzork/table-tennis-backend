package event

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

func TestGetPublicTVDashboardViewUseCase_Execute(t *testing.T) {
	newUC := func(repo *mockRepo, divRepo *mockDivisionRepo, playerRepo *mockPlayerRepo) *GetPublicTVDashboardViewUseCase {
		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		return NewGetPublicTVDashboardViewUseCase(getByID, leaderboardUC, divisionUC)
	}

	build := func(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) {
		return []BoardCard{{PlayerAName: "Alice", PlayerBName: "Bob"}},
			[]BoardCard{{PlayerAName: "Carl", PlayerBName: "Dave"}},
			[]BoardCard{{PlayerAName: "Eve", PlayerBName: "Frank"}}
	}

	t.Run("happy path aggregates board cards without search filter", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		view, err := uc.Execute(context.Background(), "t1", "", build, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(view.Scheduled) != 1 || len(view.InProgress) != 1 || len(view.Finished) != 1 {
			t.Errorf("expected all cards passed through unfiltered, got %+v", view)
		}
	})

	t.Run("event error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		_, err := uc.Execute(context.Background(), "missing", "", build, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("search filters out non-matching cards from all three buckets", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		view, err := uc.Execute(context.Background(), "t1", "Alice", build, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(view.Scheduled) != 1 {
			t.Errorf("expected scheduled to still match Alice, got %+v", view.Scheduled)
		}
		if len(view.InProgress) != 0 || len(view.Finished) != 0 {
			t.Errorf("expected in-progress and finished to be filtered out, got %+v %+v", view.InProgress, view.Finished)
		}
	})
}
