package event

import (
	"context"
	"errors"
	"testing"
	"time"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestEnrollPlayerUseCase_Execute(t *testing.T) {
	newFixtures := func() (*mockRepo, *mockPlayerRepo) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", AgeCategory: "open", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}
		return repo, playerRepo
	}

	t.Run("success with dispatcher", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		dispatcher := &mockDispatcher{}
		uc := NewEnrollPlayerUseCase(repo, playerRepo, dispatcher)

		err := uc.Execute(context.Background(), "t1", "p1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(dispatcher.dispatchedAsync) != 1 {
			t.Errorf("expected 1 dispatched event, got %d", len(dispatcher.dispatchedAsync))
		}
	})

	t.Run("success with nil dispatcher", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		uc := NewEnrollPlayerUseCase(repo, playerRepo, nil)

		err := uc.Execute(context.Background(), "t1", "p1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("repo error propagates and skips dispatch", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		repo.addParticipErr = errors.New("db error")
		dispatcher := &mockDispatcher{}
		uc := NewEnrollPlayerUseCase(repo, playerRepo, dispatcher)

		err := uc.Execute(context.Background(), "t1", "p1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if len(dispatcher.dispatchedAsync) != 0 {
			t.Errorf("expected no dispatch on error, got %d", len(dispatcher.dispatchedAsync))
		}
	})

	t.Run("event not found propagates", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		uc := NewEnrollPlayerUseCase(repo, playerRepo, nil)

		if err := uc.Execute(context.Background(), "missing-event", "p1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("player not found propagates", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		uc := NewEnrollPlayerUseCase(repo, playerRepo, nil)

		if err := uc.Execute(context.Background(), "t1", "missing-player"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("age-ineligible player is rejected and never added", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		repo.events["t13"] = &tournamentDomain.Event{ID: "t13", AgeCategory: "u13", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		dispatcher := &mockDispatcher{}
		uc := NewEnrollPlayerUseCase(repo, playerRepo, dispatcher)

		// playerRepo's "p1" fixture was born in 1990 -- an adult, ineligible
		// for a U13 event.
		err := uc.Execute(context.Background(), "t13", "p1")
		if err == nil {
			t.Fatal("expected error for age-ineligible player, got nil")
		}
		if len(dispatcher.dispatchedAsync) != 0 {
			t.Errorf("expected no dispatch when enrollment is rejected, got %d", len(dispatcher.dispatchedAsync))
		}
	})

	t.Run("age-eligible player is accepted into an age-category event", func(t *testing.T) {
		repo, playerRepo := newFixtures()
		repo.events["t13"] = &tournamentDomain.Event{ID: "t13", AgeCategory: "u13", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
		playerRepo.players["p-young"] = &playerDomain.Player{ID: "p-young", FirstName: "Young", LastName: "Player", Birthdate: time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)}
		uc := NewEnrollPlayerUseCase(repo, playerRepo, nil)

		if err := uc.Execute(context.Background(), "t13", "p-young"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}
