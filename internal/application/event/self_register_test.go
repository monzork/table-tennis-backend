package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestSelfRegisterUseCase_GetOpenTournaments(t *testing.T) {
	repo := newMockRepo()
	repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true, Status: "scheduled"}
	repo.events["t2"] = &tournamentDomain.Event{ID: "t2", RegistrationOpen: false}
	repo.events["t3"] = &tournamentDomain.Event{ID: "t3", RegistrationOpen: true, Status: "finished"}
	uc := NewSelfRegisterUseCase(repo, newMockPlayerRepo())

	open, err := uc.GetOpenTournaments(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(open) != 1 || open[0].ID != "t1" {
		t.Errorf("expected only t1 open, got %+v", open)
	}
}

func TestSelfRegisterUseCase_GetOpenTournaments_Error(t *testing.T) {
	repo := newMockRepo()
	repo.getAllErr = errors.New("db error")
	uc := NewSelfRegisterUseCase(repo, newMockPlayerRepo())

	if _, err := uc.GetOpenTournaments(context.Background()); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSelfRegisterUseCase_Execute(t *testing.T) {
	newUC := func() (*SelfRegisterUseCase, *mockRepo, *mockPlayerRepo) {
		repo := newMockRepo()
		playerRepo := newMockPlayerRepo()
		uc := NewSelfRegisterUseCase(repo, playerRepo)
		return uc, repo, playerRepo
	}

	t.Run("missing names returns error", func(t *testing.T) {
		uc, _, _ := newUC()
		_, _, err := uc.Execute(context.Background(), "t1", "", "", "", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("tournament not found", func(t *testing.T) {
		uc, _, _ := newUC()
		_, _, err := uc.Execute(context.Background(), "missing", "A", "", "B", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("registration closed", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: false}
		_, _, err := uc.Execute(context.Background(), "t1", "A", "", "B", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("finished event rejects registration", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true, Status: "finished"}
		_, _, err := uc.Execute(context.Background(), "t1", "A", "", "B", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("player search error propagates", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true}
		playerRepo.getAllErr = errors.New("boom")
		_, _, err := uc.Execute(context.Background(), "t1", "A", "", "B", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("matches existing player and registers", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true}
		existing := &playerDomain.Player{ID: "p1", FirstName: "alice", LastName: "smith", Country: "NI"}
		playerRepo.players["p1"] = existing

		got, name, err := uc.Execute(context.Background(), "t1", "Alice", "", "Smith", "", "ni", "", "", "", "", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if name != "alice smith" {
			t.Errorf("expected 'alice smith', got %s", name)
		}
		if len(got.Participants) != 1 || got.Participants[0].ID != "p1" {
			t.Errorf("expected p1 registered, got %+v", got.Participants)
		}
	})

	t.Run("creates new player when not found", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true}

		got, name, err := uc.Execute(context.Background(), "t1", "New", "", "Player", "", "NI", "Managua", "88881234", "2000-01-01", "F", "001-123456-0000A")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if name != "New Player" {
			t.Errorf("expected 'New Player', got %s", name)
		}
		if len(playerRepo.savedPlayers) != 1 {
			t.Fatalf("expected 1 new player saved, got %d", len(playerRepo.savedPlayers))
		}
		if got.Participants[0].SinglesElo != 500 {
			t.Errorf("expected default elo 500, got %d", got.Participants[0].SinglesElo)
		}
	})

	t.Run("new player invalid birthdate falls back to now", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true}

		_, _, err := uc.Execute(context.Background(), "t1", "New", "", "Player2", "", "", "", "", "not-a-date", "", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("new player save error propagates", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true}
		playerRepo.saveErr = errors.New("db error")

		_, _, err := uc.Execute(context.Background(), "t1", "New", "", "Player", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("already registered player rejected", func(t *testing.T) {
		uc, repo, playerRepo := newUC()
		existing := &playerDomain.Player{ID: "p1", FirstName: "alice", LastName: "smith"}
		playerRepo.players["p1"] = existing
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true, Participants: []*playerDomain.Player{existing}}

		_, _, err := uc.Execute(context.Background(), "t1", "Alice", "", "Smith", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("update error propagates", func(t *testing.T) {
		uc, repo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", RegistrationOpen: true}
		repo.updateErr = errors.New("db error")

		_, _, err := uc.Execute(context.Background(), "t1", "New", "", "Player", "", "", "", "", "", "", "")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
