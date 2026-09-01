package pdf

import (
	"reflect"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

func TestGetTournamentStageHeader(t *testing.T) {
	cases := map[string]string{
		"group":        "Group Stage",
		"r32":          "Round of 32",
		"r16":          "Round of 16",
		"quarterfinal": "Quarter-Finals",
		"semifinal":    "Semi-Finals",
		"final":        "Final",
		"3rd_place":    "3RD_PLACE",
		"weird_stage":  "WEIRD_STAGE",
	}
	for stage, want := range cases {
		if got := getTournamentStageHeader(stage); got != want {
			t.Errorf("getTournamentStageHeader(%q) = %q, want %q", stage, got, want)
		}
	}
}

func TestTruncateStr(t *testing.T) {
	if got := truncateStr("hello", 10); got != "hello" {
		t.Errorf("expected unchanged short string, got %q", got)
	}
	if got := truncateStr("hello world", 5); got != "hello" {
		t.Errorf("expected truncation to 5 runes, got %q", got)
	}
}

func TestNextPow2(t *testing.T) {
	cases := map[int]int{
		0: 1,
		1: 1,
		2: 2,
		3: 4,
		4: 4,
		5: 8,
		8: 8,
		9: 16,
	}
	for n, want := range cases {
		if got := nextPow2(n); got != want {
			t.Errorf("nextPow2(%d) = %d, want %d", n, got, want)
		}
	}
}

func TestGetSeedingArrangement(t *testing.T) {
	cases := map[int][]int{
		1: {1},
		2: {1, 2},
		4: {1, 4, 3, 2},
		8: {1, 8, 5, 4, 3, 6, 7, 2},
	}
	for size, want := range cases {
		got := getSeedingArrangement(size)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("getSeedingArrangement(%d) = %v, want %v", size, got, want)
		}
	}
}

func TestGetSubMatchAlignments(t *testing.T) {
	cases := []struct {
		round      int
		teamFormat string
		wantA      string
		wantB      string
	}{
		{1, "", "A & B", "X & Y"}, // empty defaults to olympic
		{1, "olympic", "A & B", "X & Y"},
		{2, "olympic", "C", "Z"},
		{3, "olympic", "A", "X"},
		{4, "olympic", "B", "Y"},
		{5, "olympic", "C", "X"},
		{6, "olympic", "", ""}, // unknown round
		{1, "corbillon", "A", "X"},
		{2, "corbillon", "B", "Y"},
		{3, "corbillon", "C", "Z"},
		{4, "corbillon", "A", "Y"},
		{5, "corbillon", "B", "X"},
		{6, "corbillon", "", ""},
	}
	for _, c := range cases {
		a, b := getSubMatchAlignments(c.round, c.teamFormat)
		if a != c.wantA || b != c.wantB {
			t.Errorf("getSubMatchAlignments(%d, %q) = (%q, %q), want (%q, %q)", c.round, c.teamFormat, a, b, c.wantA, c.wantB)
		}
	}
}

func TestBuildPdfBracketRounds_NoPlayers(t *testing.T) {
	rounds := buildPdfBracketRounds(&event.Event{}, "", nil)
	if rounds != nil {
		t.Errorf("expected nil rounds for empty players, got %v", rounds)
	}
}

func TestBuildPdfBracketRounds_TwoPlayers(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
	p2 := &player.Player{ID: "p2", FirstName: "Bob", LastName: "B"}

	ev := &event.Event{
		StageRules: []event.StageRule{{Stage: "final", BestOf: 7}},
		Matches: []event.Match{
			{ID: "m1", Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		},
	}

	rounds := buildPdfBracketRounds(ev, "", []*player.Player{p1, p2})
	if len(rounds) != 2 {
		t.Fatalf("expected 2 rounds (Final + Champion), got %d", len(rounds))
	}
	if rounds[0].Name != "🏆 Final" {
		t.Errorf("expected first round to be Final, got %s", rounds[0].Name)
	}
	if rounds[1].Name != "Champion" {
		t.Errorf("expected second round to be Champion, got %s", rounds[1].Name)
	}
	if rounds[1].Matches[0].Player1.Player.ID != "p1" {
		t.Errorf("expected champion to be p1, got %v", rounds[1].Matches[0].Player1.Player.ID)
	}
	if rounds[0].Matches[0].BestOf != 7 {
		t.Errorf("expected bestOf 7 from stage rule, got %d", rounds[0].Matches[0].BestOf)
	}
}

func TestBuildPdfBracketRounds_FourPlayers(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "P1"}
	p2 := &player.Player{ID: "p2", FirstName: "P2"}
	p3 := &player.Player{ID: "p3", FirstName: "P3"}
	p4 := &player.Player{ID: "p4", FirstName: "P4"}

	// getSeedingArrangement(4) = [1,4,3,2], pairs by 1-index: (players[0],players[3]) and (players[2],players[1])
	// i.e. semifinal pairs: p1 vs p4, p3 vs p2
	ev := &event.Event{
		StageRules: []event.StageRule{
			{Stage: "semifinal", BestOf: 5},
			{Stage: "final", BestOf: 7},
		},
		Matches: []event.Match{
			{ID: "sf1", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p4}},
			{ID: "sf2", Stage: "semifinal", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p3}, TeamB: []*player.Player{p2}},
			{ID: "f1", Stage: "final", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		},
	}

	rounds := buildPdfBracketRounds(ev, "", []*player.Player{p1, p2, p3, p4})
	if len(rounds) != 3 {
		t.Fatalf("expected 3 rounds (Semi-Finals, Final, Champion), got %d", len(rounds))
	}
	if rounds[0].Name != "Semi-Finals" {
		t.Errorf("expected first round Semi-Finals, got %s", rounds[0].Name)
	}
	if rounds[2].Name != "Champion" {
		t.Fatalf("expected champion round, got %s", rounds[2].Name)
	}
	if rounds[2].Matches[0].Player1.Player.ID != "p2" {
		t.Errorf("expected champion p2 (winner of final), got %v", rounds[2].Matches[0].Player1.Player.ID)
	}
}

func TestBuildPdfBracketRounds_ThreePlayersWithBye(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "P1"}
	p2 := &player.Player{ID: "p2", FirstName: "P2"}
	p3 := &player.Player{ID: "p3", FirstName: "P3"}

	ev := &event.Event{}
	rounds := buildPdfBracketRounds(ev, "", []*player.Player{p1, p2, p3})
	if len(rounds) == 0 {
		t.Fatalf("expected at least one round for 3 players (bracket size 4 with a bye)")
	}
}

func TestBuildPdfBracketRounds_UnresolvedWinner(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "P1"}
	p2 := &player.Player{ID: "p2", FirstName: "P2"}

	// No matching finished match -> winner should remain unresolved and
	// propagate as an unresolved slot to the next round.
	ev := &event.Event{}
	rounds := buildPdfBracketRounds(ev, "", []*player.Player{p1, p2})
	if len(rounds) != 1 {
		t.Fatalf("expected only the Final round (no champion since winner unresolved), got %d rounds", len(rounds))
	}
}

// TestBuildPdfBracketRounds_RealMatchesOverrideMismatchedSeeding is a
// regression test for a production bug: a division ("Tercera Division")
// whose knockout was fully played and had a real champion showed BYE for
// every round past the first, because the players list handed in (from
// getITTFKnockoutSeeds recomputed off *current* group standings) didn't
// reproduce the pairing actually drawn/played. The hypothetical seeded
// arrangement paired players who never actually played each other, so every
// later round's "does a real match exist between these two players" lookup
// failed. Here players is deliberately in an order that would NOT reproduce
// the real first-round pairing via getSeedingArrangement, to prove the fix
// reads the real matches directly instead.
func TestBuildPdfBracketRounds_RealMatchesOverrideMismatchedSeeding(t *testing.T) {
	p := make(map[string]*player.Player, 8)
	for i := 1; i <= 8; i++ {
		id := "p" + string(rune('0'+i))
		p[id] = &player.Player{ID: id, FirstName: id}
	}
	// players is sorted p1..p8, but the real first-round pairing below is
	// p1-p2, p3-p4, p5-p6, p7-p8 -- not the 1v8/4v5/... seeded arrangement
	// that order would imply, and not reconstructable by re-deriving seeds.
	players := []*player.Player{p["p1"], p["p2"], p["p3"], p["p4"], p["p5"], p["p6"], p["p7"], p["p8"]}

	win := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	ev := &event.Event{
		Matches: []event.Match{
			{ID: "qf-1", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p2"]}, Sets: win},
			{ID: "qf-2", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p3"]}, TeamB: []*player.Player{p["p4"]}, Sets: win},
			{ID: "qf-3", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p6"]}, Sets: win},
			{ID: "qf-4", Stage: "quarterfinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p7"]}, TeamB: []*player.Player{p["p8"]}, Sets: win},
			{ID: "sf-1", Stage: "semifinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p3"]}, Sets: win},
			{ID: "sf-2", Stage: "semifinal", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p5"]}, TeamB: []*player.Player{p["p7"]}, Sets: win},
			{ID: "final-1", Stage: "final", Status: "finished", DivisionID: "d3", WinnerTeam: "A", TeamA: []*player.Player{p["p1"]}, TeamB: []*player.Player{p["p5"]}, Sets: win},
		},
	}

	rounds := buildPdfBracketRounds(ev, "d3", players)

	stageNames := make([]string, len(rounds))
	for i, r := range rounds {
		stageNames[i] = r.Name
	}
	// 8 players / 4 first-round matches (r16-1..4) is a quarterfinal-sized
	// field: QF (4 matches) -> SF (2 matches) -> Final -> Champion.
	if len(rounds) != 4 {
		t.Fatalf("expected 4 rounds (QF, SF, Final, Champion), got %d: %v", len(rounds), stageNames)
	}
	if rounds[0].Name != "Quarter-Finals" {
		t.Errorf("expected first round Quarter-Finals, got %s", rounds[0].Name)
	}
	champRound := rounds[len(rounds)-1]
	if champRound.Name != "Champion" {
		t.Fatalf("expected last round Champion, got %s", champRound.Name)
	}
	if champRound.Matches[0].Player1 == nil || champRound.Matches[0].Player1.Player == nil {
		t.Fatalf("expected a resolved champion, got BYE/unresolved: %+v", champRound.Matches[0])
	}
	if got := champRound.Matches[0].Player1.Player.ID; got != "p1" {
		t.Errorf("expected champion p1 (winner of every real match on its path), got %s", got)
	}
}

// TestGroupPdfSlotsByRealMatches_CrossedPairing is a regression test for a
// production bug: when the real next-round matches pair non-adjacent slots
// (e.g. slot 0's winner actually played slot 3's winner, not slot 1's), the
// bracket geometry code used to assume position 2*j/2*j+1 always fed pair j
// -- drawing connector lines between the wrong boxes and positioning the box
// containing slots 0+3's winners where slots 0+1's should have been.
// parentIndex must instead record the real structural relationship: which
// pair each input slot actually ended up in.
func TestGroupPdfSlotsByRealMatches_CrossedPairing(t *testing.T) {
	mk := func(id string) *pdfMatchSlot {
		return &pdfMatchSlot{Player: &player.Player{ID: id, FirstName: id}}
	}
	slots := []*pdfMatchSlot{mk("s0"), mk("s1"), mk("s2"), mk("s3")}

	// Real matches cross the slots: s0 played s3, and s2 played s1 -- not
	// the adjacent 0-1/2-3 pairing a position-based assumption would expect.
	nextMatches := []*event.Match{
		{TeamA: []*player.Player{slots[0].Player}, TeamB: []*player.Player{slots[3].Player}},
		{TeamA: []*player.Player{slots[2].Player}, TeamB: []*player.Player{slots[1].Player}},
	}

	pairs, parentIndex := groupPdfSlotsByRealMatches(slots, nextMatches)

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d: %+v", len(pairs), pairs)
	}
	if len(parentIndex) != 4 {
		t.Fatalf("expected parentIndex of length 4, got %v", parentIndex)
	}

	// s0 (idx 0) and s3 (idx 3) must map to the same pair index.
	if parentIndex[0] != parentIndex[3] {
		t.Errorf("expected s0 and s3 to map to the same pair, got parentIndex %v", parentIndex)
	}
	// s1 (idx 1) and s2 (idx 2) must map to the same (other) pair index.
	if parentIndex[1] != parentIndex[2] {
		t.Errorf("expected s1 and s2 to map to the same pair, got parentIndex %v", parentIndex)
	}
	// The two pairs must be distinct.
	if parentIndex[0] == parentIndex[1] {
		t.Errorf("expected s0/s3's pair and s1/s2's pair to be different, got parentIndex %v", parentIndex)
	}

	// The pair s0 maps into must actually contain s0 and s3.
	got := pairs[parentIndex[0]]
	gotIDs := map[string]bool{}
	if got.P1 != nil && got.P1.Player != nil {
		gotIDs[got.P1.Player.ID] = true
	}
	if got.P2 != nil && got.P2.Player != nil {
		gotIDs[got.P2.Player.ID] = true
	}
	if !gotIDs["s0"] || !gotIDs["s3"] {
		t.Errorf("expected the pair at index %d to contain s0 and s3, got %+v", parentIndex[0], got)
	}
}

// TestGroupPdfSlotsByRealMatches_LeftoverSlotsGetOwnParentIndex covers the
// no-real-match-yet fallback path, where slots are paired positionally and
// each one (including an odd trailing solo) still gets a valid parentIndex
// pointing at its own pair.
func TestGroupPdfSlotsByRealMatches_LeftoverSlotsGetOwnParentIndex(t *testing.T) {
	mk := func(id string) *pdfMatchSlot {
		return &pdfMatchSlot{Player: &player.Player{ID: id, FirstName: id}}
	}
	slots := []*pdfMatchSlot{mk("s0"), mk("s1"), mk("s2")}

	pairs, parentIndex := groupPdfSlotsByRealMatches(slots, nil)

	if len(pairs) != 2 {
		t.Fatalf("expected 2 pairs (one full, one solo leftover), got %d: %+v", len(pairs), pairs)
	}
	if parentIndex[0] != 0 || parentIndex[1] != 0 {
		t.Errorf("expected s0 and s1 to pair up as leftover pair 0, got parentIndex %v", parentIndex)
	}
	if parentIndex[2] != 1 {
		t.Errorf("expected the solo trailing slot s2 to get its own pair 1, got parentIndex %v", parentIndex)
	}
	if pairs[1].P2 != nil {
		t.Errorf("expected the solo leftover pair to have a nil P2, got %+v", pairs[1])
	}
}

// TestBuildPdfBracketRounds_FinalNextIndexPointsAtChampion is a regression
// test for a production bug where the Final round's own date/table/score
// annotation and its connector line into the Champion box never rendered --
// the drawing code needs a Final round with a NextIndex of [0] so the
// generic connector-line/annotation logic (which used to special-case and
// skip a next round named "Champion" entirely) treats it like any other
// round.
func TestBuildPdfBracketRounds_FinalNextIndexPointsAtChampion(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
	p2 := &player.Player{ID: "p2", FirstName: "Bob", LastName: "B"}

	ev := &event.Event{
		StageRules: []event.StageRule{{Stage: "final", BestOf: 7}},
		Matches: []event.Match{
			{ID: "m1", Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		},
	}

	rounds := buildPdfBracketRounds(ev, "", []*player.Player{p1, p2})
	if len(rounds) != 2 {
		t.Fatalf("expected 2 rounds (Final + Champion), got %d", len(rounds))
	}
	finalRound := rounds[0]
	if len(finalRound.NextIndex) != 1 || finalRound.NextIndex[0] != 0 {
		t.Errorf("expected the Final round's NextIndex to be [0] (pointing at the Champion box), got %v", finalRound.NextIndex)
	}
	if finalRound.Matches[0].Match == nil {
		t.Fatalf("expected the Final round to carry its own finished match record")
	}
}

func TestGetITTFKnockoutSeeds_NoGroups(t *testing.T) {
	ev := &event.Event{}
	out := getITTFKnockoutSeeds(ev, "div1", "Division 1", nil, nil)
	if out != nil {
		t.Errorf("expected nil for no groups, got %v", out)
	}
}

func TestGetITTFKnockoutSeeds_TwoGroupsPassTwo(t *testing.T) {
	pA1 := &player.Player{ID: "pA1", FirstName: "A1", SinglesElo: 1600}
	pA2 := &player.Player{ID: "pA2", FirstName: "A2", SinglesElo: 1500}
	pA3 := &player.Player{ID: "pA3", FirstName: "A3", SinglesElo: 1400}
	pB1 := &player.Player{ID: "pB1", FirstName: "B1", SinglesElo: 1550}
	pB2 := &player.Player{ID: "pB2", FirstName: "B2", SinglesElo: 1450}
	pB3 := &player.Player{ID: "pB3", FirstName: "B3", SinglesElo: 1350}

	win3 := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}

	groupA := &event.Group{ID: "gA", Name: "Group A", Players: []*player.Player{pA1, pA2, pA3}}
	groupB := &event.Group{ID: "gB", Name: "Group B", Players: []*player.Player{pB1, pB2, pB3}}

	ev := &event.Event{
		GroupPassCount: 2,
		Matches: []event.Match{
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pA1}, TeamB: []*player.Player{pA2}, Sets: win3},
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pA1}, TeamB: []*player.Player{pA3}, Sets: win3},
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pA2}, TeamB: []*player.Player{pA3}, Sets: win3},
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pB1}, TeamB: []*player.Player{pB2}, Sets: win3},
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pB1}, TeamB: []*player.Player{pB3}, Sets: win3},
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pB2}, TeamB: []*player.Player{pB3}, Sets: win3},
		},
	}

	out := getITTFKnockoutSeeds(ev, "", "Open", nil, []*event.Group{groupA, groupB})
	if len(out) != 4 {
		t.Fatalf("expected 4 advancing players, got %d: %v", len(out), out)
	}
	wantIDs := []string{"pA1", "pB1", "pA2", "pB2"}
	for i, w := range wantIDs {
		if out[i].ID != w {
			t.Errorf("out[%d] = %s, want %s (full: %v)", i, out[i].ID, w, out)
		}
	}
}

func TestGetITTFKnockoutSeeds_PassCountExceedsGroupSize(t *testing.T) {
	pA1 := &player.Player{ID: "pA1", FirstName: "A1", SinglesElo: 1600}
	pA2 := &player.Player{ID: "pA2", FirstName: "A2", SinglesElo: 1500}
	pB1 := &player.Player{ID: "pB1", FirstName: "B1", SinglesElo: 1550}
	pB2 := &player.Player{ID: "pB2", FirstName: "B2", SinglesElo: 1450}

	win3 := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}

	groupA := &event.Group{ID: "gA", Name: "Group A", Players: []*player.Player{pA1, pA2}}
	groupB := &event.Group{ID: "gB", Name: "Group B", Players: []*player.Player{pB1, pB2}}

	ev := &event.Event{
		GroupPassCount: 3,
		Matches: []event.Match{
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pA1}, TeamB: []*player.Player{pA2}, Sets: win3},
			{Stage: "group", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{pB1}, TeamB: []*player.Player{pB2}, Sets: win3},
		},
	}

	out := getITTFKnockoutSeeds(ev, "div1", "Division 1", nil, []*event.Group{groupA, groupB})
	// passCount requested is 3 but each group only has 2 players, so take is
	// capped at 2 per group -> still 4 total advancing.
	if len(out) != 4 {
		t.Fatalf("expected 4 advancing players (capped), got %d: %v", len(out), out)
	}
}
