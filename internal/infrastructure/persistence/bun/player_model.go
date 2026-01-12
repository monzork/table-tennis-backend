package bun

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type PlayerModel struct {
	bun.BaseModel `bun:"table:players"`

	ID        uuid.UUID  `bun:"id,pk,type:uuid"`
	FirstName string     `bun:"first_name,notnull"`
	LastName  string     `bun:"last_name,notnull"`
	Birthdate time.Time  `bun:"birthdate,notnull"`
	Elo       int16      `bun:"elo,notnull,default:1000"`
	Country   string     `bun:"country,notnull"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt *time.Time `bun:"updated_at,nullzero"`
}
