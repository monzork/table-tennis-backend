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

func newEventAt(t *testing.T, name string, start time.Time, participants []*player.Player) *event.Event {
	t.Helper()
	e, err := event.NewEvent(uuid.NewString(), name, "singles", "elimination", "open", start, start.Add(24*time.Hour), nil, 2, participants, false)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	return e
}

func TestEventRepository_GetPreviousEloSnapshots(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "P", "One", "M")
	p2 := savePlayer(t, playerRepo, "P", "Two", "M")
	p3 := savePlayer(t, playerRepo, "P", "Three", "M")

	// An earlier, finished event that both p1 and p2 played: entered at
	// 1000/1050, finished at 1200/1050.
	p1.SinglesElo = 1000
	earlier := newEventAt(t, "Earlier", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p1, p2})
	if err := eventRepo.Save(ctx, earlier); err != nil {
		t.Fatalf("Save earlier: %v", err)
	}
	p1.SinglesElo = 1200
	p2.SinglesElo = 1050
	if err := eventRepo.UpdateParticipantsElo(ctx, earlier.ID, []*player.Player{p1, p2}); err != nil {
		t.Fatalf("UpdateParticipantsElo earlier: %v", err)
	}

	// A later, finished event that only p1 played: entered at 1200 (current
	// live Elo at creation time), finished at 1300. This is the single
	// latest finished tournament -- the global cutoff for everyone.
	later := newEventAt(t, "Later", time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p1})
	if err := eventRepo.Save(ctx, later); err != nil {
		t.Fatalf("Save later: %v", err)
	}
	p1.SinglesElo = 1300
	if err := eventRepo.UpdateParticipantsElo(ctx, later.ID, []*player.Player{p1}); err != nil {
		t.Fatalf("UpdateParticipantsElo later: %v", err)
	}

	// p3 has an event that was never finished (no elo_after) -- absent from
	// the result entirely.
	p3.SinglesElo = 900
	unfinished := newEventAt(t, "Unfinished", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p3})
	if err := eventRepo.Save(ctx, unfinished); err != nil {
		t.Fatalf("Save unfinished: %v", err)
	}

	got, err := eventRepo.GetPreviousEloSnapshots(ctx, "singles")
	if err != nil {
		t.Fatalf("GetPreviousEloSnapshots: %v", err)
	}

	if elo, ok := got[p1.ID]; !ok || elo != 1200 {
		t.Errorf("expected p1's previous Elo to be 1200 (from the latest finished tournament), got %v (present=%v)", elo, ok)
	}
	if _, ok := got[p2.ID]; ok {
		t.Errorf("expected p2 to be absent: it didn't play in the single latest finished tournament, and this is a global cutoff, not p2's own latest event")
	}
	if _, ok := got[p3.ID]; ok {
		t.Errorf("expected p3 to be absent (its only event was never finished)")
	}
}

// TestEventRepository_GetPreviousEloSnapshots_MultiEventTournament covers
// the parent-tournament case: several child events (categories) sharing one
// tournament_id all count as the same "latest finished tournament", even
// though each child event has its own start_date.
func TestEventRepository_GetPreviousEloSnapshots_MultiEventTournament(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "P", "One", "M")
	p2 := savePlayer(t, playerRepo, "P", "Two", "F")
	tournamentID := uuid.NewString()

	p1.SinglesElo = 1000
	menSingles := newEventAt(t, "Men's Singles", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p1})
	menSingles.TournamentID = &tournamentID
	if err := eventRepo.Save(ctx, menSingles); err != nil {
		t.Fatalf("Save menSingles: %v", err)
	}
	p1.SinglesElo = 1100
	if err := eventRepo.UpdateParticipantsElo(ctx, menSingles.ID, []*player.Player{p1}); err != nil {
		t.Fatalf("UpdateParticipantsElo menSingles: %v", err)
	}

	p2.SinglesElo = 900
	womenSingles := newEventAt(t, "Women's Singles", time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC), []*player.Player{p2})
	womenSingles.TournamentID = &tournamentID
	if err := eventRepo.Save(ctx, womenSingles); err != nil {
		t.Fatalf("Save womenSingles: %v", err)
	}
	p2.SinglesElo = 950
	if err := eventRepo.UpdateParticipantsElo(ctx, womenSingles.ID, []*player.Player{p2}); err != nil {
		t.Fatalf("UpdateParticipantsElo womenSingles: %v", err)
	}

	got, err := eventRepo.GetPreviousEloSnapshots(ctx, "singles")
	if err != nil {
		t.Fatalf("GetPreviousEloSnapshots: %v", err)
	}
	if elo, ok := got[p1.ID]; !ok || elo != 1000 {
		t.Errorf("expected p1's previous Elo to be 1000 (from its own category within the latest tournament), got %v (present=%v)", elo, ok)
	}
	if elo, ok := got[p2.ID]; !ok || elo != 900 {
		t.Errorf("expected p2's previous Elo to be 900 (a different category, same tournament), got %v (present=%v)", elo, ok)
	}
}

func TestEventRepository_GetPreviousEloSnapshots_Doubles(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "P", "One", "M")
	p1.DoublesElo = 1400
	e := newEventAt(t, "Doubles Event", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), []*player.Player{p1})
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}
	p1.DoublesElo = 1450
	if err := eventRepo.UpdateParticipantsElo(ctx, e.ID, []*player.Player{p1}); err != nil {
		t.Fatalf("UpdateParticipantsElo: %v", err)
	}

	got, err := eventRepo.GetPreviousEloSnapshots(ctx, "doubles")
	if err != nil {
		t.Fatalf("GetPreviousEloSnapshots: %v", err)
	}
	if elo, ok := got[p1.ID]; !ok || elo != 1400 {
		t.Errorf("expected p1's previous doubles Elo to be 1400, got %v (present=%v)", elo, ok)
	}

	// Event creation/finish snapshots both rank types together regardless of
	// event type, so p1 also shows up in the singles map -- at its unchanged
	// default (1000), since only doubles Elo moved for this event.
	singlesGot, err := eventRepo.GetPreviousEloSnapshots(ctx, "singles")
	if err != nil {
		t.Fatalf("GetPreviousEloSnapshots singles: %v", err)
	}
	if elo, ok := singlesGot[p1.ID]; !ok || elo != 1000 {
		t.Errorf("expected p1's previous singles Elo to be 1000 (unchanged default), got %v (present=%v)", elo, ok)
	}
}
