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
				CREATE TABLE IF NOT EXISTS "users" (
				"id"			TEXT,
				"username"		TEXT NOT NULL,
				"password"		TEXT NOT NULL,
				"created_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
				"updated_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
				PRIMARY KEY("id") );
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := DB.Exec(`DROP TABLE "users"`)
			return err
		},
	)
}
