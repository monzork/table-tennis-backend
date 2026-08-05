package leaderboard_test

import (
	"errors"
	"testing"

	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/chart"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
)

type fakeChartGen struct {
	barItems []chart.BarItem
	barErr   error
}

func (f *fakeChartGen) BarChart(items []chart.BarItem) (string, error) {
	f.barItems = items
	if f.barErr != nil {
		return "", f.barErr
	}
	return "<svg-bar/>", nil
}

func (f *fakeChartGen) LineChart(points []chart.Point) (string, error) {
	return "", nil
}

func TestGetDivisionDistributionUseCase_Execute(t *testing.T) {
	divisions := []*division.Division{
		{ID: "none", Name: "No Division"}, // must be excluded
		{ID: "low", Name: "Segunda", MinElo: 0, MaxElo: eloPtr(1000), Category: "both"},
		{ID: "high", Name: "Primera", MinElo: 1000, MaxElo: nil, Category: "both"},
	}

	t.Run("counts players per division, keeping zero-count divisions", func(t *testing.T) {
		players := []*player.Player{
			{ID: "1", SinglesElo: 900},
			{ID: "2", SinglesElo: 1500},
			{ID: "3", SinglesElo: 1600},
		}
		gen := &fakeChartGen{}
		uc := leaderboard.NewGetDivisionDistributionUseCase(gen)

		svg, err := uc.Execute(players, divisions, "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svg != "<svg-bar/>" {
			t.Errorf("expected generator output passthrough, got %q", svg)
		}
		if len(gen.barItems) != 2 {
			t.Fatalf("expected 2 buckets (No Division excluded), got %d", len(gen.barItems))
		}
		if gen.barItems[0].Label != "Segunda" || gen.barItems[0].Value != 1 {
			t.Errorf("expected Segunda=1, got %+v", gen.barItems[0])
		}
		if gen.barItems[1].Label != "Primera" || gen.barItems[1].Value != 2 {
			t.Errorf("expected Primera=2, got %+v", gen.barItems[1])
		}
	})

	t.Run("empty roster still keeps divisions as zero-count bars", func(t *testing.T) {
		gen := &fakeChartGen{}
		uc := leaderboard.NewGetDivisionDistributionUseCase(gen)

		_, err := uc.Execute(nil, divisions, "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gen.barItems) != 2 || gen.barItems[0].Value != 0 || gen.barItems[1].Value != 0 {
			t.Errorf("expected both divisions present with 0 count, got %+v", gen.barItems)
		}
	})

	t.Run("no rankable divisions skips chart entirely", func(t *testing.T) {
		gen := &fakeChartGen{}
		uc := leaderboard.NewGetDivisionDistributionUseCase(gen)

		svg, err := uc.Execute(nil, []*division.Division{{ID: "none", Name: "No Division"}}, "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svg != "" {
			t.Errorf("expected empty svg when no rankable divisions, got %q", svg)
		}
		if gen.barItems != nil {
			t.Errorf("expected generator not to be called, got %+v", gen.barItems)
		}
	})

	t.Run("doubles rank type uses doubles elo", func(t *testing.T) {
		players := []*player.Player{{ID: "1", SinglesElo: 1500, DoublesElo: 900}}
		gen := &fakeChartGen{}
		uc := leaderboard.NewGetDivisionDistributionUseCase(gen)

		if _, err := uc.Execute(players, divisions, "doubles"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if gen.barItems[0].Value != 1 || gen.barItems[1].Value != 0 {
			t.Errorf("expected player bucketed by doubles elo (900 -> Segunda), got %+v", gen.barItems)
		}
	})

	t.Run("propagates generator error", func(t *testing.T) {
		gen := &fakeChartGen{barErr: errors.New("boom")}
		uc := leaderboard.NewGetDivisionDistributionUseCase(gen)

		if _, err := uc.Execute(nil, divisions, "singles"); err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}
