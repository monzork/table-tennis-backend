//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
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

	// 1. Drop pin column from players
	if isPostgres {
		_, err = sqldb.Exec(`ALTER TABLE players DROP COLUMN IF EXISTS pin;`)
	} else {
		// SQLite doesn't support DROP COLUMN directly in older versions; skip gracefully
		_, err = sqldb.Exec(`SELECT pin FROM players LIMIT 1`)
		if err == nil {
			log.Println("Note: SQLite detected — pin column exists but cannot be dropped via ALTER. It will be ignored.")
			err = nil // Not a fatal error
		} else {
			log.Println("pin column already absent from players (SQLite).")
			err = nil
		}
	}
	if err != nil {
		log.Printf("Warning dropping pin from players: %v", err)
	} else {
		log.Println("Handled pin column removal from players.")
	}

	// 2. Add pin column to tournament_participants
	_, err = sqldb.Exec(`ALTER TABLE tournament_participants ADD COLUMN pin TEXT NOT NULL DEFAULT '0000';`)
	if err != nil {
		log.Printf("Warning adding pin to tournament_participants: %v (might already exist)", err)
	} else {
		log.Println("Added pin column to tournament_participants.")
	}

	// 3. Generate random 4-digit PINs for existing tournament_participants rows
	if isPostgres {
		_, err = sqldb.Exec(`
			UPDATE tournament_participants
			SET pin = LPAD(FLOOR(RANDOM() * 10000)::TEXT, 4, '0')
			WHERE pin = '0000';
		`)
		if err != nil {
			log.Printf("Warning generating PINs for existing participants (postgres): %v", err)
		} else {
			log.Println("Generated random PINs for existing tournament participants.")
		}
	} else {
		// SQLite: fetch all rows and update individually
		rows, err := sqldb.Query(`SELECT tournament_id, player_id FROM tournament_participants WHERE pin = '0000'`)
		if err != nil {
			log.Printf("Warning fetching participants for PIN generation: %v", err)
		} else {
			defer rows.Close()
			count := 0
			for rows.Next() {
				var tournamentID, playerID string
				if err := rows.Scan(&tournamentID, &playerID); err != nil {
					continue
				}
				pin := fmt.Sprintf("%04d", rand.Intn(10000))
				sqldb.Exec(`UPDATE tournament_participants SET pin = ? WHERE tournament_id = ? AND player_id = ?`, pin, tournamentID, playerID)
				count++
			}
			log.Printf("Generated random PINs for %d existing tournament participants (SQLite).", count)
		}
	}

	log.Println("Migration 024 complete.")
}
