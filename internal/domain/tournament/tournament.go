package tournament

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/event"
	"time"
)

var (
	ErrInvalidTournamentName  = errors.New("tournament name is required")
	ErrInvalidDivisionIDs     = errors.New("at least one division ID is required")
	ErrInvalidTournamentDates = errors.New("tournament end date must be after start date")
)

type Repository interface {
	Save(ctx context.Context, e *Tournament) error
	Update(ctx context.Context, e *Tournament) error
	GetByID(ctx context.Context, id string) (*Tournament, error)
	GetByIDDeep(ctx context.Context, id string) (*Tournament, error)
	GetAll(ctx context.Context) ([]*Tournament, error)
	Delete(ctx context.Context, id string) error
	DeleteEvents(ctx context.Context, ids []string) error
}

type Tournament struct {
	ID                 string
	Name               string
	DivisionIDs        []string
	SkipElo            bool
	StartDate          time.Time
	EndDate            time.Time
	NumTables          int
	TablePriorities    map[string][]int
	FederationEndorsed bool
	// IncludeIDPhotosInReport controls whether the exported tournament PDF
	// appends each player's cédula de identidad photos -- off by default
	// since downloading/embedding every player's ID photo is the slowest
	// and most memory-heavy part of report generation (a large tournament
	// caused an OOM before photos were downscaled/re-encoded; keeping this
	// opt-in bounds the cost to tournaments that actually need it).
	IncludeIDPhotosInReport bool
	Events                  []*event.Event
}

func NewTournament(id string, name string, divisionIDs []string, skipElo bool, start, end time.Time) (*Tournament, error) {
	if name == "" {
		return nil, ErrInvalidTournamentName
	}
	if !skipElo && len(divisionIDs) == 0 {
		return nil, ErrInvalidDivisionIDs
	}
	if end.Before(start) {
		return nil, ErrInvalidTournamentDates
	}

	return &Tournament{
		ID:          id,
		Name:        name,
		DivisionIDs: divisionIDs,
		SkipElo:     skipElo,
		StartDate:   start,
		EndDate:     end,
		NumTables:   4,
		Events:      []*event.Event{},
	}, nil
}

// TablePriorityFor returns the preferred table assignment order for a division,
// or nil if the tournament has no configured priority for it.
func (t *Tournament) TablePriorityFor(divisionID string) []int {
	if t.TablePriorities == nil {
		return nil
	}
	return t.TablePriorities[divisionID]
}
