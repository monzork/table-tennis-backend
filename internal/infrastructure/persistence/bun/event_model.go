package bun

import (
	"time"

	"table-tennis-backend/internal/domain/event"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

type EventModel struct {
	bun.BaseModel `bun:"table:events"`

	ID                            uuid.UUID         `bun:"id,pk,type:uuid"`
	Name                          string            `bun:"name,notnull"`
	Type                          string            `bun:"type,notnull,default:'singles'"`
	Format                        string            `bun:"format,notnull,default:'elimination'"`
	Status                        string            `bun:"status,notnull,default:'in_progress'"`
	EventCategory                 string            `bun:"tournament_category,notnull,default:'open'"`
	StartDate                     time.Time         `bun:"start_date,notnull"`
	EndDate                       time.Time         `bun:"end_date,notnull"`
	GroupPassCount                int               `bun:"group_pass_count,notnull,default:2"`
	LosersGroupPassCount          int               `bun:"losers_group_pass_count,notnull,default:0"`
	RegistrationOpen              bool              `bun:"registration_open,notnull,default:false"`
	EventID                       *uuid.UUID        `bun:"tournament_id,type:uuid"`
	SkipElo                       bool              `bun:"skip_elo,notnull,default:false"`
	TeamFormat                    string            `bun:"team_format,nullzero"`
	DivisionFormats               map[string]string `bun:"division_formats,type:json"`
	DivisionGroupPassCounts       map[string]int    `bun:"division_group_pass_counts,type:json"`
	DivisionLosersGroupPassCounts map[string]int    `bun:"division_losers_group_pass_counts,type:json"`
	DivisionGroupCounts           map[string]int    `bun:"division_group_counts,type:json"`

	WinnerName            string                   `bun:"winner_name,nullzero"`
	NumTables             int                      `bun:"num_tables,notnull,default:0"`
	HasThirdPlaceMatch    bool                     `bun:"has_third_place_match,notnull,default:false"`
	KnockoutBracketsCount int                      `bun:"knockout_brackets_count,notnull,default:1"`
	Metrics               *event.TournamentMetrics `bun:"metrics,type:jsonb"`
	ManualSeedingLocked   bool                     `bun:"manual_seeding_locked,notnull,default:false"`
	CreatedAt             time.Time                `bun:"created_at,notnull,default:current_timestamp"`
	UpdatedAt             *time.Time               `bun:"updated_at,nullzero"`

	Participants  []EventParticipantModel `bun:"rel:has-many,join:id=event_id"`
	Groups        []GroupModel            `bun:"rel:has-many,join:id=event_id"`
	Teams         []TeamModel             `bun:"rel:has-many,join:id=event_id"`
	Matches       []MatchModel            `bun:"rel:has-many,join:id=event_id"`
	StageRules    []StageRuleModel        `bun:"rel:has-many,join:id=event_id"`
	DivisionRules []DivisionRuleModel     `bun:"rel:has-many,join:id=event_id"`
}

// join table — no back-refs to avoid circular resolution at RegisterModel time
type EventParticipantModel struct {
	bun.BaseModel `bun:"table:event_participants"`

	TournamentID uuid.UUID `bun:"event_id,pk,type:uuid"`
	PlayerID     uuid.UUID `bun:"player_id,pk,type:uuid"`
	Pin          string    `bun:"pin,notnull,default:'0000'"`

	EloBeforeSingles *int16 `bun:"elo_before_singles"`
	EloBeforeDoubles *int16 `bun:"elo_before_doubles"`
	EloAfterSingles  *int16 `bun:"elo_after_singles"`
	EloAfterDoubles  *int16 `bun:"elo_after_doubles"`

	Player *PlayerModel `bun:"rel:belongs-to,join:player_id=id"`
}

type EventOfficialModel struct {
	bun.BaseModel `bun:"table:event_officials"`

	TournamentID uuid.UUID `bun:"event_id,pk,type:uuid"`
	PlayerID     uuid.UUID `bun:"player_id,pk,type:uuid"`
	Pin          string    `bun:"pin,notnull"`
}

type GroupModel struct {
	bun.BaseModel `bun:"table:groups"`

	ID           uuid.UUID `bun:"id,pk,type:uuid"`
	TournamentID uuid.UUID `bun:"event_id,type:uuid"`
	Name         string    `bun:"name,notnull"`

	Participants []GroupParticipantModel `bun:"rel:has-many,join:id=group_id"`
}

// join table — no back-refs to avoid circular resolution at RegisterModel time
type GroupParticipantModel struct {
	bun.BaseModel `bun:"table:group_participants"`

	GroupID  uuid.UUID `bun:"group_id,pk,type:uuid"`
	PlayerID uuid.UUID `bun:"player_id,pk,type:uuid"`
	Position int       `bun:"position,notnull,default:0"`

	Player *PlayerModel `bun:"rel:belongs-to,join:player_id=id"`
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
	TournamentID uuid.UUID `bun:"event_id,type:uuid"`
	Name         string    `bun:"name,notnull"`

	TeamPlayers []TeamPlayerModel `bun:"rel:has-many,join:id=team_id"`
}

type TeamPlayerModel struct {
	bun.BaseModel `bun:"table:team_players"`

	TeamID   uuid.UUID `bun:"team_id,pk,type:uuid"`
	PlayerID uuid.UUID `bun:"player_id,pk,type:uuid"`

	Player *PlayerModel `bun:"rel:belongs-to,join:player_id=id"`
}
