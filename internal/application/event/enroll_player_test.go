package event

import (
	"context"
	"errors"
	"testing"
)

func TestEnrollPlayerUseCase_Execute(t *testing.T) {
	t.Run("success with dispatcher", func(t *testing.T) {
		repo := newMockRepo()
		dispatcher := &mockDispatcher{}
		uc := NewEnrollPlayerUseCase(repo, dispatcher)

		err := uc.Execute(context.Background(), "t1", "p1", 1000, 1000)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(dispatcher.dispatchedAsync) != 1 {
			t.Errorf("expected 1 dispatched event, got %d", len(dispatcher.dispatchedAsync))
		}
	})

	t.Run("success with nil dispatcher", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewEnrollPlayerUseCase(repo, nil)

		err := uc.Execute(context.Background(), "t1", "p1", 1000, 1000)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("repo error propagates and skips dispatch", func(t *testing.T) {
		repo := newMockRepo()
		repo.addParticipErr = errors.New("db error")
		dispatcher := &mockDispatcher{}
		uc := NewEnrollPlayerUseCase(repo, dispatcher)

		err := uc.Execute(context.Background(), "t1", "p1", 1000, 1000)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if len(dispatcher.dispatchedAsync) != 0 {
			t.Errorf("expected no dispatch on error, got %d", len(dispatcher.dispatchedAsync))
		}
	})
}
