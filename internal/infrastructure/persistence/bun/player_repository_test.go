package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func newTestPlayer(t *testing.T, first, last, gender string) *player.Player {
	t.Helper()
	p, err := player.NewPlayer(uuid.NewString(), first, last, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), gender, "NIC", "Managua", "")
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	return p
}

func TestPlayerRepository_SaveAndGetById(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p := newTestPlayer(t, "John", "Doe", "M")
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetById(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.FirstName != "John" || got.LastName != "Doe" || got.SinglesElo != 1000 {
		t.Fatalf("unexpected player: %+v", got)
	}
}

func TestPlayerRepository_GetById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	if _, err := repo.GetById(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing player, got nil")
	}
}

func TestPlayerRepository_GetById_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	if _, err := repo.GetById(ctx, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestPlayerRepository_Save_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p := newTestPlayer(t, "Jane", "Smith", "F")
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	p.UpdateSinglesElo(1200)
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save (update): %v", err)
	}

	got, err := repo.GetById(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.SinglesElo != 1200 {
		t.Fatalf("expected updated elo 1200, got %d", got.SinglesElo)
	}
}

func TestPlayerRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p := newTestPlayer(t, "Del", "Ete", "M")
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := repo.Delete(ctx, p.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := repo.GetById(ctx, p.ID); err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestPlayerRepository_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	if err := repo.Delete(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestPlayerRepository_GetAllSinglesAndDoubles(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := newTestPlayer(t, "Low", "Elo", "M")
	p1.SinglesElo = 900
	p1.DoublesElo = 1100
	p2 := newTestPlayer(t, "High", "Elo", "F")
	p2.SinglesElo = 1300
	p2.DoublesElo = 800

	if err := repo.Save(ctx, p1); err != nil {
		t.Fatalf("Save p1: %v", err)
	}
	if err := repo.Save(ctx, p2); err != nil {
		t.Fatalf("Save p2: %v", err)
	}

	singles, err := repo.GetAllSingles(ctx)
	if err != nil {
		t.Fatalf("GetAllSingles: %v", err)
	}
	if len(singles) != 2 || singles[0].ID != p2.ID {
		t.Fatalf("expected singles ordered desc by elo, got %+v", singles)
	}

	doubles, err := repo.GetAllDoubles(ctx)
	if err != nil {
		t.Fatalf("GetAllDoubles: %v", err)
	}
	if len(doubles) != 2 || doubles[0].ID != p1.ID {
		t.Fatalf("expected doubles ordered desc by elo, got %+v", doubles)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 players, got %d", len(all))
	}
}

func TestPlayerRepository_GetSinglesAndDoublesByGender(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	m := newTestPlayer(t, "Male", "Player", "M")
	f := newTestPlayer(t, "Female", "Player", "F")
	if err := repo.Save(ctx, m); err != nil {
		t.Fatalf("Save m: %v", err)
	}
	if err := repo.Save(ctx, f); err != nil {
		t.Fatalf("Save f: %v", err)
	}

	males, err := repo.GetSinglesByGender(ctx, "M")
	if err != nil {
		t.Fatalf("GetSinglesByGender: %v", err)
	}
	if len(males) != 1 || males[0].ID != m.ID {
		t.Fatalf("expected 1 male player, got %+v", males)
	}

	females, err := repo.GetDoublesByGender(ctx, "F")
	if err != nil {
		t.Fatalf("GetDoublesByGender: %v", err)
	}
	if len(females) != 1 || females[0].ID != f.ID {
		t.Fatalf("expected 1 female player, got %+v", females)
	}

	allSingles, err := repo.GetSinglesByGender(ctx, "")
	if err != nil {
		t.Fatalf("GetSinglesByGender (all): %v", err)
	}
	if len(allSingles) != 2 {
		t.Fatalf("expected 2 players with empty gender filter, got %d", len(allSingles))
	}
}

func TestPlayerRepository_GetByIDs(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := newTestPlayer(t, "A", "One", "M")
	p2 := newTestPlayer(t, "B", "Two", "M")
	if err := repo.Save(ctx, p1); err != nil {
		t.Fatalf("Save p1: %v", err)
	}
	if err := repo.Save(ctx, p2); err != nil {
		t.Fatalf("Save p2: %v", err)
	}

	got, err := repo.GetByIDs(ctx, []string{p1.ID, p2.ID, "invalid-uuid"})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 players, got %d", len(got))
	}

	empty, err := repo.GetByIDs(ctx, nil)
	if err != nil {
		t.Fatalf("GetByIDs (empty): %v", err)
	}
	if empty != nil {
		t.Fatalf("expected nil for empty ids, got %+v", empty)
	}

	onlyInvalid, err := repo.GetByIDs(ctx, []string{"not-a-uuid"})
	if err != nil {
		t.Fatalf("GetByIDs (only invalid): %v", err)
	}
	if onlyInvalid != nil {
		t.Fatalf("expected nil when no valid ids, got %+v", onlyInvalid)
	}
}

func TestPlayerRepository_SearchAndSearchForSelection(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := newTestPlayer(t, "Alice", "Wonderland", "F")
	p2 := newTestPlayer(t, "Bob", "Builder", "M")
	if err := repo.Save(ctx, p1); err != nil {
		t.Fatalf("Save p1: %v", err)
	}
	if err := repo.Save(ctx, p2); err != nil {
		t.Fatalf("Save p2: %v", err)
	}

	results, err := repo.Search(ctx, "alice")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].ID != p1.ID {
		t.Fatalf("expected to find Alice, got %+v", results)
	}

	all, err := repo.Search(ctx, "")
	if err != nil {
		t.Fatalf("Search (empty query): %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 players for empty query, got %d", len(all))
	}

	none, err := repo.Search(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Search (no match): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(none))
	}

	selection, err := repo.SearchForSelection(ctx, "bob", "M")
	if err != nil {
		t.Fatalf("SearchForSelection: %v", err)
	}
	if len(selection) != 1 || selection[0].ID != p2.ID {
		t.Fatalf("expected to find Bob, got %+v", selection)
	}

	wrongGender, err := repo.SearchForSelection(ctx, "bob", "F")
	if err != nil {
		t.Fatalf("SearchForSelection (wrong gender): %v", err)
	}
	if len(wrongGender) != 0 {
		t.Fatalf("expected 0 matches for wrong gender, got %d", len(wrongGender))
	}
}

func TestPlayerRepository_Search_AccentInsensitive(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := newTestPlayer(t, "José", "Muñoz", "M")
	p2 := newTestPlayer(t, "René", "Peña", "M")
	if err := repo.Save(ctx, p1); err != nil {
		t.Fatalf("Save p1: %v", err)
	}
	if err := repo.Save(ctx, p2); err != nil {
		t.Fatalf("Save p2: %v", err)
	}

	// Unaccented query should find accented names.
	results, err := repo.Search(ctx, "jose munoz")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(results) != 1 || results[0].ID != p1.ID {
		t.Fatalf("expected to find José Muñoz via unaccented query, got %+v", results)
	}

	// Accented query should still find accented names.
	results, err = repo.Search(ctx, "José")
	if err != nil {
		t.Fatalf("Search (accented query): %v", err)
	}
	if len(results) != 1 || results[0].ID != p1.ID {
		t.Fatalf("expected to find José via accented query, got %+v", results)
	}

	// Unaccented query should find "Peña" via SearchForSelection too.
	selection, err := repo.SearchForSelection(ctx, "pena", "M")
	if err != nil {
		t.Fatalf("SearchForSelection: %v", err)
	}
	if len(selection) != 1 || selection[0].ID != p2.ID {
		t.Fatalf("expected to find Peña via unaccented query, got %+v", selection)
	}
}

func TestPlayerRepository_SaveMultiple(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	if err := repo.SaveMultiple(ctx, nil); err != nil {
		t.Fatalf("SaveMultiple (empty): %v", err)
	}

	p1 := newTestPlayer(t, "Multi", "One", "M")
	p2 := newTestPlayer(t, "Multi", "Two", "F")
	if err := repo.SaveMultiple(ctx, []*player.Player{p1, p2}); err != nil {
		t.Fatalf("SaveMultiple: %v", err)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 players, got %d", len(all))
	}

	p1.LastName = "OneUpdated"
	if err := repo.SaveMultiple(ctx, []*player.Player{p1}); err != nil {
		t.Fatalf("SaveMultiple (update): %v", err)
	}
	got, err := repo.GetById(ctx, p1.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.LastName != "OneUpdated" {
		t.Fatalf("expected updated last name, got %q", got.LastName)
	}
}

func TestPlayerRepository_SaveMultiple_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	bad := &player.Player{ID: "not-a-uuid", FirstName: "X", LastName: "Y"}
	if err := repo.SaveMultiple(ctx, []*player.Player{bad}); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestPlayerRepository_Save_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	bad := &player.Player{ID: "not-a-uuid", FirstName: "X", LastName: "Y"}
	if err := repo.Save(ctx, bad); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestPlayerRepository_GetAllSinglesAndDoubles_QueryError(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancelled context forces the underlying query to fail

	if _, err := repo.GetAllSingles(ctx); err == nil {
		t.Fatal("expected error from GetAllSingles with a cancelled context")
	}
	if _, err := repo.GetAllDoubles(ctx); err == nil {
		t.Fatal("expected error from GetAllDoubles with a cancelled context")
	}
	if _, err := repo.GetSinglesByGender(ctx, "M"); err == nil {
		t.Fatal("expected error from GetSinglesByGender with a cancelled context")
	}
	if _, err := repo.GetDoublesByGender(ctx, "F"); err == nil {
		t.Fatal("expected error from GetDoublesByGender with a cancelled context")
	}
}

func TestPlayerRepository_GuardianAccountID_RoundTrip(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	accountID := uuid.NewString()
	p, err := player.NewGuardianChildPlayer(uuid.NewString(), accountID, "Kid", "Smith", time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "Managua")
	if err != nil {
		t.Fatalf("NewGuardianChildPlayer: %v", err)
	}

	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetById(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.GuardianAccountID == nil || *got.GuardianAccountID != accountID {
		t.Fatalf("expected GuardianAccountID %q round-tripped, got %+v", accountID, got.GuardianAccountID)
	}
}

func TestPlayerRepository_Save_NilGuardianAccountID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p := newTestPlayer(t, "No", "Guardian", "M")
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetById(ctx, p.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.GuardianAccountID != nil {
		t.Fatalf("expected nil GuardianAccountID, got %v", got.GuardianAccountID)
	}
}

func TestPlayerRepository_Save_InvalidGuardianAccountID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p := newTestPlayer(t, "Bad", "Guardian", "M")
	bad := "not-a-uuid"
	p.GuardianAccountID = &bad

	if err := repo.Save(ctx, p); err == nil {
		t.Fatal("expected error for invalid GuardianAccountID UUID")
	}
}

func TestPlayerRepository_GetByGuardianAccountID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	accountID := uuid.NewString()
	otherAccountID := uuid.NewString()

	child1, _ := player.NewGuardianChildPlayer(uuid.NewString(), accountID, "Kid1", "Smith", time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "")
	child2, _ := player.NewGuardianChildPlayer(uuid.NewString(), accountID, "Kid2", "Smith", time.Date(2014, 1, 1, 0, 0, 0, 0, time.UTC), "F", "NIC", "")
	otherChild, _ := player.NewGuardianChildPlayer(uuid.NewString(), otherAccountID, "Other", "Kid", time.Date(2013, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "")
	unlinked := newTestPlayer(t, "Adult", "Player", "M")

	for _, p := range []*player.Player{child1, child2, otherChild, unlinked} {
		if err := repo.Save(ctx, p); err != nil {
			t.Fatalf("Save: %v", err)
		}
	}

	linked, err := repo.GetByGuardianAccountID(ctx, accountID)
	if err != nil {
		t.Fatalf("GetByGuardianAccountID: %v", err)
	}
	if len(linked) != 2 {
		t.Fatalf("expected 2 linked players, got %d: %+v", len(linked), linked)
	}

	none, err := repo.GetByGuardianAccountID(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("GetByGuardianAccountID: %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected no players for unused account, got %d", len(none))
	}

	if _, err := repo.GetByGuardianAccountID(ctx, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid account UUID")
	}
}

func TestPlayerRepository_SaveMultiple_GuardianAccountID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	accountID := uuid.NewString()
	child, _ := player.NewGuardianChildPlayer(uuid.NewString(), accountID, "Kid", "Smith", time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "")

	if err := repo.SaveMultiple(ctx, []*player.Player{child}); err != nil {
		t.Fatalf("SaveMultiple: %v", err)
	}

	got, err := repo.GetById(ctx, child.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.GuardianAccountID == nil || *got.GuardianAccountID != accountID {
		t.Fatalf("expected GuardianAccountID round-tripped via SaveMultiple, got %+v", got.GuardianAccountID)
	}
}
