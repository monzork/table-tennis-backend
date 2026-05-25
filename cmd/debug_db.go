//go:build ignore

package main

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite"
)

func main() {
	db, err := sql.Open("sqlite", "table_tennis.db")
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer db.Close()

	// Find doubles tournaments
	rows, _ := db.Query("SELECT id, name, type, format FROM tournaments WHERE type IN ('doubles','mixed_doubles','teams')")
	defer rows.Close()
	for rows.Next() {
		var id, name, typ, format string
		rows.Scan(&id, &name, &typ, &format)
		fmt.Printf("\n=== Tournament: %s (type=%s, format=%s, id=%s) ===\n", name, typ, format, id)

		// Count teams
		var teamCount int
		db.QueryRow("SELECT count(*) FROM teams WHERE tournament_id=?", id).Scan(&teamCount)
		fmt.Printf("  Teams: %d\n", teamCount)

		// List teams
		trows, _ := db.Query("SELECT id, name FROM teams WHERE tournament_id=?", id)
		for trows.Next() {
			var tid, tname string
			trows.Scan(&tid, &tname)
			fmt.Printf("    Team: %s (id=%s)\n", tname, tid)
		}
		trows.Close()

		// Count groups
		var groupCount int
		db.QueryRow("SELECT count(*) FROM groups WHERE tournament_id=?", id).Scan(&groupCount)
		fmt.Printf("  Groups: %d\n", groupCount)

		// List groups and their participants
		grows, _ := db.Query("SELECT id, name FROM groups WHERE tournament_id=?", id)
		for grows.Next() {
			var gid, gname string
			grows.Scan(&gid, &gname)
			
			var gpCount int
			db.QueryRow("SELECT count(*) FROM group_participants WHERE group_id=?", gid).Scan(&gpCount)
			fmt.Printf("    Group: %s (id=%s) - %d participants\n", gname, gid, gpCount)

			// List participants
			gprows, _ := db.Query("SELECT player_id, position FROM group_participants WHERE group_id=? ORDER BY position", gid)
			for gprows.Next() {
				var pid string
				var pos int
				gprows.Scan(&pid, &pos)
				
				// Check if it exists in players table
				var pname string
				err := db.QueryRow("SELECT first_name || ' ' || last_name FROM players WHERE id=?", pid).Scan(&pname)
				if err != nil {
					// Check teams table
					var tname string
					err2 := db.QueryRow("SELECT name FROM teams WHERE id=?", pid).Scan(&tname)
					if err2 != nil {
						fmt.Printf("      [%d] player_id=%s NOT FOUND in players OR teams\n", pos, pid)
					} else {
						fmt.Printf("      [%d] player_id=%s -> team: %s\n", pos, pid, tname)
					}
				} else {
					fmt.Printf("      [%d] player_id=%s -> player: %s\n", pos, pid, pname)
				}
			}
			gprows.Close()
		}
		grows.Close()
	}
}
