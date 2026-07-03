package pdf

import (
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type Generator interface {
	GenerateTournamentReport(t *tournament.Tournament, divs []*division.Division) ([]byte, error)
	GenerateEventReport(e *event.Event, divs []*division.Division) ([]byte, error)
}
