package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestRegenerateGroupSeedsUseCase_Execute(t *testing.T) {
	newUC := func() (*RegenerateGroupSeedsUseCase, *mockRepo, *mockMatchRepo, *mockDivisionRepo) {
		repo := newMockRepo()
		matchRepo := &mockMatchRepo{}
		divRepo := &mockDivisionRepo{}
		uc := NewRegenerateGroupSeedsUseCase(repo, matchRepo, divRepo)
		return uc, repo, matchRepo, divRepo
	}

	t.Run("happy path regenerates and deletes matches", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{
			ID: "t1", Status: "scheduled", Format: "round_robin", SkipElo: true,
			Participants: []*playerDomain.Player{p1},
		}

		if err := uc.Execute(context.Background(), "t1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if matchRepo.deleteByTournErr != nil {
			t.Fatalf("unexpected delete err set")
		}
		if repo.updateGroupsCalls != 1 {
			t.Errorf("expected 1 UpdateGroups call, got %d", repo.updateGroupsCalls)
		}
	})

	t.Run("get error propagates", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.getErr = errors.New("db error")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("has activity error propagates", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		matchRepo.hasActivityErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("division get all error propagates", func(t *testing.T) {
		uc, repo, _, divRepo := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		divRepo.getAllErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("finished event rejects regen", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "finished"}
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("manual seeding locked rejects regen", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", ManualSeedingLocked: true}
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("existing activity rejects regen", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1"}
		matchRepo.hasActivity = true
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("delete by tournament error propagates", func(t *testing.T) {
		uc, repo, matchRepo, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Format: "elimination", SkipElo: true}
		matchRepo.deleteByTournErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("update groups error propagates", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Format: "elimination", SkipElo: true}
		repo.updateGroupsErr = errors.New("boom")
		if err := uc.Execute(context.Background(), "t1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
