package db

import (
	"database/sql"
	"log"
	"sync"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"github.com/uptrace/bun/migrate"
	_ "modernc.org/sqlite"
)

var (
	db   *bun.DB
	once sync.Once
)

var Migrations = migrate.NewMigrations()

func Connect() *bun.DB {
	once.Do(func() {
		dsn := "file:data.db?cache=shared&mode=rwc"

		sqlDB, err := sql.Open("sqlite", dsn)

		if err != nil {
			log.Fatal("Failed to connect to SQLite:", err)
		}

		db = bun.NewDB(sqlDB, sqlitedialect.New())
	})
	return db
}
