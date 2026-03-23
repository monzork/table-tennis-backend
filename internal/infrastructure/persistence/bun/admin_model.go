package bun

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type AdminModel struct {
	bun.BaseModel `bun:"table:admins"`

	ID           uuid.UUID `bun:"id,pk,type:uuid"`
	Username     string    `bun:"username,notnull,unique"`
	PasswordHash string    `bun:"password_hash,notnull"`
	CreatedAt    time.Time `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    time.Time `bun:"updated_at,nullzero"`
}
