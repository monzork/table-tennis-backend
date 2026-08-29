package bracket_test

import (
	"fmt"
	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestBracketGenerator_LosersGroupPassCount(t *testing.T) {
	// Setup players
	players := []*player.Player{
		{ID: "p1", FirstName: "Player", LastName: "1", SinglesElo: 1200, Gender: "M"},
		{ID: "p2", FirstName: "Player", LastName: "2", SinglesElo: 1100, Gender: "M"},
		{ID: "p3", FirstName: "Player", LastName: "3", SinglesElo: 1000, Gender: "M"},
		{ID: "p4", FirstName: "Player", LastName: "4", SinglesElo: 900, Gender: "M"},
		{ID: "p5", FirstName: "Player", LastName: "5", SinglesElo: 800, Gender: "M"},
		{ID: "p6", FirstName: "Player", LastName: "6", SinglesElo: 700, Gender: "M"},
		{ID: "p7", FirstName: "Player", LastName: "7", SinglesElo: 600, Gender: "M"},
		{ID: "p8", FirstName: "Player", LastName: "8", SinglesElo: 500, Gender: "M"},
	}

	tourney := &event.Event{
		ID:                    "t1",
		Name:                  "Test Tournament",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 2,
		GroupPassCount:        2,
		LosersGroupPassCount:  1,
		Participants:          players,
	}

	group1 := event.Group{
		ID:      "g1",
		EventID: "t1",
		Name:    "Open - Group A",
		Players: []*player.Player{players[0], players[1], players[2], players[3]},
		Matches: []event.Match{},
	}

	group1.Matches = append(group1.Matches, createFinishedMatch("m1", players[0], players[1], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m2", players[0], players[2], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m3", players[0], players[3], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m4", players[1], players[2], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m5", players[1], players[3], "A"))
	group1.Matches = append(group1.Matches, createFinishedMatch("m6", players[2], players[3], "A"))

	group2 := event.Group{
		ID:      "g2",
		EventID: "t1",
		Name:    "Open - Group B",
		Players: []*player.Player{players[4], players[5], players[6], players[7]},
		Matches: []event.Match{},
	}
	group2.Matches = append(group2.Matches, createFinishedMatch("m7", players[4], players[5], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m8", players[4], players[6], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m9", players[4], players[7], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m10", players[5], players[6], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m11", players[5], players[7], "A"))
	group2.Matches = append(group2.Matches, createFinishedMatch("m12", players[6], players[7], "A"))

	tourney.Groups = []event.Group{group1, group2}
	tourney.Matches = append(tourney.Matches, group1.Matches...)
	tourney.Matches = append(tourney.Matches, group2.Matches...)

	divs := []*division.Division{
		{ID: "div1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, map[string]string{})
	views := br.Divisions

	if len(views) != 1 {
		t.Fatalf("Expected 1 division view, got %d", len(views))
	}

	divView := views[0]
	if len(divView.KnockoutBrackets) != 2 {
		t.Fatalf("Expected 2 brackets (tiers), got %d", len(divView.KnockoutBrackets))
	}

	tier1 := divView.KnockoutBrackets[0]
	tier2 := divView.KnockoutBrackets[1]

	if len(tier1.Rounds[0].Matches) != 2 {
		t.Errorf("Tier 1 should have 2 matches in round 1, got %d", len(tier1.Rounds[0].Matches))
	}

	if len(tier2.Rounds[0].Matches) != 1 {
		t.Errorf("Tier 2 should have 1 match in round 1, got %d", len(tier2.Rounds[0].Matches))
	}

	tourney.LosersGroupPassCount = 2

	br = bracket.BuildBracket(tourney, divs, map[string]string{})
	views = br.Divisions
	divView = views[0]
	tier2 = divView.KnockoutBrackets[1]

	if len(tier2.Rounds[0].Matches) != 2 {
		t.Errorf("With override, Tier 2 should have 2 matches in round 1, got %d (Rounds len: %d, tierAdvancing len: %d)", len(tier2.Rounds[0].Matches), len(tier2.Rounds), len(tier2.Advancing))
	}
}

// TestBracketGenerator_SameGroupSeparationHoldsForHigherPassCounts reproduces
// a real production bug: with GroupPassCount >= 3, the seeding used to only
// separate advancing players into 2 bracket halves, which by the pigeonhole
// principle can't keep 3+ same-group players apart — group-mates who already
// played each other in the group stage could land on the bracket's fixed
// first-round-adjacent seed pair (e.g. seeds 6 & 11 in a 16-bracket always
// face each other in round 1). The fix spreads a group's players across as
// many regions (quarters, eighths, ...) as its pass count needs.
func TestBracketGenerator_SameGroupSeparationHoldsForHigherPassCounts(t *testing.T) {
	// 5 groups of 4 players each, ranked p0 > p1 > p2 > p3 within every group
	// (p0 beats everyone, p1 beats p2/p3, p2 beats p3) so group standings are
	// unambiguous.
	const numGroups = 5
	const groupSize = 4

	var allPlayers []*player.Player
	var groups []event.Group
	var allMatches []event.Match
	for gi := 0; gi < numGroups; gi++ {
		groupName := string(rune('A' + gi))
		var gp []*player.Player
		for pi := 0; pi < groupSize; pi++ {
			gp = append(gp, &player.Player{
				ID:         fmt.Sprintf("g%d-p%d", gi, pi),
				FirstName:  fmt.Sprintf("G%dP%d", gi, pi),
				LastName:   "Test",
				SinglesElo: int16(2000 - gi*100 - pi*10),
				Gender:     "M",
			})
		}
		allPlayers = append(allPlayers, gp...)

		g := event.Group{
			ID:      fmt.Sprintf("g%d", gi),
			EventID: "t1",
			Name:    groupName,
			Players: gp,
		}
		for i := 0; i < groupSize; i++ {
			for j := i + 1; j < groupSize; j++ {
				mID := fmt.Sprintf("g%d-m%d-%d", gi, i, j)
				g.Matches = append(g.Matches, createFinishedMatch(mID, gp[i], gp[j], "A"))
			}
		}
		groups = append(groups, g)
		allMatches = append(allMatches, g.Matches...)
	}

	divs := []*division.Division{
		{ID: "div1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	for _, passCount := range []int{2, 3, 4} {
		t.Run(fmt.Sprintf("passCount=%d", passCount), func(t *testing.T) {
			tourney := &event.Event{
				ID:                    "t1",
				Name:                  "Test Tournament",
				Type:                  "singles",
				Format:                "groups_elimination",
				EventCategory:         "open",
				KnockoutBracketsCount: 1,
				GroupPassCount:        passCount,
				Participants:          allPlayers,
				Groups:                groups,
				Matches:               allMatches,
			}

			br := bracket.BuildBracket(tourney, divs, map[string]string{})
			if len(br.Divisions) != 1 {
				t.Fatalf("expected 1 division view, got %d", len(br.Divisions))
			}
			dv := br.Divisions[0]
			if len(dv.KnockoutBrackets) != 1 {
				t.Fatalf("expected 1 bracket tier, got %d", len(dv.KnockoutBrackets))
			}

			advancing := dv.KnockoutBrackets[0].Advancing
			if len(advancing) != numGroups*passCount {
				t.Fatalf("expected %d advancing players, got %d", numGroups*passCount, len(advancing))
			}

			if err := bracket.ValidateSameGroupSeparation(dv.Groups, advancing); err != nil {
				t.Errorf("same-group separation violated: %v", err)
			}
		})
	}
}

func TestBracketGenerator_SkipDivisionSplit(t *testing.T) {
	// Two players from clearly different Elo bands, hand-picked into one
	// "Open" event that must NOT be fragmented back into per-division
	// sub-brackets by the global division list.
	players := []*player.Player{
		{ID: "p1", FirstName: "Player", LastName: "1", SinglesElo: 2000, Gender: "M"},
		{ID: "p2", FirstName: "Player", LastName: "2", SinglesElo: 500, Gender: "M"},
	}

	div1MaxElo := int16(1200)
	divs := []*division.Division{
		{ID: "div1", Name: "Division 1", Category: "both", MinElo: 0, MaxElo: &div1MaxElo},
		{ID: "div2", Name: "Division 2", Category: "both", MinElo: 1201, MaxElo: nil},
	}

	tourney := &event.Event{
		ID:                "t1",
		Name:              "Open Tournament",
		Type:              "singles",
		Format:            "groups_elimination",
		EventCategory:     "open",
		GroupPassCount:    2,
		Participants:      players,
		SkipDivisionSplit: true,
	}

	br := bracket.BuildBracket(tourney, divs, map[string]string{})
	if len(br.Divisions) != 1 {
		t.Fatalf("expected a single flat bracket when SkipDivisionSplit is set, got %d", len(br.Divisions))
	}
	if len(br.Divisions[0].Players) != 2 {
		t.Fatalf("expected both players in the single bracket, got %d", len(br.Divisions[0].Players))
	}

	// Sanity check: without the flag, the same roster fragments into two
	// division buckets — proving the flag is what changes the outcome.
	tourney.SkipDivisionSplit = false
	br = bracket.BuildBracket(tourney, divs, map[string]string{})
	if len(br.Divisions) != 2 {
		t.Fatalf("expected the roster to split by division without the flag, got %d", len(br.Divisions))
	}
}

func TestBracketGenerator_GenderScopedDivisions(t *testing.T) {
	// Women's Elo is now shifted ~700 below men's on a shared scale, so
	// gender-specific division bands must only apply to the matching
	// gender's event -- otherwise a women's event would see none of its
	// players fall into a men's-scaled Elo band. Gender-specific divisions
	// only take effect for events that opt in via UseGenderDivisions --
	// otherwise every pre-existing event (which never sets this field) would
	// suddenly start seeing new gender-specific division rows.
	menDiv1200 := int16(1600)
	womenDiv1200 := int16(900)
	divs := []*division.Division{
		{ID: "men1", Name: "Men's 1st", Category: "both", Gender: "M", MinElo: menDiv1200, MaxElo: nil},
		{ID: "women1", Name: "Women's 1st", Category: "both", Gender: "F", MinElo: womenDiv1200, MaxElo: nil},
	}

	womenPlayers := []*player.Player{
		{ID: "p1", FirstName: "Player", LastName: "1", SinglesElo: 1000, Gender: "F"},
		{ID: "p2", FirstName: "Player", LastName: "2", SinglesElo: 950, Gender: "F"},
	}
	womensEvent := &event.Event{
		ID:                 "t1",
		Name:               "Women's Singles",
		Type:               "singles",
		Format:             "groups_elimination",
		EventCategory:      "women",
		GroupPassCount:     2,
		Participants:       womenPlayers,
		UseGenderDivisions: true,
	}

	br := bracket.BuildBracket(womensEvent, divs, map[string]string{})
	if len(br.Divisions) != 1 || br.Divisions[0].Name != "Women's 1st" {
		t.Fatalf("expected women's event to only use the women's division band, got %+v", br.Divisions)
	}

	menPlayers := []*player.Player{
		{ID: "p3", FirstName: "Player", LastName: "3", SinglesElo: 1700, Gender: "M"},
		{ID: "p4", FirstName: "Player", LastName: "4", SinglesElo: 1650, Gender: "M"},
	}
	mensEvent := &event.Event{
		ID:                 "t2",
		Name:               "Men's Singles",
		Type:               "singles",
		Format:             "groups_elimination",
		EventCategory:      "men",
		GroupPassCount:     2,
		Participants:       menPlayers,
		UseGenderDivisions: true,
	}

	br = bracket.BuildBracket(mensEvent, divs, map[string]string{})
	if len(br.Divisions) != 1 || br.Divisions[0].Name != "Men's 1st" {
		t.Fatalf("expected men's event to only use the men's division band, got %+v", br.Divisions)
	}
}

func TestBracketGenerator_GenderDivisionsHiddenWithoutOptIn(t *testing.T) {
	// Same gender-specific bands as above, but neither event sets
	// UseGenderDivisions -- this is every event that existed before the flag
	// was introduced, and it must see none of these bands (falling back to
	// "Unclassified" here since the fixture has no gender-agnostic "both"
	// bands at all), so pre-existing tournaments are never affected by
	// newly-added gender-specific division rows.
	divs := []*division.Division{
		{ID: "men1", Name: "Men's 1st", Category: "both", Gender: "M", MinElo: 1600, MaxElo: nil},
		{ID: "women1", Name: "Women's 1st", Category: "both", Gender: "F", MinElo: 900, MaxElo: nil},
	}

	womenPlayers := []*player.Player{
		{ID: "p1", FirstName: "Player", LastName: "1", SinglesElo: 1000, Gender: "F"},
		{ID: "p2", FirstName: "Player", LastName: "2", SinglesElo: 950, Gender: "F"},
	}
	womensEvent := &event.Event{
		ID:             "t3",
		Name:           "Women's Singles",
		Type:           "singles",
		Format:         "groups_elimination",
		EventCategory:  "women",
		GroupPassCount: 2,
		Participants:   womenPlayers,
	}

	br := bracket.BuildBracket(womensEvent, divs, map[string]string{})
	if len(br.Divisions) != 1 || !br.Divisions[0].IsUnclassified {
		t.Fatalf("expected women's event without UseGenderDivisions to ignore gender-specific bands, got %+v", br.Divisions)
	}
}

func TestBracketGenerator_GenderDivisionsOptedInButMixedCategory(t *testing.T) {
	// A "mixed"/"open" event has no single gender for gendered Elo bands to
	// be scaled against, so even with UseGenderDivisions=true it must never
	// see gender-specific divisions.
	divs := []*division.Division{
		{ID: "men1", Name: "Men's 1st", Category: "both", Gender: "M", MinElo: 0, MaxElo: nil},
	}
	players := []*player.Player{
		{ID: "p1", FirstName: "Player", LastName: "1", SinglesElo: 1000, Gender: "M"},
		{ID: "p2", FirstName: "Player", LastName: "2", SinglesElo: 950, Gender: "F"},
	}
	mixedEvent := &event.Event{
		ID:                 "t4",
		Name:               "Open Singles",
		Type:               "singles",
		Format:             "groups_elimination",
		EventCategory:      "open",
		GroupPassCount:     2,
		Participants:       players,
		UseGenderDivisions: true,
	}

	br := bracket.BuildBracket(mixedEvent, divs, map[string]string{})
	if len(br.Divisions) != 1 || !br.Divisions[0].IsUnclassified {
		t.Fatalf("expected mixed-category event to ignore gender-specific bands even when opted in, got %+v", br.Divisions)
	}
}

func createFinishedMatch(id string, p1, p2 *player.Player, winner string) event.Match {
	scoreA := 0
	scoreB := 0
	if winner == "A" {
		scoreA = 11
		scoreB = 5
	} else {
		scoreA = 5
		scoreB = 11
	}
	return event.Match{
		ID:         id,
		TeamA:      []*player.Player{p1},
		TeamB:      []*player.Player{p2},
		Status:     "finished",
		Stage:      "group",
		WinnerTeam: winner,
		Sets: []event.MatchSet{
			{ScoreA: scoreA, ScoreB: scoreB},
			{ScoreA: scoreA, ScoreB: scoreB},
			{ScoreA: scoreA, ScoreB: scoreB},
		},
	}
}

// TestBracketMatch_ScoreAndWinnerFollowPlayerSlotNotTeamAssumption reproduces a real
// production bug: Player1/Player2 are assigned by bracket seed position, which does not
// necessarily line up with which player is TeamA vs TeamB in the underlying match record.
// Jorge (TeamB, won 3 sets to 2) ends up seeded into the Player1 slot here — the display
// must still show him with score 3 and as the winner, not TeamA's score/status.
func TestBracketMatch_ScoreAndWinnerFollowPlayerSlotNotTeamAssumption(t *testing.T) {
	orlando := &player.Player{ID: "orlando", FirstName: "Orlando", LastName: "Montiel"}
	jorge := &player.Player{ID: "jorge", FirstName: "Jorge", LastName: "Bermúdez"}

	m := &event.Match{
		ID:         "m1",
		TeamA:      []*player.Player{orlando},
		TeamB:      []*player.Player{jorge},
		Status:     "finished",
		WinnerTeam: "B", // Jorge (TeamB) won
		Sets: []event.MatchSet{
			{ScoreA: 9, ScoreB: 11},
			{ScoreA: 11, ScoreB: 7},
			{ScoreA: 11, ScoreB: 5},
			{ScoreA: 10, ScoreB: 12},
			{ScoreA: 6, ScoreB: 11},
		}, // Orlando (A) wins 2 sets, Jorge (B) wins 3 sets
	}

	// Player1 is Jorge (seeded there), Player2 is Orlando — the "wrong side" relative to TeamA/TeamB.
	bm := bracket.BracketMatch{
		Player1: &bracket.MatchSlot{Seed: 1, Player: jorge},
		Player2: &bracket.MatchSlot{Seed: 5, Player: orlando},
		Match:   m,
	}

	if got := bm.Player1Score(); got != 3 {
		t.Errorf("expected Player1 (Jorge, TeamB) score 3, got %d", got)
	}
	if got := bm.Player2Score(); got != 2 {
		t.Errorf("expected Player2 (Orlando, TeamA) score 2, got %d", got)
	}
	if !bm.Player1Won() {
		t.Error("expected Player1 (Jorge) to be flagged as the winner")
	}
	if bm.Player2Won() {
		t.Error("expected Player2 (Orlando) to NOT be flagged as the winner")
	}
}

func TestBracketMatch_ScoreAndWinner_Player1IsTeamA(t *testing.T) {
	p1 := &player.Player{ID: "a", FirstName: "A"}
	p2 := &player.Player{ID: "b", FirstName: "B"}

	m := &event.Match{
		ID:         "m2",
		TeamA:      []*player.Player{p1},
		TeamB:      []*player.Player{p2},
		Status:     "finished",
		WinnerTeam: "A",
		Sets: []event.MatchSet{
			{ScoreA: 11, ScoreB: 5},
			{ScoreA: 11, ScoreB: 5},
			{ScoreA: 11, ScoreB: 5},
		},
	}

	bm := bracket.BracketMatch{
		Player1: &bracket.MatchSlot{Seed: 1, Player: p1},
		Player2: &bracket.MatchSlot{Seed: 2, Player: p2},
		Match:   m,
	}

	if got := bm.Player1Score(); got != 3 {
		t.Errorf("expected Player1 score 3, got %d", got)
	}
	if got := bm.Player2Score(); got != 0 {
		t.Errorf("expected Player2 score 0, got %d", got)
	}
	if !bm.Player1Won() || bm.Player2Won() {
		t.Errorf("expected Player1 to win, got Player1Won=%v Player2Won=%v", bm.Player1Won(), bm.Player2Won())
	}
}

func TestBracketMatch_ScoreAndWinner_NoMatch(t *testing.T) {
	bm := bracket.BracketMatch{
		Player1: &bracket.MatchSlot{Seed: 1, Player: &player.Player{ID: "a"}},
	}
	if bm.Player1Score() != 0 || bm.Player2Score() != 0 {
		t.Error("expected zero scores when Match is nil")
	}
	if bm.Player1Won() || bm.Player2Won() {
		t.Error("expected no winner when Match is nil")
	}
}
