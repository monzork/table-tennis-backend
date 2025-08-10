package db

import (
	"database/sql"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	"log"
	_ "modernc.org/sqlite"
)

var DB *bun.DB

func Connect() {
	dsn := "file:data.db?cache=shared&mode=rwc"

	sqlDB, err := sql.Open("sqlite", dsn)

	if err != nil {
		log.Fatal("Failed to connect to SQLite:", err)
	}

	DB = bun.NewDB(sqlDB, sqlitedialect.New())

	createTables()
}

func createTables() {
	_, err := DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		)`)

	if err != nil {
		log.Fatal("Failed to create users table:", err)
	}

	log.Println("SQLite initialized with users table.")
}
