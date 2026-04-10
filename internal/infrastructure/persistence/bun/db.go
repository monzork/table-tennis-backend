package bun

import (
	"database/sql"
	"log"
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
	)
}
