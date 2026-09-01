package bun

import (
	"context"
	"database/sql"
	"errors"
	"table-tennis-backend/internal/domain/account"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AccountRepository struct {
	db *bun.DB
}

func NewAccountRepository(db *bun.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

func modelToAccount(m *AccountModel) *account.Account {
	return &account.Account{
		ID:         m.ID.String(),
		GoogleSub:  m.GoogleSub,
		Email:      m.Email,
		Name:       m.Name,
		PictureURL: m.PictureURL,
		CreatedAt:  m.CreatedAt,
	}
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*account.Account, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	var model AccountModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&model).Where("id = ?", uid).Scan(ctx); err != nil {
		return nil, err
	}
	return modelToAccount(&model), nil
}

func (r *AccountRepository) GetByIDs(ctx context.Context, ids []string) ([]*account.Account, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	uids := make([]uuid.UUID, 0, len(ids))
	for _, idStr := range ids {
		if uid, err := uuid.Parse(idStr); err == nil {
			uids = append(uids, uid)
		}
	}
	if len(uids) == 0 {
		return nil, nil
	}
	var models []AccountModel
	if err := ExtractDB(ctx, r.db).NewSelect().Model(&models).Where("id IN (?)", bun.List(uids)).Scan(ctx); err != nil {
		return nil, err
	}
	accounts := make([]*account.Account, len(models))
	for i := range models {
		accounts[i] = modelToAccount(&models[i])
	}
	return accounts, nil
}

func (r *AccountRepository) GetByGoogleSub(ctx context.Context, sub string) (*account.Account, error) {
	var model AccountModel
	err := ExtractDB(ctx, r.db).NewSelect().Model(&model).Where("google_sub = ?", sub).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return modelToAccount(&model), nil
}

func (r *AccountRepository) GetByEmail(ctx context.Context, email string) (*account.Account, error) {
	var model AccountModel
	err := ExtractDB(ctx, r.db).NewSelect().Model(&model).Where("email = ?", email).Scan(ctx)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return modelToAccount(&model), nil
}

func (r *AccountRepository) Save(ctx context.Context, a *account.Account) error {
	id, err := uuid.Parse(a.ID)
	if err != nil {
		return err
	}

	model := &AccountModel{
		ID:         id,
		GoogleSub:  a.GoogleSub,
		Email:      a.Email,
		Name:       a.Name,
		PictureURL: a.PictureURL,
	}

	_, err = ExtractDB(ctx, r.db).NewInsert().
		Model(model).
		On("CONFLICT (id) DO UPDATE").
		Set("google_sub = EXCLUDED.google_sub").
		Set("email = EXCLUDED.email").
		Set("name = EXCLUDED.name").
		Set("picture_url = EXCLUDED.picture_url").
		Exec(ctx)
	return err
}
