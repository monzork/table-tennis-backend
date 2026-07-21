package bun_test

import (
	"context"
	"errors"
	"testing"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

func TestExtractDB_NoTxReturnsFallback(t *testing.T) {
	db := setupTestDB(t)
	got := bunRepo.ExtractDB(context.Background(), db)
	if got != db {
		t.Fatalf("expected fallback db, got different IDB")
	}
}

func TestExtractDB_WithTxReturnsTx(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		injected := bunRepo.InjectTx(ctx, tx)
		got := bunRepo.ExtractDB(injected, db)
		if got != tx {
			t.Fatalf("expected ExtractDB to return the injected tx")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("RunInTx: %v", err)
	}
}

func TestRunInTx_CommitsOnSuccess(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := bunRepo.NewAdminRepository(db)

	err := bunRepo.RunInTx(ctx, db, func(ctx context.Context, tx bun.Tx) error {
		a := &bunRepo.AdminModel{ID: uuid.New(), Username: "txuser", PasswordHash: "hash"}
		_, err := tx.NewInsert().Model(a).Exec(ctx)
		return err
	})
	if err != nil {
		t.Fatalf("RunInTx: %v", err)
	}

	got, err := repo.GetByUsername(ctx, "txuser")
	if err != nil {
		t.Fatalf("GetByUsername: %v", err)
	}
	if got.Username != "txuser" {
		t.Fatalf("unexpected admin: %+v", got)
	}
}

func TestRunInTx_RollsBackOnError(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	repo := bunRepo.NewAdminRepository(db)

	wantErr := errors.New("boom")
	err := bunRepo.RunInTx(ctx, db, func(ctx context.Context, tx bun.Tx) error {
		a := &bunRepo.AdminModel{ID: uuid.New(), Username: "rollback-user", PasswordHash: "hash"}
		if _, err := tx.NewInsert().Model(a).Exec(ctx); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected wrapped wantErr, got %v", err)
	}

	if _, err := repo.GetByUsername(ctx, "rollback-user"); err == nil {
		t.Fatal("expected rolled-back insert to not be visible")
	}
}

func TestRunInTx_ReusesExistingTx(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	err := db.RunInTx(ctx, nil, func(ctx context.Context, outerTx bun.Tx) error {
		injected := bunRepo.InjectTx(ctx, outerTx)
		return bunRepo.RunInTx(injected, db, func(innerCtx context.Context, innerTx bun.Tx) error {
			if innerTx != outerTx {
				t.Fatalf("expected RunInTx to reuse the outer transaction")
			}
			return nil
		})
	})
	if err != nil {
		t.Fatalf("RunInTx: %v", err)
	}
}
