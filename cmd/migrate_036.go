//go:build ignore

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
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

func main() {
	_ = godotenv.Load()

	var sqldb *sql.DB
	var bunDB *bun.DB
	var err error

	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		log.Println("Using PostgreSQL migration...")
		if u, err := url.Parse(dsn); err == nil {
			q := u.Query()
			if q.Has("channel_binding") {
				q.Del("channel_binding")
				u.RawQuery = q.Encode()
				dsn = u.String()
			}
		}
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		bunDB = bun.NewDB(sqldb, pgdialect.New())
	} else {
		log.Println("Using SQLite migration...")
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "table_tennis.db"
		}
		sqldb, err = sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
		if err != nil {
			log.Fatal(err)
		}
		bunDB = bun.NewDB(sqldb, sqlitedialect.New())
	}
	defer sqldb.Close()

	ctx := context.Background()

	// Handle UUID generation based on database dialect
	uuidDefault := "gen_random_uuid()" // PostgreSQL
	if dsn == "" {
		// SQLite doesn't have gen_random_uuid() built-in, but we generate UUIDs in code.
		// So we just omit the DEFAULT clause for SQLite or set it to empty string.
		uuidDefault = "NULL" // or handle SQLite specifically if needed
	}

	createTableQuery := `CREATE TABLE IF NOT EXISTS push_subscriptions (
		id UUID PRIMARY KEY DEFAULT ` + uuidDefault + `,
		endpoint TEXT NOT NULL UNIQUE,
		p256dh TEXT NOT NULL,
		auth TEXT NOT NULL,
		user_agent TEXT,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`

	queries := []string{
		createTableQuery,
	}

	for _, q := range queries {
		if _, err := bunDB.NewRaw(q).Exec(ctx); err != nil {
			log.Printf("Failed to execute query %s: %v\n", q, err)
		}
	}

	log.Println("Migration 036 complete: Added push_subscriptions table")
}
