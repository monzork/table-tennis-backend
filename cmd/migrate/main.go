package main

import (
	"context"
	"github.com/monzork/table-tennis-backend/internal/db"
	_ "github.com/monzork/table-tennis-backend/internal/db/migrations"
	"github.com/uptrace/bun/migrate"
	"log"
)

func main() {
	ctx := context.Background()
	dbConn := db.Connect()

	migrator := migrate.NewMigrator(dbConn, db.Migrations)

	if err := migrator.Init(ctx); err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}

	if err := migrator.Lock(ctx); err != nil {
		log.Fatalf("cannot get migration lock: %v", err)
	}
	defer migrator.Unlock(ctx)

	group, err := migrator.Migrate(ctx)
	if err != nil {
		log.Fatalf("migration failed: %v", err)
	}

	log.Printf("migrated %v migrations", group.Migrations)
}
