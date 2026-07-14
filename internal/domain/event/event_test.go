package event_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/event"
)

func TestNewTournament_Valid(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)
	participants := []*player.Player{
		{ID: "p1", Gender: "M"},
		{ID: "p2", Gender: "M"},
	}

	tourn, err := event.NewTournament("t1", "Test Tourn", "singles", "elimination", "men", start, end, nil, 2, participants, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if tourn.ID != "t1" {
		t.Errorf("expected t1, got %s", tourn.ID)
	}
	if tourn.EventCategory != "men" {
		t.Errorf("expected men, got %s", tourn.EventCategory)
	}
}

func TestNewTournament_InvalidDates(t *testing.T) {
	start := time.Now()
	end := start.Add(-24 * time.Hour) // Ends before starts

	_, err := event.NewTournament("t1", "Test Tourn", "singles", "elimination", "open", start, end, nil, 2, nil, false)
	if err != event.ErrInvalidDates {
		t.Fatalf("expected ErrInvalidDates, got %v", err)
	}
}

func TestNewTournament_CategoryValidation(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)
	participants := []*player.Player{
		{ID: "p1", Gender: "F"}, // Female in a men's event
	}

	_, err := event.NewTournament("t1", "Test Tourn", "singles", "elimination", "men", start, end, nil, 2, participants, false)
	if err == nil {
		t.Fatalf("expected error for gender mismatch, got nil")
	}
}

func TestTournament_GetEffectiveStageRule(t *testing.T) {
	tourn := &event.Event{
		StageRules: []event.StageRule{
			{Stage: "final", BestOf: 7, PointsToWin: 11, PointsMargin: 2},
		},
		DivisionRules: []event.DivisionRule{
			{DivisionID: "div1", Stage: "final", BestOf: 5, PointsToWin: 11, PointsMargin: 2},
		},
	}

	// Should prioritize division rule
	rule := tourn.GetEffectiveStageRule("final", "div1")
	if rule.BestOf != 5 {
		t.Errorf("expected division rule bestOf 5, got %d", rule.BestOf)
	}

	// Should fallback to stage rule
	rule2 := tourn.GetEffectiveStageRule("final", "div2")
	if rule2.BestOf != 7 {
		t.Errorf("expected stage rule bestOf 7, got %d", rule2.BestOf)
	}
}
