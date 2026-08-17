package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/account"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestAccountRepository_SaveAndGetByGoogleSub(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAccountRepository(db)
	ctx := context.Background()

	a, err := account.NewAccount(uuid.NewString(), "sub-1", "alice@x.com", "Alice", "http://pic")
	if err != nil {
		t.Fatalf("NewAccount: %v", err)
	}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByGoogleSub(ctx, "sub-1")
	if err != nil {
		t.Fatalf("GetByGoogleSub: %v", err)
	}
	if got == nil || got.ID != a.ID || got.Email != "alice@x.com" || got.Name != "Alice" || got.PictureURL != "http://pic" {
		t.Fatalf("unexpected account: %+v", got)
	}
}

func TestAccountRepository_GetByGoogleSub_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAccountRepository(db)
	ctx := context.Background()

	got, err := repo.GetByGoogleSub(ctx, "nobody")
	if err != nil {
		t.Fatalf("expected no error for missing sub, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil account, got %+v", got)
	}
}

func TestAccountRepository_GetByEmail(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAccountRepository(db)
	ctx := context.Background()

	a, _ := account.NewAccount(uuid.NewString(), "sub-2", "bob@x.com", "Bob", "")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByEmail(ctx, "bob@x.com")
	if err != nil || got == nil || got.ID != a.ID {
		t.Fatalf("unexpected result: %+v, err=%v", got, err)
	}

	notFound, err := repo.GetByEmail(ctx, "missing@x.com")
	if err != nil {
		t.Fatalf("expected no error for missing email, got %v", err)
	}
	if notFound != nil {
		t.Fatalf("expected nil for missing email, got %+v", notFound)
	}
}

func TestAccountRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAccountRepository(db)
	ctx := context.Background()

	a, _ := account.NewAccount(uuid.NewString(), "sub-3", "carol@x.com", "Carol", "")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repo.GetByID(ctx, a.ID)
	if err != nil || got == nil || got.Email != "carol@x.com" {
		t.Fatalf("unexpected result: %+v, err=%v", got, err)
	}

	if _, err := repo.GetByID(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing account ID")
	}

	if _, err := repo.GetByID(ctx, "not-a-uuid"); err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestAccountRepository_Save_UpsertsOnConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAccountRepository(db)
	ctx := context.Background()

	id := uuid.NewString()
	a, _ := account.NewAccount(id, "sub-4", "dave@x.com", "Dave", "")
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("Save: %v", err)
	}

	a2, _ := account.NewAccount(id, "sub-4", "dave2@x.com", "Dave Updated", "http://pic2")
	if err := repo.Save(ctx, a2); err != nil {
		t.Fatalf("Save (update): %v", err)
	}

	got, err := repo.GetByGoogleSub(ctx, "sub-4")
	if err != nil {
		t.Fatalf("GetByGoogleSub: %v", err)
	}
	if got.Email != "dave2@x.com" || got.Name != "Dave Updated" || got.PictureURL != "http://pic2" {
		t.Fatalf("expected updated fields, got %+v", got)
	}
}

func TestAccountRepository_Save_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewAccountRepository(db)
	ctx := context.Background()

	a := &account.Account{ID: "not-a-uuid", GoogleSub: "s", Email: "e@x.com"}
	if err := repo.Save(ctx, a); err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}
