package bun

import (
	"context"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/google/uuid"
	bun "github.com/uptrace/bun"
)

// DivisionRuleModel represents the tournament_division_rules table.
type DivisionRuleModel struct {
	bun.BaseModel `bun:"table:tournament_division_rules,alias:dr"`

	ID           string `bun:"id,pk"`
	TournamentID string `bun:"tournament_id,notnull"`
	DivisionID   string `bun:"division_id,notnull"`
	Stage        string `bun:"stage,notnull"`
	BestOf       int    `bun:"best_of,notnull"`
	PointsToWin  int    `bun:"points_to_win,notnull"`
	PointsMargin int    `bun:"points_margin,notnull"`
	CreatedAt    string `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    string `bun:"updated_at,notnull,default:current_timestamp"`
}

// ToDomain converts a DivisionRuleModel to a domain DivisionRule.
func (m *DivisionRuleModel) ToDomain() tournament.DivisionRule {
	return tournament.DivisionRule{
		ID:           m.ID,
		TournamentID: m.TournamentID,
		DivisionID:   m.DivisionID,
		Stage:        m.Stage,
		BestOf:       m.BestOf,
		PointsToWin:  m.PointsToWin,
		PointsMargin: m.PointsMargin,
	}
}

// FromDomain converts a domain DivisionRule to a DivisionRuleModel.
func FromDomainDivisionRule(dr tournament.DivisionRule) *DivisionRuleModel {
	id := dr.ID
	if id == "" {
		id = uuid.NewString()
	} else if _, err := uuid.Parse(id); err != nil {
		// Convert deterministic domain string to a valid UUID format
		id = uuid.NewSHA1(uuid.NameSpaceURL, []byte(dr.ID)).String()
	}

	return &DivisionRuleModel{
		ID:           id,
		TournamentID: dr.TournamentID,
		DivisionID:   dr.DivisionID,
		Stage:        dr.Stage,
		BestOf:       dr.BestOf,
		PointsToWin:  dr.PointsToWin,
		PointsMargin: dr.PointsMargin,
	}
}

// LoadDivisionRules fetches all division rules for a tournament.
func LoadDivisionRules(ctx context.Context, db *bun.DB, tournamentID string) []tournament.DivisionRule {
	var models []DivisionRuleModel
	_ = db.NewSelect().Model(&models).Where("tournament_id = ?", tournamentID).Scan(ctx)

	rules := make([]tournament.DivisionRule, len(models))
	for i, m := range models {
		rules[i] = m.ToDomain()
	}
	return rules
}

// SaveDivisionRules inserts all division rules inside a transaction, stamping
// tournamentID onto each rule so callers don't need to set it themselves.
func SaveDivisionRules(ctx context.Context, tx bun.IDB, tournamentID string, rules []tournament.DivisionRule) error {
	if len(rules) == 0 {
		return nil
	}
	models := make([]DivisionRuleModel, len(rules))
	for i, r := range rules {
		r.TournamentID = tournamentID
		models[i] = *FromDomainDivisionRule(r)
	}
	_, err := tx.NewInsert().Model(&models).Exec(ctx)
	return err
}

// ReplaceDivisionRules deletes old rules and re-inserts new ones inside a transaction.
func ReplaceDivisionRules(ctx context.Context, tx bun.IDB, tournamentID string, rules []tournament.DivisionRule) error {
	if _, err := tx.NewDelete().TableExpr("tournament_division_rules").
		Where("tournament_id = ?", tournamentID).Exec(ctx); err != nil {
		return err
	}
	return SaveDivisionRules(ctx, tx, tournamentID, rules)
}

// GetDivisionRule retrieves a specific division rule for a tournament.
func GetDivisionRule(ctx context.Context, db *bun.DB, tournamentID, divisionID string) (*DivisionRuleModel, error) {
	m := new(DivisionRuleModel)
	err := db.NewSelect().Model(m).
		Where("tournament_id = ?", tournamentID).
		Where("division_id = ?", divisionID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// DeleteDivisionRule deletes a specific division rule.
func DeleteDivisionRule(ctx context.Context, db *bun.DB, tournamentID, divisionID string) error {
	_, err := db.NewDelete().TableExpr("tournament_division_rules").
		Where("tournament_id = ?", tournamentID).
		Where("division_id = ?", divisionID).
		Exec(ctx)
	return err
}
