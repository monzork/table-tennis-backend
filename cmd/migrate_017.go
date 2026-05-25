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

	// Fix team_match_id FK: change from referencing team_matches(id) to matches(id)
	// This aligns with the code which stores parent team matches in the matches table.
	_, err = sqldb.Exec(`ALTER TABLE matches DROP CONSTRAINT IF EXISTS matches_team_match_id_fkey;`)
	if err != nil {
		log.Printf("Warning dropping old constraint: %v", err)
	}
	_, err = sqldb.Exec(`ALTER TABLE matches ADD CONSTRAINT matches_team_match_id_fkey FOREIGN KEY (team_match_id) REFERENCES matches(id) ON DELETE SET NULL;`)
	if err != nil {
		log.Fatalf("Error adding new constraint: %v", err)
	}
	log.Println("Successfully updated team_match_id FK to reference matches(id).")
}
