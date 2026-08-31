package bun

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type TournamentModel struct {
	bun.BaseModel `bun:"table:tournaments"`

	ID                      uuid.UUID        `bun:"id,pk,type:uuid"`
	Name                    string           `bun:"name,notnull"`
	DivisionIDs             []string         `bun:"division_ids,array"`
	SkipElo                 bool             `bun:"skip_elo,notnull,default:false"`
	StartDate               time.Time        `bun:"start_date,notnull"`
	EndDate                 time.Time        `bun:"end_date,notnull"`
	NumTables               int              `bun:"num_tables,notnull,default:4"`
	TablePriorities         map[string][]int `bun:"table_priorities,type:jsonb"`
	FederationEndorsed      bool             `bun:"federation_endorsed,notnull,default:false"`
	IncludeIDPhotosInReport bool             `bun:"include_id_photos_in_report,notnull,default:false"`
	CreatedAt               time.Time        `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt               *time.Time       `bun:"updated_at,nullzero"`
}
