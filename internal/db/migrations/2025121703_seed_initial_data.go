package migrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun"
)

func init() {
	db.Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {

			tournamentID := uuid.NewString()

			_, err := db.Exec(`
			-- Rules
			INSERT OR IGNORE INTO rules (id, name, description)
			VALUES
				(1, 'Best of 5', 'First to 3 sets wins'),
				(2, '11 Point Sets', 'Sets played to 11 points');

			-- Tournament
			INSERT OR IGNORE INTO tournaments (
				id, name, description, start_date, end_date
			) VALUES (
				?, 'Test Open', 'Seed tournament',
				CURRENT_TIMESTAMP,
				CURRENT_TIMESTAMP
			);

			-- Tournament rules
			INSERT OR IGNORE INTO tournament_rules (
				tournament_id, rule_id
			) VALUES
				(?, 1),
				(?, 2);
			`, tournamentID, tournamentID, tournamentID)

			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			// Seeds usually not rolled back
			return nil
		},
	)
}
