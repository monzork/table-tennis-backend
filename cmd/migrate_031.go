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

	// Create division_rules table (no foreign key to avoid type mismatch)
	if isPostgres {
		_, err = bunDB.NewRaw(`
			CREATE TABLE IF NOT EXISTS tournament_division_rules (
				id TEXT PRIMARY KEY,
				tournament_id TEXT NOT NULL,
				division_id TEXT NOT NULL,
				best_of INTEGER NOT NULL,
				points_to_win INTEGER NOT NULL,
				points_margin INTEGER NOT NULL,
				UNIQUE(tournament_id, division_id)
			)
		`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create division_rules table (PostgreSQL):", err)
		}
	} else {
		_, err = bunDB.NewRaw(`
			CREATE TABLE IF NOT EXISTS tournament_division_rules (
				id TEXT PRIMARY KEY,
				tournament_id TEXT NOT NULL,
				division_id TEXT NOT NULL,
				best_of INTEGER NOT NULL,
				points_to_win INTEGER NOT NULL,
				points_margin INTEGER NOT NULL,
				UNIQUE(tournament_id, division_id)
			)
		`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to create division_rules table (SQLite):", err)
		}
	}

	// Add division_id column to matches table
	if isPostgres {
		_, err = bunDB.NewRaw(`ALTER TABLE matches ADD COLUMN IF NOT EXISTS division_id VARCHAR(255)`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to add division_id column to matches (PostgreSQL):", err)
		}
	} else {
		var hasCol int
		_ = bunDB.NewRaw(`SELECT COUNT(*) FROM pragma_table_info('matches') WHERE name = 'division_id'`).Scan(ctx, &hasCol)
		if hasCol == 0 {
			_, err = bunDB.NewRaw(`ALTER TABLE matches ADD COLUMN division_id VARCHAR(255)`).Exec(ctx)
			if err != nil {
				log.Fatal("Failed to add division_id column to matches (SQLite):", err)
			}
		}
	}

	log.Println("Migration 031 complete: division_rules table created and matches.division_id added.")
}
