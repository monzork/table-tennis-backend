package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/identity"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

// TestCreateEventUseCase_RealDB_GenderedDivisionsDoNotCrossOver is a
// regression test for a bug where creating a tournament with a mix of
// male- and female-gendered divisions selected produced an extra,
// wrongly-labeled sub-event for a single player (e.g. a lone male player
// ended up with both a correct "Men's Singles (2nd Division (Men))" event
// AND a bogus "Men's Singles (1st Division (Women))" event). Root cause:
// DivisionRepository.GetById (the fetch path CreateEventUseCase uses for
// each admin-selected division ID) never mapped the Gender column, so
// every division loaded that way came back gender-agnostic and
// division.MatchesGender treated it as matching every category. GetAll
// mapped Gender correctly, which is why the bug only showed up through
// real tournament creation, never through code paths using GetAll.
func TestCreateEventUseCase_RealDB_GenderedDivisionsDoNotCrossOver(t *testing.T) {
	idgen.Register(identity.NewUUIDGenerator())
	db := setupTestDB(t)
	ctx := context.Background()

	eventRepo := bunRepo.NewEventRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db, eventRepo)
	playerRepo := bunRepo.NewPlayerRepository(db)
	divisionRepo := bunRepo.NewDivisionRepository(db)

	maxSecondMale := int16(2000)
	maxSecondFemale := int16(1300)
	fixtures := []struct {
		id, name string
		order    int
		min      int16
		max      *int16
		gender   string
	}{
		{"gdt-div-first-male", "1st Division (Men)", 10, 2000, nil, "M"},
		{"gdt-div-second-male", "2nd Division (Men)", 11, 1300, &maxSecondMale, "M"},
		{"gdt-div-first-female", "1st Division (Women)", 12, 1300, nil, "F"},
		{"gdt-div-second-female", "2nd Division (Women)", 13, 0, &maxSecondFemale, "F"},
	}
	var divIDs []string
	for _, f := range fixtures {
		d, err := divisionDomain.NewDivision(f.id, f.name, f.order, f.min, f.max, "both", "#000")
		if err != nil {
			t.Fatalf("NewDivision: %v", err)
		}
		d.Gender = f.gender
		if err := divisionRepo.Save(ctx, d); err != nil {
			t.Fatalf("Save division %s: %v", f.id, err)
		}
		divIDs = append(divIDs, f.id)
	}

	p, _ := player.NewPlayer(idgen.Generate(), "Solo", "Male", time.Now(), "M", "", "", "")
	p.SinglesElo = 1500 // falls in 2nd Division (Men): 1300-1999
	if err := playerRepo.Save(ctx, p); err != nil {
		t.Fatalf("Save player: %v", err)
	}

	uc := tournament.NewCreateEventUseCase(tournamentRepo, eventRepo, playerRepo, divisionRepo)
	res, err := uc.Execute(
		ctx,
		"Gendered Creation Regression",
		divIDs,
		false,
		"2026-10-01",
		"2026-10-02",
		tournament.CategoryConfig{Auto: true, Format: "elimination", PlayerIDs: []string{p.ID}}, // singlesMen
		tournament.CategoryConfig{},
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
		t.Fatalf("Execute: %v", err)
	}

	if len(res.Events) != 1 {
		t.Fatalf("expected exactly 1 sub-event for a single male player, got %d: %+v", len(res.Events), res.Events)
	}
	ev := res.Events[0]
	if ev.EventCategory != "men" {
		t.Errorf("expected the sub-event's category to be 'men', got %q", ev.EventCategory)
	}
	if !ev.UseGenderDivisions {
		t.Errorf("expected UseGenderDivisions=true on the created sub-event")
	}
}
