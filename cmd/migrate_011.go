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

	queries := []string{
		`ALTER TABLE tournaments ADD COLUMN group_pass_count INTEGER NOT NULL DEFAULT 2;`,
		`ALTER TABLE tournaments ADD COLUMN registration_open BOOLEAN NOT NULL DEFAULT 0;`,
	}
	for _, q := range queries {
		_, err = db.Exec(q)
		if err != nil {
			log.Printf("Query (%s) error: %v", q, err)
		} else {
			log.Printf("Query executed: %s", q)
		}
	}
}
