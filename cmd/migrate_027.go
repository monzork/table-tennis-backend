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

	// ── Index: event_participants.player_id ────────────────────────────
	// Used when computing per-player event history and participation counts.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_event_participants_player_id
		ON event_participants(player_id)
	`).Exec(ctx)

	// ── Index: event_participants.event_id ────────────────────────
	// All participant lookups are filtered by event_id.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_event_participants_event_id
		ON event_participants(event_id)
	`).Exec(ctx)

	// ── Index: group_participants.group_id ─────────────────────────────────
	// Batch-loaded for every event fetch that has groups.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_group_participants_group_id
		ON group_participants(group_id)
	`).Exec(ctx)

	// ── Index: groups.event_id ────────────────────────────────────────
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_groups_event_id
		ON groups(event_id)
	`).Exec(ctx)

	// ── Index: team_players.team_id ────────────────────────────────────────
	// Bulk-loaded when assembling team rosters.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_team_players_team_id
		ON team_players(team_id)
	`).Exec(ctx)

	// ── Index: teams.event_id ─────────────────────────────────────────
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_teams_event_id
		ON teams(event_id)
	`).Exec(ctx)

	// ── Index: matches.event_id ───────────────────────────────────────
	// Almost every match query filters by event_id.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_matches_event_id
		ON matches(event_id)
	`).Exec(ctx)

	// ── Index: matches.team_match_id ───────────────────────────────────────
	// Used to find sub-matches of a team match.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_matches_team_match_id
		ON matches(team_match_id)
	`).Exec(ctx)

	// ── Index: matches.status ──────────────────────────────────────────────
	// Filters like "status != 'finished'" and "status = 'in_progress'" are common.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_matches_status
		ON matches(status)
	`).Exec(ctx)

	// ── Index: matches.pin ─────────────────────────────────────────────────
	// Used by GenerateUniquePin to check PIN uniqueness among active matches.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_matches_pin
		ON matches(pin)
	`).Exec(ctx)

	// ── Index: players full-name search (SQLite-friendly) ─────────────────
	// Player search does LIKE '%query%' on first_name and last_name.
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_players_first_name
		ON players(first_name)
	`).Exec(ctx)
	_, _ = bunDB.NewRaw(`
		CREATE INDEX IF NOT EXISTS idx_players_last_name
		ON players(last_name)
	`).Exec(ctx)

	log.Println("Migration 027 complete: database indexes added.")
}
