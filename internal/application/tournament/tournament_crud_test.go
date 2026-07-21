package tournament_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	playerDomain "table-tennis-backend/internal/domain/player"
	eventDomain "table-tennis-backend/internal/domain/tournament"
)

func TestCreateEventUseCase_Execute(t *testing.T) {
	eventRepo := newMockEventRepo()
	subTourneyRepo := newMockSubTourneyRepo()
	playerRepo := newMockPlayerRepo()
	divRepo := newMockDivisionRepo()

	uc := tournament.NewCreateEventUseCase(eventRepo, subTourneyRepo, playerRepo, divRepo)
	ctx := context.Background()

	p1 := &playerDomain.Player{ID: "p1", Gender: "M", SinglesElo: 1000, DoublesElo: 1100}
	p2 := &playerDomain.Player{ID: "p2", Gender: "F", SinglesElo: 1500, DoublesElo: 1600}
	playerRepo.players["p1"] = p1
	playerRepo.players["p2"] = p2

	div1, _ := divisionDomain.NewDivision("d1", "Div 1", 1, 1200, nil, "singles", "#000")
	div2, _ := divisionDomain.NewDivision("d2", "Div 2", 1, 2000, nil, "singles", "#fff")
	divRepo.divisions["d1"] = div1
	divRepo.divisions["d2"] = div2

	// Test valid
	res, err := uc.Execute(
		ctx,
		"Test Event",
		[]string{"d1", "d2"},
		false,
		"2026-10-01",
		"2026-10-02",
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"p1"}},
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"p2"}},
		tournament.CategoryConfig{Auto: true, Format: "doubles", PlayerIDs: []string{"p1", "p2"}},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{Auto: true, Format: "doubles", PlayerIDs: []string{"p1", "p2"}}, // mixed
		tournament.CategoryConfig{Auto: true, Format: "teams", PlayerIDs: []string{"p1"}},
		tournament.CategoryConfig{Auto: true, Format: "teams", PlayerIDs: []string{"p2"}},
		[]string{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("result is nil")
	}

	// Test errors
	_, err = uc.Execute(ctx, "Test", []string{"invalid_div"}, false, "2026-10-01", "2026-10-02", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil)
	if err == nil {
		t.Errorf("expected error for invalid div")
	}

	_, err = uc.Execute(ctx, "Test", nil, true, "bad-date", "2026-10-02", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil)
	if err == nil {
		t.Errorf("expected error for bad start date")
	}

	_, err = uc.Execute(ctx, "Test", nil, true, "2026-10-01", "bad-date", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil)
	if err == nil {
		t.Errorf("expected error for bad end date")
	}

	// NewEvent validation error (empty name) with a non-skip-elo division set.
	_, err = uc.Execute(ctx, "", []string{"d1"}, false, "2026-10-01", "2026-10-02", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil)
	if err == nil {
		t.Errorf("expected error for empty event name")
	}

	// Unknown player ID should be skipped (not resolvable from the cache) and
	// existingTournamentIDs containing both a real and a blank entry should
	// still succeed, exercising UpdateEventIDBulk.
	res2, err := uc.Execute(
		ctx,
		"Test Event 2",
		[]string{"d1", "d2"},
		false,
		"2026-10-01",
		"2026-10-02",
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"p1", "p_unknown"}},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		[]string{"existing1", ""},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res2 == nil {
		t.Fatalf("result is nil")
	}

	// skipElo=true (no divisions loaded) exercises the "flat" sub-tournament
	// branch and its M/F/mixed catArg mapping.
	res3, err := uc.Execute(
		ctx,
		"Test Event 3",
		nil,
		true,
		"2026-10-01",
		"2026-10-02",
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"p1"}},
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"p2"}},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{Auto: true, Format: "doubles", PlayerIDs: []string{"p1", "p2"}}, // mixed
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res3 == nil {
		t.Fatalf("result is nil")
	}
}

func TestUpdateEventUseCase_Execute(t *testing.T) {
	eventRepo := newMockEventRepo()
	uc := tournament.NewUpdateEventUseCase(eventRepo)
	ctx := context.Background()

	now := time.Now()
	e, _ := eventDomain.NewEvent("e1", "Event 1", nil, true, now, now)
	eventRepo.Save(ctx, e)

	_, err := uc.Execute(ctx, "e1", "Updated Name", "2026-10-10", "2026-10-12", 5, map[string][]int{"t": {1}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err = uc.Execute(ctx, "invalid", "", "", "", 0, nil)
	if err == nil {
		t.Errorf("expected error for invalid id")
	}

	eventRepo.updateErr = errors.New("update failed")
	_, err = uc.Execute(ctx, "e1", "Another Name", "", "", 0, nil)
	if err == nil {
		t.Errorf("expected error when repo Update fails")
	}
}

func TestGetAllEventsUseCase_Execute(t *testing.T) {
	eventRepo := newMockEventRepo()
	uc := tournament.NewGetAllEventsUseCase(eventRepo)
	ctx := context.Background()

	res, err := uc.Execute(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 0 {
		t.Errorf("expected 0")
	}
}

func TestGetEventByIDUseCase_Execute(t *testing.T) {
	eventRepo := newMockEventRepo()
	uc := tournament.NewGetEventByIDUseCase(eventRepo)
	ctx := context.Background()

	e, _ := eventDomain.NewEvent("e1", "Event 1", nil, true, time.Now(), time.Now())
	eventRepo.Save(ctx, e)

	res, err := uc.Execute(ctx, "e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "e1" {
		t.Errorf("expected e1")
	}
}
