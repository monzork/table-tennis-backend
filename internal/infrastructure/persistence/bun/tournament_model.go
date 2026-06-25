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
	Type      string     `bun:"type,notnull,default:'singles'"`
	Format    string     `bun:"format,notnull,default:'elimination'"`
	Status    string     `bun:"status,notnull,default:'in_progress'"`
	EventCategory  string     `bun:"event_category,notnull,default:'open'"`
	StartDate      time.Time  `bun:"start_date,notnull"`
	EndDate        time.Time  `bun:"end_date,notnull"`
	GroupPassCount   int        `bun:"group_pass_count,notnull,default:2"`
	RegistrationOpen bool       `bun:"registration_open,notnull,default:false"`
	EventID        *uuid.UUID `bun:"event_id,type:uuid"`
	SkipElo        bool       `bun:"skip_elo,notnull,default:false"`
	TeamFormat     string     `bun:"team_format,nullzero"`
	WinnerName     string     `bun:"winner_name,nullzero"`
	NumTables      int        `bun:"num_tables,notnull,default:0"`
	CreatedAt        time.Time  `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt      *time.Time `bun:"updated_at,nullzero"`
}

// join table — no back-refs to avoid circular resolution at RegisterModel time
type TournamentParticipantModel struct {
	bun.BaseModel `bun:"table:tournament_participants"`

	TournamentID uuid.UUID `bun:"tournament_id,pk,type:uuid"`
	PlayerID     uuid.UUID `bun:"player_id,pk,type:uuid"`
	Pin          string    `bun:"pin,notnull,default:'0000'"`

	EloBeforeSingles *int16 `bun:"elo_before_singles"`
	EloBeforeDoubles *int16 `bun:"elo_before_doubles"`
	EloAfterSingles  *int16 `bun:"elo_after_singles"`
	EloAfterDoubles  *int16 `bun:"elo_after_doubles"`
}

type GroupModel struct {
	bun.BaseModel `bun:"table:groups"`

	ID           uuid.UUID `bun:"id,pk,type:uuid"`
	TournamentID uuid.UUID `bun:"tournament_id,type:uuid"`
	Name         string    `bun:"name,notnull"`
}

// join table — no back-refs to avoid circular resolution at RegisterModel time
type GroupParticipantModel struct {
	bun.BaseModel `bun:"table:group_participants"`

	GroupID  uuid.UUID `bun:"group_id,pk,type:uuid"`
	PlayerID uuid.UUID `bun:"player_id,pk,type:uuid"`
	Position int       `bun:"position,notnull,default:0"`
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

type TeamModel struct {
	bun.BaseModel `bun:"table:teams"`

	ID           uuid.UUID `bun:"id,pk,type:uuid"`
	TournamentID uuid.UUID `bun:"tournament_id,type:uuid"`
	Name         string    `bun:"name,notnull"`
}

type TeamPlayerModel struct {
	bun.BaseModel `bun:"table:team_players"`

	TeamID   uuid.UUID `bun:"team_id,pk,type:uuid"`
	PlayerID uuid.UUID `bun:"player_id,pk,type:uuid"`
}
