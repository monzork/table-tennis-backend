package bun

import (
	"context"
	"database/sql"
	"log"
	"net/url"
	"os"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/driver/pgdriver"
	_ "modernc.org/sqlite"
)

var DB *bun.DB

func Connect() {
	var sqldb *sql.DB
	var bunDB *bun.DB

	dsn := os.Getenv("DATABASE_URL")
	if dsn != "" {
		// Clean DSN by removing channel_binding parameter
		if u, err := url.Parse(dsn); err == nil {
			q := u.Query()
			if q.Has("channel_binding") {
				q.Del("channel_binding")
				u.RawQuery = q.Encode()
				dsn = u.String()
			}
		}

		// Use PostgreSQL if DATABASE_URL is present
		sqldb = sql.OpenDB(pgdriver.NewConnector(pgdriver.WithDSN(dsn)))
		bunDB = bun.NewDB(sqldb, pgdialect.New())
	} else {
		// Fallback to SQLite (local development)
		dbPath := os.Getenv("DB_PATH")
		if dbPath == "" {
			dbPath = "table_tennis.db"
		}
		var err error
		sqldb, err = sql.Open("sqlite", dbPath+"?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
		if err != nil {
			log.Fatal(err)
		}
		sqldb.SetMaxOpenConns(1)
		bunDB = bun.NewDB(sqldb, sqlitedialect.New())
	}

	DB = bunDB

	// Bun requires join-table models to be registered for any lookup
	DB.RegisterModel(
		(*TournamentParticipantModel)(nil),
		(*GroupParticipantModel)(nil),
		(*TeamModel)(nil),
		(*TeamPlayerModel)(nil),
	)

	// Self-healing seed for No Division fallback to prevent FK violations on Skip-Elo Events
	_, _ = DB.NewRaw("INSERT INTO divisions (id, name, display_order, min_elo, max_elo, category, color) VALUES ('none', 'No Division', 99, 0, 9999, 'both', '#7B8794') ON CONFLICT (id) DO NOTHING").Exec(context.Background())
}
