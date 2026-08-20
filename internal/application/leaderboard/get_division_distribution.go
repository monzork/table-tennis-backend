package leaderboard

import (
	"table-tennis-backend/internal/domain/chart"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
)

type GetDivisionDistributionUseCase struct {
	chartGen chart.Generator
}

func NewGetDivisionDistributionUseCase(chartGen chart.Generator) *GetDivisionDistributionUseCase {
	return &GetDivisionDistributionUseCase{chartGen: chartGen}
}

// Execute renders a bar chart of player counts per division for the given
// rank type, built from the full unfiltered roster. Unlike groupPlayers
// (used for the ranking table itself), zero-count divisions are kept as
// empty bars -- meaningful signal for a distribution view.
func (uc *GetDivisionDistributionUseCase) Execute(players []*player.Player, divisions []*division.Division, rankType string) (string, error) {
	rankable := filterRankableDivisions(divisions)
	if len(rankable) == 0 {
		return "", nil
	}

	items := make([]chart.BarItem, len(rankable))
	for i, d := range rankable {
		count := 0
		for _, p := range players {
			if d.MatchesGender(p.Gender) && d.ContainsElo(eloOf(p, rankType)) {
				count++
			}
		}
		items[i] = chart.BarItem{Label: d.Name, Value: float64(count)}
	}

	return uc.chartGen.BarChart(items)
}
