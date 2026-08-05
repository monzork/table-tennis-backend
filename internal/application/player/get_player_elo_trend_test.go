package player_test

import (
	"errors"
	"testing"
	"time"

	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/domain/chart"
	tournamentEvent "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/tournament"
)

type fakeChartGen struct {
	barItems  []chart.BarItem
	linePts   []chart.Point
	barErr    error
	lineErr   error
	barSVG    string
	lineSVG   string
	barCalls  int
	lineCalls int
}

func (f *fakeChartGen) BarChart(items []chart.BarItem) (string, error) {
	f.barCalls++
	f.barItems = items
	if f.barErr != nil {
		return "", f.barErr
	}
	if f.barSVG != "" {
		return f.barSVG, nil
	}
	return "<svg-bar/>", nil
}

func (f *fakeChartGen) LineChart(points []chart.Point) (string, error) {
	f.lineCalls++
	f.linePts = points
	if f.lineErr != nil {
		return "", f.lineErr
	}
	if f.lineSVG != "" {
		return f.lineSVG, nil
	}
	return "<svg-line/>", nil
}

func i16(v int16) *int16 { return &v }

func TestGetPlayerEloTrendUseCase_Execute(t *testing.T) {
	t1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) // intentionally out of order

	history := []player.PlayerTournamentView{
		{
			Tournament: &tournament.Tournament{ID: "t1", Name: "March Cup", StartDate: t1},
			Events: []player.PlayerEventStatsView{
				{Event: &tournamentEvent.Event{ID: "e1"}, Participated: true, EloBeforeSingles: i16(1000), EloAfterSingles: i16(1020)},
			},
		},
		{
			Tournament: &tournament.Tournament{ID: "t2", Name: "January Cup", StartDate: t2},
			Events: []player.PlayerEventStatsView{
				{Event: &tournamentEvent.Event{ID: "e2"}, Participated: true, EloBeforeSingles: i16(950), EloAfterSingles: i16(1000)},
			},
		},
	}

	t.Run("sorts chronologically and uses after-elo, falling back to before", func(t *testing.T) {
		gen := &fakeChartGen{}
		uc := player.NewGetPlayerEloTrendUseCase(gen)

		svg, err := uc.Execute(history, "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if svg != "<svg-line/>" {
			t.Errorf("expected generator output passthrough, got %q", svg)
		}
		if len(gen.linePts) != 2 {
			t.Fatalf("expected 2 points, got %d", len(gen.linePts))
		}
		if gen.linePts[0].Label != "January Cup" || gen.linePts[1].Label != "March Cup" {
			t.Errorf("expected chronological order (Jan before March), got %+v", gen.linePts)
		}
		if gen.linePts[0].Y != 1000 || gen.linePts[1].Y != 1020 {
			t.Errorf("expected after-elo values, got %+v", gen.linePts)
		}
	})

	t.Run("skips events the player did not participate in", func(t *testing.T) {
		h := []player.PlayerTournamentView{
			{
				Tournament: &tournament.Tournament{ID: "t1", Name: "Cup", StartDate: t1},
				Events: []player.PlayerEventStatsView{
					{Event: &tournamentEvent.Event{ID: "e1"}, Participated: false, EloAfterSingles: i16(1500)},
				},
			},
		}
		gen := &fakeChartGen{}
		uc := player.NewGetPlayerEloTrendUseCase(gen)
		if _, err := uc.Execute(h, "singles"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gen.linePts) != 0 {
			t.Errorf("expected 0 points for non-participated event, got %+v", gen.linePts)
		}
	})

	t.Run("skips events with no elo data for the requested rank type", func(t *testing.T) {
		h := []player.PlayerTournamentView{
			{
				Tournament: &tournament.Tournament{ID: "t1", Name: "Cup", StartDate: t1},
				Events: []player.PlayerEventStatsView{
					{Event: &tournamentEvent.Event{ID: "e1"}, Participated: true}, // no before/after elo at all
				},
			},
		}
		gen := &fakeChartGen{}
		uc := player.NewGetPlayerEloTrendUseCase(gen)
		if _, err := uc.Execute(h, "singles"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gen.linePts) != 0 {
			t.Errorf("expected 0 points when neither before nor after elo present, got %+v", gen.linePts)
		}
	})

	t.Run("doubles rank type reads doubles elo fields", func(t *testing.T) {
		h := []player.PlayerTournamentView{
			{
				Tournament: &tournament.Tournament{ID: "t1", Name: "Cup", StartDate: t1},
				Events: []player.PlayerEventStatsView{
					{Event: &tournamentEvent.Event{ID: "e1"}, Participated: true, EloBeforeDoubles: i16(900), EloAfterDoubles: i16(940)},
				},
			},
		}
		gen := &fakeChartGen{}
		uc := player.NewGetPlayerEloTrendUseCase(gen)
		if _, err := uc.Execute(h, "doubles"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(gen.linePts) != 1 || gen.linePts[0].Y != 940 {
			t.Errorf("expected 1 point at doubles after-elo 940, got %+v", gen.linePts)
		}
	})

	t.Run("propagates generator error", func(t *testing.T) {
		gen := &fakeChartGen{lineErr: errors.New("boom")}
		uc := player.NewGetPlayerEloTrendUseCase(gen)
		if _, err := uc.Execute(history, "singles"); err == nil {
			t.Fatal("expected error to propagate")
		}
	})
}
