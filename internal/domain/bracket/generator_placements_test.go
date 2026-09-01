package bracket_test

import (
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

// TestPlayerPlacements_FullBracketWithThirdPlace covers the standard case a
// tournament report cares about: champion, finalist, 3rd/4th place (from a
// played 3rd-place match), and the exit round for everyone eliminated
// earlier (quarterfinalists here).
func TestPlayerPlacements_FullBracketWithThirdPlace(t *testing.T) {
	p := make(map[string]*player.Player, 8)
	for i := 1; i <= 8; i++ {
		id := "p" + string(rune('0'+i))
		p[id] = &player.Player{ID: id, FirstName: id}
	}
	players := []*player.Player{p["p1"], p["p2"], p["p3"], p["p4"], p["p5"], p["p6"], p["p7"], p["p8"]}

	win := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	ev := &event.Event{
		HasThirdPlaceMatch: true,
		Matches: []event.Match{
			{ID: "qf-1", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p2"]}, Sets: win},
			{ID: "qf-2", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p3"]}, TeamB: []*player.Player{p["p4"]}, Sets: win},
			{ID: "qf-3", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p6"]}, Sets: win},
			{ID: "qf-4", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p7"]}, TeamB: []*player.Player{p["p8"]}, Sets: win},
			{ID: "sf-1", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p3"]}, Sets: win},
			{ID: "sf-2", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p7"]}, Sets: win},
			{ID: "final-1", Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p5"]}, Sets: win},
			{ID: "3rd-1", Stage: "3rd_place", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p3"]}, TeamB: []*player.Player{p["p7"]}, Sets: win},
		},
	}

	rounds := bracket.BuildBracketRoundsForTest(ev, "d1", players, 0)
	placements := bracket.PlayerPlacements(rounds)

	want := map[string]string{
		"p1": "Champion",
		"p5": "Finalist",
		"p3": "3rd Place",
		"p7": "4th Place",
		"p2": "Quarter-Finals",
		"p4": "Quarter-Finals",
		"p6": "Quarter-Finals",
		"p8": "Quarter-Finals",
	}
	for id, wantLabel := range want {
		if got := placements[id]; got != wantLabel {
			t.Errorf("placements[%s] = %q, want %q (full: %+v)", id, got, wantLabel, placements)
		}
	}
}

// TestPlayerPlacements_NoThirdPlaceMatchLabelsSemifinalLosers covers the
// common case where the event has no 3rd-place decider: both semifinal
// losers just get labeled with the semifinal round name, not "3rd Place".
func TestPlayerPlacements_NoThirdPlaceMatchLabelsSemifinalLosers(t *testing.T) {
	p := make(map[string]*player.Player, 4)
	for i := 1; i <= 4; i++ {
		id := "p" + string(rune('0'+i))
		p[id] = &player.Player{ID: id, FirstName: id}
	}
	players := []*player.Player{p["p1"], p["p2"], p["p3"], p["p4"]}

	win := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	ev := &event.Event{
		Matches: []event.Match{
			{ID: "sf-1", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p2"]}, Sets: win},
			{ID: "sf-2", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p3"]}, TeamB: []*player.Player{p["p4"]}, Sets: win},
			{ID: "final-1", Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p3"]}, Sets: win},
		},
	}

	rounds := bracket.BuildBracketRoundsForTest(ev, "d1", players, 0)
	placements := bracket.PlayerPlacements(rounds)

	if placements["p1"] != "Champion" {
		t.Errorf("expected p1 Champion, got %q", placements["p1"])
	}
	if placements["p3"] != "Finalist" {
		t.Errorf("expected p3 Finalist, got %q", placements["p3"])
	}
	if placements["p2"] != "Semi-Finals" || placements["p4"] != "Semi-Finals" {
		t.Errorf("expected both semifinal losers labeled Semi-Finals, got p2=%q p4=%q", placements["p2"], placements["p4"])
	}
}

// TestPlayerPlacements_UnplayedBracketReturnsEmpty covers the not-started
// case: no finished matches anywhere means no placements at all.
func TestPlayerPlacements_UnplayedBracketReturnsEmpty(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "A"}
	p2 := &player.Player{ID: "p2", FirstName: "B"}
	ev := &event.Event{}

	rounds := bracket.BuildBracketRoundsForTest(ev, "d1", []*player.Player{p1, p2}, 0)
	placements := bracket.PlayerPlacements(rounds)

	if len(placements) != 0 {
		t.Errorf("expected no placements for an unplayed bracket, got %+v", placements)
	}
}
