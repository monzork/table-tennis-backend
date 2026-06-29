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

	// ── Index: matches.tournament_id_table_number_status (Partial) ─────────
	// Used for scoped tournament table lookups and active table status queries.
	// Only indexes active ('in_progress') matches to keep the index tiny and ultra-fast.
	_, err = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_matches_tourney_table_status
		ON matches(tournament_id, table_number)
		WHERE status = 'in_progress'
	`).Exec(ctx)
	if err != nil {
		log.Fatal("Failed to create index:", err)
	}

	log.Println("Migration 028 complete: composite index idx_matches_tourney_table_status added.")
}
