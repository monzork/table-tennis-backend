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
				ALTER TABLE "players" ADD COLUMN "email" TEXT;
				ALTER TABLE "players" ADD COLUMN "phone" TEXT;
			`)
			return err
		},
		func(ctx context.Context, db *bun.DB) error {
			_, err := DB.Exec(`
				ALTER TABLE "players" DROP COLUMN "email"; 
		    	ALTER TABLE "players" DROP COLUMN "phone";
			`)
			return err
		},
	)
}
