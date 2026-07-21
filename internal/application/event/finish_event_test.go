package event

import (
	"context"
	"errors"
	"testing"
	"time"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestFinishTournamentUseCase_Execute(t *testing.T) {
	newUC := func() (*FinishTournamentUseCase, *mockRepo, *mockMatchRepo, *mockPlayerRepo) {
		repo := newMockRepo()
		matchRepo := &mockMatchRepo{}
		playerRepo := newMockPlayerRepo()
		uc := NewFinishTournamentUseCase(repo, matchRepo, playerRepo)
		return uc, repo, matchRepo, playerRepo
	}

	t.Run("already finished returns error", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "finished"}

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("get error propagates", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.getErr = errors.New("db error")

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("unfinished count error propagates", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "in_progress"}
		matchRepo.unfinishedErr = errors.New("boom")

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("unfinished matches block finish", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "in_progress"}
		matchRepo.unfinishedCount = 2

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("finished count error propagates", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "in_progress"}
		matchRepo.finishedErr = errors.New("boom")

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("not all rounds played blocks finish", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "B", LastName: "B"}
		p3 := &playerDomain.Player{ID: "p3", FirstName: "C", LastName: "C"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "in_progress", Participants: []*playerDomain.Player{p1, p2, p3}}
		matchRepo.finishedCount = 1 // needs at least 2 (participantCount-1)

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("happy path elimination format with elo, sets winner and finishes", func(t *testing.T) {
		uc, repo, matchRepo, playerRepo := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1000}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1000}
		playerRepo.players["p1"] = p1
		playerRepo.players["p2"] = p2

		now := time.Now()
		finalMatch := tournamentDomain.Match{
			ID:         "m1",
			MatchType:  "singles",
			TeamA:      []*playerDomain.Player{p1},
			TeamB:      []*playerDomain.Player{p2},
			Status:     "finished",
			Stage:      "final",
			WinnerTeam: "A",
			UpdatedAt:  &now,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Status:       "in_progress",
			Format:       "elimination",
			Participants: []*playerDomain.Player{p1, p2},
			Matches:      []tournamentDomain.Match{finalMatch},
		}
		matchRepo.finishedCount = 1 // participantCount - 1 = 1

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		got := repo.events["t1"]
		if got.Status != "finished" {
			t.Errorf("expected status finished, got %s", got.Status)
		}
		if got.WinnerName == "" {
			t.Errorf("expected winner name to be set")
		}
		if got.Metrics == nil {
			t.Errorf("expected metrics to be calculated")
		}
	})

	t.Run("skip elo events do not touch elo but still finish", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Status:       "in_progress",
			Format:       "round_robin",
			SkipElo:      true,
			Participants: []*playerDomain.Player{p1},
		}
		matchRepo.finishedCount = 0

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.events["t1"].Status != "finished" {
			t.Errorf("expected finished status")
		}
	})

	t.Run("snapshots error propagates when elo not skipped", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Status:       "in_progress",
			Format:       "elimination",
			Participants: []*playerDomain.Player{p1},
		}
		matchRepo.finishedCount = 0
		repo.snapshotsErr = errors.New("boom")

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("round robin winner determined via standings", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", SinglesElo: 1000}
		p2 := &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", SinglesElo: 1000}
		now := time.Now()
		m := tournamentDomain.Match{
			ID: "m1", MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2},
			Status: "finished", Stage: "group", WinnerTeam: "A", UpdatedAt: &now,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Status: "in_progress", Format: "round_robin", SkipElo: true,
			Participants: []*playerDomain.Player{p1, p2},
			Matches:      []tournamentDomain.Match{m},
		}
		matchRepo.finishedCount = 1

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.events["t1"].WinnerName == "" {
			t.Errorf("expected winner name to be determined from standings")
		}
	})

	t.Run("doubles match type updates doubles elo", func(t *testing.T) {
		uc, repo, matchRepo, playerRepo := newUC()
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
			ID: "t1", Status: "in_progress", Format: "elimination", Type: "doubles",
			Participants: []*playerDomain.Player{p1, p2, p3, p4},
			Matches:      []tournamentDomain.Match{m},
		}
		matchRepo.finishedCount = 3

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.events["t1"].Status != "finished" {
			t.Errorf("expected status finished")
		}
	})

	t.Run("final update error propagates", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Status: "in_progress", Format: "elimination", SkipElo: true,
			Participants: []*playerDomain.Player{p1},
		}
		matchRepo.finishedCount = 0
		repo.updateErr = errors.New("boom")

		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
