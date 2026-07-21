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

func TestGetEventDetailViewUseCase_Execute(t *testing.T) {
	newUC := func(repo *mockRepo, divRepo *mockDivisionRepo, playerRepo *mockPlayerRepo) *GetEventDetailViewUseCase {
		getByID := NewGetTournamentByIDUseCase(repo, divRepo)
		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
		divisionUC := division.NewDivisionUseCase(divRepo)
		return NewGetEventDetailViewUseCase(getByID, leaderboardUC, divisionUC)
	}

	minElo := int16(0)
	maxElo := int16(1500)

	t.Run("happy path builds rows, seeds, groups, divisions, and available participants", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1400}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1200}
		team := &tournamentDomain.Team{ID: "team1", Name: "T1", Players: []*playerDomain.Player{p1}}

		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Type:         "singles",
			Participants: []*playerDomain.Player{p1, p2},
			Teams:        []*tournamentDomain.Team{team},
			Groups:       []tournamentDomain.Group{{ID: "g1", Name: "Open Bracket - Group A", Players: []*playerDomain.Player{p1, p2}}},
			Matches: []tournamentDomain.Match{
				{ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			},
		}
		repo.snapshots = []tournamentDomain.ParticipantSnapshot{{PlayerID: "p1", Pin: "1111"}}
		divRepo := &mockDivisionRepo{divisions: []*divisionDomain.Division{{ID: "d1", Name: "Div1", Category: "both", MinElo: minElo, MaxElo: &maxElo}}}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		view, err := uc.Execute(context.Background(), "t1", "all", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(view.ParticipantRows) != 2 {
			t.Fatalf("expected 2 participant rows, got %d", len(view.ParticipantRows))
		}
		if len(view.AvailableParticipants) != 1 || view.AvailableParticipants[0].ID != "p2" {
			t.Errorf("expected only p2 available (p1 assigned to team), got %+v", view.AvailableParticipants)
		}
		if view.PlayerPins["p1"] != "1111" {
			t.Errorf("expected pin 1111 for p1, got %s", view.PlayerPins["p1"])
		}
	})

	t.Run("event error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		_, err := uc.Execute(context.Background(), "missing", "all", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("filters matches by status and player search", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B"}
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Participants: []*playerDomain.Player{p1, p2},
			Matches: []tournamentDomain.Match{
				{ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
				{ID: "m2", Status: "finished", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			},
		}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		view, err := uc.Execute(context.Background(), "t1", "finished", "alice")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(view.Event.Matches) != 1 || view.Event.Matches[0].ID != "m2" {
			t.Fatalf("expected only finished match m2, got %+v", view.Event.Matches)
		}
	})

	t.Run("player search excludes non-matching matches", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B"}
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Participants: []*playerDomain.Player{p1, p2},
			Matches: []tournamentDomain.Match{
				{ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			},
		}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		view, err := uc.Execute(context.Background(), "t1", "all", "nonexistent")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(view.Event.Matches) != 0 {
			t.Fatalf("expected no matches for nonexistent player search, got %+v", view.Event.Matches)
		}
	})

	t.Run("doubles elo sort order used for doubles events", func(t *testing.T) {
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 900, DoublesElo: 1300}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1300, DoublesElo: 900}
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Type: "doubles", Participants: []*playerDomain.Player{p1, p2},
		}
		divRepo := &mockDivisionRepo{}
		playerRepo := newMockPlayerRepo()

		uc := newUC(repo, divRepo, playerRepo)
		view, err := uc.Execute(context.Background(), "t1", "all", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if view.ParticipantRows[0].Player.ID != "p1" {
			t.Errorf("expected p1 (higher doubles elo) seeded first, got %+v", view.ParticipantRows)
		}
	})
}
