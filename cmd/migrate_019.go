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

	// 1. Add pin to players
	_, err = sqldb.Exec(`ALTER TABLE players ADD COLUMN pin TEXT NOT NULL DEFAULT '1234';`)
	if err != nil {
		log.Printf("Warning adding pin to players: %v (might already exist)", err)
	} else {
		log.Println("Added pin column to players.")
	}

	// 2. Add num_tables to tournaments
	_, err = sqldb.Exec(`ALTER TABLE tournaments ADD COLUMN num_tables INT NOT NULL DEFAULT 4;`)
	if err != nil {
		log.Printf("Warning adding num_tables to tournaments: %v (might already exist)", err)
	} else {
		log.Println("Added num_tables column to tournaments.")
	}

	// 3. Add num_tables to events
	_, err = sqldb.Exec(`ALTER TABLE events ADD COLUMN num_tables INT NOT NULL DEFAULT 0;`)
	if err != nil {
		log.Printf("Warning adding num_tables to events: %v (might already exist)", err)
	} else {
		log.Println("Added num_tables column to events.")
	}

	// 4. Add referee_id to matches
	_, err = sqldb.Exec(`ALTER TABLE matches ADD COLUMN referee_id TEXT DEFAULT NULL;`)
	if err != nil {
		log.Printf("Warning adding referee_id to matches: %v (might already exist)", err)
	} else {
		log.Println("Added referee_id column to matches.")
	}

	// 5. Add table_number to matches
	_, err = sqldb.Exec(`ALTER TABLE matches ADD COLUMN table_number INT DEFAULT NULL;`)
	if err != nil {
		log.Printf("Warning adding table_number to matches: %v (might already exist)", err)
	} else {
		log.Println("Added table_number column to matches.")
	}
}
