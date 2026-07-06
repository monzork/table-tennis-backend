package bun

import (
	"context"
	"database/sql"

	"github.com/uptrace/bun"
)

type txKey struct{}

// InjectTx injects a bun.Tx into the context.
func InjectTx(ctx context.Context, tx bun.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// ExtractDB extracts the bun.IDB from the context if a transaction is present,
// otherwise it returns the default fallback db.
func ExtractDB(ctx context.Context, fallback *bun.DB) bun.IDB {
	if tx, ok := ctx.Value(txKey{}).(bun.Tx); ok {
		return tx
	}
	return fallback
}

// RunInTx runs the given function in a transaction. If a transaction is already present
// in the context, it reuses it (and does NOT commit/rollback).
// If no transaction is present, it starts a new one, and commits/rolls it back.
func RunInTx(ctx context.Context, db *bun.DB, fn func(ctx context.Context, tx bun.Tx) error) error {
	if existingTx, ok := ctx.Value(txKey{}).(bun.Tx); ok {
		return fn(ctx, existingTx)
	}
	return db.RunInTx(ctx, &sql.TxOptions{}, func(ctx context.Context, tx bun.Tx) error {
		return fn(InjectTx(ctx, tx), tx)
	})
}
