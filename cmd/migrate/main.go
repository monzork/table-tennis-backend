package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/monzork/table-tennis-backend/internal/db"
	_ "github.com/monzork/table-tennis-backend/internal/db/migrations"
	"github.com/uptrace/bun/migrate"
)

func main() {
	ctx := context.Background()
	dbConn := db.Connect()

	migrator := migrate.NewMigrator(dbConn, db.Migrations)

	if err := migrator.Init(ctx); err != nil {
		log.Fatalf("failed to initialize migrator: %v", err)
	}

	if err := migrator.Unlock(ctx); err != nil {
		log.Printf("failed to unlock migrator: %v", err)
	}

	if err := migrator.Lock(ctx); err != nil {
		log.Fatalf("cannot get migration lock: %v", err)
	}

	if len(os.Args) < 2 {
		log.Fatalf("usage: %s [migrate|rollback|status]", os.Args[0])
	}

	switch os.Args[1] {
	case "migrate":
		print("")
		group, err := migrator.Migrate(ctx)
		if err != nil {
			log.Fatalf("migration failed: %v", err)
		}
		if group.IsZero() {
			fmt.Println("No new migrations to run.")
		} else {
			fmt.Printf("Migrated %d migrations\n", len(group.Migrations))
		}

	case "rollback":
		group, err := migrator.Rollback(ctx)
		if err != nil {
			log.Fatalf("rollback failed: %v", err)
		}
		if group.IsZero() {
			fmt.Println("No migrations to roll back.")
		} else {
			fmt.Printf("Rolled back %d migrations\n", len(group.Migrations))
		}

	default:
		log.Fatalf("unknown command: %s", os.Args[1])
	}
}
