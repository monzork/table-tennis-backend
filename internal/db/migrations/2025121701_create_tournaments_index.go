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
			CREATE INDEX IF NOT EXISTS idx_tournaments_dates
				ON tournaments (start_date, end_date);

			CREATE INDEX IF NOT EXISTS idx_matches_tournament
				ON matches (tournament_id);

			CREATE INDEX IF NOT EXISTS idx_matches_status
				ON matches (status);

			CREATE INDEX IF NOT EXISTS idx_match_teams_match
				ON match_teams (match_id);

			CREATE INDEX IF NOT EXISTS idx_match_team_players_team
				ON match_team_players (match_team_id);

			CREATE INDEX IF NOT EXISTS idx_match_sets_match
				ON match_sets (match_id);

			CREATE INDEX IF NOT EXISTS idx_elo_history_player
				ON elo_history (player_id);
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(`
			DROP INDEX IF EXISTS idx_tournaments_dates;
			DROP INDEX IF EXISTS idx_matches_tournament;
			DROP INDEX IF EXISTS idx_matches_status;
			DROP INDEX IF EXISTS idx_match_teams_match;
			DROP INDEX IF EXISTS idx_match_team_players_team;
			DROP INDEX IF EXISTS idx_match_sets_match;
			DROP INDEX IF EXISTS idx_elo_history_player;
			`)
			return err
		},
	)
}
