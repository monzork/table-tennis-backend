package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestMetricsCalculator_Calculate(t *testing.T) {
	calc := NewMetricsCalculator()

	// Helper functions to create pointers
	int16Ptr := func(i int16) *int16 { return &i }

	// Create a dummy event with matches
	event := &Event{
		ID:   "tourney-1",
		Type: "singles",
		Matches: []Match{
			{
				Status:     "finished",
				DivisionID: "div-1",
				WinnerTeam: "A",
				Sets: []MatchSet{
					{Number: 1, ScoreA: 11, ScoreB: 5},
					{Number: 2, ScoreA: 11, ScoreB: 7},
					{Number: 3, ScoreA: 12, ScoreB: 10},
				},
				TeamA: []*player.Player{{ID: "p1"}},
				TeamB: []*player.Player{{ID: "p2"}},
			},
			{
				Status:     "finished",
				DivisionID: "div-1",
				WinnerTeam: "A",
				Sets: []MatchSet{
					{Number: 1, ScoreA: 11, ScoreB: 0},
					{Number: 2, ScoreA: 11, ScoreB: 0},
					{Number: 3, ScoreA: 11, ScoreB: 0}, // Clean sweep simulated by big margin/0
				},
				TeamA: []*player.Player{{ID: "p1"}},
				TeamB: []*player.Player{{ID: "p3"}},
			},
			{
				Status:     "in_progress",
				DivisionID: "div-1",
			},
		},
	}

	// Create dummy snapshots for Elo
	snapshots := []ParticipantSnapshot{
		{
			PlayerID:         "p1",
			EloBeforeSingles: int16Ptr(1500),
			EloAfterSingles:  int16Ptr(1520),
		},
		{
			PlayerID:         "p2",
			EloBeforeSingles: int16Ptr(1600),
			EloAfterSingles:  int16Ptr(1580),
		},
		{
			PlayerID:         "p3",
			EloBeforeSingles: int16Ptr(1400),
			EloAfterSingles:  int16Ptr(1400),
		},
	}

	metrics := calc.Calculate(event, snapshots)

	if metrics.SchemaVersion != 1 {
		t.Errorf("Expected SchemaVersion 1, got %d", metrics.SchemaVersion)
	}

	// Average Elo at Start = (1500 + 1600 + 1400) / 3 = 1500
	if metrics.AverageEloAtStart != 1500.0 {
		t.Errorf("Expected AverageEloAtStart 1500, got %v", metrics.AverageEloAtStart)
	}

	// Most Elo Gained = p1 (+20)
	if metrics.MostEloGainedPlayerID != "p1" {
		t.Errorf("Expected MostEloGainedPlayerID 'p1', got '%s'", metrics.MostEloGainedPlayerID)
	}

	// Total Matches Played (only finished)
	if metrics.TotalMatchesPlayed != 2 {
		t.Errorf("Expected 2 matches played, got %d", metrics.TotalMatchesPlayed)
	}

	// Total Sets: 3 + 3 = 6
	if metrics.TotalSetsPlayed != 6 {
		t.Errorf("Expected 6 sets played, got %d", metrics.TotalSetsPlayed)
	}

	// Total Points: (11+5 + 11+7 + 12+10) + (11+0 + 11+0 + 11+0) = 56 + 33 = 89
	if metrics.TotalPointsScored != 89 {
		t.Errorf("Expected 89 points, got %d", metrics.TotalPointsScored)
	}

	// Average Points Per Match: 89 / 2 = 44.5
	if metrics.AveragePointsPerMatch != 44.5 {
		t.Errorf("Expected 44.5 AveragePointsPerMatch, got %v", metrics.AveragePointsPerMatch)
	}

	// Average Sets Per Match: 6 / 2 = 3
	if metrics.AverageSetsPerMatch != 3.0 {
		t.Errorf("Expected 3.0 AverageSetsPerMatch, got %v", metrics.AverageSetsPerMatch)
	}

	// Clean Sweeps: (11-0, 11-0, 11-0 match -> has scoreB = 0)
	if metrics.CleanSweeps != 2 {
		t.Errorf("Expected 2 CleanSweep, got %d", metrics.CleanSweeps)
	}

	// Division Metrics check
	if len(metrics.DivisionMetrics) != 1 {
		t.Fatalf("Expected 1 DivisionMetric, got %d", len(metrics.DivisionMetrics))
	}
	divMetric := metrics.DivisionMetrics["div-1"]
	if divMetric.TotalMatchesPlayed != 2 {
		t.Errorf("Expected div metric to have 2 matches, got %d", divMetric.TotalMatchesPlayed)
	}
	if divMetric.AveragePointsPerMatch != 44.5 {
		t.Errorf("Expected div metric average points 44.5, got %v", divMetric.AveragePointsPerMatch)
	}
}

// A forfeit is persisted with no sets at all (see MatchRepository.UpdateScore),
// so it still counts toward TotalMatchesPlayed but contributes no sets/points/
// clean-sweeps -- nothing here needs to special-case IsForfeit. This also
// guards the len(m.Sets) > 0 checks below: a finished match with zero real
// sets (only possible for a forfeit) must not be misread as a 0-0 "clean sweep".
func TestMetricsCalculator_ForfeitWithNoPersistedSets(t *testing.T) {
	calc := NewMetricsCalculator()

	event := &Event{
		ID:   "tourney-2",
		Type: "singles",
		Matches: []Match{
			{
				Status:     "finished",
				DivisionID: "div-1",
				WinnerTeam: "A",
				IsForfeit:  true,
				TeamA:      []*player.Player{{ID: "p1"}},
				TeamB:      []*player.Player{{ID: "p2"}},
			},
		},
	}

	metrics := calc.Calculate(event, nil)

	if metrics.TotalMatchesPlayed != 1 {
		t.Errorf("expected the forfeit to still count as a match played, got %d", metrics.TotalMatchesPlayed)
	}
	if metrics.TotalSetsPlayed != 0 || metrics.TotalPointsScored != 0 || metrics.CleanSweeps != 0 {
		t.Errorf("expected no sets/points/clean-sweeps with none persisted, got sets=%d points=%d cleanSweeps=%d",
			metrics.TotalSetsPlayed, metrics.TotalPointsScored, metrics.CleanSweeps)
	}
}
