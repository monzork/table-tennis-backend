package player_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/player"
	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

// fakeDivisionRepo returns an empty division list — BuildBoardCards degrades
// gracefully (no divisions matched) with no divisions configured, which is
// fine for these tests since they only assert on real (non-virtual) matches.
type fakeDivisionRepo struct {
	divisionDomain.Repository
}

func (f *fakeDivisionRepo) GetAll(ctx context.Context) ([]*divisionDomain.Division, error) {
	return nil, nil
}

func TestGetPlayerPendingMatchesUseCase_Execute(t *testing.T) {
	p1 := &playerDomain.Player{ID: "p1", FirstName: "Ana"}
	opponent := &playerDomain.Player{ID: "p2", FirstName: "Beto"}

	t.Run("flattens pending matches across every event the player is in", func(t *testing.T) {
		eventRepo := &fakeEventRepo{
			events: []*eventDomain.Event{
				{
					ID:         "e1",
					Name:       "Men's Singles",
					StageRules: []eventDomain.StageRule{{Stage: "group", BestOf: 3, PointsToWin: 11, PointsMargin: 2}},
					Matches: []eventDomain.Match{
						{ID: "m1", Status: "scheduled", Stage: "group", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{opponent}},
						{ID: "m2", Status: "finished", WinnerTeam: "A", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{opponent}},
					},
				},
				{
					ID:   "e2",
					Name: "Mixed Doubles",
					Matches: []eventDomain.Match{
						{ID: "m3", Status: "in_progress", TeamA: []*playerDomain.Player{opponent}, TeamB: []*playerDomain.Player{p1}},
					},
				},
			},
		}

		uc := player.NewGetPlayerPendingMatchesUseCase(eventRepo, &fakeDivisionRepo{})
		pending, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(pending) != 2 {
			t.Fatalf("expected 2 pending matches (finished one excluded), got %d: %+v", len(pending), pending)
		}
		if pending[0].EventName != "Men's Singles" || pending[1].EventName != "Mixed Doubles" {
			t.Errorf("expected EventName to be stamped from the owning event, got %+v", pending)
		}
		if pending[0].BestOf != 3 {
			t.Errorf("expected BestOf 3 from the event's configured group stage rule, got %d", pending[0].BestOf)
		}
		if pending[1].BestOf != 5 {
			t.Errorf("expected BestOf to fall back to the default WTT rule (5) when unconfigured, got %d", pending[1].BestOf)
		}
	})

	t.Run("propagates repository error", func(t *testing.T) {
		eventRepo := &fakeEventRepo{eventsErr: errors.New("boom")}
		uc := player.NewGetPlayerPendingMatchesUseCase(eventRepo, &fakeDivisionRepo{})
		if _, err := uc.Execute(context.Background(), "p1"); err == nil {
			t.Fatal("expected error to propagate")
		}
	})

	t.Run("surfaces a potential round-robin matchup with no Match row yet", func(t *testing.T) {
		p3 := &playerDomain.Player{ID: "p3", FirstName: "Caro"}
		eventRepo := &fakeEventRepo{
			events: []*eventDomain.Event{
				{
					ID:            "e3",
					Name:          "Primera Division",
					Type:          "singles",
					EventCategory: "open",
					Format:        "round_robin",
					Participants:  []*playerDomain.Player{p1, opponent, p3},
					// No Matches at all — group play hasn't been "started".
				},
			},
		}

		uc := player.NewGetPlayerPendingMatchesUseCase(eventRepo, &fakeDivisionRepo{})
		pending, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// p1 plays both opponent and p3 in a 3-player round robin.
		if len(pending) != 2 {
			t.Fatalf("expected 2 potential matchups for p1, got %d: %+v", len(pending), pending)
		}
		for _, d := range pending {
			if d.MatchID != "" {
				t.Errorf("expected no MatchID on a not-yet-created matchup, got %q", d.MatchID)
			}
			if d.EventID != "e3" || d.EventName != "Primera Division" {
				t.Errorf("expected EventID/EventName stamped from the owning event, got %+v", d)
			}
			if d.OpponentID == "" || d.Opponent == "" {
				t.Errorf("expected opponent identity populated, got %+v", d)
			}
			if d.BestOf == 0 {
				t.Errorf("expected BestOf populated on a potential matchup too, got %+v", d)
			}
		}
	})
}
