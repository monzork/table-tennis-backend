package event

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/division"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

func noopBuildBoardCards(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) {
	return []BoardCard{{DivisionName: "Open"}}, nil, nil
}

func noopFilterBoardCards(cards []BoardCard, q string, divs []string) []BoardCard {
	return cards
}

func TestGetBoardViewUseCase_Execute(t *testing.T) {
	t.Run("happy path aggregates event, divisions, and cards", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Groups: []tournamentDomain.Group{{ID: "g1"}}}
		divRepo := &mockDivisionRepo{divisions: []*divisionDomain.Division{{ID: "d1", Name: "Open"}}}
		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		uc := NewGetBoardViewUseCase(getByID, divisionUC)

		view, err := uc.Execute(context.Background(), "t1", "", nil, noopBuildBoardCards, noopFilterBoardCards)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if view.Event.ID != "t1" {
			t.Errorf("expected event t1, got %s", view.Event.ID)
		}
		if len(view.AllDivs) != 1 || view.AllDivs[0] != "Open" {
			t.Errorf("expected AllDivs to contain Open, got %v", view.AllDivs)
		}
	})

	t.Run("event error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		divRepo := &mockDivisionRepo{}
		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		uc := NewGetBoardViewUseCase(getByID, divisionUC)

		_, err := uc.Execute(context.Background(), "missing", "", nil, noopBuildBoardCards, noopFilterBoardCards)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("filters applied when query or divs present", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo := &mockDivisionRepo{}
		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		uc := NewGetBoardViewUseCase(getByID, divisionUC)

		filterCalled := false
		filter := func(cards []BoardCard, q string, divs []string) []BoardCard {
			filterCalled = true
			return cards
		}

		_, err := uc.Execute(context.Background(), "t1", "Search", []string{"Open"}, noopBuildBoardCards, filter)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if !filterCalled {
			t.Errorf("expected filter function to be invoked when query is set")
		}
	})
}
