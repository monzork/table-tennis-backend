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

	// 1. Create teams table
	_, err = sqldb.Exec(`
		CREATE TABLE IF NOT EXISTS teams (
			id ` + uuidType + ` PRIMARY KEY,
			tournament_id ` + uuidType + ` NOT NULL,
			name TEXT NOT NULL,
			FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatalf("Error creating teams table: %v", err)
	}
	log.Println("Successfully created teams table.")

	// 2. Create team_players table
	_, err = sqldb.Exec(`
		CREATE TABLE IF NOT EXISTS team_players (
			team_id ` + uuidType + ` NOT NULL,
			player_id ` + uuidType + ` NOT NULL,
			PRIMARY KEY (team_id, player_id),
			FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
			FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		log.Fatalf("Error creating team_players table: %v", err)
	}
	log.Println("Successfully created team_players table.")
}
