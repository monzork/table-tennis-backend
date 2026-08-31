package pdf

import (
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type Generator interface {
	GenerateTournamentReport(t *event.Event, divs []*division.Division) ([]byte, error)
	// includeIDPhotos appends each player's cédula de identidad photos at
	// the end of the report -- the caller should pass the owning
	// tournament's IncludeIDPhotosInReport setting.
	GenerateEventReport(e *tournament.Tournament, divs []*division.Division, includeIDPhotos bool) ([]byte, error)
}
