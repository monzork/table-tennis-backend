package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestUpdateParticipantEloBeforeUseCase_Execute(t *testing.T) {
	t.Run("happy path updates elo then regenerates seeds", func(t *testing.T) {
		repo := newMockRepo()
		p1 := &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "A"}
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Format: "round_robin", SkipElo: true, Participants: []*playerDomain.Player{p1}}
		matchRepo := &mockMatchRepo{}
		divRepo := &mockDivisionRepo{}
		regen := NewRegenerateGroupSeedsUseCase(repo, matchRepo, divRepo)
		uc := NewUpdateParticipantEloBeforeUseCase(repo, regen)

		err := uc.Execute(context.Background(), "t1", "p1", 1200, 1100)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.updateGroupsCalls != 1 {
			t.Errorf("expected regenerate seeds to call UpdateGroups, got %d calls", repo.updateGroupsCalls)
		}
	})

	t.Run("update elo before error propagates without calling regenerate", func(t *testing.T) {
		repo := newMockRepo()
		repo.updateEloBeforeErr = errors.New("db error")
		matchRepo := &mockMatchRepo{}
		divRepo := &mockDivisionRepo{}
		regen := NewRegenerateGroupSeedsUseCase(repo, matchRepo, divRepo)
		uc := NewUpdateParticipantEloBeforeUseCase(repo, regen)

		err := uc.Execute(context.Background(), "t1", "p1", 1200, 1100)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("regenerate seeds error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("not found")
		matchRepo := &mockMatchRepo{}
		divRepo := &mockDivisionRepo{}
		regen := NewRegenerateGroupSeedsUseCase(repo, matchRepo, divRepo)
		uc := NewUpdateParticipantEloBeforeUseCase(repo, regen)

		err := uc.Execute(context.Background(), "t1", "p1", 1200, 1100)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
