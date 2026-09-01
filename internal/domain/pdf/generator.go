package pdf

import (
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type Generator interface {
	// lang selects the report's rendered language ("es" for Spanish; any
	// other value, including "en" or empty, renders in English).
	GenerateTournamentReport(t *event.Event, divs []*division.Division, lang string) ([]byte, error)
	// includeIDPhotos appends each player's cédula de identidad photos at
	// the end of the report -- the caller should pass the owning
	// tournament's IncludeIDPhotosInReport setting. lang selects the
	// report's rendered language, same as GenerateTournamentReport.
	GenerateEventReport(e *tournament.Tournament, divs []*division.Division, includeIDPhotos bool, lang string) ([]byte, error)
}
