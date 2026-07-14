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

	// Step 1: Add new array column for division_ids
	if isPostgres {
		_, err = bunDB.NewRaw(`ALTER TABLE tournaments ADD COLUMN IF NOT EXISTS division_ids text[] DEFAULT '{}'`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to add division_ids column:", err)
		}
	} else {
		// SQLite JSON support for array mapping in Bun
		// First check if column exists
		var hasCol int
		_ = bunDB.NewRaw(`SELECT COUNT(*) FROM pragma_table_info('tournaments') WHERE name = 'division_ids'`).Scan(ctx, &hasCol)
		if hasCol == 0 {
			_, err = bunDB.NewRaw(`ALTER TABLE tournaments ADD COLUMN division_ids TEXT`).Exec(ctx)
			if err != nil {
				log.Fatal("Failed to add division_ids column (sqlite):", err)
			}
		}
	}

	// Step 2: Drop the old division_id column
	if isPostgres {
		_, err = bunDB.NewRaw(`ALTER TABLE tournaments DROP COLUMN IF EXISTS division_id`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to drop old division_id column:", err)
		}
	} else {
		// SQLite DROP COLUMN requires newer versions, but it's supported in modernc
		_, err = bunDB.NewRaw(`ALTER TABLE tournaments DROP COLUMN division_id`).Exec(ctx)
		if err != nil {
			log.Println("Notice: Failed to drop old division_id column (sqlite), this is fine on older SQLite versions:", err)
		}
	}

	log.Println("Migration 030 complete: tournament division_ids added and division_id dropped.")
}
