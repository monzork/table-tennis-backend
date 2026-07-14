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
		// Clean DSN by removing channel_binding parameter
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

	// Define column types
	uuidType := "UUID"
	if !isPostgres {
		uuidType = "TEXT"
	}

	// 1. Add team_format to events table
	_, _ = sqldb.Exec(`ALTER TABLE events ADD COLUMN team_format TEXT;`)
	log.Println("Tried adding team_format to events.")

	// 2. Create team_matches table
	_, err = sqldb.Exec(`
		CREATE TABLE IF NOT EXISTS team_matches (
			id ` + uuidType + ` PRIMARY KEY,
			event_id ` + uuidType + ` NOT NULL,
			team_a_id ` + uuidType + ` NOT NULL,
			team_b_id ` + uuidType + ` NOT NULL,
			status TEXT NOT NULL DEFAULT 'scheduled',
			winner_team TEXT,
			FOREIGN KEY (event_id) REFERENCES events(id) ON DELETE CASCADE,
			FOREIGN KEY (team_a_id) REFERENCES teams(id) ON DELETE CASCADE,
			FOREIGN KEY (team_b_id) REFERENCES teams(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatalf("Error creating team_matches table: %v", err)
	}
	log.Println("Successfully created team_matches table.")

	// 3. Add team_match_id to matches table
	_, _ = sqldb.Exec(`ALTER TABLE matches ADD COLUMN team_match_id ` + uuidType + ` REFERENCES team_matches(id) ON DELETE SET NULL;`)
	log.Println("Tried adding team_match_id to matches.")
}
