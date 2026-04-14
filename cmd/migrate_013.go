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

	_, err = db.Exec(`ALTER TABLE tournaments ADD COLUMN status TEXT NOT NULL DEFAULT 'in_progress';`)
	if err != nil {
		log.Printf("Query error: %v", err)
	} else {
		log.Printf("Successfully added status to tournaments.")
	}
}
