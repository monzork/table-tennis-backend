package event

import (
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	"testing"
)

func TestPlacementEloBonus_Elimination(t *testing.T) {
	john := &player.Player{ID: "john"}
	jane := &player.Player{ID: "jane"}
	bob := &player.Player{ID: "bob"}
	alice := &player.Player{ID: "alice"}
	charlie := &player.Player{ID: "charlie"}

	ev := &Event{
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{
				Stage:      "final",
				Status:     "finished",
				WinnerTeam: "A",
				TeamA:      []*player.Player{john},
				TeamB:      []*player.Player{jane},
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				WinnerTeam: "A",
				TeamA:      []*player.Player{bob},
				TeamB:      []*player.Player{alice}, // semi loser -> 3rd
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				WinnerTeam: "B",
				TeamA:      []*player.Player{charlie}, // semi loser -> 3rd
				TeamB:      []*player.Player{john},
			},
		},
	}

	bonus := PlacementEloBonus(ev, nil)
	if bonus["john"] != FirstPlaceEloBonus {
		t.Errorf("expected champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["john"])
	}
	if bonus["jane"] != SecondPlaceEloBonus {
		t.Errorf("expected runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["jane"])
	}
	if bonus["alice"] != ThirdPlaceEloBonus || bonus["charlie"] != ThirdPlaceEloBonus {
		t.Errorf("expected both semifinal losers at %.1f, got alice=%v charlie=%v", ThirdPlaceEloBonus, bonus["alice"], bonus["charlie"])
	}
	if _, ok := bonus["bob"]; ok {
		t.Errorf("expected bob (a semifinal winner, not on the podium) to have no bonus entry, got %v", bonus["bob"])
	}
}

func TestPlacementResults_LabelsMatchAmounts(t *testing.T) {
	john := &player.Player{ID: "john"}
	jane := &player.Player{ID: "jane"}
	bob := &player.Player{ID: "bob"}
	alice := &player.Player{ID: "alice"}

	ev := &Event{
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{Stage: "final", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{john}, TeamB: []*player.Player{jane}},
			{Stage: "semifinal", Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{bob}, TeamB: []*player.Player{alice}},
		},
	}

	results := PlacementResults(ev, nil)
	if got := results["john"]; got.Placement != PlacementChampion || got.BonusElo != FirstPlaceEloBonus {
		t.Errorf("expected john labeled champion with %.1f, got %+v", FirstPlaceEloBonus, got)
	}
	if got := results["jane"]; got.Placement != PlacementRunnerUp || got.BonusElo != SecondPlaceEloBonus {
		t.Errorf("expected jane labeled runner_up with %.1f, got %+v", SecondPlaceEloBonus, got)
	}
	if got := results["alice"]; got.Placement != PlacementThird || got.BonusElo != ThirdPlaceEloBonus {
		t.Errorf("expected alice labeled third with %.1f, got %+v", ThirdPlaceEloBonus, got)
	}
	if _, ok := results["bob"]; ok {
		t.Errorf("expected bob (semifinal winner, not on the podium) absent, got %+v", results["bob"])
	}

	// PlacementEloBonus must stay in lockstep with PlacementResults -- same
	// underlying computation, just without the label.
	amounts := PlacementEloBonus(ev, nil)
	for id, detail := range results {
		if amounts[id] != detail.BonusElo {
			t.Errorf("expected PlacementEloBonus[%s]=%.1f to match PlacementResults, got %.1f", id, detail.BonusElo, amounts[id])
		}
	}
}

func TestPlacementEloBonus_NoFinishedFinal(t *testing.T) {
	ev := &Event{
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{Stage: "final", Status: "in_progress", TeamA: []*player.Player{{ID: "a"}}, TeamB: []*player.Player{{ID: "b"}}},
		},
	}
	bonus := PlacementEloBonus(ev, nil)
	if len(bonus) != 0 {
		t.Errorf("expected no bonus for unfinished final, got %v", bonus)
	}
}

func TestPlacementEloBonus_RoundRobin(t *testing.T) {
	p1 := &player.Player{ID: "p1"}
	p2 := &player.Player{ID: "p2"}
	p3 := &player.Player{ID: "p3"}
	p4 := &player.Player{ID: "p4"}

	ev := &Event{
		Format:       "round_robin",
		Type:         "singles",
		Participants: []*player.Player{p1, p2, p3, p4},
		Matches: []Match{
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p3}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p1}, TeamB: []*player.Player{p4}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p3}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p2}, TeamB: []*player.Player{p4}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{p3}, TeamB: []*player.Player{p4}},
		},
	}

	bonus := PlacementEloBonus(ev, nil)
	if bonus["p1"] != FirstPlaceEloBonus {
		t.Errorf("expected p1 (3 wins) as champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["p1"])
	}
	if bonus["p2"] != SecondPlaceEloBonus {
		t.Errorf("expected p2 (2 wins) as runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["p2"])
	}
	if bonus["p3"] != ThirdPlaceEloBonus {
		t.Errorf("expected p3 (1 win) at bonus %.1f, got %v", ThirdPlaceEloBonus, bonus["p3"])
	}
	if _, ok := bonus["p4"]; ok {
		t.Errorf("expected last place (p4) to have no bonus entry, got %v", bonus["p4"])
	}
}

func TestPlacementEloBonus_TeamsFormat(t *testing.T) {
	johnM := &player.Player{ID: "john"}
	janeM := &player.Player{ID: "jane"}
	bobM := &player.Player{ID: "bob"}
	aliceM := &player.Player{ID: "alice"}

	teamA := &Team{ID: "teamA", Name: "Team A", Players: []*player.Player{johnM}}
	teamB := &Team{ID: "teamB", Name: "Team B", Players: []*player.Player{janeM}}
	teamC := &Team{ID: "teamC", Name: "Team C", Players: []*player.Player{bobM}}
	teamD := &Team{ID: "teamD", Name: "Team D", Players: []*player.Player{aliceM}}

	ev := &Event{
		Format: "elimination",
		Type:   "teams",
		Teams:  []*Team{teamA, teamB, teamC, teamD},
		Matches: []Match{
			{
				Stage:      "final",
				Status:     "finished",
				MatchType:  "teams",
				WinnerTeam: "A",
				TeamA:      []*player.Player{{ID: "teamA", FirstName: "Team A"}},
				TeamB:      []*player.Player{{ID: "teamB", FirstName: "Team B"}},
			},
			{
				Stage:      "semifinal",
				Status:     "finished",
				MatchType:  "teams",
				WinnerTeam: "A",
				TeamA:      []*player.Player{{ID: "teamC", FirstName: "Team C"}},
				TeamB:      []*player.Player{{ID: "teamD", FirstName: "Team D"}},
			},
		},
	}

	bonus := PlacementEloBonus(ev, nil)
	if bonus["john"] != FirstPlaceEloBonus {
		t.Errorf("expected team A member john as champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["john"])
	}
	if bonus["jane"] != SecondPlaceEloBonus {
		t.Errorf("expected team B member jane as runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["jane"])
	}
	if bonus["alice"] != ThirdPlaceEloBonus {
		t.Errorf("expected team D member alice (eliminated in semis) at bonus %.1f, got %v", ThirdPlaceEloBonus, bonus["alice"])
	}
}

func TestPlacementEloBonus_UnknownFormat(t *testing.T) {
	ev := &Event{Format: "single_division_multiple_brackets"}
	if bonus := PlacementEloBonus(ev, nil); bonus != nil {
		t.Errorf("expected nil bonus for unhandled format, got %v", bonus)
	}
}

func TestPlacementEloBonus_RoundRobinEmptyParticipants(t *testing.T) {
	ev := &Event{Format: "round_robin", Type: "singles"}
	if bonus := PlacementEloBonus(ev, nil); bonus != nil {
		t.Errorf("expected nil bonus with no participants, got %v", bonus)
	}
}

func TestPlacementEloBonus_KnockoutPerDivisionMultiplier(t *testing.T) {
	elite1 := &player.Player{ID: "elite1"}
	elite2 := &player.Player{ID: "elite2"}
	rookie1 := &player.Player{ID: "rookie1"}
	rookie2 := &player.Player{ID: "rookie2"}

	ev := &Event{
		Format: "elimination",
		Type:   "singles",
		Matches: []Match{
			{
				Stage:      "final",
				Status:     "finished",
				DivisionID: "elite",
				WinnerTeam: "A",
				TeamA:      []*player.Player{elite1},
				TeamB:      []*player.Player{elite2},
			},
			{
				Stage:      "final",
				Status:     "finished",
				DivisionID: "rookie",
				WinnerTeam: "A",
				TeamA:      []*player.Player{rookie1},
				TeamB:      []*player.Player{rookie2},
			},
		},
	}

	divisions := []*division.Division{
		{ID: "elite", PlacementEloMultiplier: 0.5},
		{ID: "rookie", PlacementEloMultiplier: 2},
	}

	bonus := PlacementEloBonus(ev, divisions)
	if bonus["elite1"] != 16 { // 0.5 * 32
		t.Errorf("expected elite champion bonus 16, got %v", bonus["elite1"])
	}
	if bonus["elite2"] != 8 { // half of 16
		t.Errorf("expected elite runner-up bonus 8, got %v", bonus["elite2"])
	}
	if bonus["rookie1"] != 64 { // 2 * 32
		t.Errorf("expected rookie champion bonus 64, got %v", bonus["rookie1"])
	}
	if bonus["rookie2"] != 32 { // half of 64
		t.Errorf("expected rookie runner-up bonus 32, got %v", bonus["rookie2"])
	}
}

// TestPlacementEloBonus_RoundRobinIgnoresStaleMatchDivisionID is a
// regression test: a player (b1 here) who defaulted every one of their
// round-robin matches had those matches recorded with a stale/incorrect
// DivisionID while every other match in the same single round-robin group
// carried none. Bucketing standings by each match's own DivisionID split
// one real group into a bogus second "division" whose standings were
// computed from only that player's forfeits, corrupting both the real
// champion's rank and their placement bonus multiplier -- round-robin now
// always treats the whole event as one table/division (see
// roundRobinPlacementBonus), so this must resolve as a single group of 4.
func TestPlacementEloBonus_RoundRobinIgnoresStaleMatchDivisionID(t *testing.T) {
	a1 := &player.Player{ID: "a1", SinglesElo: 2300}
	a2 := &player.Player{ID: "a2", SinglesElo: 2300}
	a3 := &player.Player{ID: "a3", SinglesElo: 2300}
	b1 := &player.Player{ID: "b1", SinglesElo: 2300} // defaults every match

	ev := &Event{
		Format: "round_robin", Type: "singles", EventCategory: "men", UseGenderDivisions: true,
		Participants: []*player.Player{a1, a2, a3, b1},
		Matches: []Match{
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{a1}, TeamB: []*player.Player{a2}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{a1}, TeamB: []*player.Player{a3}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{a2}, TeamB: []*player.Player{a3}},
			// b1 forfeits every match -- stale DivisionID "div-first" (a
			// real, but wrong/legacy, division for this event) recorded on
			// these specifically, none of the real matches above.
			{Status: "finished", IsForfeit: true, DivisionID: "div-first", WinnerTeam: "A", TeamA: []*player.Player{a1}, TeamB: []*player.Player{b1}},
			{Status: "finished", IsForfeit: true, DivisionID: "div-first", WinnerTeam: "A", TeamA: []*player.Player{a2}, TeamB: []*player.Player{b1}},
			{Status: "finished", IsForfeit: true, DivisionID: "div-first", WinnerTeam: "A", TeamA: []*player.Player{a3}, TeamB: []*player.Player{b1}},
		},
	}

	divisions := []*division.Division{
		{ID: "div-first", MinElo: 1600, MaxElo: nil, Category: "both", Gender: "both", PlacementEloMultiplier: 2},
		{ID: "div-first-male", MinElo: 2000, MaxElo: nil, Category: "both", Gender: "M", PlacementEloMultiplier: 0.5},
	}

	bonus := PlacementEloBonus(ev, divisions)
	// a1 (3 wins) is the real champion of the whole group -- and should get
	// the gender-specific division's multiplier (0.5x), not the stale
	// legacy division's (2x) that only b1's forfeits carried.
	if bonus["a1"] != 16 {
		t.Errorf("expected champion bonus 16 (0.5x K-factor), got %v", bonus["a1"])
	}
	if _, ok := bonus["b1"]; ok {
		t.Errorf("expected b1 (0 wins, defaulted every match) to have no bonus, got %v", bonus["b1"])
	}
}

// TestPlacementEloBonus_ResolvesDivisionWhenMatchesCarryNone is a regression
// test: some tournaments create one whole event per division (e.g. "II
// Ranking Nacional por Divisiones" makes a separate "...(1st Division
// (Women))" event per division) and never stamp DivisionID on their
// matches, since the division scoping already happened at event-creation
// time. Without resolving the division from a representative participant's
// Elo, the bonus silently fell back to the default 2x-K-factor multiplier
// for every such event regardless of which division's own multiplier
// (e.g. 0.5x for an elite division) was actually configured.
func TestPlacementEloBonus_ResolvesDivisionWhenMatchesCarryNone(t *testing.T) {
	champ := &player.Player{ID: "champ", SinglesElo: 1769}
	runnerUp := &player.Player{ID: "runner", SinglesElo: 1700}
	third := &player.Player{ID: "third", SinglesElo: 1650}
	fourth := &player.Player{ID: "fourth", SinglesElo: 1600}

	ev := &Event{
		Format: "round_robin", Type: "singles",
		EventCategory: "women", UseGenderDivisions: true,
		Participants: []*player.Player{champ, runnerUp, third, fourth},
		Matches: []Match{
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{champ}, TeamB: []*player.Player{runnerUp}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{champ}, TeamB: []*player.Player{third}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{champ}, TeamB: []*player.Player{fourth}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{runnerUp}, TeamB: []*player.Player{third}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{runnerUp}, TeamB: []*player.Player{fourth}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{third}, TeamB: []*player.Player{fourth}},
		},
	}

	secondDivMax := int16(1300)
	divisions := []*division.Division{
		{ID: "div-first-female", MinElo: 1300, MaxElo: nil, Category: "both", Gender: "F", PlacementEloMultiplier: 0.5},
		{ID: "div-second-female", MinElo: 0, MaxElo: &secondDivMax, Category: "both", Gender: "F", PlacementEloMultiplier: 1},
		{ID: "div-first-male", MinElo: 1300, MaxElo: nil, Category: "both", Gender: "M", PlacementEloMultiplier: 2},
	}

	bonus := PlacementEloBonus(ev, divisions)
	if bonus["champ"] != 16 { // 0.5 * 32
		t.Errorf("expected champion bonus 16 (0.5x K-factor for div-first-female), got %v", bonus["champ"])
	}
	if bonus["runner"] != 8 {
		t.Errorf("expected runner-up bonus 8, got %v", bonus["runner"])
	}
	if bonus["third"] != 4 {
		t.Errorf("expected 3rd place bonus 4, got %v", bonus["third"])
	}
}

func TestPlacementEloBonus_RoundRobinTeamsFormat(t *testing.T) {
	john := &player.Player{ID: "john"}
	jane := &player.Player{ID: "jane"}
	bob := &player.Player{ID: "bob"}

	teamA := &Team{ID: "teamA", Players: []*player.Player{john}}
	teamB := &Team{ID: "teamB", Players: []*player.Player{jane}}
	teamC := &Team{ID: "teamC", Players: []*player.Player{bob}}

	ev := &Event{
		Format: "round_robin",
		Type:   "teams",
		Teams:  []*Team{teamA, teamB, teamC},
		Matches: []Match{
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{{ID: "teamA"}}, TeamB: []*player.Player{{ID: "teamB"}}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{{ID: "teamA"}}, TeamB: []*player.Player{{ID: "teamC"}}},
			{Status: "finished", WinnerTeam: "A", TeamA: []*player.Player{{ID: "teamB"}}, TeamB: []*player.Player{{ID: "teamC"}}},
		},
	}

	bonus := PlacementEloBonus(ev, nil)
	if bonus["john"] != FirstPlaceEloBonus {
		t.Errorf("expected team A member john as champion bonus %.1f, got %v", FirstPlaceEloBonus, bonus["john"])
	}
	if bonus["jane"] != SecondPlaceEloBonus {
		t.Errorf("expected team B member jane as runner-up bonus %.1f, got %v", SecondPlaceEloBonus, bonus["jane"])
	}
}
