package bun

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

var DB *bun.DB

func Connect() {
	sqldb, err := sql.Open("sqlite", "table_tennis.db?_pragma=busy_timeout(10000)&_pragma=journal_mode(WAL)")
	if err == nil {
		sqldb.SetMaxOpenConns(1)
	}

	if err != nil {
		log.Fatal(err)
	}

	DB = bun.NewDB(sqldb, sqlitedialect.New())

	// Bun requires join-table models to be registered for any lookup
	DB.RegisterModel(
		(*TournamentParticipantModel)(nil),
		(*GroupParticipantModel)(nil),
	)
}
