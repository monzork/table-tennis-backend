//go:build ignore

package main

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"

	"github.com/joho/godotenv"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

func main() {
	_ = godotenv.Load()

	var sqldb *sql.DB
	var bunDB *bun.DB
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
		bunDB = bun.NewDB(sqldb, pgdialect.New())
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
		bunDB = bun.NewDB(sqldb, sqlitedialect.New())
	}
	defer sqldb.Close()

	ctx := context.Background()
	
	queries := []string{
		`CREATE INDEX IF NOT EXISTS idx_events_tournament_id ON events(tournament_id);`,
		`CREATE INDEX IF NOT EXISTS idx_event_participants_tourn_id ON event_participants(event_id);`,
		`CREATE INDEX IF NOT EXISTS idx_event_participants_player_id ON event_participants(player_id);`,
		`CREATE INDEX IF NOT EXISTS idx_groups_tourn_id ON groups(event_id);`,
		`CREATE INDEX IF NOT EXISTS idx_group_participants_group_id ON group_participants(group_id);`,
		`CREATE INDEX IF NOT EXISTS idx_group_participants_player_id ON group_participants(player_id);`,
		`CREATE INDEX IF NOT EXISTS idx_matches_tourn_id ON matches(event_id);`,
		`CREATE INDEX IF NOT EXISTS idx_matches_team_a_1 ON matches(team_a_player_1_id);`,
		`CREATE INDEX IF NOT EXISTS idx_matches_team_b_1 ON matches(team_b_player_1_id);`,
		`CREATE INDEX IF NOT EXISTS idx_matches_division_id ON matches(division_id);`,
		`CREATE INDEX IF NOT EXISTS idx_matches_team_match_id ON matches(team_match_id);`,
		`CREATE INDEX IF NOT EXISTS idx_matches_referee_id ON matches(referee_id);`,
		`CREATE INDEX IF NOT EXISTS idx_match_sets_match_id ON match_sets(match_id);`,
	}

	for _, q := range queries {
		if _, err := bunDB.NewRaw(q).Exec(ctx); err != nil {
			log.Printf("Failed to execute query %s: %v\n", q, err)
		}
	}

	log.Println("Migration 035 complete: Added performance indexes on foreign keys")
}
