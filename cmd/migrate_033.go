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

	if isPostgres {
		_, err = bunDB.NewRaw(`
			ALTER TABLE tournaments 
			ADD COLUMN IF NOT EXISTS division_formats JSON DEFAULT '{}'
		`).Exec(ctx)
		if err != nil {
			log.Fatal("Failed to add division_formats column (PostgreSQL):", err)
		}
	} else {
		// SQLite doesn't support ADD COLUMN IF NOT EXISTS in older versions
		var hasCol int
		_ = bunDB.NewRaw(`SELECT COUNT(*) FROM pragma_table_info('tournaments') WHERE name = 'division_formats'`).Scan(ctx, &hasCol)

		if hasCol == 0 {
			_, err = bunDB.NewRaw(`ALTER TABLE tournaments ADD COLUMN division_formats JSON DEFAULT '{}'`).Exec(ctx)
			if err != nil {
				log.Fatal("Failed to add division_formats column (SQLite):", err)
			}
		}
	}

	log.Println("Migration 033 complete: Added division_formats column to tournaments")
}
