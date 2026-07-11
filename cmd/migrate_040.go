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

	// Add division_group_counts column to tournaments table
	_, err := bun.DB.ExecContext(ctx, `
		ALTER TABLE tournaments 
		ADD COLUMN division_group_counts JSONB NOT NULL DEFAULT '{}'::jsonb;
	`)
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Migration 040 completed successfully!")
}
