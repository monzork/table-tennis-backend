package uow

import "context"

// TransactionManager defines an interface for managing transactions in the application layer.
type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
