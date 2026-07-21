package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
)

func TestAddGroupUseCase_Execute(t *testing.T) {
	t.Run("adds first group with letter A", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "scheduled"}
		uc := NewAddGroupUseCase(repo)

		if err := uc.Execute(context.Background(), "t1", "First Division"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		groups := repo.events["t1"].Groups
		if len(groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(groups))
		}
		if groups[0].Name != "First Division - Group A" {
			t.Errorf("expected 'First Division - Group A', got %s", groups[0].Name)
		}
	})

	t.Run("adds subsequent group with incremented letter", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:     "t1",
			Status: "scheduled",
			Groups: []tournamentDomain.Group{{Name: "First Division - Group A"}},
		}
		uc := NewAddGroupUseCase(repo)

		if err := uc.Execute(context.Background(), "t1", "First Division"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		groups := repo.events["t1"].Groups
		if len(groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(groups))
		}
		if groups[1].Name != "First Division - Group B" {
			t.Errorf("expected 'First Division - Group B', got %s", groups[1].Name)
		}
	})

	t.Run("finished event rejects add", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "finished"}
		uc := NewAddGroupUseCase(repo)

		err := uc.Execute(context.Background(), "t1", "First Division")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("get error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		uc := NewAddGroupUseCase(repo)

		if err := uc.Execute(context.Background(), "missing", "Div"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("update groups error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{ID: "t1", Status: "scheduled"}
		repo.updateGroupsErr = errors.New("boom")
		uc := NewAddGroupUseCase(repo)

		if err := uc.Execute(context.Background(), "t1", "Div"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
