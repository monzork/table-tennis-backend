package migrations

import (
	"context"

	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun"
)

func init() {
	db.Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			// Rules
			_, err := db.Exec(`
			CREATE TABLE IF NOT EXISTS "rules" (
				"id"          INTEGER PRIMARY KEY AUTOINCREMENT,
				"name"        TEXT NOT NULL,
				"description" TEXT,
				"created_at"  TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at"  TEXT,
				"deleted_at"  TEXT DEFAULT NULL
			);
		`)
			if err != nil {
				return err
			}

			// Tournaments
			_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS "tournaments" (
				"id"          TEXT PRIMARY KEY,
				"name"        TEXT NOT NULL,
				"description" TEXT,
				"start_date"  TEXT NOT NULL,
				"end_date"    TEXT NOT NULL,
				"created_at"  TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at"  TEXT,
				"deleted_at"  TEXT DEFAULT NULL
			);
		`)
			if err != nil {
				return err
			}

			// TournamentRule (m2m)
			_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS "tournament_rules" (
				"id"            INTEGER PRIMARY KEY AUTOINCREMENT,
				"tournament_id" TEXT NOT NULL,
				"rule_id"       INTEGER NOT NULL,
				"created_at"    TEXT DEFAULT CURRENT_TIMESTAMP,
				FOREIGN KEY("tournament_id") REFERENCES "tournaments"("id"),
				FOREIGN KEY("rule_id") REFERENCES "rules"("id")
			);
		`)
			if err != nil {
				return err
			}

			// Matches
			_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS "matches" (
				"id"            TEXT PRIMARY KEY,
				"tournament_id" TEXT NOT NULL,
				"player_a"      TEXT NOT NULL,
				"player_b"      TEXT NOT NULL,
				"created_at"    TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at"    TEXT,
				"deleted_at"    TEXT DEFAULT NULL,
				FOREIGN KEY("tournament_id") REFERENCES "tournaments"("id")
			);
		`)
			if err != nil {
				return err
			}

			// Sets
			_, err = db.Exec(`
			CREATE TABLE IF NOT EXISTS "sets" (
				"id"         TEXT PRIMARY KEY,
				"match_id"   TEXT NOT NULL,
				"number"     INTEGER NOT NULL,
				"score_a"    INTEGER NOT NULL,
				"score_b"    INTEGER NOT NULL,
				"created_at" TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at" TEXT,
				"deleted_at" TEXT DEFAULT NULL,
				FOREIGN KEY("match_id") REFERENCES "matches"("id")
			);
		`)
			if err != nil {
				return err
			}

			return nil
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := db.Exec(`
			DROP TABLE IF EXISTS "sets";
			DROP TABLE IF EXISTS "matches";
			DROP TABLE IF EXISTS "tournament_rules";
			DROP TABLE IF EXISTS "tournaments";
			DROP TABLE IF EXISTS "rules";
		`)
			return err
		},
	)
}
