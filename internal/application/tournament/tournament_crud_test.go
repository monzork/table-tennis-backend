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
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"p1", "p2"}}, // open singles
		[]tournament.CustomEventConfig{
			{Name: "Group A", Format: "elimination", PlayerIDs: []string{"p1", "p2"}},
		},
		[]string{},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatalf("result is nil")
	}

	// Test errors
	_, err = uc.Execute(ctx, "Test", []string{"invalid_div"}, false, "2026-10-01", "2026-10-02", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil, nil)
	if err == nil {
		t.Errorf("expected error for invalid div")
	}

	_, err = uc.Execute(ctx, "Test", nil, true, "bad-date", "2026-10-02", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil, nil)
	if err == nil {
		t.Errorf("expected error for bad start date")
	}

	_, err = uc.Execute(ctx, "Test", nil, true, "2026-10-01", "bad-date", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil, nil)
	if err == nil {
		t.Errorf("expected error for bad end date")
	}

	// NewEvent validation error (empty name) with a non-skip-elo division set.
	_, err = uc.Execute(ctx, "", []string{"d1"}, false, "2026-10-01", "2026-10-02", tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{}, nil, nil)
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
		tournament.CategoryConfig{},
		nil,
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
		tournament.CategoryConfig{},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res3 == nil {
		t.Fatalf("result is nil")
	}
	for _, ev := range res3.Events {
		if !ev.SkipDivisionSplit {
			t.Errorf("expected event %q created via the flat (no-division) branch to have SkipDivisionSplit=true", ev.Name)
		}
	}
}

func TestCreateEventUseCase_Execute_GenderDivisions(t *testing.T) {
	eventRepo := newMockEventRepo()
	subTourneyRepo := newMockSubTourneyRepo()
	playerRepo := newMockPlayerRepo()
	divRepo := newMockDivisionRepo()

	uc := tournament.NewCreateEventUseCase(eventRepo, subTourneyRepo, playerRepo, divRepo)
	ctx := context.Background()

	// A man just above the male 1st-division threshold and a woman just
	// above the (much lower) female 1st-division threshold -- if gender
	// isn't checked when grouping by division, the man's Elo would also
	// satisfy the female band's numeric range, and vice versa is possible
	// too depending on the bands, so this must not leak across categories.
	pMale := &playerDomain.Player{ID: "pm", Gender: "M", SinglesElo: 2100}
	pFemale := &playerDomain.Player{ID: "pf", Gender: "F", SinglesElo: 1400}
	playerRepo.players["pm"] = pMale
	playerRepo.players["pf"] = pFemale

	divMale := &divisionDomain.Division{ID: "div-first-male", Name: "1st Division (Men)", DisplayOrder: 10, MinElo: 2000, MaxElo: nil, Category: "both", Gender: "M", Color: "#000"}
	divFemale := &divisionDomain.Division{ID: "div-first-female", Name: "1st Division (Women)", DisplayOrder: 12, MinElo: 1300, MaxElo: nil, Category: "both", Gender: "F", Color: "#000"}
	divRepo.divisions["div-first-male"] = divMale
	divRepo.divisions["div-first-female"] = divFemale

	res, err := uc.Execute(
		ctx,
		"Gendered Tournament",
		[]string{"div-first-male", "div-first-female"},
		false,
		"2026-10-01",
		"2026-10-02",
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"pm"}},
		tournament.CategoryConfig{Auto: true, Format: "single", PlayerIDs: []string{"pf"}},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		tournament.CategoryConfig{},
		nil,
		nil,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Events) != 2 {
		t.Fatalf("expected exactly 2 sub-events (one per gender), got %d: %+v", len(res.Events), res.Events)
	}

	for _, ev := range res.Events {
		if !ev.UseGenderDivisions {
			t.Errorf("expected sub-event %q created from a gender-specific division to have UseGenderDivisions=true", ev.Name)
		}
		switch ev.EventCategory {
		case "men":
			if len(ev.Participants) != 1 || ev.Participants[0].ID != "pm" {
				t.Errorf("expected men's sub-event to contain only the male player, got %+v", ev.Participants)
			}
		case "women":
			if len(ev.Participants) != 1 || ev.Participants[0].ID != "pf" {
				t.Errorf("expected women's sub-event to contain only the female player, got %+v", ev.Participants)
			}
		default:
			t.Errorf("unexpected event category %q for event %q", ev.EventCategory, ev.Name)
		}
	}
}

func TestUpdateEventUseCase_Execute(t *testing.T) {
	eventRepo := newMockEventRepo()
	uc := tournament.NewUpdateEventUseCase(eventRepo)
	ctx := context.Background()

	now := time.Now()
	e, _ := eventDomain.NewTournament("e1", "Event 1", nil, true, now, now)
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

	e, _ := eventDomain.NewTournament("e1", "Event 1", nil, true, time.Now(), time.Now())
	eventRepo.Save(ctx, e)

	res, err := uc.Execute(ctx, "e1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ID != "e1" {
		t.Errorf("expected e1")
	}
}
