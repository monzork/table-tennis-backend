package storage

import "github.com/uptrace/bun"

type SQLiteTournamentRepository struct {
	db *bun.DB
}

func NewSQLiteTournamentRepository(db *bun.DB) *SQLiteTournamentRepository {
	return &SQLiteTournamentRepository{db: db}
}
