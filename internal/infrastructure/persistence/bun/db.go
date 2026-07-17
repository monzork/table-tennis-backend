package bun

import (
	"database/sql"
	"log/slog"
	"net/url"
	"os"
	"time"

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

		// Configure Connection Pool
		sqldb.SetMaxOpenConns(25)
		sqldb.SetMaxIdleConns(25) // Match MaxOpenConns to avoid connection churn
		sqldb.SetConnMaxLifetime(5 * time.Minute)

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
			slog.Error("Failed to open db", "err", err)
			os.Exit(1)
		}
		sqldb.SetMaxOpenConns(1) // SQLite works best with 1 connection when using modernc.org/sqlite
		bunDB = bun.NewDB(sqldb, sqlitedialect.New())
	}

	DB = bunDB

	// Bun requires join-table models to be registered for any lookup
	DB.RegisterModel(
		(*EventParticipantModel)(nil),
		(*GroupParticipantModel)(nil),
		(*TeamModel)(nil),
		(*TeamPlayerModel)(nil),
	)

}
