package event

import (
	"context"
	"errors"
	"table-tennis-backend/internal/domain/tournament"
	"time"
)

var (
	ErrInvalidEventName  = errors.New("event name is required")
	ErrInvalidDivisionID = errors.New("division ID is required")
	ErrInvalidEventDates = errors.New("event end date must be after start date")
)

type Repository interface {
	Save(ctx context.Context, e *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetAll(ctx context.Context) ([]*Event, error)
	Delete(ctx context.Context, id string) error
	DeleteEvents(ctx context.Context, ids []string) error
}

type Event struct {
	ID          string
	Name        string
	DivisionID  string
	SkipElo     bool
	StartDate   time.Time
	EndDate     time.Time
	NumTables   int
	Tournaments []*tournament.Tournament
}

func NewEvent(id string, name string, divisionID string, skipElo bool, start, end time.Time) (*Event, error) {
	if name == "" {
		return nil, ErrInvalidEventName
	}
	if !skipElo && divisionID == "" {
		return nil, ErrInvalidDivisionID
	}
	if end.Before(start) {
		return nil, ErrInvalidEventDates
	}

	return &Event{
		ID:          id,
		Name:        name,
		DivisionID:  divisionID,
		SkipElo:     skipElo,
		StartDate:   start,
		EndDate:     end,
		NumTables:   4,
		Tournaments: []*tournament.Tournament{},
	}, nil
}
