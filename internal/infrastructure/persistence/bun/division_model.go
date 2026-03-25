package bun

import (
	"time"

	"github.com/uptrace/bun"
)

type DivisionModel struct {
	bun.BaseModel `bun:"table:divisions"`

	ID           string     `bun:"id,pk,type:text"`
	Name         string     `bun:"name,notnull"`
	DisplayOrder int        `bun:"display_order,notnull,default:0"`
	MinElo       int16      `bun:"min_elo,notnull,default:0"`
	MaxElo       *int16     `bun:"max_elo"` // can be null
	Category     string     `bun:"category,notnull,default:'both'"`
	Color        string     `bun:"color,notnull,default:'#ffffff'"`
	CreatedAt    time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt    *time.Time `bun:"updated_at,nullzero"`
}
