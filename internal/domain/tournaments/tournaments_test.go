package tournaments_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/tournaments"
)

func TestPlayerEnrolledEvent_EventName(t *testing.T) {
	evt := tournaments.PlayerEnrolledEvent{
		TournamentID: "tourn-1",
		PlayerID:     "player-1",
	}

	if evt.EventName() != tournaments.PlayerEnrolledEventName {
		t.Errorf("expected EventName '%s', got '%s'", tournaments.PlayerEnrolledEventName, evt.EventName())
	}
}

func TestInMemoryDispatcher_Dispatch(t *testing.T) {
	dispatcher := tournaments.NewInMemoryDispatcher()

	var calledCount int
	var receivedEvent tournaments.Tournament

	handler := func(ctx context.Context, tr tournaments.Tournament) error {
		calledCount++
		receivedEvent = tr
		return nil
	}

	dispatcher.Subscribe(tournaments.PlayerEnrolledEventName, handler)

	evt := tournaments.PlayerEnrolledEvent{
		TournamentID: "t-100",
		PlayerID:     "p-200",
	}

	err := dispatcher.Dispatch(context.Background(), evt)
	if err != nil {
		t.Fatalf("expected no error from Dispatch, got %v", err)
	}

	if calledCount != 1 {
		t.Errorf("expected handler to be called once, got %d", calledCount)
	}

	if receivedEvent != evt {
		t.Errorf("expected received event %v, got %v", evt, receivedEvent)
	}
}

func TestInMemoryDispatcher_Dispatch_ErrorPropagation(t *testing.T) {
	dispatcher := tournaments.NewInMemoryDispatcher()

	expectedErr := errors.New("handler failed")

	handler1 := func(ctx context.Context, tr tournaments.Tournament) error {
		return expectedErr
	}

	handler2Called := false
	handler2 := func(ctx context.Context, tr tournaments.Tournament) error {
		handler2Called = true
		return nil
	}

	dispatcher.Subscribe(tournaments.PlayerEnrolledEventName, handler1)
	dispatcher.Subscribe(tournaments.PlayerEnrolledEventName, handler2)

	evt := tournaments.PlayerEnrolledEvent{}

	err := dispatcher.Dispatch(context.Background(), evt)
	if err != expectedErr {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}

	if handler2Called {
		t.Error("expected handler2 not to be called after handler1 failed")
	}
}

func TestInMemoryDispatcher_DispatchAsync(t *testing.T) {
	dispatcher := tournaments.NewInMemoryDispatcher()

	var wg sync.WaitGroup
	wg.Add(1)

	var receivedEvent tournaments.Tournament

	handler := func(ctx context.Context, tr tournaments.Tournament) error {
		defer wg.Done()
		receivedEvent = tr
		return nil
	}

	dispatcher.Subscribe(tournaments.PlayerEnrolledEventName, handler)

	evt := tournaments.PlayerEnrolledEvent{
		TournamentID: "t-300",
		PlayerID:     "p-400",
	}

	dispatcher.DispatchAsync(context.Background(), evt)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Async execution completed
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for async handler execution")
	}

	if receivedEvent != evt {
		t.Errorf("expected received event %v, got %v", evt, receivedEvent)
	}
}

func TestInMemoryDispatcher_NoHandlers(t *testing.T) {
	dispatcher := tournaments.NewInMemoryDispatcher()

	evt := tournaments.PlayerEnrolledEvent{
		TournamentID: "t-1",
		PlayerID:     "p-1",
	}

	err := dispatcher.Dispatch(context.Background(), evt)
	if err != nil {
		t.Errorf("expected no error when no handlers subscribed, got %v", err)
	}
}
