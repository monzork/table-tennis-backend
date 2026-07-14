package main

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/driver/pgdriver"
)

func main() {
	_ = godotenv.Load()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL not set")
	}

	if u, err := url.Parse(dsn); err == nil {
		q := u.Query()
		if q.Has("channel_binding") {
			q.Del("channel_binding")
			u.RawQuery = q.Encode()
			dsn = u.String()
		}
	}

	sqldb := sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
	bunDB := bun.NewDB(sqldb, pgdialect.New())
	defer sqldb.Close()

	ctx := context.Background()

	// Add missing columns
	_, err := bunDB.NewRaw(`ALTER TABLE event_division_rules ADD COLUMN IF NOT EXISTS created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`).Exec(ctx)
	if err != nil {
		log.Fatal("Failed to add created_at column:", err)
	}

	_, err = bunDB.NewRaw(`ALTER TABLE event_division_rules ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`).Exec(ctx)
	if err != nil {
		log.Fatal("Failed to add updated_at column:", err)
	}

	log.Println("Successfully added created_at and updated_at columns")
}
