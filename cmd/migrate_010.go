//go:build ignore

package main

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "table_tennis.db?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec(`ALTER TABLE tournaments ADD COLUMN event_category TEXT NOT NULL DEFAULT 'open';`)
	if err != nil {
		// Tolerate duplicate column if it already ran
		log.Printf("Alter table exec result: %v", err)
	} else {
		log.Println("Successfully added event_category to tournaments table.")
	}
}
