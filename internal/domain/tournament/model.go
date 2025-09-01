package tournament

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Tournament struct {
	bun.BaseModel `bun:"table:tournaments"`
	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	Name          string     `bun:"name,notnull"`
	Description   string     `bun:"description"`
	StartDate     time.Time  `bun:"start_date,notnull"`
	EndDate       time.Time  `bun:"end_date,notnull"`
	Rules         []Rule     `bun:"m2m:tournament_rules,join:ID=TournamentID"`
	Matches       []Match    `bun:"rel:has-many,join:ID=TournamentID"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time `bun:"updated_at,notnull,default:current_timestamp"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}

type Rule struct {
	bun.BaseModel `bun:"table:rules"`
	ID            int          `bun:"id,pk,autoincrement,type:int"`
	Name          string       `bun:"name,notnull"`
	Description   string       `bun:"description"`
	Tournaments   []Tournament `bun:"m2m:tournament_rules,join:ID=RuleID"`
	CreatedAt     time.Time    `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time   `bun:"updated_at"`
	DeletedAt     *time.Time   `bun:"deleted_at,soft_delete,nullzero"`
}

type TournamentRule struct {
	bun.BaseModel `bun:"table:tournament_rules"`
	ID            int `bun:"id,pk,autoincrement,type:int"`
	TournamentID  uuid.UUID
	RuleID        int
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type Match struct {
	bun.BaseModel `bun:"table:matches"`
	ID            uuid.UUID   `bun:"id,pk,type:uuid"`
	TournamentID  uuid.UUID   `bun:"tournament_id,notnull"`
	PlayerA       uuid.UUID   `bun:"player_a,notnull"`
	PlayerB       uuid.UUID   `bun:"player_b,notnull"`
	Sets          []Set       `bun:"rel:has-many,join:ID=MatchID"`
	CreatedAt     time.Time   `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time  `bun:"updated_at"`
	DeletedAt     *time.Time  `bun:"deleted_at,soft_delete,nullzero"`
	Tournament    *Tournament `bun:"rel:belongs-to,join:TournamentID=ID"`
}

type Set struct {
	bun.BaseModel `bun:"table:sets"`
	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	MatchID       uuid.UUID  `bun:"match_id,notnull"`
	Number        int        `bun:"number,notnull"`
	ScoreA        int        `bun:"score_a,notnull"`
	ScoreB        int        `bun:"score_b,notnull"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time `bun:"updated_at"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero"`
	Match         *Match     `bun:"rel:belongs-to,join:MatchID=ID"`
}
