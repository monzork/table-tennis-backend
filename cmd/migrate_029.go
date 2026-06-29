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

	// ── Indexes: players.singles_elo and players.doubles_elo ─────────
	// Used for leaderboard sorting and fast retrieval by rating.
	_, err = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_players_singles_elo
		ON players(singles_elo DESC)
	`).Exec(ctx)
	if err != nil {
		log.Fatal("Failed to create singles_elo index:", err)
	}

	_, err = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_players_doubles_elo
		ON players(doubles_elo DESC)
	`).Exec(ctx)
	if err != nil {
		log.Fatal("Failed to create doubles_elo index:", err)
	}

	log.Println("Migration 029 complete: indexes idx_players_singles_elo and idx_players_doubles_elo added.")
}
