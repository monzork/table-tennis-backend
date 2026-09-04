package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/inactivity"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestInactivitySettingsRepository_GetDefaultsAndUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewInactivitySettingsRepository(db)
	ctx := context.Background()

	// No row yet: Get falls back to the documented defaults instead of erroring.
	got, err := repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.TournamentThreshold != 4 || got.EloPenalty != 50 {
		t.Fatalf("expected default settings, got %+v", got)
	}

	want := &inactivity.Settings{TournamentThreshold: 6, EloPenalty: 30}
	if err := repo.Update(ctx, want); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if *got != *want {
		t.Fatalf("expected %+v, got %+v", want, got)
	}

	// Update again to exercise the ON CONFLICT upsert path.
	want2 := &inactivity.Settings{TournamentThreshold: 5, EloPenalty: 40}
	if err := repo.Update(ctx, want2); err != nil {
		t.Fatalf("second Update: %v", err)
	}
	got, err = repo.Get(ctx)
	if err != nil {
		t.Fatalf("Get after second update: %v", err)
	}
	if *got != *want2 {
		t.Fatalf("expected %+v, got %+v", want2, got)
	}
}
