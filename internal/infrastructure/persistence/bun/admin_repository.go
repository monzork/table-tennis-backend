package bun

import (
	"context"
	"table-tennis-backend/internal/domain/admin"

	"github.com/uptrace/bun"
)

type AdminRepository struct {
	db *bun.DB
}

func NewAdminRepository(db *bun.DB) *AdminRepository {
	return &AdminRepository{db: db}
}

func (r *AdminRepository) GetByUsername(ctx context.Context, username string) (*admin.Admin, error) {
	var model AdminModel
	err := r.db.NewSelect().Model(&model).Where("username = ?", username).Scan(ctx)
	if err != nil {
		return nil, err
	}

	return &admin.Admin{
		ID:           model.ID,
		Username:     model.Username,
		PasswordHash: model.PasswordHash,
	}, nil
}

func (r *AdminRepository) Save(ctx context.Context, a *admin.Admin) error {
	model := &AdminModel{
		ID:           a.ID,
		Username:     a.Username,
		PasswordHash: a.PasswordHash,
	}

	_, err := r.db.NewInsert().
		Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("username = EXCLUDED.username").
		Set("password_hash = EXCLUDED.password_hash").
		Exec(ctx)
	return err
}

func (r *AdminRepository) Count(ctx context.Context) (int, error) {
	return r.db.NewSelect().Model((*AdminModel)(nil)).Count(ctx)
}
