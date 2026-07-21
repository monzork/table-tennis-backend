package event

import (
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func winSets(scoreA, scoreB int) []MatchSet {
	sets := []MatchSet{}
	for scoreA > 0 || scoreB > 0 {
		if scoreA > 0 {
			sets = append(sets, MatchSet{Number: len(sets) + 1, ScoreA: 11, ScoreB: 5})
			scoreA--
		} else {
			sets = append(sets, MatchSet{Number: len(sets) + 1, ScoreA: 5, ScoreB: 11})
			scoreB--
		}
	}
	return sets
}

func TestBuildStandings_Basic(t *testing.T) {
	p1 := &player.Player{ID: "p1", SinglesElo: 1500}
	p2 := &player.Player{ID: "p2", SinglesElo: 1400}
	p3 := &player.Player{ID: "p3", SinglesElo: 1300}

	matches := []Match{
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, Sets: winSets(3, 0)},
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}, Sets: winSets(3, 1)},
		{Status: "finished", Stage: "group", WinnerTeam: "B", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}, Sets: winSets(1, 3)},
		// Non-group / unfinished matches should be ignored
		{Status: "in_progress", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}},
		{Status: "finished", Stage: "final", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}},
	}

	standings := BuildStandings([]*player.Player{p1, p2, p3}, matches)
	if len(standings) != 3 {
		t.Fatalf("Expected 3 standings entries, got %d", len(standings))
	}

	if standings[0].Player.ID != "p1" {
		t.Errorf("Expected p1 to rank first (2 wins), got %s", standings[0].Player.ID)
	}
	if standings[0].Rank != 1 {
		t.Errorf("Expected rank 1, got %d", standings[0].Rank)
	}
	if standings[0].Wins != 2 {
		t.Errorf("Expected 2 wins for p1, got %d", standings[0].Wins)
	}
	if standings[0].WinPercentage != "100" {
		t.Errorf("Expected 100 win pct for p1, got %s", standings[0].WinPercentage)
	}

	// p2 and p3 each have 1 win, tied - need H2H resolution; p3 beat p2 head-to-head.
	if standings[1].Player.ID != "p3" {
		t.Errorf("Expected p3 to rank second via H2H tiebreak, got %s", standings[1].Player.ID)
	}
	if standings[2].Player.ID != "p2" {
		t.Errorf("Expected p2 to rank third, got %s", standings[2].Player.ID)
	}
}

func TestBuildStandings_NoMatchesPlayed(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	standings := BuildStandings([]*player.Player{p1}, nil)
	if len(standings) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(standings))
	}
	if standings[0].Played != 0 || standings[0].WinPercentage != "0" {
		t.Errorf("Expected 0 played and 0%% win rate, got played=%d pct=%s", standings[0].Played, standings[0].WinPercentage)
	}
}

func TestResolveITTFTies_SingleOrEmpty(t *testing.T) {
	if got := ResolveITTFTies(nil, nil, 0); got != nil {
		t.Errorf("Expected nil for empty input")
	}

	p1 := &player.Player{ID: "p1"}
	single := []*PlayerStanding{{Player: p1}}
	got := ResolveITTFTies(single, nil, 0)
	if len(got) != 1 || got[0].Player.ID != "p1" {
		t.Errorf("Expected single-element passthrough")
	}
}

func TestResolveITTFTies_SetRatioTiebreak(t *testing.T) {
	// Two H2H legs between the same pair (aggregate) give both players 1 win
	// each (tied on criterion 1) but different set ratios, so criterion 2
	// (set ratio) must break the tie.
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}

	matches := []Match{
		// p1 wins 3-1 -> p1 setsWon 3/1, p2 setsWon 1/3
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, Sets: winSets(3, 1)},
		// p2 wins 3-2 -> p2 setsWon 3/2, p1 setsWon 2/3
		{Status: "finished", Stage: "group", WinnerTeam: "B", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, Sets: winSets(2, 3)},
		// noise entries exercising matchesBetween's filter branches
		{Status: "in_progress", Stage: "group", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		{Status: "finished", Stage: "final", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		{Status: "finished", Stage: "group", TeamA: nil, TeamB: []*player.Player{p2}},
	}

	tied := []*PlayerStanding{
		{Player: p1, Wins: 1},
		{Player: p2, Wins: 1},
	}

	// p1 total sets: 3+2=5 won, 1+3=4 lost -> ratio 1.25
	// p2 total sets: 1+3=4 won, 3+2=5 lost -> ratio 0.8
	result := ResolveITTFTies(tied, matches, 0)
	if len(result) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(result))
	}
	if result[0].Player.ID != "p1" {
		t.Errorf("Expected p1 first due to higher set ratio, got %s", result[0].Player.ID)
	}
}

func TestResolveITTFTies_WinsDiffRecursion(t *testing.T) {
	p1 := &player.Player{ID: "p1", SinglesElo: 1500}
	p2 := &player.Player{ID: "p2", SinglesElo: 1400}

	matches := []Match{
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, Sets: winSets(3, 0)},
	}

	tied := []*PlayerStanding{
		{Player: p1, Wins: 1},
		{Player: p2, Wins: 1},
	}

	result := ResolveITTFTies(tied, matches, 0)
	if result[0].Player.ID != "p1" {
		t.Errorf("Expected p1 to win the H2H tiebreak, got %s", result[0].Player.ID)
	}
}

func TestResolveITTFTies_PointRatioTiebreak(t *testing.T) {
	// A 3-way round-robin triangle where every player wins one and loses one
	// (tied on wins) and each match is a symmetric 3-0 sweep (tied on set
	// ratio, all 1.0), but the point margins per set are rigged differently
	// so the aggregate point ratios diverge, forcing criterion 3.
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	p3 := &player.Player{ID: "p3"}

	closeSets := func(a, b int) []MatchSet {
		return []MatchSet{{Number: 1, ScoreA: a, ScoreB: b}, {Number: 2, ScoreA: a, ScoreB: b}, {Number: 3, ScoreA: a, ScoreB: b}}
	}

	matches := []Match{
		// p1 beats p2 3-0, tight sets 11-9
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}, Sets: closeSets(11, 9)},
		// p2 beats p3 3-0, blowout sets 11-0
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}, Sets: closeSets(11, 0)},
		// p3 beats p1 3-0, mid-margin sets 11-5
		{Status: "finished", Stage: "group", WinnerTeam: "A", TeamA: []*player.Player{p3}, TeamB: []*player.Player{p1}, Sets: closeSets(11, 5)},
	}

	tied := []*PlayerStanding{
		{Player: p1, Wins: 1},
		{Player: p2, Wins: 1},
		{Player: p3, Wins: 1},
	}

	// p1: ptsWon 33+15=48, ptsLost 27+33=60 -> ratio 0.8
	// p2: ptsWon 27+33=60, ptsLost 33+0=33   -> ratio ~1.818
	// p3: ptsWon 0+33=33,  ptsLost 33+15=48  -> ratio ~0.6875
	result := ResolveITTFTies(tied, matches, 0)
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result[0].Player.ID != "p2" {
		t.Errorf("Expected p2 first (highest point ratio), got %s", result[0].Player.ID)
	}
	if result[1].Player.ID != "p1" {
		t.Errorf("Expected p1 second, got %s", result[1].Player.ID)
	}
	if result[2].Player.ID != "p3" {
		t.Errorf("Expected p3 third (lowest point ratio), got %s", result[2].Player.ID)
	}
}

func TestResolveITTFTies_CompletelyTiedFallsBackToElo(t *testing.T) {
	p1 := &player.Player{ID: "p1", SinglesElo: 1300}
	p2 := &player.Player{ID: "p2", SinglesElo: 1500}
	p3 := &player.Player{ID: "p3", SinglesElo: 1400}

	tied := []*PlayerStanding{
		{Player: p1, Wins: 0},
		{Player: p2, Wins: 0},
		{Player: p3, Wins: 0},
	}

	result := ResolveITTFTies(tied, nil, 0)
	if len(result) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(result))
	}
	if result[0].Player.ID != "p2" || result[1].Player.ID != "p3" || result[2].Player.ID != "p1" {
		t.Errorf("Expected Elo-descending fallback order p2,p3,p1, got %s,%s,%s", result[0].Player.ID, result[1].Player.ID, result[2].Player.ID)
	}
}
