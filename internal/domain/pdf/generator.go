package pdf

import (
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/domain/event"
)

type Generator interface {
	GenerateTournamentReport(t *event.Event, divs []*division.Division) ([]byte, error)
	GenerateEventReport(e *tournament.Tournament, divs []*division.Division) ([]byte, error)
}
