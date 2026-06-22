package event

import (
	"errors"
	"table-tennis-backend/internal/domain/tournament"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidEventName  = errors.New("event name is required")
	ErrInvalidDivisionID = errors.New("division ID is required")
	ErrInvalidEventDates = errors.New("event end date must be after start date")
)

type Event struct {
	ID          uuid.UUID
	Name        string
	DivisionID  string
	SkipElo     bool
	StartDate   time.Time
	EndDate     time.Time
	NumTables   int
	Tournaments []*tournament.Tournament
}

func NewEvent(name string, divisionID string, skipElo bool, start, end time.Time) (*Event, error) {
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
		ID:          uuid.New(),
		Name:        name,
		DivisionID:  divisionID,
		SkipElo:     skipElo,
		StartDate:   start,
		EndDate:     end,
		NumTables:   4,
		Tournaments: []*tournament.Tournament{},
	}, nil
}
