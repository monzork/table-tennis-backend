package bun

import (
	"context"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	bun "github.com/uptrace/bun"
)

type StageRuleModel struct {
	bun.BaseModel `bun:"table:tournament_stage_rules,alias:sr"`

	ID           string `bun:"id,pk"`
	TournamentID string `bun:"tournament_id,notnull"`
	Stage        string `bun:"stage,notnull"`
	BestOf       int    `bun:"best_of,notnull"`
	PointsToWin  int    `bun:"points_to_win,notnull"`
	PointsMargin int    `bun:"points_margin,notnull"`
}

func stageRuleToModel(r tournament.StageRule) *StageRuleModel {
	return &StageRuleModel{
		ID:           r.ID.String(),
		TournamentID: r.TournamentID.String(),
		Stage:        r.Stage,
		BestOf:       r.BestOf,
		PointsToWin:  r.PointsToWin,
		PointsMargin: r.PointsMargin,
	}
}

func stageRuleToDomain(m StageRuleModel) tournament.StageRule {
	id, _ := uuid.Parse(m.ID)
	tid, _ := uuid.Parse(m.TournamentID)
	return tournament.StageRule{
		ID:           id,
		TournamentID: tid,
		Stage:        m.Stage,
		BestOf:       m.BestOf,
		PointsToWin:  m.PointsToWin,
		PointsMargin: m.PointsMargin,
	}
}

// loadStageRules fetches all stage rules for a tournament (shared helper).
func loadStageRules(ctx context.Context, db *bun.DB, tournamentID uuid.UUID) []tournament.StageRule {
	var models []StageRuleModel
	_ = db.NewSelect().Model(&models).Where("tournament_id = ?", tournamentID).Scan(ctx)
	rules := make([]tournament.StageRule, len(models))
	for i, m := range models {
		rules[i] = stageRuleToDomain(m)
	}
	return rules
}

// saveStageRules inserts all stage rules inside a transaction.
func saveStageRules(ctx context.Context, tx bun.IDB, rules []tournament.StageRule) error {
	for _, r := range rules {
		m := stageRuleToModel(r)
		if _, err := tx.NewInsert().Model(m).Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// replaceStageRules deletes old rules and re-inserts new ones inside a transaction.
func replaceStageRules(ctx context.Context, tx bun.IDB, tournamentID uuid.UUID, rules []tournament.StageRule) error {
	if _, err := tx.NewDelete().TableExpr("tournament_stage_rules").
		Where("tournament_id = ?", tournamentID).Exec(ctx); err != nil {
		return err
	}
	return saveStageRules(ctx, tx, rules)
}
