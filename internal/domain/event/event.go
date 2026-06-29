package event

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/tournament"
	"time"
)

var (
	ErrInvalidEventName  = errors.New("event name is required")
	ErrInvalidDivisionIDs = errors.New("at least one division ID is required")
	ErrInvalidEventDates = errors.New("event end date must be after start date")
)

type Repository interface {
	Save(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetByIDDeep(ctx context.Context, id string) (*Event, error)
	GetAll(ctx context.Context) ([]*Event, error)
	Delete(ctx context.Context, id string) error
	DeleteEvents(ctx context.Context, ids []string) error
}

type Event struct {
	ID          string
	Name        string
	DivisionIDs []string
	SkipElo     bool
	StartDate   time.Time
	EndDate     time.Time
	NumTables   int
	Tournaments []*tournament.Tournament
}

func NewEvent(id string, name string, divisionIDs []string, skipElo bool, start, end time.Time) (*Event, error) {
	if name == "" {
		return nil, ErrInvalidEventName
	}
	if !skipElo && len(divisionIDs) == 0 {
		return nil, ErrInvalidDivisionIDs
	}
	if end.Before(start) {
		return nil, ErrInvalidEventDates
	}

	return &Event{
		ID:          id,
		Name:        name,
		DivisionIDs: divisionIDs,
		SkipElo:     skipElo,
		StartDate:   start,
		EndDate:     end,
		NumTables:   4,
		Tournaments: []*tournament.Tournament{},
	}, nil
}
