package migrations

import (
	"context"
	"fmt"

	"github.com/monzork/table-tennis-backend/internal/db"
	"github.com/uptrace/bun/migrate"
)

var Migrations = migrate.NewMigrations()

func init() {

	ctx := context.Background()

	// Run migrations
	migrator := migrate.NewMigrator(db.DB, Migrations)
	err := migrator.Lock(ctx)
	if err != nil {
		fmt.Printf("Failed to lock migrations: %v\n", err)
		return
	}
	defer migrator.Unlock(ctx)

	group, err := migrator.Migrate(ctx)

	if err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		return
	}
	fmt.Printf("Migrated %v migrations\n", group.Migrations)

}
