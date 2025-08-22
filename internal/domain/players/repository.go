package players

import (
	"context"

	"github.com/google/uuid"
)

type Repository interface {
	Create(ctx context.Context, p *Players) error
	GetAll(ctx context.Context) (*[]Players, error)
	Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) (*Players, error)
}
