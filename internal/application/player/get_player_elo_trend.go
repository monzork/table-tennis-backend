package player

import (
	"sort"

	"table-tennis-backend/internal/domain/chart"
)

type GetPlayerEloTrendUseCase struct {
	chartGen chart.Generator
}

func NewGetPlayerEloTrendUseCase(chartGen chart.Generator) *GetPlayerEloTrendUseCase {
	return &GetPlayerEloTrendUseCase{chartGen: chartGen}
}

// Execute renders a chronological Elo trend chart for one rank type
// ("singles" or "doubles") from a player's already-fetched tournament
// history. No I/O — purely reshapes data already in hand.
func (uc *GetPlayerEloTrendUseCase) Execute(history []PlayerTournamentView, rankType string) (string, error) {
	points := buildEloTrendPoints(history, rankType)
	return uc.chartGen.LineChart(points)
}

func buildEloTrendPoints(history []PlayerTournamentView, rankType string) []chart.Point {
	var points []chart.Point
	for _, tv := range history {
		if tv.Tournament == nil {
			continue
		}
		for _, ev := range tv.Events {
			if !ev.Participated {
				continue
			}

			before, after := ev.EloBeforeSingles, ev.EloAfterSingles
			if rankType == "doubles" {
				before, after = ev.EloBeforeDoubles, ev.EloAfterDoubles
			}

			var val *int16
			switch {
			case after != nil:
				val = after
			case before != nil:
				val = before
			default:
				continue // no Elo data for this rank type: skip, never plot as 0
			}

			points = append(points, chart.Point{
				X:     tv.Tournament.StartDate,
				Y:     float64(*val),
				Label: tv.Tournament.Name,
			})
		}
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].X.Before(points[j].X)
	})
	return points
}
