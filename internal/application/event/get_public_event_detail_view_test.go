package event

import (
	"context"
	"errors"
	"testing"
	"time"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestGetPublicEventDetailViewUseCase_Execute(t *testing.T) {
	newUC := func(repo *mockRepo, divRepo *mockDivisionRepo, playerRepo *mockPlayerRepo) *GetPublicEventDetailViewUseCase {
		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		return NewGetPublicEventDetailViewUseCase(getByID, leaderboardUC, divisionUC)
	}

	t.Run("happy path builds JSONLD, matches, and referee names", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1200}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1100}
		refID := "p1"
		m := tournamentDomain.Match{
			ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, RefereeID: &refID,
		}
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Name: `Cup "2026"`, Type: "singles",
			StartDate: time.Now(), EndDate: time.Now().Add(24 * time.Hour),
			Participants: []*playerDomain.Player{p1, p2},
			Matches:      []tournamentDomain.Match{m},
		}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = p1

		uc := newUC(repo, divRepo, playerRepo)
		build := func(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) {
			return nil, nil, nil
		}

		view, err := uc.Execute(context.Background(), "t1", "all", "", "https://example.com/t1", build, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if view.JSONLD == "" {
			t.Errorf("expected non-empty JSONLD")
		}
		if view.RefereeNames["p1"] == "" {
			t.Errorf("expected referee name to be resolved for p1")
		}
	})

	t.Run("event error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		build := func(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) {
			return nil, nil, nil
		}
		_, err := uc.Execute(context.Background(), "missing", "all", "", "", build, nil)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("adds virtual scheduled matches from buildBoardCards for singles", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1200}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1100}
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Type: "singles",
			StartDate: time.Now(), EndDate: time.Now().Add(24 * time.Hour),
			Participants: []*playerDomain.Player{p1, p2},
		}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		build := func(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) {
			return []BoardCard{{MatchID: "", P1Id: "p1", P2Id: "p2", Stage: "final"}}, nil, nil
		}

		view, err := uc.Execute(context.Background(), "t1", "all", "", "", build, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		found := false
		for _, m := range view.Event.Matches {
			if m.Stage == "final" && len(m.TeamA) > 0 && m.TeamA[0].ID == "p1" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected virtual match to be appended, got %+v", view.Event.Matches)
		}
	})

	t.Run("player search filters matches by name terms", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B"}
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", StartDate: time.Now(), EndDate: time.Now().Add(time.Hour),
			Participants: []*playerDomain.Player{p1, p2},
			Matches: []tournamentDomain.Match{
				{ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			},
		}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		build := func(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) {
			return nil, nil, nil
		}

		view, err := uc.Execute(context.Background(), "t1", "all", "nonexistent", "", build, nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(view.Event.Matches) != 0 {
			t.Errorf("expected no matches for nonexistent search, got %+v", view.Event.Matches)
		}
	})
}
