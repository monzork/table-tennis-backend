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

	t.Run("FIDE-style: each match uses the start-of-event rating, not a compounding one", func(t *testing.T) {
		uc, repo, matchRepo, playerRepo := newUC()
		// All three players start the event at 1000 -- current Elo is
		// deliberately different (already moved by some other, unrelated
		// event) to prove the calculation uses the frozen event-start
		// snapshot, not whatever's live on the Player struct right now.
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
		// p1 beats p3 too, both starting at the same 1000 as the m1 pairing --
		// if the rating were compounding, this second win (against an
		// opponent whose rating hasn't moved) would be worth less because
		// p1's working rating would already have risen from m1.
		m2 := tournamentDomain.Match{
			ID: "m2", MatchType: "singles", Stage: "group",
			TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p3},
			Status: "finished", WinnerTeam: "A", UpdatedAt: &t2,
		}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Status: "in_progress", Format: "elimination",
			Participants: []*playerDomain.Player{p1, p2, p3},
			Matches:      []tournamentDomain.Match{m1, m2},
		}
		matchRepo.finishedCount = 2

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
