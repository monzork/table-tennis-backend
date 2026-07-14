//go:build ignore

package main

import (
	"database/sql"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

func main() {
	_ = godotenv.Load()

	var sqldb *sql.DB
	var err error
	isPostgres := false

	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		log.Println("Using PostgreSQL migration...")
		isPostgres = true
		if u, err := url.Parse(dsn); err == nil {
			q := u.Query()
			if q.Has("channel_binding") {
				q.Del("channel_binding")
				u.RawQuery = q.Encode()
				dsn = u.String()
			}
		}
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
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
	}
	defer sqldb.Close()

	// Create event_officials table
	var createTableQuery string
	if isPostgres {
		createTableQuery = `
			CREATE TABLE IF NOT EXISTS event_officials (
				event_id UUID NOT NULL,
				player_id UUID NOT NULL,
				pin TEXT NOT NULL,
				PRIMARY KEY (event_id, player_id)
			);
		`
	} else {
		createTableQuery = `
			CREATE TABLE IF NOT EXISTS event_officials (
				event_id TEXT NOT NULL,
				player_id TEXT NOT NULL,
				pin TEXT NOT NULL,
				PRIMARY KEY (event_id, player_id)
			);
		`
	}

	_, err = sqldb.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating event_officials table: %v", err)
	}

	log.Println("Migration 025 complete: event_officials table created.")
}
