package dashboard_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/dashboard"
	"table-tennis-backend/internal/domain/chart"
	dashboardDomain "table-tennis-backend/internal/domain/dashboard"
)

type fakeDashboardRepo struct {
	stats         dashboardDomain.Stats
	statsErr      error
	byCountry     []chart.BarItem
	byCountryErr  error
	byFormat      []chart.BarItem
	byFormatErr   error
	byMonth       []chart.BarItem
	byMonthErr    error
	topGainers    []chart.BarItem
	topGainersErr error
}

func (f *fakeDashboardRepo) GetStats(ctx context.Context) (dashboardDomain.Stats, error) {
	return f.stats, f.statsErr
}
func (f *fakeDashboardRepo) GetPlayersByCountry(ctx context.Context) ([]chart.BarItem, error) {
	return f.byCountry, f.byCountryErr
}
func (f *fakeDashboardRepo) GetEventsByFormat(ctx context.Context) ([]chart.BarItem, error) {
	return f.byFormat, f.byFormatErr
}
func (f *fakeDashboardRepo) GetTournamentActivityByMonth(ctx context.Context) ([]chart.BarItem, error) {
	return f.byMonth, f.byMonthErr
}
func (f *fakeDashboardRepo) GetTopEloGainers(ctx context.Context, limit int) ([]chart.BarItem, error) {
	return f.topGainers, f.topGainersErr
}

type fakeChartGen struct{}

func (f *fakeChartGen) BarChart(items []chart.BarItem) (string, error) {
	if len(items) == 0 {
		return "", nil
	}
	return "<svg-bar/>", nil
}
func (f *fakeChartGen) LineChart(points []chart.Point) (string, error) { return "", nil }

func TestGetPublicDashboardViewUseCase_Execute(t *testing.T) {
	t.Run("assembles stats and charts from all repository calls", func(t *testing.T) {
		repo := &fakeDashboardRepo{
			stats:      dashboardDomain.Stats{TotalPlayers: 10, TotalTournaments: 3, TotalMatches: 50},
			byCountry:  []chart.BarItem{{Label: "NIC", Value: 8}},
			byFormat:   []chart.BarItem{{Label: "singles", Value: 5}},
			byMonth:    []chart.BarItem{{Label: "Jan 2026", Value: 2}},
			topGainers: []chart.BarItem{{Label: "Alice A", Value: 100}},
		}
		uc := dashboard.NewGetPublicDashboardViewUseCase(repo, &fakeChartGen{})

		view, err := uc.Execute(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if view.Stats.TotalPlayers != 10 {
			t.Errorf("expected stats passthrough, got %+v", view.Stats)
		}
		if view.PlayersByCountrySVG != "<svg-bar/>" || view.EventsByFormatSVG != "<svg-bar/>" ||
			view.ActivityByMonthSVG != "<svg-bar/>" || view.TopGainersSVG != "<svg-bar/>" {
			t.Errorf("expected all 4 charts rendered, got %+v", view)
		}
	})

	t.Run("empty buckets render empty charts", func(t *testing.T) {
		repo := &fakeDashboardRepo{}
		uc := dashboard.NewGetPublicDashboardViewUseCase(repo, &fakeChartGen{})

		view, err := uc.Execute(context.Background())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if view.PlayersByCountrySVG != "" || view.EventsByFormatSVG != "" ||
			view.ActivityByMonthSVG != "" || view.TopGainersSVG != "" {
			t.Errorf("expected all charts empty for empty buckets, got %+v", view)
		}
	})

	t.Run("propagates stats error", func(t *testing.T) {
		repo := &fakeDashboardRepo{statsErr: errors.New("boom")}
		uc := dashboard.NewGetPublicDashboardViewUseCase(repo, &fakeChartGen{})

		if _, err := uc.Execute(context.Background()); err == nil {
			t.Fatal("expected error to propagate")
		}
	})

	t.Run("propagates chart data fetch error", func(t *testing.T) {
		repo := &fakeDashboardRepo{byCountryErr: errors.New("boom")}
		uc := dashboard.NewGetPublicDashboardViewUseCase(repo, &fakeChartGen{})

		if _, err := uc.Execute(context.Background()); err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}
