package player_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/player"
	eventDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestGetPlayerPendingMatchesUseCase_Execute(t *testing.T) {
	p1 := &playerDomain.Player{ID: "p1", FirstName: "Ana"}
	opponent := &playerDomain.Player{ID: "p2", FirstName: "Beto"}

	t.Run("flattens pending matches across every event the player is in", func(t *testing.T) {
		eventRepo := &fakeEventRepo{
			events: []*eventDomain.Event{
				{
					ID: "e1",
					Matches: []eventDomain.Match{
						{ID: "m1", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{opponent}},
						{ID: "m2", Status: "finished", WinnerTeam: "A", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{opponent}},
					},
				},
				{
					ID: "e2",
					Matches: []eventDomain.Match{
						{ID: "m3", Status: "in_progress", TeamA: []*playerDomain.Player{opponent}, TeamB: []*playerDomain.Player{p1}},
					},
				},
			},
		}

		uc := player.NewGetPlayerPendingMatchesUseCase(eventRepo)
		pending, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pending) != 2 {
			t.Fatalf("expected 2 pending matches (finished one excluded), got %d: %+v", len(pending), pending)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		eventRepo := &fakeEventRepo{eventsErr: errors.New("boom")}
		uc := player.NewGetPlayerPendingMatchesUseCase(eventRepo)
		if _, err := uc.Execute(context.Background(), "p1"); err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}
