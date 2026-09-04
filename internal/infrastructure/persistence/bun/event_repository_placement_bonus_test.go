package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestEventRepository_SavePlacementResults_And_GetPlacementHistoryByPlayerID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "P", "One", "M")
	p2 := savePlayer(t, playerRepo, "P", "Two", "M")

	ev := newEventAt(t, "Champ Event", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p1, p2})
	if err := eventRepo.Save(ctx, ev); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := eventRepo.SavePlacementResults(ctx, ev.ID, nil); err != nil {
		t.Fatalf("SavePlacementResults (empty): %v", err)
	}

	results := map[string]event.PlacementDetail{
		p1.ID: {Placement: event.PlacementChampion, BonusElo: 64},
		p2.ID: {Placement: event.PlacementRunnerUp, BonusElo: 32},
	}
	if err := eventRepo.SavePlacementResults(ctx, ev.ID, results); err != nil {
		t.Fatalf("SavePlacementResults: %v", err)
	}

	history, err := eventRepo.GetPlacementHistoryByPlayerID(ctx, p1.ID)
	if err != nil {
		t.Fatalf("GetPlacementHistoryByPlayerID: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("expected one history entry, got %+v", history)
	}
	if history[0].EventID != ev.ID || history[0].EventName != "Champ Event" || history[0].Placement != event.PlacementChampion || history[0].BonusElo != 64 {
		t.Errorf("unexpected history entry: %+v", history[0])
	}

	// A later tournament's bonus doesn't erase the earlier one -- p1's
	// history keeps growing, newest first.
	later := newEventAt(t, "Later Event", time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p1})
	if err := eventRepo.Save(ctx, later); err != nil {
		t.Fatalf("Save later: %v", err)
	}
	if err := eventRepo.SavePlacementResults(ctx, later.ID, map[string]event.PlacementDetail{
		p1.ID: {Placement: event.PlacementThird, BonusElo: 16},
	}); err != nil {
		t.Fatalf("SavePlacementResults later: %v", err)
	}

	history, err = eventRepo.GetPlacementHistoryByPlayerID(ctx, p1.ID)
	if err != nil {
		t.Fatalf("GetPlacementHistoryByPlayerID after later event: %v", err)
	}
	if len(history) != 2 {
		t.Fatalf("expected both events in history, got %+v", history)
	}
	if history[0].EventID != later.ID || history[1].EventID != ev.ID {
		t.Errorf("expected newest-tournament-first order, got %+v", history)
	}

	// p2 never appears in the later event, and its own earlier bonus stays intact.
	p2History, err := eventRepo.GetPlacementHistoryByPlayerID(ctx, p2.ID)
	if err != nil {
		t.Fatalf("GetPlacementHistoryByPlayerID p2: %v", err)
	}
	if len(p2History) != 1 || p2History[0].BonusElo != 32 {
		t.Errorf("expected p2's original bonus untouched, got %+v", p2History)
	}
}

func TestEventRepository_SavePlacementResults_InvalidIDs(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if err := eventRepo.SavePlacementResults(ctx, "not-a-uuid", map[string]event.PlacementDetail{"p1": {}}); err == nil {
		t.Fatal("expected error for invalid event ID, got nil")
	}

	// An invalid player ID within the map is skipped rather than failing
	// the whole batch.
	validEventID := uuid.NewString()
	if err := eventRepo.SavePlacementResults(ctx, validEventID, map[string]event.PlacementDetail{"not-a-uuid": {Placement: "champion", BonusElo: 1}}); err != nil {
		t.Fatalf("expected invalid player ID to be skipped, got error: %v", err)
	}
}

func TestEventRepository_GetLatestTournamentPlacementBonuses(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	got, err := eventRepo.GetLatestTournamentPlacementBonuses(ctx, "singles")
	if err != nil {
		t.Fatalf("GetLatestTournamentPlacementBonuses (none finished): %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}

	champ := savePlayer(t, playerRepo, "Champ", "One", "M")

	singles := newEventAt(t, "Singles Final", time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), []*player.Player{champ})
	if err := eventRepo.Save(ctx, singles); err != nil {
		t.Fatalf("Save singles: %v", err)
	}
	if err := eventRepo.SavePlacementResults(ctx, singles.ID, map[string]event.PlacementDetail{
		champ.ID: {Placement: event.PlacementChampion, BonusElo: 64},
	}); err != nil {
		t.Fatalf("SavePlacementResults singles: %v", err)
	}
	singles.Status = "finished"
	if err := eventRepo.Update(ctx, singles); err != nil {
		t.Fatalf("Update singles: %v", err)
	}

	got, err = eventRepo.GetLatestTournamentPlacementBonuses(ctx, "singles")
	if err != nil {
		t.Fatalf("GetLatestTournamentPlacementBonuses: %v", err)
	}
	if got[champ.ID] != 64 {
		t.Errorf("expected champ's bonus 64, got %v", got)
	}

	// A doubles-ranking lookup at this point sees nothing: the latest
	// finished tournament has no doubles/mixed_doubles event.
	gotDoubles, err := eventRepo.GetLatestTournamentPlacementBonuses(ctx, "doubles")
	if err != nil {
		t.Fatalf("GetLatestTournamentPlacementBonuses (doubles): %v", err)
	}
	if len(gotDoubles) != 0 {
		t.Errorf("expected no doubles bonuses from a singles-only tournament, got %v", gotDoubles)
	}
}
