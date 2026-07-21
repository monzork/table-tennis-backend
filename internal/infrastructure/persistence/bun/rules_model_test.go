package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/event"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// Exercises the exported helper functions in division_rule_model.go and
// stage_rule_model.go directly, since most of them are only partially
// touched by the higher-level repository flows.

func TestDivisionRuleModel_ToDomainAndFromDomain(t *testing.T) {
	m := &bunRepo.DivisionRuleModel{
		ID:           "rule-1",
		TournamentID: "t-1",
		DivisionID:   "d-1",
		Stage:        "final",
		BestOf:       5,
		PointsToWin:  11,
		PointsMargin: 2,
	}
	d := m.ToDomain()
	if d.ID != "rule-1" || d.DivisionID != "d-1" || d.BestOf != 5 {
		t.Fatalf("unexpected domain conversion: %+v", d)
	}

	back := bunRepo.FromDomainDivisionRule(d)
	if back.TournamentID != "t-1" || back.Stage != "final" {
		t.Fatalf("unexpected model conversion: %+v", back)
	}

	// A non-UUID ID should be deterministically converted to a UUID-shaped ID.
	dr2 := event.DivisionRule{ID: "not-a-uuid", TournamentID: "t-1", DivisionID: "d-2", Stage: "group", BestOf: 3, PointsToWin: 11, PointsMargin: 2}
	m2 := bunRepo.FromDomainDivisionRule(dr2)
	if m2.ID == "not-a-uuid" || m2.ID == "" {
		t.Fatalf("expected deterministic UUID conversion, got %q", m2.ID)
	}

	// An empty ID should generate a fresh UUID.
	dr3 := event.DivisionRule{TournamentID: "t-1", DivisionID: "d-3", Stage: "group"}
	m3 := bunRepo.FromDomainDivisionRule(dr3)
	if m3.ID == "" {
		t.Fatal("expected a generated UUID for empty ID")
	}
}

func TestDivisionRuleModel_SaveLoadGetReplaceDelete(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	tournamentID := "tournament-with-rules"

	rules := []event.DivisionRule{
		{DivisionID: "d-1", Stage: "final", BestOf: 7, PointsToWin: 11, PointsMargin: 2},
		{DivisionID: "d-2", Stage: "semifinal", BestOf: 5, PointsToWin: 11, PointsMargin: 2},
	}

	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return bunRepo.SaveDivisionRules(ctx, tx, tournamentID, rules)
	})
	if err != nil {
		t.Fatalf("SaveDivisionRules: %v", err)
	}

	loaded := bunRepo.LoadDivisionRules(ctx, db, tournamentID)
	if len(loaded) != 2 {
		t.Fatalf("expected 2 loaded rules, got %d", len(loaded))
	}

	got, err := bunRepo.GetDivisionRule(ctx, db, tournamentID, "d-1")
	if err != nil {
		t.Fatalf("GetDivisionRule: %v", err)
	}
	if got.Stage != "final" || got.BestOf != 7 {
		t.Fatalf("unexpected division rule: %+v", got)
	}

	if _, err := bunRepo.GetDivisionRule(ctx, db, tournamentID, "missing-division"); err == nil {
		t.Fatal("expected error for missing division rule")
	}

	replacement := []event.DivisionRule{
		{DivisionID: "d-3", Stage: "group", BestOf: 3, PointsToWin: 11, PointsMargin: 2},
	}
	err = db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return bunRepo.ReplaceDivisionRules(ctx, tx, tournamentID, replacement)
	})
	if err != nil {
		t.Fatalf("ReplaceDivisionRules: %v", err)
	}
	loaded = bunRepo.LoadDivisionRules(ctx, db, tournamentID)
	if len(loaded) != 1 || loaded[0].DivisionID != "d-3" {
		t.Fatalf("expected replaced rules, got %+v", loaded)
	}

	if err := bunRepo.DeleteDivisionRule(ctx, db, tournamentID, "d-3"); err != nil {
		t.Fatalf("DeleteDivisionRule: %v", err)
	}
	loaded = bunRepo.LoadDivisionRules(ctx, db, tournamentID)
	if len(loaded) != 0 {
		t.Fatalf("expected 0 rules after delete, got %d", len(loaded))
	}
}

func TestDivisionRuleModel_SaveDivisionRules_Empty(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	err := db.RunInTx(ctx, nil, func(ctx context.Context, tx bun.Tx) error {
		return bunRepo.SaveDivisionRules(ctx, tx, "t-1", nil)
	})
	if err != nil {
		t.Fatalf("expected no-op for empty rules, got %v", err)
	}
}

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
