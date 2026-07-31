package bun_test

import (
	"context"
	"testing"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// Exercises the exported helper functions in stage_rule_model.go directly,
// since most of them are only partially touched by the higher-level
// repository flows.

func TestStageRuleModel_GetStageRule(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	e := newBareEvent(t, "Stage Rule Event", nil)
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	tID, err := uuid.Parse(e.ID)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	got, err := bunRepo.GetStageRule(ctx, db, tID, "group")
	if err != nil {
		t.Fatalf("GetStageRule: %v", err)
	}
	if got.BestOf != 5 || got.PointsToWin != 11 {
		t.Fatalf("unexpected stage rule: %+v", got)
	}

	if _, err := bunRepo.GetStageRule(ctx, db, tID, "nonexistent-stage"); err == nil {
		t.Fatal("expected error for missing stage rule")
	}
}
