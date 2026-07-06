package bun

import (
	"context"
	"database/sql"
	"table-tennis-backend/internal/domain/uow"

	"github.com/uptrace/bun"
)

type BunTransactionManager struct {
	db *bun.DB
}

func NewBunTransactionManager(db *bun.DB) uow.TransactionManager {
	return &BunTransactionManager{db: db}
}

func (m *BunTransactionManager) RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return m.db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		ctxWithTx := InjectTx(ctx, tx)
		return fn(ctxWithTx)
	})
}
