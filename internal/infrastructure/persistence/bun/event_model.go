package bun

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type EventModel struct {
	bun.BaseModel `bun:"table:events"`

	ID         uuid.UUID  `bun:"id,pk,type:uuid"`
	Name       string     `bun:"name,notnull"`
	DivisionID string     `bun:"division_id,notnull"`
	SkipElo    bool       `bun:"skip_elo,notnull,default:false"`
	StartDate  time.Time  `bun:"start_date,notnull"`
	EndDate    time.Time  `bun:"end_date,notnull"`
	CreatedAt  time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt  *time.Time `bun:"updated_at,nullzero"`
}
