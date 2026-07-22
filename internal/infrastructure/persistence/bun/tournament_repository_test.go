package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func newTestTournament(t *testing.T, name string) *tournament.Tournament {
	t.Helper()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(48 * time.Hour)
	tr, err := tournament.NewEvent(uuid.NewString(), name, []string{"div-1"}, false, start, end)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	return tr
}

func newTestEvent(t *testing.T, name string) *event.Event {
	t.Helper()
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	e, err := event.NewTournament(uuid.NewString(), name, "singles", "elimination", "open", start, end, nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	return e
}

// attachEvent mirrors what the application layer does before persisting a
// Tournament: it stamps the parent tournament's ID onto each nested Event's
// EventID field (the FK column is literally named tournament_id).
func attachEvent(tr *tournament.Tournament, e *event.Event) {
	id := tr.ID
	e.EventID = &id
	tr.Events = append(tr.Events, e)
}

func TestTournamentRepository_SaveAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	tr := newTestTournament(t, "Summer Open")
	attachEvent(tr, newTestEvent(t, "Singles"))

	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Summer Open" || len(got.DivisionIDs) != 1 || got.DivisionIDs[0] != "div-1" {
		t.Fatalf("unexpected tournament: %+v", got)
	}
	if len(got.Events) != 1 || got.Events[0].Name != "Singles" {
		t.Fatalf("expected nested event to be persisted, got %+v", got.Events)
	}
}

func TestTournamentRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing tournament, got nil")
	}
}

func TestTournamentRepository_GetByID_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	if _, err := repo.GetByID(ctx, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestTournamentRepository_GetByIDDeep(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	tr := newTestTournament(t, "Deep Cup")
	attachEvent(tr, newTestEvent(t, "Deep Singles"))
	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByIDDeep(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetByIDDeep: %v", err)
	}
	if len(got.Events) != 1 {
		t.Fatalf("expected 1 nested event, got %d", len(got.Events))
	}
}

func TestTournamentRepository_GetByIDDeep_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	if _, err := repo.GetByIDDeep(ctx, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestTournamentRepository_GetByIDDeep_NotFound(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	if _, err := repo.GetByIDDeep(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing tournament, got nil")
	}
}

func TestTournamentRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	tr := newTestTournament(t, "Before Rename")
	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tr.Name = "After Rename"
	tr.NumTables = 8
	if err := repo.Update(ctx, tr); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := repo.GetByID(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "After Rename" || got.NumTables != 8 {
		t.Fatalf("expected updated tournament, got %+v", got)
	}
}

func TestTournamentRepository_GetAll(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	empty, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 tournaments, got %d", len(empty))
	}

	p1 := savePlayer(t, playerRepo, "GetAll", "One", "M")
	p2 := savePlayer(t, playerRepo, "GetAll", "Two", "M")

	tr1 := newTestTournament(t, "Tournament A")
	ev1 := newBareEvent(t, "Event A1", []*player.Player{p1})
	attachEvent(tr1, ev1)
	if err := repo.Save(ctx, tr1); err != nil {
		t.Fatalf("Save tr1: %v", err)
	}

	// Attach a team + team player to the saved event so GetAll's batch team-loading
	// branches (allTeamModels/allTPModels non-empty) are exercised too.
	team, err := event.NewTeam(uuid.NewString(), ev1.ID, "Team GetAll")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	if err := eventRepo.SaveTeam(ctx, team); err != nil {
		t.Fatalf("SaveTeam: %v", err)
	}
	if err := eventRepo.AddPlayerToTeam(ctx, team.ID, p2.ID); err != nil {
		t.Fatalf("AddPlayerToTeam: %v", err)
	}

	tr2 := newTestTournament(t, "Tournament B")
	if err := repo.Save(ctx, tr2); err != nil {
		t.Fatalf("Save tr2: %v", err)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 tournaments, got %d", len(all))
	}

	var gotTr1 *tournament.Tournament
	for _, tr := range all {
		if tr.ID == tr1.ID {
			gotTr1 = tr
		}
	}
	if gotTr1 == nil || len(gotTr1.Events) != 1 {
		t.Fatalf("expected tr1 with 1 event, got %+v", gotTr1)
	}
	if len(gotTr1.Events[0].Participants) != 1 {
		t.Fatalf("expected 1 participant on event A1, got %+v", gotTr1.Events[0].Participants)
	}
	if len(gotTr1.Events[0].Teams) != 1 || len(gotTr1.Events[0].Teams[0].Players) != 1 {
		t.Fatalf("expected 1 team with 1 player on event A1, got %+v", gotTr1.Events[0].Teams)
	}
}

func TestTournamentRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	tr := newTestTournament(t, "To Delete")
	attachEvent(tr, newTestEvent(t, "Nested Event"))
	if err := repo.Save(ctx, tr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := repo.Delete(ctx, tr.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err := repo.GetByID(ctx, tr.ID); err == nil {
		t.Fatal("expected error after delete, got nil")
	}
	// The nested event should be cascade-deleted too.
	if _, err := eventRepo.GetByID(ctx, tr.Events[0].ID); err == nil {
		t.Fatal("expected nested event to be cascade-deleted")
	}
}

func TestTournamentRepository_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	if err := repo.Delete(ctx, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestTournamentRepository_DeleteEvents(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	// no-op cases
	if err := repo.DeleteEvents(ctx, nil); err != nil {
		t.Fatalf("DeleteEvents (nil): %v", err)
	}
	if err := repo.DeleteEvents(ctx, []string{"not-a-uuid"}); err != nil {
		t.Fatalf("DeleteEvents (only invalid): %v", err)
	}

	tr1 := newTestTournament(t, "Tournament 1")
	attachEvent(tr1, newTestEvent(t, "Ev1"))
	if err := repo.Save(ctx, tr1); err != nil {
		t.Fatalf("Save tr1: %v", err)
	}
	tr2 := newTestTournament(t, "Tournament 2")
	if err := repo.Save(ctx, tr2); err != nil {
		t.Fatalf("Save tr2: %v", err)
	}

	if err := repo.DeleteEvents(ctx, []string{tr1.ID}); err != nil {
		t.Fatalf("DeleteEvents: %v", err)
	}

	if _, err := repo.GetByID(ctx, tr1.ID); err == nil {
		t.Fatal("expected tr1 to be deleted")
	}
	if _, err := repo.GetByID(ctx, tr2.ID); err != nil {
		t.Fatalf("expected tr2 to remain: %v", err)
	}
}

func TestTournamentRepository_DB(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	repo := bunRepo.NewTournamentRepository(db, eventRepo)
	if repo.DB() != db {
		t.Fatal("expected DB() to return the underlying bun.DB")
	}
}
