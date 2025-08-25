package migrations

import (
	"context"

	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun"
)

func init() {
	DB := db.Connect()

	db.Migrations.MustRegister(
		func(ctx context.Context, db *bun.DB) error {
			_, err := DB.Exec(`
				CREATE TABLE IF NOT EXISTS "players" (
				"id"			TEXT,
				"name"			TEXT NOT NULL,
				"sex"			TEXT NOT NULL,
				"country"		TEXT NOT NULL,
				"city"			TEXT NOT NULL,
				"birthdate"		TEXT NOT NULL,
				"elo"		 	INTEGER NOT NULL DEFAULT 1000,
				"created_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
				"deleted_at" 	TEXT,
				PRIMARY KEY("id")
				);
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := DB.Exec(`DROP TABLE "players"`)
			return err
		},
	)
}
