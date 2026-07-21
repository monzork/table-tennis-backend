package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/division"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestDivisionRepository_SaveGetAllGetByIdDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewDivisionRepository(db)
	ctx := context.Background()

	maxElo := int16(1500)
	d1, err := division.NewDivision(uuid.NewString(), "Division 1", 1, 1000, &maxElo, "singles", "#ff0000")
	if err != nil {
		t.Fatalf("NewDivision: %v", err)
	}
	d2, err := division.NewDivision(uuid.NewString(), "Division 2", 2, 1500, nil, "both", "")
	if err != nil {
		t.Fatalf("NewDivision: %v", err)
	}

	if err := repo.Save(ctx, d1); err != nil {
		t.Fatalf("Save d1: %v", err)
	}
	if err := repo.Save(ctx, d2); err != nil {
		t.Fatalf("Save d2: %v", err)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 divisions, got %d", len(all))
	}
	if all[0].DisplayOrder > all[1].DisplayOrder {
		t.Fatalf("expected divisions ordered by display_order asc")
	}

	got, err := repo.GetById(ctx, d1.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Name != "Division 1" || got.Color != "#ff0000" || *got.MaxElo != 1500 {
		t.Fatalf("unexpected division: %+v", got)
	}

	if err := repo.Delete(ctx, d1.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	all, err = repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll after delete: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 division after delete, got %d", len(all))
	}
}

func TestDivisionRepository_GetById_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewDivisionRepository(db)
	ctx := context.Background()

	if _, err := repo.GetById(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing division, got nil")
	}
}

func TestDivisionRepository_Save_Upsert(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewDivisionRepository(db)
	ctx := context.Background()

	id := uuid.NewString()
	d, _ := division.NewDivision(id, "Original", 1, 0, nil, "both", "")
	if err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save: %v", err)
	}

	d.Name = "Renamed"
	if err := repo.Save(ctx, d); err != nil {
		t.Fatalf("Save (update): %v", err)
	}

	got, err := repo.GetById(ctx, id)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.Name != "Renamed" {
		t.Fatalf("expected updated name, got %q", got.Name)
	}
}
