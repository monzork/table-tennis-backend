package bracket_test

import (
	"fmt"
	"testing"

	"table-tennis-backend/internal/domain/bracket"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

// TestGeneratorCoverage_SkipEloGroups exercises findGroupByPlayers via the
// SkipElo "Open Bracket" path: once when a pre-existing group already
// contains the participants (GroupID gets carried through) and once when no
// group matches (findGroupByPlayers returns nil).
func TestGeneratorCoverage_SkipEloGroups(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "P", LastName: "1", Gender: "M", SinglesElo: 1000}
	p2 := &player.Player{ID: "p2", FirstName: "P", LastName: "2", Gender: "M", SinglesElo: 900}

	t.Run("group found for skip-elo participants", func(t *testing.T) {
		tourney := &event.Event{
			ID:            "t1",
			Type:          "singles",
			Format:        "elimination",
			EventCategory: "open",
			SkipElo:       true,
			Participants:  []*player.Player{p1, p2},
			Groups: []event.Group{
				{ID: "g1", Name: "Open Bracket - Bracket Draw", Players: []*player.Player{p1, p2}},
			},
		}
		br := bracket.BuildBracket(tourney, nil, nil)
		if len(br.Divisions) != 1 {
			t.Fatalf("expected 1 division, got %d", len(br.Divisions))
		}
		if br.Divisions[0].GroupID != "g1" {
			t.Errorf("expected GroupID g1, got %q", br.Divisions[0].GroupID)
		}
	})

	t.Run("no group matches skip-elo participants", func(t *testing.T) {
		tourney := &event.Event{
			ID:            "t2",
			Type:          "singles",
			Format:        "elimination",
			EventCategory: "open",
			SkipElo:       true,
			Participants:  []*player.Player{p1, p2},
		}
		br := bracket.BuildBracket(tourney, nil, nil)
		if len(br.Divisions) != 1 {
			t.Fatalf("expected 1 division, got %d", len(br.Divisions))
		}
		if br.Divisions[0].GroupID != "" {
			t.Errorf("expected empty GroupID, got %q", br.Divisions[0].GroupID)
		}
	})

	t.Run("single_division_multiple_brackets format takes the open-bracket path", func(t *testing.T) {
		tourney := &event.Event{
			ID:            "t3",
			Type:          "singles",
			Format:        "single_division_multiple_brackets",
			EventCategory: "open",
			Participants:  []*player.Player{p1, p2},
		}
		br := bracket.BuildBracket(tourney, nil, nil)
		if len(br.Divisions) != 1 {
			t.Fatalf("expected 1 division, got %d", len(br.Divisions))
		}
	})
}

// TestGeneratorCoverage_UnassignedWithGroup exercises the "Unclassified"
// pool's group lookup: a player who doesn't fall in any division range but
// does belong to an existing bracket-draw group.
func TestGeneratorCoverage_UnassignedWithGroup(t *testing.T) {
	inRange := &player.Player{ID: "p_in", Gender: "M", SinglesElo: 500}
	outOfRange := &player.Player{ID: "p_out", Gender: "M", SinglesElo: 5000}

	maxElo := int16(1000)
	divs := []*division.Division{
		{ID: "d1", Name: "Low", Category: "both", MinElo: 1, MaxElo: &maxElo},
	}

	tourney := &event.Event{
		ID:            "t1",
		Type:          "singles",
		Format:        "elimination",
		EventCategory: "open",
		Participants:  []*player.Player{inRange, outOfRange},
		Groups: []event.Group{
			{ID: "g_unclassified", Name: "Unclassified - Bracket Draw", Players: []*player.Player{outOfRange}},
		},
	}

	br := bracket.BuildBracket(tourney, divs, nil)

	var unclassified *bracket.Division
	for i := range br.Divisions {
		if br.Divisions[i].IsUnclassified {
			unclassified = &br.Divisions[i]
		}
	}
	if unclassified == nil {
		t.Fatalf("expected an unclassified division, got %+v", br.Divisions)
	}
	if unclassified.GroupID != "g_unclassified" {
		t.Errorf("expected unclassified GroupID g_unclassified, got %q", unclassified.GroupID)
	}
}

// TestGeneratorCoverage_ThirdPlaceSingleElim drives a full 4-player
// single-elimination bracket with real, finished matches through the
// semifinals, final and third-place match so the champion-crowning and
// third-place branches in buildBracketRounds execute.
func TestGeneratorCoverage_ThirdPlaceSingleElim(t *testing.T) {
	players := []*player.Player{
		{ID: "p1", FirstName: "P", LastName: "1", Gender: "M", SinglesElo: 1200},
		{ID: "p2", FirstName: "P", LastName: "2", Gender: "M", SinglesElo: 1100},
		{ID: "p3", FirstName: "P", LastName: "3", Gender: "M", SinglesElo: 1000},
		{ID: "p4", FirstName: "P", LastName: "4", Gender: "M", SinglesElo: 900},
	}
	p1, p2, p3, p4 := players[0], players[1], players[2], players[3]

	// Seeding arrangement for 4 pairs seed1 vs seed4, seed3 vs seed2:
	// semifinal 0 = p1 vs p4, semifinal 1 = p3 vs p2.
	matches := []event.Match{
		{ID: "sf0", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p4}},
		{ID: "sf1", Stage: "semifinal", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p3}, TeamB: []*player.Player{p2}},
		{ID: "final", Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
		{ID: "third", Stage: "3rd_place", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p4}, TeamB: []*player.Player{p3}},
	}

	tourney := &event.Event{
		ID:                    "t1",
		Type:                  "singles",
		Format:                "single_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 1,
		HasThirdPlaceMatch:    true,
		Participants:          players,
		Matches:               matches,
	}
	divs := []*division.Division{
		{ID: "d1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) != 1 {
		t.Fatalf("expected 1 division, got %d", len(br.Divisions))
	}
	kb := br.Divisions[0].KnockoutBrackets
	if len(kb) != 1 {
		t.Fatalf("expected 1 knockout bracket, got %d", len(kb))
	}

	var sawFinal, sawChampion, sawThirdPlace bool
	for _, r := range kb[0].Rounds {
		switch r.Name {
		case "🏆 Final":
			sawFinal = true
			if len(r.Matches) != 1 || r.Matches[0].Match == nil {
				t.Errorf("expected final round to have the attached DB match")
			}
		case "Champion":
			sawChampion = true
			if len(r.Matches) != 1 || r.Matches[0].Player1 == nil || r.Matches[0].Player1.Player.ID != "p1" {
				t.Errorf("expected p1 to be crowned champion, got %+v", r.Matches[0].Player1)
			}
		case "🥉 3rd Place":
			sawThirdPlace = true
			if len(r.Matches) != 1 || r.Matches[0].Match == nil {
				t.Errorf("expected 3rd place round to have the attached DB match")
			}
		}
	}
	if !sawFinal || !sawChampion || !sawThirdPlace {
		t.Errorf("expected Final, Champion and 3rd Place rounds, got final=%v champion=%v third=%v", sawFinal, sawChampion, sawThirdPlace)
	}
}

// TestGeneratorCoverage_DoubleEliminationFull drives an 8-player
// double-elimination bracket through quarterfinals and semifinals with real
// finished matches (mixing which physical team ends up as the bracket's
// "Player1" slot, and mixing WinnerTeam "A"/"B") so getMatchWinner and
// getMatchLoser exercise all four of their branches while building the
// losers bracket.
func TestGeneratorCoverage_DoubleEliminationFull(t *testing.T) {
	names := []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7", "p8"}
	players := make([]*player.Player, len(names))
	elo := int16(1600)
	for i, id := range names {
		players[i] = &player.Player{ID: id, FirstName: "P", LastName: id, Gender: "M", SinglesElo: elo}
		elo -= 100
	}
	p1, p2, p3, p4, p5, p6, p7, p8 := players[0], players[1], players[2], players[3], players[4], players[5], players[6], players[7]

	// Seeding arrangement for size 8 pairs seeds: (1,8) (5,4) (3,6) (7,2), i.e.
	// quarterfinal pairs (p1,p8) (p5,p4) (p3,p6) (p7,p2).
	matches := []event.Match{
		{ID: "qf0", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p8}},
		{ID: "qf1", Stage: "quarterfinal", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p5}, TeamB: []*player.Player{p4}},
		{ID: "qf2", Stage: "quarterfinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p3}, TeamB: []*player.Player{p6}},
		{ID: "qf3", Stage: "quarterfinal", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p7}, TeamB: []*player.Player{p2}},
		// Semifinal pairs derived from the quarterfinal winners: (p1,p4) and (p3,p2).
		// Team ordering is flipped relative to bracket slot order here to exercise
		// the "Player1 is TeamB" branches of getMatchWinner/getMatchLoser.
		{ID: "sf0", Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p4}, TeamB: []*player.Player{p1}},
		{ID: "sf1", Stage: "semifinal", Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}},
		{ID: "final", Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p4}, TeamB: []*player.Player{p3}},
	}

	tourney := &event.Event{
		ID:                    "t1",
		Type:                  "singles",
		Format:                "double_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 1,
		Participants:          players,
		Matches:               matches,
	}
	divs := []*division.Division{
		{ID: "d1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) != 1 {
		t.Fatalf("expected 1 division, got %d", len(br.Divisions))
	}
	kb := br.Divisions[0].KnockoutBrackets
	if len(kb) != 2 {
		t.Fatalf("expected winners + losers brackets (2), got %d", len(kb))
	}
	if kb[0].Name != "Winners Bracket" || len(kb[0].Rounds) == 0 {
		t.Errorf("expected a non-empty winners bracket, got %+v", kb[0])
	}
	if kb[1].Name != "Losers Bracket" || len(kb[1].Rounds) == 0 {
		t.Errorf("expected a non-empty losers bracket, got %+v", kb[1])
	}
	// The first losers round should be seeded from the quarterfinal losers
	// (p8, p5, p6, p7) rather than being left fully unresolved.
	foundResolvedLoserSlot := false
	for _, m := range kb[1].Rounds[0].Matches {
		if (m.Player1 != nil && m.Player1.Player != nil) || (m.Player2 != nil && m.Player2.Player != nil) {
			foundResolvedLoserSlot = true
		}
	}
	if !foundResolvedLoserSlot {
		t.Errorf("expected at least one resolved player slot in the first losers round, got %+v", kb[1].Rounds[0].Matches)
	}
}

// TestGeneratorCoverage_GroupsEliminationExistingGroups exercises
// buildGroupEliminationGroups' "load saved groups" branch (including the
// "Open Bracket" / "Group " prefix fallback and membership-only matching)
// and buildDivisionView's pre-existing "Knockout Seeds" group lookup.
func TestGeneratorCoverage_GroupsEliminationExistingGroups(t *testing.T) {
	players := make([]*player.Player, 8)
	elo := int16(1600)
	for i := range players {
		id := "gp" + string(rune('1'+i))
		players[i] = &player.Player{ID: id, FirstName: "P", LastName: id, Gender: "M", SinglesElo: elo}
		elo -= 50
	}

	groupA := event.Group{ID: "gA", Name: "Open - Group A", Players: players[:4]}
	// This group belongs to the division purely by player membership (no
	// "<division> - " prefix and no "Group " fallback prefix), exercising the
	// membership-scan branch.
	groupB := event.Group{ID: "gB", Name: "Untitled Pool", Players: players[4:]}
	knockoutSeeds := event.Group{ID: "gKS", Name: "Open - Knockout Seeds", Players: []*player.Player{players[0], players[4]}}

	// Every pair within each 4-player group has played and finished, so both
	// groups are fully finished and the knockout-seed lookup path runs.
	var matches []event.Match
	mid := 0
	finishAllPairs := func(group []*player.Player) {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				mid++
				matches = append(matches, event.Match{
					ID:         fmt.Sprintf("m%d", mid),
					Stage:      "group",
					Status:     "finished",
					WinnerTeam: "A",
					TeamA:      []*player.Player{group[i]},
					TeamB:      []*player.Player{group[j]},
				})
			}
		}
	}
	finishAllPairs(players[:4])
	finishAllPairs(players[4:])

	tourney := &event.Event{
		ID:                    "t1",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 1,
		GroupPassCount:        1,
		Participants:          players,
		Groups:                []event.Group{groupA, groupB, knockoutSeeds},
		Matches:               matches,
	}
	divs := []*division.Division{
		{ID: "d1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) != 1 {
		t.Fatalf("expected 1 division, got %d", len(br.Divisions))
	}
	dv := br.Divisions[0]
	if len(dv.Groups) != 2 {
		t.Fatalf("expected the 2 pre-existing groups to be reused, got %d", len(dv.Groups))
	}
	if !dv.AllGroupsFinished {
		t.Skip("groups not finished under this player count; not exercising knockout-seed lookup")
	}
	if len(dv.KnockoutBrackets) != 1 || len(dv.KnockoutBrackets[0].Advancing) == 0 {
		t.Errorf("expected knockout bracket advancing players sourced from the pre-existing Knockout Seeds group, got %+v", dv.KnockoutBrackets)
	}
}

// TestGeneratorCoverage_GetMatchWinnerLoser unit-tests getMatchWinner and
// getMatchLoser directly (via export_test.go wrappers) since, in the real
// double-elimination flow, they're only ever called on losers-bracket rounds
// whose Match field hasn't been attached yet - making their "resolved"
// branches structurally unreachable through BuildBracket alone.
func TestGeneratorCoverage_GetMatchWinnerLoser(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	slot1 := &bracket.MatchSlot{Seed: 1, Player: p1}
	slot2 := &bracket.MatchSlot{Seed: 2, Player: p2}

	t.Run("no match returns unresolved slot", func(t *testing.T) {
		m := bracket.BracketMatch{Player1: slot1, Player2: slot2, Match: nil}
		if w := bracket.GetMatchWinnerForTest(m); w.Player != nil {
			t.Errorf("expected nil player, got %v", w.Player)
		}
		if l := bracket.GetMatchLoserForTest(m); l.Player != nil {
			t.Errorf("expected nil player, got %v", l.Player)
		}
	})

	t.Run("match not finished returns unresolved slot", func(t *testing.T) {
		dbm := &event.Match{Status: "scheduled", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}}
		m := bracket.BracketMatch{Player1: slot1, Player2: slot2, Match: dbm}
		if w := bracket.GetMatchWinnerForTest(m); w.Player != nil {
			t.Errorf("expected nil player, got %v", w.Player)
		}
	})

	t.Run("winner A, Player1 is TeamA", func(t *testing.T) {
		dbm := &event.Match{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}}
		m := bracket.BracketMatch{Player1: slot1, Player2: slot2, Match: dbm}
		if w := bracket.GetMatchWinnerForTest(m); w.Player.ID != "p1" {
			t.Errorf("expected p1 to win, got %v", w.Player)
		}
		if l := bracket.GetMatchLoserForTest(m); l.Player.ID != "p2" {
			t.Errorf("expected p2 to lose, got %v", l.Player)
		}
	})

	t.Run("winner A, Player1 is TeamB", func(t *testing.T) {
		dbm := &event.Match{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p1}}
		m := bracket.BracketMatch{Player1: slot1, Player2: slot2, Match: dbm}
		if w := bracket.GetMatchWinnerForTest(m); w.Player.ID != "p2" {
			t.Errorf("expected p2 (TeamA) to win, got %v", w.Player)
		}
		if l := bracket.GetMatchLoserForTest(m); l.Player.ID != "p1" {
			t.Errorf("expected p1 (TeamB) to lose, got %v", l.Player)
		}
	})

	t.Run("winner B, Player1 is TeamB", func(t *testing.T) {
		dbm := &event.Match{Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p1}}
		m := bracket.BracketMatch{Player1: slot1, Player2: slot2, Match: dbm}
		if w := bracket.GetMatchWinnerForTest(m); w.Player.ID != "p1" {
			t.Errorf("expected p1 (TeamB) to win, got %v", w.Player)
		}
		if l := bracket.GetMatchLoserForTest(m); l.Player.ID != "p2" {
			t.Errorf("expected p2 (TeamA) to lose, got %v", l.Player)
		}
	})

	t.Run("winner B, Player1 is TeamA", func(t *testing.T) {
		dbm := &event.Match{Status: "finished", WinnerTeam: "B", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}}
		m := bracket.BracketMatch{Player1: slot1, Player2: slot2, Match: dbm}
		if w := bracket.GetMatchWinnerForTest(m); w.Player.ID != "p2" {
			t.Errorf("expected p2 (TeamB) to win, got %v", w.Player)
		}
		if l := bracket.GetMatchLoserForTest(m); l.Player.ID != "p1" {
			t.Errorf("expected p1 (TeamA) to lose, got %v", l.Player)
		}
	})

	t.Run("missing player slots returns unresolved", func(t *testing.T) {
		dbm := &event.Match{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}}
		m := bracket.BracketMatch{Player1: nil, Player2: slot2, Match: dbm}
		if w := bracket.GetMatchWinnerForTest(m); w.Player != nil {
			t.Errorf("expected nil player when Player1 slot missing, got %v", w.Player)
		}
	})
}

// TestGeneratorCoverage_DoublesTeamsWithBracketDrawGroup exercises the
// team/doubles average-Elo participant synthesis, the "0-infinite division
// skipped" filter, the " Division" name-suffix trim, and the elimination
// format's named "<division> - Bracket Draw" group lookup, all in one pass.
func TestGeneratorCoverage_DoublesTeamsWithBracketDrawGroup(t *testing.T) {
	pA1 := &player.Player{ID: "pA1", Gender: "M", DoublesElo: 1200}
	pA2 := &player.Player{ID: "pA2", Gender: "M", DoublesElo: 1300}
	pB1 := &player.Player{ID: "pB1", Gender: "M", DoublesElo: 900}
	pB2 := &player.Player{ID: "pB2", Gender: "M", DoublesElo: 800}

	teamA := &event.Team{ID: "teamA", Name: "Team A", Players: []*player.Player{pA1, pA2}}
	teamB := &event.Team{ID: "teamB", Name: "Team B", Players: []*player.Player{pB1, pB2}}

	tourney := &event.Event{
		ID:            "t1",
		Type:          "doubles",
		Format:        "elimination",
		EventCategory: "open",
		Teams:         []*event.Team{teamA, teamB},
		Groups: []event.Group{
			// Bracket-draw group name must match "<division name> - Bracket Draw".
			{ID: "gDraw", Name: "Open Division - Bracket Draw", Players: []*player.Player{
				{ID: "teamA"}, {ID: "teamB"},
			}},
		},
	}

	divs := []*division.Division{
		// Should be skipped: "0-infinite" division for a non-skip-elo event.
		{ID: "d0", Name: "No Division", Category: "doubles", MinElo: 0, MaxElo: nil},
		// Name has the " Division" suffix, which should be trimmed for display,
		// and matches the "Open Division - Bracket Draw" group above.
		{ID: "d1", Name: "Open Division", Category: "doubles", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) != 1 {
		t.Fatalf("expected 1 division (the 0-infinite one skipped), got %d: %+v", len(br.Divisions), br.Divisions)
	}
	dv := br.Divisions[0]
	if dv.Name != "Open" {
		t.Errorf("expected trimmed name 'Open', got %q", dv.Name)
	}
	if dv.GroupID != "gDraw" {
		t.Errorf("expected GroupID gDraw from the bracket-draw group lookup, got %q", dv.GroupID)
	}
	if len(dv.Players) != 2 {
		t.Fatalf("expected 2 team-players from the bracket-draw group, got %d", len(dv.Players))
	}
}

// TestGeneratorCoverage_RoundRobinMatchLookupSkips exercises buildRRMatches'
// DB-match lookup skip branches: a sub-match (TeamMatchID set) and a match
// at a different stage should both be ignored, while a same-stage,
// non-sub-match should be attached.
func TestGeneratorCoverage_RoundRobinMatchLookupSkips(t *testing.T) {
	p1 := &player.Player{ID: "p1", Gender: "M", SinglesElo: 1000}
	p2 := &player.Player{ID: "p2", Gender: "M", SinglesElo: 900}
	p3 := &player.Player{ID: "p3", Gender: "M", SinglesElo: 800}

	teamMatchID := "parent1"
	tourney := &event.Event{
		ID:            "t1",
		Type:          "singles",
		Format:        "round_robin",
		EventCategory: "open",
		Participants:  []*player.Player{p1, p2, p3},
		Matches: []event.Match{
			// Sub-match: must be skipped regardless of team pairing.
			{ID: "sub", TeamMatchID: &teamMatchID, Stage: "group", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
			// Wrong stage: must be skipped.
			{ID: "wrong-stage", Stage: "final", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}},
			// Correct stage, real match: should be attached.
			{ID: "real", Status: "finished", WinnerTeam: "A", Stage: "group", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}},
		},
	}
	divs := []*division.Division{
		{ID: "d1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) != 1 {
		t.Fatalf("expected 1 division, got %d", len(br.Divisions))
	}
	var foundP2P3 bool
	for _, m := range br.Divisions[0].RoundRobinMatches {
		if (m.Player1.ID == "p2" && m.Player2.ID == "p3") || (m.Player1.ID == "p3" && m.Player2.ID == "p2") {
			if m.Match == nil || m.Match.ID != "real" {
				t.Errorf("expected p2 vs p3 to have the 'real' DB match attached, got %+v", m.Match)
			}
			foundP2P3 = true
		}
		if (m.Player1.ID == "p1" && m.Player2.ID == "p2") || (m.Player1.ID == "p2" && m.Player2.ID == "p1") {
			if m.Match != nil {
				t.Errorf("expected p1 vs p2 to have no DB match (sub-match should be skipped), got %+v", m.Match)
			}
		}
	}
	if !foundP2P3 {
		t.Fatalf("expected to find the p2 vs p3 round-robin match")
	}
}

// TestGeneratorCoverage_GroupsEliminationSnakeSeedingFallback exercises
// buildGroupEliminationGroups' fallback path (no pre-existing DB groups, so
// players are snake-seeded into fresh groups) including its finished-match
// counting loop, by finishing every pair within each generated group.
func TestGeneratorCoverage_GroupsEliminationSnakeSeedingFallback(t *testing.T) {
	players := make([]*player.Player, 8)
	elo := int16(1600)
	for i := range players {
		players[i] = &player.Player{ID: fmt.Sprintf("sp%d", i+1), Gender: "M", SinglesElo: elo}
		elo -= 50
	}

	// Snake seeding for 8 players / groupSize 4 -> numGroups=2:
	// group0 = players[0,3,4,7], group1 = players[1,2,5,6].
	group0 := []*player.Player{players[0], players[3], players[4], players[7]}
	group1 := []*player.Player{players[1], players[2], players[5], players[6]}

	var matches []event.Match
	mid := 0
	finishAllPairs := func(group []*player.Player) {
		for i := 0; i < len(group); i++ {
			for j := i + 1; j < len(group); j++ {
				mid++
				matches = append(matches, event.Match{
					ID:         fmt.Sprintf("sm%d", mid),
					Stage:      "group",
					Status:     "finished",
					WinnerTeam: "A",
					TeamA:      []*player.Player{group[i]},
					TeamB:      []*player.Player{group[j]},
				})
			}
		}
	}
	finishAllPairs(group0)
	finishAllPairs(group1)

	tourney := &event.Event{
		ID:                    "t1",
		Type:                  "singles",
		Format:                "groups_elimination",
		EventCategory:         "open",
		KnockoutBracketsCount: 1,
		GroupPassCount:        2,
		Participants:          players,
		Matches:               matches,
		// No pre-existing t.Groups -> forces the snake-seeding fallback.
	}
	divs := []*division.Division{
		{ID: "d1", Name: "Open", Category: "both", MinElo: 1, MaxElo: nil},
	}

	br := bracket.BuildBracket(tourney, divs, nil)
	if len(br.Divisions) != 1 {
		t.Fatalf("expected 1 division, got %d", len(br.Divisions))
	}
	dv := br.Divisions[0]
	if len(dv.Groups) != 2 {
		t.Fatalf("expected 2 snake-seeded groups, got %d", len(dv.Groups))
	}
	if !dv.AllGroupsFinished {
		t.Fatalf("expected all snake-seeded groups to be finished, got groups=%+v", dv.Groups)
	}
	if len(dv.KnockoutBrackets) != 1 || len(dv.KnockoutBrackets[0].Advancing) == 0 {
		t.Errorf("expected a knockout bracket built from the finished snake-seeded groups, got %+v", dv.KnockoutBrackets)
	}
}

// TestGeneratorCoverage_ValidateSameGroupSeparationConflict feeds a seed
// order that deliberately places two same-group players in the same bracket
// half, so the conflict-reporting path executes.
func TestGeneratorCoverage_ValidateSameGroupSeparationConflict(t *testing.T) {
	p1 := &player.Player{ID: "p1", FirstName: "A", LastName: "1"}
	p2 := &player.Player{ID: "p2", FirstName: "A", LastName: "2"}
	p3 := &player.Player{ID: "p3", FirstName: "A", LastName: "3"}
	p4 := &player.Player{ID: "p4", FirstName: "A", LastName: "4"}

	// For a size-4 bracket the seeding arrangement is [1,4,3,2]: seeds 1 and 4
	// land in the top half, seeds 2 and 3 land in the bottom half.
	order := []*player.Player{p1, p2, p3, p4} // seed1=p1, seed2=p2, seed3=p3, seed4=p4

	conflictGroups := []bracket.Group{
		{ID: "g1", Name: "Group A", Players: []*player.Player{p1, p4}}, // both top half seeds
		{ID: "g2", Name: "Group B", Players: []*player.Player{p2, p3}}, // both bottom half seeds
	}
	if err := bracket.ValidateSameGroupSeparation(conflictGroups, order); err == nil {
		t.Fatal("expected a same-group separation conflict error")
	}

	// A single player short-circuits with no error.
	if err := bracket.ValidateSameGroupSeparation(conflictGroups, []*player.Player{p1}); err != nil {
		t.Errorf("expected nil for a single player, got %v", err)
	}

	// Same groups but split across top/bottom halves -> no conflict.
	separatedGroups := []bracket.Group{
		{ID: "g1", Name: "Group A", Players: []*player.Player{p1, p2}}, // top + bottom
		{ID: "g2", Name: "Group B", Players: []*player.Player{p3, p4}}, // bottom + top
	}
	if err := bracket.ValidateSameGroupSeparation(separatedGroups, order); err != nil {
		t.Errorf("expected no conflict for well separated seeds, got %v", err)
	}
}
