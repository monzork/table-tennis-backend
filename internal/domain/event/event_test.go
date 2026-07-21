package event_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
)

type stubIDGen struct{}

func (stubIDGen) Generate() string { return "generated-id" }

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

func TestNewTournament_WomenCategoryValidation(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)
	participants := []*player.Player{
		{ID: "p1", Gender: "M"}, // Male in a women's event
	}

	_, err := event.NewTournament("t1", "Test Tourn", "singles", "elimination", "women", start, end, nil, 2, participants, false)
	if err == nil {
		t.Fatalf("expected error for gender mismatch, got nil")
	}
}

func TestNewTournament_DefaultsApplied(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)

	tourn, err := event.NewTournament("t1", "Test Tourn", "", "", "", start, end, nil, 2, nil, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if tourn.Type != "singles" {
		t.Errorf("expected default type singles, got %s", tourn.Type)
	}
	if tourn.Format != "elimination" {
		t.Errorf("expected default format elimination, got %s", tourn.Format)
	}
	if tourn.EventCategory != "open" {
		t.Errorf("expected default category open, got %s", tourn.EventCategory)
	}
}

func TestNewTournament_GroupsElimination_AssignsGroups(t *testing.T) {
	idgen.Register(stubIDGen{})
	start := time.Now()
	end := start.Add(24 * time.Hour)
	participants := []*player.Player{
		{ID: "p1", SinglesElo: 1500},
		{ID: "p2", SinglesElo: 1400},
		{ID: "p3", SinglesElo: 1300},
	}

	tourn, err := event.NewTournament("t1", "Test Tourn", "singles", "groups_elimination", "open", start, end, nil, 2, participants, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(tourn.Groups) == 0 {
		t.Errorf("expected groups to be assigned for groups_elimination format")
	}
}

func TestNewTournament_RoundRobin_AssignsGroups(t *testing.T) {
	idgen.Register(stubIDGen{})
	start := time.Now()
	end := start.Add(24 * time.Hour)
	participants := []*player.Player{
		{ID: "p1", SinglesElo: 1500},
		{ID: "p2", SinglesElo: 1400},
	}

	tourn, err := event.NewTournament("t1", "Test Tourn", "singles", "round_robin", "open", start, end, nil, 2, participants, false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(tourn.Groups) != 1 {
		t.Fatalf("expected a single round robin group, got %d", len(tourn.Groups))
	}
	if len(tourn.Groups[0].Players) != 2 {
		t.Errorf("expected 2 players in the round robin group, got %d", len(tourn.Groups[0].Players))
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
