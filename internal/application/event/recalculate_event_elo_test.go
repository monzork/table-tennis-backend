package event

import (
	"context"
	"errors"
	"testing"
	"time"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestRecalculateTournamentEloUseCase_Execute(t *testing.T) {
	newUC := func() (*RecalculateTournamentEloUseCase, *mockRepo, *mockPlayerRepo) {
		repo := newMockRepo()
		playerRepo := newMockPlayerRepo()
		uc := NewRecalculateTournamentEloUseCase(repo, &mockMatchRepo{}, playerRepo)
		return uc, repo, playerRepo
	}
	newUCWithMatchRepo := func() (*RecalculateTournamentEloUseCase, *mockRepo, *mockMatchRepo, *mockPlayerRepo) {
		repo := newMockRepo()
		matchRepo := &mockMatchRepo{}
		playerRepo := newMockPlayerRepo()
		uc := NewRecalculateTournamentEloUseCase(repo, matchRepo, playerRepo)
		return uc, repo, matchRepo, playerRepo
	}

	t.Run("get error propagates", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.getErr = errors.New("db error")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("skip elo event rejected", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", SkipElo: true}
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("snapshots error propagates", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		repo.snapshotsErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("happy path recalculates elo and updates event", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1000}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1000}
		playerRepo.players["p1"] = p1
		playerRepo.players["p2"] = p2

		now := time.Now()
		eloBefore1 := int16(1000)
		m := tournamentDomain.Match{
			ID: "m1", MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2},
			Status: "finished", Stage: "final", WinnerTeam: "A", UpdatedAt: &now,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Participants: []*playerDomain.Player{p1, p2},
			Matches: []tournamentDomain.Match{m},
		}
		repo.snapshots = []tournamentDomain.ParticipantSnapshot{
			{PlayerID: "p1", EloBeforeSingles: &eloBefore1},
		}

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.updateCalls != 1 {
			t.Errorf("expected event to be updated, got %d calls", repo.updateCalls)
		}
		if repo.events["t1"].Metrics == nil {
			t.Errorf("expected metrics recalculated")
		}
	})

	t.Run("final update error propagates", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		repo.updateErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("doubles match type updates doubles elo", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", DoublesElo: 1000}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", DoublesElo: 1000}
		p3 := &playerDomain.Player{ID: "p3", FirstName: "Carl", LastName: "C", DoublesElo: 1000}
		p4 := &playerDomain.Player{ID: "p4", FirstName: "Dave", LastName: "D", DoublesElo: 1000}
		playerRepo.players["p1"] = p1
		playerRepo.players["p2"] = p2
		playerRepo.players["p3"] = p3
		playerRepo.players["p4"] = p4

		now := time.Now()
		m := tournamentDomain.Match{
			ID: "m1", MatchType: "doubles", TeamA: []*playerDomain.Player{p1, p2}, TeamB: []*playerDomain.Player{p3, p4},
			Status: "finished", Stage: "final", WinnerTeam: "A", UpdatedAt: &now,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Type: "doubles",
			Participants: []*playerDomain.Player{p1, p2, p3, p4},
			Matches:      []tournamentDomain.Match{m},
		}

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("teams match type skipped in elo processing", func(t *testing.T) {
		uc, repo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A", SinglesElo: 1000}
		now := time.Now()
		m := tournamentDomain.Match{
			ID: "m1", MatchType: "teams", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{},
			Status: "finished", WinnerTeam: "A", UpdatedAt: &now,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Participants: []*playerDomain.Player{p1},
			Matches: []tournamentDomain.Match{m},
		}

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("FIDE-style: each match uses the start-of-event rating, not a compounding one", func(t *testing.T) {
		uc, repo, matchRepo, playerRepo := newUCWithMatchRepo()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1234}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1234}
		p3 := &playerDomain.Player{ID: "p3", FirstName: "Carl", LastName: "C", SinglesElo: 1234}
		playerRepo.players["p1"] = p1
		playerRepo.players["p2"] = p2
		playerRepo.players["p3"] = p3

		startElo := int16(1000)
		repo.snapshots = []tournamentDomain.ParticipantSnapshot{
			{PlayerID: "p1", EloBeforeSingles: &startElo},
			{PlayerID: "p2", EloBeforeSingles: &startElo},
			{PlayerID: "p3", EloBeforeSingles: &startElo},
		}

		t1 := time.Now()
		t2 := t1.Add(time.Minute)
		m1 := tournamentDomain.Match{
			ID: "m1", MatchType: "singles", Stage: "group",
			TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2},
			Status: "finished", WinnerTeam: "A", UpdatedAt: &t1,
		}
		m2 := tournamentDomain.Match{
			ID: "m2", MatchType: "singles", Stage: "group",
			TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p3},
			Status: "finished", WinnerTeam: "A", UpdatedAt: &t2,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Participants: []*playerDomain.Player{p1, p2, p3},
			Matches:      []tournamentDomain.Match{m1, m2},
		}

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		d1 := matchRepo.eloDeltas["m1"]
		d2 := matchRepo.eloDeltas["m2"]
		if d1[0] == nil || d2[0] == nil {
			t.Fatalf("expected both matches to have a recorded Elo delta, got %+v / %+v", d1, d2)
		}
		if *d1[0] != *d2[0] {
			t.Errorf("expected identical deltas for two matches played from the same starting rating against equally-rated opponents, got m1=%v m2=%v", *d1[0], *d2[0])
		}

		wantFinal := startElo + int16(*d1[0]) + int16(*d2[0])
		if p1.SinglesElo != wantFinal {
			t.Errorf("expected p1's final Elo to be start (%d) + both match deltas = %d, got %d", startElo, wantFinal, p1.SinglesElo)
		}
	})
}
