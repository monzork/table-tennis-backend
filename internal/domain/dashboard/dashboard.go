package dashboard

import (
	"context"

	"table-tennis-backend/internal/domain/chart"
)

// Stats holds the headline counts shown on the public dashboard.
type Stats struct {
	TotalPlayers     int
	TotalTournaments int
	TotalMatches     int
}

// Repository provides read-only aggregate data for the public dashboard.
// Every method is expected to be backed by a single narrow SQL aggregate
// query (COUNT/GROUP BY) rather than loading and counting full domain
// object graphs.
type Repository interface {
	GetStats(ctx context.Context) (Stats, error)
	GetPlayersByCountry(ctx context.Context) ([]chart.BarItem, error)
	GetEventsByFormat(ctx context.Context) ([]chart.BarItem, error)
	GetTournamentActivityByMonth(ctx context.Context) ([]chart.BarItem, error)
	GetTopEloGainers(ctx context.Context, limit int) ([]chart.BarItem, error)
}
