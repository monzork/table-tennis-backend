package pdf

import (
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type Generator interface {
	GenerateTournamentReport(t *tournament.Tournament) ([]byte, error)
	GenerateEventReport(e *event.Event) ([]byte, error)
}
