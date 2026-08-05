package dashboard

import (
	"context"

	"table-tennis-backend/internal/domain/chart"
	"table-tennis-backend/internal/domain/dashboard"

	"golang.org/x/sync/errgroup"
)

// PublicDashboardView is the fully-assembled view model for the public
// dashboard page: headline counts plus pre-rendered chart markup.
type PublicDashboardView struct {
	Stats               dashboard.Stats
	PlayersByCountrySVG string
	EventsByFormatSVG   string
	ActivityByMonthSVG  string
	TopGainersSVG       string
}

type GetPublicDashboardViewUseCase struct {
	repo     dashboard.Repository
	chartGen chart.Generator
}

func NewGetPublicDashboardViewUseCase(repo dashboard.Repository, chartGen chart.Generator) *GetPublicDashboardViewUseCase {
	return &GetPublicDashboardViewUseCase{repo: repo, chartGen: chartGen}
}

// Execute fetches every dashboard data point concurrently (each backed by a
// single narrow SQL aggregate query, not full object hydration) and renders
// the bar charts.
func (uc *GetPublicDashboardViewUseCase) Execute(ctx context.Context) (*PublicDashboardView, error) {
	view := &PublicDashboardView{}

	eg, egCtx := errgroup.WithContext(ctx)

	eg.Go(func() (err error) {
		view.Stats, err = uc.repo.GetStats(egCtx)
		return err
	})
	eg.Go(func() error {
		items, err := uc.repo.GetPlayersByCountry(egCtx)
		if err != nil {
			return err
		}
		view.PlayersByCountrySVG, err = uc.chartGen.BarChart(items)
		return err
	})
	eg.Go(func() error {
		items, err := uc.repo.GetEventsByFormat(egCtx)
		if err != nil {
			return err
		}
		view.EventsByFormatSVG, err = uc.chartGen.BarChart(items)
		return err
	})
	eg.Go(func() error {
		items, err := uc.repo.GetTournamentActivityByMonth(egCtx)
		if err != nil {
			return err
		}
		view.ActivityByMonthSVG, err = uc.chartGen.BarChart(items)
		return err
	})
	eg.Go(func() error {
		items, err := uc.repo.GetTopEloGainers(egCtx, 10)
		if err != nil {
			return err
		}
		view.TopGainersSVG, err = uc.chartGen.BarChart(items)
		return err
	})

	if err := eg.Wait(); err != nil {
		return nil, err
	}
	return view, nil
}
