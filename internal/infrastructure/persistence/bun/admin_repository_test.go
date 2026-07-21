package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/admin"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestAdminRepository_SaveAndGetByUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	a, err := admin.NewAdmin(uuid.NewString(), "alice", "hashedpw")
	if err != nil {
		t.Fatalf("NewAdmin: %v", err)
	}

	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByUsername(ctx, "alice")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.ID != a.ID || got.Username != "alice" || got.PasswordHash != "hashedpw" {
		t.Fatalf("unexpected admin: %+v", got)
	}
}

func TestAdminRepository_Save_UpsertsOnConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	id := uuid.NewString()
	a, _ := admin.NewAdmin(id, "bob", "pw1")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	a2, _ := admin.NewAdmin(id, "bob2", "pw2")
	if err := repo.Save(ctx, a2); err != nil {
		t.Fatalf("Save (update): %v", err)
	}

	got, err := repo.GetByUsername(ctx, "bob2")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.PasswordHash != "pw2" {
		t.Fatalf("expected updated password hash, got %q", got.PasswordHash)
	}
}

func TestAdminRepository_GetByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	if _, err := repo.GetByUsername(ctx, "nobody"); err == nil {
		t.Fatal("expected error for missing admin, got nil")
	}
}

func TestAdminRepository_Save_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	a := &admin.Admin{ID: "not-a-uuid", Username: "x", PasswordHash: "y"}
	if err := repo.Save(ctx, a); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestAdminRepository_Count(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	count, err := repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 admins, got %d", count)
	}

	a, _ := admin.NewAdmin(uuid.NewString(), "carol", "pw")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	count, err = repo.Count(ctx)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 admin, got %d", count)
	}
}
