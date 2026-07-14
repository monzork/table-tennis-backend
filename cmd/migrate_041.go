//go:build ignore

package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

func main() {
	godotenv.Load()
	bun.Connect()
	ctx := context.Background()

	sql, err := os.ReadFile("cmd/migrations/041_swap_event_and_tournament.sql")
	if err != nil {
		log.Fatal("Failed to read migration file:", err)
	}

	_, err = bun.DB.ExecContext(ctx, string(sql))
	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	log.Println("Migration 041 completed successfully!")
}
