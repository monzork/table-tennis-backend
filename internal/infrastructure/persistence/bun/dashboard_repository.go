package bun

import (
	"context"
	"time"

	"table-tennis-backend/internal/domain/chart"
	"table-tennis-backend/internal/domain/dashboard"

	"github.com/uptrace/bun"
	"golang.org/x/sync/errgroup"
)

// DashboardRepository implements dashboard.Repository. Every method is a
// single narrow SQL aggregate query (COUNT/GROUP BY) -- never a full
// hydration of the Player/Tournament/Event domain object graphs -- since
// this data backs a public, unauthenticated page.
type DashboardRepository struct {
	db *bun.DB
}

func NewDashboardRepository(db *bun.DB) *DashboardRepository {
	return &DashboardRepository{db: db}
}

func (r *DashboardRepository) GetStats(ctx context.Context) (dashboard.Stats, error) {
	var stats dashboard.Stats
	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() (err error) {
		stats.TotalPlayers, err = ExtractDB(ctx, r.db).NewSelect().Model((*PlayerModel)(nil)).Count(egCtx)
		return err
	})
	eg.Go(func() (err error) {
		stats.TotalTournaments, err = ExtractDB(ctx, r.db).NewSelect().Model((*TournamentModel)(nil)).Count(egCtx)
		return err
	})
	eg.Go(func() (err error) {
		stats.TotalMatches, err = ExtractDB(ctx, r.db).NewSelect().Model((*MatchModel)(nil)).Where("status = ?", "finished").Count(egCtx)
		return err
	})

	if err := eg.Wait(); err != nil {
		return dashboard.Stats{}, err
	}
	return stats, nil
}

type countRow struct {
	Label string `bun:"label"`
	Count int    `bun:"count"`
}

func (r *DashboardRepository) GetPlayersByCountry(ctx context.Context) ([]chart.BarItem, error) {
	var rows []countRow
	err := ExtractDB(ctx, r.db).NewSelect().
		Model((*PlayerModel)(nil)).
		ColumnExpr("country AS label").
		ColumnExpr("count(*) AS count").
		GroupExpr("country").
		OrderExpr("count DESC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	return countRowsToBarItems(rows), nil
}

func (r *DashboardRepository) GetEventsByFormat(ctx context.Context) ([]chart.BarItem, error) {
	var rows []countRow
	err := ExtractDB(ctx, r.db).NewSelect().
		Model((*EventModel)(nil)).
		ColumnExpr("type AS label").
		ColumnExpr("count(*) AS count").
		GroupExpr("type").
		OrderExpr("count DESC").
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	return countRowsToBarItems(rows), nil
}

// GetTournamentActivityByMonth buckets tournaments by start month. Bucketing
// happens in Go (over just the start_date column, not full Tournament
// hydration) rather than via a SQL date_trunc, to stay portable across the
// Postgres (production) and SQLite (test) dialects this repository runs under.
func (r *DashboardRepository) GetTournamentActivityByMonth(ctx context.Context) ([]chart.BarItem, error) {
	var startDates []time.Time
	err := ExtractDB(ctx, r.db).NewSelect().
		Model((*TournamentModel)(nil)).
		ColumnExpr("start_date").
		OrderExpr("start_date ASC").
		Scan(ctx, &startDates)
	if err != nil {
		return nil, err
	}

	var order []string
	counts := make(map[string]int)
	for _, d := range startDates {
		key := d.Format("Jan 2006")
		if _, ok := counts[key]; !ok {
			order = append(order, key)
		}
		counts[key]++
	}

	items := make([]chart.BarItem, len(order))
	for i, k := range order {
		items[i] = chart.BarItem{Label: k, Value: float64(counts[k])}
	}
	return items, nil
}

type gainerRow struct {
	FirstName string `bun:"first_name"`
	LastName  string `bun:"last_name"`
	Gain      int64  `bun:"gain"`
}

func (r *DashboardRepository) GetTopEloGainers(ctx context.Context, limit int) ([]chart.BarItem, error) {
	var rows []gainerRow
	err := ExtractDB(ctx, r.db).NewSelect().
		TableExpr("event_participants AS ep").
		Join("JOIN players AS p ON p.id = ep.player_id").
		ColumnExpr("p.first_name AS first_name").
		ColumnExpr("p.last_name AS last_name").
		ColumnExpr("SUM(COALESCE(ep.elo_after_singles - ep.elo_before_singles, 0) + COALESCE(ep.elo_after_doubles - ep.elo_before_doubles, 0)) AS gain").
		GroupExpr("p.id, p.first_name, p.last_name").
		Having("SUM(COALESCE(ep.elo_after_singles - ep.elo_before_singles, 0) + COALESCE(ep.elo_after_doubles - ep.elo_before_doubles, 0)) > 0").
		OrderExpr("gain DESC").
		Limit(limit).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}

	items := make([]chart.BarItem, len(rows))
	for i, row := range rows {
		items[i] = chart.BarItem{Label: row.FirstName + " " + row.LastName, Value: float64(row.Gain)}
	}
	return items, nil
}

func countRowsToBarItems(rows []countRow) []chart.BarItem {
	items := make([]chart.BarItem, len(rows))
	for i, row := range rows {
		items[i] = chart.BarItem{Label: row.Label, Value: float64(row.Count)}
	}
	return items
}
