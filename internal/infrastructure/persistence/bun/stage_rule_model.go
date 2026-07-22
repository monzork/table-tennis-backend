package bun

import (
	"context"
	"table-tennis-backend/internal/domain/event"

	"github.com/google/uuid"
	bun "github.com/uptrace/bun"
)

type StageRuleModel struct {
	bun.BaseModel `bun:"table:event_stage_rules,alias:sr"`

	ID           string `bun:"id,pk"`
	TournamentID string `bun:"event_id,notnull"`
	Stage        string `bun:"stage,notnull"`
	BestOf       int    `bun:"best_of,notnull"`
	PointsToWin  int    `bun:"points_to_win,notnull"`
	PointsMargin int    `bun:"points_margin,notnull"`
}

func stageRuleToModel(r event.StageRule) *StageRuleModel {
	id := r.ID
	if id == "" {
		id = uuid.NewString()
	} else if _, err := uuid.Parse(id); err != nil {
		// Convert deterministic domain string to a valid UUID format
		id = uuid.NewSHA1(uuid.NameSpaceURL, []byte(r.ID)).String()
	}
	return &StageRuleModel{
		ID:           id,
		TournamentID: r.TournamentID,
		Stage:        r.Stage,
		BestOf:       r.BestOf,
		PointsToWin:  r.PointsToWin,
		PointsMargin: r.PointsMargin,
	}
}

func stageRuleToDomain(m StageRuleModel) event.StageRule {
	return event.StageRule{
		ID:           m.ID,
		TournamentID: m.TournamentID,
		Stage:        m.Stage,
		BestOf:       m.BestOf,
		PointsToWin:  m.PointsToWin,
		PointsMargin: m.PointsMargin,
	}
}

// saveStageRules inserts all stage rules inside a transaction.
func saveStageRules(ctx context.Context, tx bun.IDB, rules []event.StageRule) error {
	if len(rules) == 0 {
		return nil
	}
	models := make([]StageRuleModel, len(rules))
	for i, r := range rules {
		models[i] = *stageRuleToModel(r)
	}
	_, err := tx.NewInsert().Model(&models).Exec(ctx)
	return err
}

// replaceStageRules deletes old rules and re-inserts new ones inside a transaction.
func replaceStageRules(ctx context.Context, tx bun.IDB, tournamentID string, rules []event.StageRule) error {
	if _, err := tx.NewDelete().TableExpr("event_stage_rules").
		Where("event_id = ?", tournamentID).Exec(ctx); err != nil {
		return err
	}
	return saveStageRules(ctx, tx, rules)
}

// GetStageRule retrieves a specific stage rule for a event by stage name.
func GetStageRule(ctx context.Context, db *bun.DB, tournamentID uuid.UUID, stage string) (*StageRuleModel, error) {
	m := new(StageRuleModel)
	err := db.NewSelect().Model(m).
		Where("event_id = ?", tournamentID.String()).
		Where("stage = ?", stage).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}
