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
	isPostgres := false
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
		bunDB = bun.NewDB(sqldb, pgdialect.New())
	} else {
		log.Println("Using SQLite migration (skipping pg_trgm creation)...")
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

	if isPostgres {
		_, err = bunDB.NewRaw(`CREATE EXTENSION IF NOT EXISTS pg_trgm;`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create pg_trgm extension:", err)
		}
		_, err = bunDB.NewRaw(`CREATE INDEX IF NOT EXISTS idx_player_fname_trgm ON players USING gin (LOWER(first_name) gin_trgm_ops);`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create fname trigram index:", err)
		}
		_, err = bunDB.NewRaw(`CREATE INDEX IF NOT EXISTS idx_player_sname_trgm ON players USING gin (LOWER(second_name) gin_trgm_ops);`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create sname trigram index:", err)
		}
		_, err = bunDB.NewRaw(`CREATE INDEX IF NOT EXISTS idx_player_lname_trgm ON players USING gin (LOWER(last_name) gin_trgm_ops);`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create lname trigram index:", err)
		}
		_, err = bunDB.NewRaw(`CREATE INDEX IF NOT EXISTS idx_player_slname_trgm ON players USING gin (LOWER(second_last_name) gin_trgm_ops);`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create slname trigram index:", err)
		}
		log.Println("Migration 034 complete: Added pg_trgm search indexes to players table")
	} else {
		log.Println("Migration 034 skipped: not using PostgreSQL")
	}
}
