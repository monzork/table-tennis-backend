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

	tourn, err := event.NewEvent("t1", "Test Tourn", "singles", "elimination", "men", "", start, end, nil, 2, participants, false)
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

	_, err := event.NewEvent("t1", "Test Tourn", "singles", "elimination", "open", "", start, end, nil, 2, nil, false)
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

	_, err := event.NewEvent("t1", "Test Tourn", "singles", "elimination", "men", "", start, end, nil, 2, participants, false)
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

	_, err := event.NewEvent("t1", "Test Tourn", "singles", "elimination", "women", "", start, end, nil, 2, participants, false)
	if err == nil {
		t.Fatalf("expected error for gender mismatch, got nil")
	}
}

func TestNewTournament_DefaultsApplied(t *testing.T) {
	start := time.Now()
	end := start.Add(24 * time.Hour)

	tourn, err := event.NewEvent("t1", "Test Tourn", "", "", "", "", start, end, nil, 2, nil, false)
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
	if tourn.AgeCategory != "open" {
		t.Errorf("expected default age category open, got %s", tourn.AgeCategory)
	}
}

func TestNewTournament_AgeCategoryValidation(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	t.Run("rejects a participant too old for the bracket", func(t *testing.T) {
		adult := &player.Player{ID: "p1", Gender: "M", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}
		_, err := event.NewEvent("t1", "U13 Event", "singles", "elimination", "open", "u13", start, end, nil, 2, []*player.Player{adult}, false)
		if err == nil {
			t.Fatal("expected error for age-ineligible participant, got nil")
		}
	})

	t.Run("accepts an age-eligible participant", func(t *testing.T) {
		child := &player.Player{ID: "p1", Gender: "M", Birthdate: time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC)}
		tourn, err := event.NewEvent("t1", "U13 Event", "singles", "elimination", "open", "u13", start, end, nil, 2, []*player.Player{child}, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if tourn.AgeCategory != "u13" {
			t.Errorf("expected AgeCategory u13, got %s", tourn.AgeCategory)
		}
	})

	t.Run("play-up: a younger player may enter an older bracket", func(t *testing.T) {
		child := &player.Player{ID: "p1", Gender: "M", Birthdate: time.Date(2018, 1, 1, 0, 0, 0, 0, time.UTC)} // 8 in 2026, U11-eligible
		_, err := event.NewEvent("t1", "U19 Event", "singles", "elimination", "open", "u19", start, end, nil, 2, []*player.Player{child}, false)
		if err != nil {
			t.Fatalf("expected play-up into an older bracket to succeed, got %v", err)
		}
	})

	t.Run("open has no age ceiling", func(t *testing.T) {
		adult := &player.Player{ID: "p1", Gender: "M", Birthdate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}
		_, err := event.NewEvent("t1", "Open Event", "singles", "elimination", "open", "open", start, end, nil, 2, []*player.Player{adult}, false)
		if err != nil {
			t.Fatalf("expected no error for open category, got %v", err)
		}
	})
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

	tourn, err := event.NewEvent("t1", "Test Tourn", "singles", "groups_elimination", "open", "", start, end, nil, 2, participants, false)
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

	tourn, err := event.NewEvent("t1", "Test Tourn", "singles", "round_robin", "open", "", start, end, nil, 2, participants, false)
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
	}

	rule := tourn.GetEffectiveStageRule("final")
	if rule.BestOf != 7 {
		t.Errorf("expected stage rule bestOf 7, got %d", rule.BestOf)
	}

	// Should fallback to default WTT rule when stage isn't configured
	rule2 := tourn.GetEffectiveStageRule("semifinal")
	if rule2.BestOf != 5 {
		t.Errorf("expected default bestOf 5, got %d", rule2.BestOf)
	}
}
