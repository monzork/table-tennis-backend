package players

import "context"

type Repository interface {
	Create(ctx context.Context, p *Players) error
	GetAll(ctx context.Context) (*[]Players, error)
}
