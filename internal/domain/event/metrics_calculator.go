package event

type MetricsCalculator struct{}

func NewMetricsCalculator() *MetricsCalculator {
	return &MetricsCalculator{}
}

func (c *MetricsCalculator) Calculate(t *Event, snapshots []ParticipantSnapshot) *TournamentMetrics {
	metrics := &TournamentMetrics{
		SchemaVersion:   1,
		DivisionMetrics: make(map[string]DivisionMetric),
	}

	var avgEloSum int
	var avgEloCount int
	for _, snap := range snapshots {
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			if snap.EloBeforeDoubles != nil {
				avgEloSum += int(*snap.EloBeforeDoubles)
				avgEloCount++
			}
		} else {
			if snap.EloBeforeSingles != nil {
				avgEloSum += int(*snap.EloBeforeSingles)
				avgEloCount++
			}
		}
	}
	if avgEloCount > 0 {
		metrics.AverageEloAtStart = float64(avgEloSum) / float64(avgEloCount)
	}

	var maxEloGain int
	for _, snap := range snapshots {
		gain := 0
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			if snap.EloBeforeDoubles != nil && snap.EloAfterDoubles != nil {
				gain = int(*snap.EloAfterDoubles) - int(*snap.EloBeforeDoubles)
			}
		} else {
			if snap.EloBeforeSingles != nil && snap.EloAfterSingles != nil {
				gain = int(*snap.EloAfterSingles) - int(*snap.EloBeforeSingles)
			}
		}
		if gain > maxEloGain {
			maxEloGain = gain
			metrics.MostEloGainedPlayerID = snap.PlayerID
		}
	}

	divisionStats := make(map[string]*struct {
		matches int
		points  int
	})

	for _, m := range t.Matches {
		if m.Status == "finished" {
			metrics.TotalMatchesPlayed++

			divID := m.DivisionID
			if divID == "" {
				divID = "default"
			}
			if divisionStats[divID] == nil {
				divisionStats[divID] = &struct {
					matches int
					points  int
				}{}
			}
			divisionStats[divID].matches++

			if m.WinnerTeam != "" {
				scoreA := m.ScoreA()
				scoreB := m.ScoreB()
				if scoreA == 0 || scoreB == 0 {
					metrics.CleanSweeps++
				}
				if len(m.Sets) > 0 {
					if scoreA > 0 && scoreB > 0 && (scoreA-scoreB == 1 || scoreB-scoreA == 1) {
						metrics.DecidingSets++
					}
				}
			}
			for _, s := range m.Sets {
				metrics.TotalSetsPlayed++
				pts := s.ScoreA + s.ScoreB
				metrics.TotalPointsScored += pts
				divisionStats[divID].points += pts
			}
		}
	}

	if metrics.TotalMatchesPlayed > 0 {
		metrics.AveragePointsPerMatch = float64(metrics.TotalPointsScored) / float64(metrics.TotalMatchesPlayed)
		metrics.AverageSetsPerMatch = float64(metrics.TotalSetsPlayed) / float64(metrics.TotalMatchesPlayed)
	}

	for divID, stats := range divisionStats {
		dm := DivisionMetric{
			TotalMatchesPlayed: stats.matches,
		}
		if stats.matches > 0 {
			dm.AveragePointsPerMatch = float64(stats.points) / float64(stats.matches)
		}
		metrics.DivisionMetrics[divID] = dm
	}

	// Wait, to calculate BiggestEloUpset we would need the player's Elo before the match.
	// Since we only have the start/end event snapshots here and not per-match snapshots,
	// we will leave BiggestEloUpset empty for now unless we do a complex calculation.
	// For now, it will remain empty.

	return metrics
}
