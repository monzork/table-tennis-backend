package tournament

import (
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type Tournament struct {
	bun.BaseModel `bun:"table:tournaments"`

	ID          uuid.UUID  `bun:"id,pk,type:uuid"`
	Name        string     `bun:"name,notnull"`
	Description string     `bun:"description"`
	StartDate   time.Time  `bun:"start_date,notnull"`
	EndDate     time.Time  `bun:"end_date,notnull"`
	Rules       []Rule     `bun:"m2m:tournament_rules,join:ID=TournamentID"`
	Matches     []Match    `bun:"rel:has-many,join:ID=TournamentID"`
	CreatedAt   time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt   *time.Time `bun:"updated_at,nullzero"`
	DeletedAt   *time.Time `bun:"deleted_at,soft_delete,nullzero"`
}

type MatchPlayer struct {
	bun.BaseModel `bun:"table:match_players"`
	MatchId       uuid.UUID `bun:"match_id,pk"`
	PlayerID      uuid.UUID `bun:"player_id,pk"`
	Side          string    `bun:"side, notnull"` // "A" | "B"
}

type Rule struct {
	bun.BaseModel `bun:"table:rules"`
	ID            int          `bun:"id,pk,autoincrement,type:int"`
	Name          string       `bun:"name,notnull"`
	Description   string       `bun:"description"`
	Tournaments   []Tournament `bun:"m2m:tournament_rules,join:ID=RuleID"`
	CreatedAt     time.Time    `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time   `bun:"updated_at,nullzero"`
	DeletedAt     *time.Time   `bun:"deleted_at,soft_delete,nullzero"`
}

type TournamentRule struct {
	bun.BaseModel `bun:"table:tournament_rules"`
	TournamentID  uuid.UUID `bun:"tournament_id,pk"`
	RuleID        int       `bun:"rule_id,pk"`
	CreatedAt     time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type Match struct {
	bun.BaseModel `bun:"table:matches"`
	ID            uuid.UUID   `bun:"id,pk,type:uuid"`
	TournamentID  uuid.UUID   `bun:"tournament_id,notnull"`
	Format        string      `bun:"format,notnull"`
	Status        string      `bun:"status,notnull,default:'scheduled'"` // in_progress | finished | walkover
	WinnerTeamID  *uuid.UUID  `bun:"winner_team_id,nullzero"`
	Teams         []MatchTeam `bun:rel:has-many,join:ID=MatchID"`
	MatchSets     []MatchSet  `bun:"rel:has-many,join:ID=MatchID"`
	StartedAt     *time.Time  `bun:"started_at"`
	FinishedAt    *time.Time  `bun:"finished_at"`
	CreatedAt     time.Time   `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time  `bun:"updated_at,nullzero"`
	DeletedAt     *time.Time  `bun:"deleted_at,soft_delete,nullzero"`
	Tournament    *Tournament `bun:"rel:belongs-to,join:TournamentID=ID"`
}

type MatchTeam struct {
	bun.BaseModel `bun:"table:match_teams"`

	ID        uuid.UUID         `bun:"id,pk,type:uuid"`
	MatchID   uuid.UUID         `bun:"match_id,notnull"`
	Side      string            `bun:"side,notnull"` // "A" | "B"
	Players   []MatchTeamPlayer `bun:"rel:has-many,join:ID=MatchTeamID"`
	CreatedAt time.Time         `bun:"created_at,notnull,default:current_timestamp"`
}

type MatchTeamPlayer struct {
	bun.BaseModel `bun:table:match_team_players"`

	MatchTeamID uuid.UUID `bun:"match_team_id,pk"`
	PlayerID    uuid.UUID `bun:"player_id,pk"`
	Position    int       `bun:"position,notnull"` // 1,2 (useful for ordering in UI)
	CreatedAt   time.Time `bun:"created_at,notnull,default:current_timestamp"`
}

type MatchSet struct {
	bun.BaseModel `bun:"table:match_sets"`
	ID            uuid.UUID  `bun:"id,pk,type:uuid"`
	MatchID       uuid.UUID  `bun:"match_id,notnull"`
	Number        int        `bun:"number,notnull"` //unique per match
	ScoreA        int        `bun:"score_a,notnull"`
	ScoreB        int        `bun:"score_b,notnull"`
	CreatedAt     time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt     *time.Time `bun:"updated_at,nullzero"`
	DeletedAt     *time.Time `bun:"deleted_at,soft_delete,nullzero"`
	Match         *Match     `bun:"rel:belongs-to,join:MatchID=ID"`
}

type EloHistory struct {
	PlayerId  uuid.UUID
	MatchID   uuid.UUID
	Before    int
	After     int
	CreatedAt time.Time
}
