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

	dsn := os.Getenv("DATABASE_URL")
	isPostgres := dsn != ""
	if isPostgres {
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

	if isPostgres {
		// PostgreSQL DROP NOT NULL
		_, err = sqldb.Exec("ALTER TABLE players ALTER COLUMN second_name DROP NOT NULL;")
		if err != nil {
			log.Printf("Postgres ALTER COLUMN second_name error: %v", err)
		}
		_, err = sqldb.Exec("ALTER TABLE players ALTER COLUMN second_last_name DROP NOT NULL;")
		if err != nil {
			log.Printf("Postgres ALTER COLUMN second_last_name error: %v", err)
		}
		_, err = sqldb.Exec("ALTER TABLE players ALTER COLUMN department DROP NOT NULL;")
		if err != nil {
			log.Printf("Postgres ALTER COLUMN department error: %v", err)
		}
	} else {
		// SQLite table info check and recreation if needed
		rows, err := sqldb.Query("PRAGMA table_info(players);")
		if err == nil {
			defer rows.Close()
			needsRecreation := false
			for rows.Next() {
				var cid int
				var name, colType string
				var notnull int
				var dfltValue sql.NullString
				var pk int
				if err := rows.Scan(&cid, &name, &colType, &notnull, &dfltValue, &pk); err == nil {
					if (name == "second_name" || name == "second_last_name" || name == "department") && notnull == 1 {
						needsRecreation = true
					}
				}
			}
			if needsRecreation {
				log.Println("Migrating SQLite players table to drop NOT NULL constraints for optional fields...")
				_, _ = sqldb.Exec("PRAGMA foreign_keys=OFF;")
				
				// Create new table
				_, _ = sqldb.Exec(`
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
						pin TEXT NOT NULL DEFAULT '1234',
						national_id TEXT,
						created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
						updated_at TEXT
					);
				`)
				
				// Copy data
				_, _ = sqldb.Exec(`
					INSERT INTO players_new (id, first_name, second_name, last_name, second_last_name, birthdate, gender, singles_elo, doubles_elo, country, department, whatsapp_number, pin, national_id, created_at, updated_at)
					SELECT id, first_name, NULLIF(second_name, ''), last_name, NULLIF(second_last_name, ''), birthdate, gender, singles_elo, doubles_elo, country, NULLIF(department, ''), whatsapp_number, pin, national_id, created_at, updated_at FROM players;
				`)
				
				// Drop and rename
				_, _ = sqldb.Exec("DROP TABLE players;")
				_, _ = sqldb.Exec("ALTER TABLE players_new RENAME TO players;")
				_, _ = sqldb.Exec("PRAGMA foreign_keys=ON;")
				log.Println("SQLite migration successful.")
			} else {
				log.Println("SQLite migration not needed (NOT NULL already dropped).")
			}
		} else {
			log.Printf("SQLite PRAGMA table_info error: %v", err)
		}
	}

	// Update empty strings to NULL
	_, _ = sqldb.Exec("UPDATE players SET second_name = NULL WHERE second_name = '';")
	_, _ = sqldb.Exec("UPDATE players SET second_last_name = NULL WHERE second_last_name = '';")
	_, _ = sqldb.Exec("UPDATE players SET department = NULL WHERE department = '';")
	_, _ = sqldb.Exec("UPDATE players SET whatsapp_number = NULL WHERE whatsapp_number = '';")
	_, _ = sqldb.Exec("UPDATE players SET national_id = NULL WHERE national_id = '';")
	log.Println("Data update complete (empty strings converted to NULL).")
}
