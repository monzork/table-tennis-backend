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
		`ALTER TABLE tournament_participants ADD COLUMN elo_before_singles INTEGER;`,
		`ALTER TABLE tournament_participants ADD COLUMN elo_before_doubles INTEGER;`,
		`ALTER TABLE tournament_participants ADD COLUMN elo_after_singles INTEGER;`,
		`ALTER TABLE tournament_participants ADD COLUMN elo_after_doubles INTEGER;`,
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
