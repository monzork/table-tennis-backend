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

	// Self-healing seed for No Division fallback to prevent FK violations on Skip-Elo Events
	_, _ = bunDB.NewRaw("INSERT INTO divisions (id, name, display_order, min_elo, max_elo, category, color) VALUES ('none', 'No Division', 99, 0, 9999, 'both', '#7B8794') ON CONFLICT (id) DO NOTHING").Exec(context.Background())

	// Ensure tournaments table has winner_name column
	_, _ = bunDB.NewRaw("ALTER TABLE tournaments ADD COLUMN winner_name TEXT DEFAULT ''").Exec(context.Background())
	// Ensure players table has national_id column
	_, _ = bunDB.NewRaw("ALTER TABLE players ADD COLUMN national_id TEXT DEFAULT ''").Exec(context.Background())
	// Ensure players table has second_name column
	_, _ = bunDB.NewRaw("ALTER TABLE players ADD COLUMN second_name TEXT").Exec(context.Background())
	// Ensure players table has second_last_name column
	_, _ = bunDB.NewRaw("ALTER TABLE players ADD COLUMN second_last_name TEXT").Exec(context.Background())
	// Ensure tournament_participants has pin column (migration 024)
	_, _ = bunDB.NewRaw("ALTER TABLE tournament_participants ADD COLUMN pin TEXT NOT NULL DEFAULT '0000'").Exec(context.Background())
	// Ensure events has num_tables
	_, _ = bunDB.NewRaw("ALTER TABLE events ADD COLUMN num_tables INT NOT NULL DEFAULT 4").Exec(context.Background())
	// Ensure tournaments has num_tables
	_, _ = bunDB.NewRaw("ALTER TABLE tournaments ADD COLUMN num_tables INT NOT NULL DEFAULT 0").Exec(context.Background())

	// Migrate existing optional fields with empty strings to NULL
	if !isPostgres {
		type TableInfo struct {
			Cid       int     `bun:"cid"`
			Name      string  `bun:"name"`
			Type      string  `bun:"type"`
			Notnull   int     `bun:"notnull"`
			DfltValue *string `bun:"dflt_value"`
			Pk        int     `bun:"pk"`
		}
		var info []TableInfo
		err := bunDB.NewRaw("PRAGMA table_info(players)").Scan(context.Background(), &info)
		if err == nil {
			needsRecreation := false
			for _, col := range info {
				if (col.Name == "second_name" || col.Name == "second_last_name" || col.Name == "department") && col.Notnull == 1 {
					needsRecreation = true
					break
				}
			}
			if needsRecreation {
				log.Println("Migrating SQLite players table to drop NOT NULL constraints for optional fields...")
				_, _ = bunDB.NewRaw("PRAGMA foreign_keys=OFF").Exec(context.Background())
				
				// Create new table (without pin column)
				_, _ = bunDB.NewRaw(`
					CREATE TABLE players_new (
						id TEXT PRIMARY KEY,
						first_name TEXT NOT NULL,
						second_name TEXT,
						last_name TEXT NOT NULL,
						second_last_name TEXT,
						birthdate TEXT NOT NULL,
						gender TEXT NOT NULL DEFAULT 'M',
						singles_elo INTEGER NOT NULL DEFAULT 1000,
						doubles_elo INTEGER NOT NULL DEFAULT 1000,
						country TEXT NOT NULL,
						department TEXT,
						whatsapp_number TEXT,
						national_id TEXT,
						created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at TEXT
					)
				`).Exec(context.Background())
				
				// Copy data
				_, _ = bunDB.NewRaw(`
					INSERT INTO players_new (id, first_name, second_name, last_name, second_last_name, birthdate, gender, singles_elo, doubles_elo, country, department, whatsapp_number, national_id, created_at, updated_at)
					SELECT id, first_name, NULLIF(second_name, ''), last_name, NULLIF(second_last_name, ''), birthdate, gender, singles_elo, doubles_elo, country, NULLIF(department, ''), whatsapp_number, national_id, created_at, updated_at FROM players
				`).Exec(context.Background())
				
				// Drop and rename
				_, _ = bunDB.NewRaw("DROP TABLE players").Exec(context.Background())
				_, _ = bunDB.NewRaw("ALTER TABLE players_new RENAME TO players").Exec(context.Background())
				_, _ = bunDB.NewRaw("PRAGMA foreign_keys=ON").Exec(context.Background())
			}
		}
	} else {
		// PostgreSQL drop NOT NULL constraints
		_, _ = bunDB.NewRaw("ALTER TABLE players ALTER COLUMN second_name DROP NOT NULL").Exec(context.Background())
		_, _ = bunDB.NewRaw("ALTER TABLE players ALTER COLUMN second_last_name DROP NOT NULL").Exec(context.Background())
		_, _ = bunDB.NewRaw("ALTER TABLE players ALTER COLUMN department DROP NOT NULL").Exec(context.Background())
	}

	// Sync empty strings to NULL
	_, _ = bunDB.NewRaw("UPDATE players SET second_name = NULL WHERE second_name = ''").Exec(context.Background())
	_, _ = bunDB.NewRaw("UPDATE players SET second_last_name = NULL WHERE second_last_name = ''").Exec(context.Background())
	_, _ = bunDB.NewRaw("UPDATE players SET department = NULL WHERE department = ''").Exec(context.Background())
	_, _ = bunDB.NewRaw("UPDATE players SET whatsapp_number = NULL WHERE whatsapp_number = ''").Exec(context.Background())
	_, _ = bunDB.NewRaw("UPDATE players SET national_id = NULL WHERE national_id = ''").Exec(context.Background())

	// Add has_third_place_match if missing
	_, _ = bunDB.NewRaw("ALTER TABLE tournaments ADD COLUMN has_third_place_match BOOLEAN NOT NULL DEFAULT false").Exec(context.Background())

	log.Println("Migration 026 complete: ad-hoc alters applied.")
}
