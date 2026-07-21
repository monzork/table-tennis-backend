package bun_test

import (
	"context"
	"errors"
	"testing"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestBunTransactionManager_RunInTransaction_Commits(t *testing.T) {
	db := setupTestDB(t)
	mgr := bunRepo.NewBunTransactionManager(db)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	a := &bunRepo.AdminModel{ID: uuid.New(), Username: "uowuser", PasswordHash: "hash"}
	err := mgr.RunInTransaction(ctx, func(ctx context.Context) error {
		_, err := bunRepo.ExtractDB(ctx, db).NewInsert().Model(a).Exec(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("RunInTransaction: %v", err)
	}

	got, err := repo.GetByUsername(ctx, "uowuser")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.Username != "uowuser" {
		t.Fatalf("unexpected admin: %+v", got)
	}
}

func TestBunTransactionManager_RunInTransaction_RollsBackOnError(t *testing.T) {
	db := setupTestDB(t)
	mgr := bunRepo.NewBunTransactionManager(db)
	repo := bunRepo.NewAdminRepository(db)
	ctx := context.Background()

	wantErr := errors.New("boom")
	a := &bunRepo.AdminModel{ID: uuid.New(), Username: "uow-rollback", PasswordHash: "hash"}
	err := mgr.RunInTransaction(ctx, func(ctx context.Context) error {
		if _, err := bunRepo.ExtractDB(ctx, db).NewInsert().Model(a).Exec(ctx); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got %v", err)
	}

	if _, err := repo.GetByUsername(ctx, "uow-rollback"); err == nil {
		t.Fatal("expected rolled-back insert to not be visible")
	}
}
