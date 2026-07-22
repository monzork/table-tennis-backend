package tournament

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/event"
	"time"
)

var (
	ErrInvalidEventName   = errors.New("tournament name is required")
	ErrInvalidDivisionIDs = errors.New("at least one division ID is required")
	ErrInvalidEventDates  = errors.New("tournament end date must be after start date")
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
	ID              string
	Name            string
	DivisionIDs     []string
	SkipElo         bool
	StartDate       time.Time
	EndDate         time.Time
	NumTables       int
	TablePriorities map[string][]int
	Events          []*event.Event
}

func NewEvent(id string, name string, divisionIDs []string, skipElo bool, start, end time.Time) (*Tournament, error) {
	if name == "" {
		return nil, ErrInvalidEventName
	}
	if !skipElo && len(divisionIDs) == 0 {
		return nil, ErrInvalidDivisionIDs
	}
	if end.Before(start) {
		return nil, ErrInvalidEventDates
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
