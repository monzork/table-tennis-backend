package migrations

import (
	"context"

	"github.com/uptrace/bun"
)

// Define migration
func init() {
	Migrations.MustRegister(func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`CREATE TABLE "players" IF NOT EXISTS (
	"id"	TEXT,
	"name"	TEXT NOT NULL,
	"sex"	TEXT NOT NULL,
	"country"	TEXT NOT NULL,
	"city"	TEXT NOT NULL,
	"birthdate"	TEXT NOT NULL,
	"elo"	INTEGER NOT NULL DEFAULT 1000,
	"created_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
	"updated_at"	TEXT DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY("id")
);`)
		return err
	}, func(ctx context.Context, db *bun.DB) error {
		_, err := db.Exec(`DROP TABLE players`)
		return err
	})
}
