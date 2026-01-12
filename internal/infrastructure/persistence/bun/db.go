package bun

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
)

var DB *bun.DB

func Connect() {
	sqldb, err := sql.Open("sqlite3", "table_tennis.db")

	if err != nil {
		log.Fatal(err)
	}

	DB = bun.NewDB(sqldb, sqlitedialect.New())
}
