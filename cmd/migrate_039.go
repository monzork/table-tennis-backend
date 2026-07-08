//go:build ignore

package main

import (
	"context"
	"log"

	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

func main() {
	bun.Connect()
	ctx := context.Background()

	// Add manual_seeding_locked column to tournaments table
	_, err := bun.DB.ExecContext(ctx, `
		ALTER TABLE tournaments 
		ADD COLUMN manual_seeding_locked BOOLEAN NOT NULL DEFAULT false;
	`)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Migration 039 completed successfully!")
}
