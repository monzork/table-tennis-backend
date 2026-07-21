package event

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestGetEditFormViewUseCase_Execute(t *testing.T) {
	t.Run("happy path aggregates event, players, divisions", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo := &mockDivisionRepo{divisions: []*divisionDomain.Division{{ID: "d1", Name: "Open"}}}
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "B"}

		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		uc := NewGetEditFormViewUseCase(getByID, leaderboardUC, divisionUC)

		view, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if view.Event.ID != "t1" {
			t.Errorf("expected event t1, got %s", view.Event.ID)
		}
		if len(view.Divisions) != 1 {
			t.Errorf("expected 1 division, got %d", len(view.Divisions))
		}
	})

	t.Run("event error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		uc := NewGetEditFormViewUseCase(getByID, leaderboardUC, divisionUC)

		_, err := uc.Execute(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
