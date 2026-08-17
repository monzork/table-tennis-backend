package bun

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AccountModel struct {
	bun.BaseModel `bun:"table:accounts"`

	ID         uuid.UUID `bun:"id,pk,type:uuid"`
	GoogleSub  string    `bun:"google_sub,notnull,unique"`
	Email      string    `bun:"email,notnull"`
	Name       string    `bun:"name"`
	PictureURL string    `bun:"picture_url"`
	CreatedAt  time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt  time.Time `bun:"updated_at,nullzero"`
}
