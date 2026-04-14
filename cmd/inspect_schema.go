//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "table_tennis.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	tables := []string{"tournaments", "tournament_participants", "matches", "players"}
	for _, t := range tables {
		rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, t))
		if err != nil {
			continue
		}
		fmt.Printf("\nColumns in %s:\n", t)
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dflt sql.NullString
			rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk)
			fmt.Printf("  %s %s\n", name, typ)
		}
		rows.Close()
	}
}
