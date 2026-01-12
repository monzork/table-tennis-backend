package bun

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TournamentModel struct {
	bun.BaseModel `bun:"table:tournaments"`

	ID        uuid.UUID  `bun:"id,pk,type:uuid"`
	Name      string     `bun:"name,notnull"`
	StartDate time.Time  `bun:"start_date,notnull"`
	EndDate   time.Time  `bun:"end_date,notnull"`
	CreatedAt time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt *time.Time `bun:"updated_at,nullzero"`
}

// -------------------------
// Rule
// -------------------------
type RuleModel struct {
	bun.BaseModel `bun:"table:rules"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid"`
	Name        string     `bun:"name,notnull"`
	Description string     `bun:"description"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   *time.Time `bun:"updated_at,nullzero"`
}
