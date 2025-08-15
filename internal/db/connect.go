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
		dsn := "file:data.db?cache=shared&mode=rwc&_journal_mode=WAL"

		sqlDB, err := sql.Open("sqlite", dsn)

		if err != nil {
			log.Fatal("Failed to connect to SQLite:", err)
		}

		db = bun.NewDB(sqlDB, sqlitedialect.New())
		db.Exec("PRAGMA journal_mode=WAL;")   // allows concurrent reads/writes
		db.Exec("PRAGMA synchronous=NORMAL;") // faster than FULL, still safe
		db.Exec("PRAGMA cache_size=10000;")   // increase in-memory cache
		db.Exec("PRAGMA temp_store=MEMORY;")  // store temp tables in memory
	})
	return db
}
