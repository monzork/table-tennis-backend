package user

import (
	"context"

	"github.com/monzork/table-tennis-backend/internal/domain/user"
	"github.com/uptrace/bun"
)

type SQLiteUserRepository struct {
	db *bun.DB
}

func NewSQLiteUserRepository(db *bun.DB) *SQLiteUserRepository {
	return &SQLiteUserRepository{db: db}
}

func (r *SQLiteUserRepository) Create(ctx context.Context, u *user.User) error {
	_, err := r.db.NewInsert().Model(u).Exec(ctx)
	return err
}
