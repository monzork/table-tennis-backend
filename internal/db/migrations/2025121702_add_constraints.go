package migrations

import (
	"context"

	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun"
)

func init() {
	db.Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {

			_, err := db.Exec(`
			-- Only one A/B team per match
			CREATE UNIQUE INDEX IF NOT EXISTS ux_match_teams_match_side
				ON match_teams (match_id, side);

			-- Prevent duplicate set numbers per match
			CREATE UNIQUE INDEX IF NOT EXISTS ux_match_sets_match_number
				ON match_sets (match_id, number);

			-- Prevent duplicate player positions per team
			CREATE UNIQUE INDEX IF NOT EXISTS ux_match_team_players_position
				ON match_team_players (match_team_id, position);
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(`
			DROP INDEX IF EXISTS ux_match_teams_match_side;
			DROP INDEX IF EXISTS ux_match_sets_match_number;
			DROP INDEX IF EXISTS ux_match_team_players_position;
			`)
			return err
		},
	)
}
