package leaderboard_test

import (
	"testing"

	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
)

func eloPtr(v int16) *int16 { return &v }

func TestBuildRanking_AlwaysOneFlatListRegardlessOfDivisions(t *testing.T) {
	divisions := []*division.Division{
		{ID: "none", Name: "No Division"},
		{ID: "low", Name: "Segunda", MinElo: 0, MaxElo: eloPtr(1000), Category: "both"},
		{ID: "high", Name: "Primera", MinElo: 1000, MaxElo: nil, Category: "both"},
	}
	players := []*player.Player{
		{ID: "1", FirstName: "A", SinglesElo: 900},
		{ID: "2", FirstName: "B", SinglesElo: 1000},
		{ID: "3", FirstName: "C", SinglesElo: 1500},
	}

	result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
		RankType:  "singles",
		SortOrder: "points_desc",
	})

	if result.IsDivisional {
		t.Fatalf("expected the public ranking to never group by division")
	}
	if len(result.Groups) != 1 || result.Groups[0].Division != nil {
		t.Fatalf("expected a single ungrouped list, got %+v", result.Groups)
	}
	if len(result.Groups[0].Players) != 3 {
		t.Errorf("expected all 3 players in the one list, got %+v", result.Groups[0].Players)
	}
}

func TestBuildRanking_SearchQuery(t *testing.T) {
	players := []*player.Player{
		{ID: "1", FirstName: "Alice", LastName: "Smith", Country: "NIC", SinglesElo: 1200},
		{ID: "2", FirstName: "Bob", LastName: "Jones", Country: "CUB", SinglesElo: 1100},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		Query:     "cub",
		SortOrder: "points_desc",
	})

	if len(result.Groups) != 1 || len(result.Groups[0].Players) != 1 || result.Groups[0].Players[0].ID != "2" {
		t.Errorf("expected only Bob (country CUB) to match query, got %+v", result.Groups)
	}
}

func TestBuildRanking_SortOrders(t *testing.T) {
	players := []*player.Player{
		{ID: "1", FirstName: "Zed", SinglesElo: 1000},
		{ID: "2", FirstName: "Amy", SinglesElo: 1500},
	}

	t.Run("points_asc", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
			RankType: "singles", SortOrder: "points_asc",
		})
		got := result.Groups[0].Players
		if got[0].ID != "1" || got[1].ID != "2" {
			t.Errorf("expected ascending Elo order [1,2], got [%s,%s]", got[0].ID, got[1].ID)
		}
	})

	t.Run("name_asc", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
			RankType: "singles", SortOrder: "name_asc",
		})
		got := result.Groups[0].Players
		if got[0].ID != "2" || got[1].ID != "1" {
			t.Errorf("expected name order [Amy, Zed], got [%s,%s]", got[0].FirstName, got[1].FirstName)
		}
	})
}

func TestBuildRanking_SinglesIsOneCombinedPoolRegardlessOfGender(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "M1", Gender: "M", SinglesElo: 2000},
		{ID: "f1", FirstName: "F1", Gender: "F", SinglesElo: 1000},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		SortOrder: "points_desc",
	})

	if len(result.Groups) != 1 || len(result.Groups[0].Players) != 2 {
		t.Fatalf("expected both players in one combined group, got %+v", result.Groups)
	}
	if result.Groups[0].Players[0].ID != "m1" || result.Groups[0].Players[0].Rank != 1 {
		t.Errorf("expected higher-Elo player ranked 1 regardless of gender, got %+v", result.Groups[0].Players)
	}
	if result.Groups[0].Players[1].ID != "f1" || result.Groups[0].Players[1].Rank != 2 {
		t.Errorf("expected lower-Elo player ranked 2 regardless of gender, got %+v", result.Groups[0].Players)
	}
}

func TestBuildRanking_DoublesIsOneCombinedPoolRegardlessOfGender(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "M1", Gender: "M", DoublesElo: 2000},
		{ID: "f1", FirstName: "F1", Gender: "F", DoublesElo: 1000},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "doubles",
		SortOrder: "points_desc",
	})

	if len(result.Groups) != 1 || len(result.Groups[0].Players) != 2 {
		t.Fatalf("expected both players in one combined group, got %+v", result.Groups)
	}
	if result.Groups[0].Players[0].ID != "m1" || result.Groups[0].Players[1].ID != "f1" {
		t.Errorf("expected players ranked by Elo regardless of gender, got %+v", result.Groups[0].Players)
	}
}

func TestBuildRanking_DivisionFilter(t *testing.T) {
	divisions := []*division.Division{
		{ID: "low", Name: "Segunda", MinElo: 0, MaxElo: eloPtr(1000), Category: "both"},
		{ID: "high", Name: "Primera", MinElo: 1000, Category: "both"},
	}
	players := []*player.Player{
		{ID: "1", FirstName: "A", SinglesElo: 900},
		{ID: "2", FirstName: "B", SinglesElo: 1500},
	}

	t.Run("all keeps every player", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
			RankType: "singles", SortOrder: "points_desc", DivisionFilter: "all",
		})
		if len(result.Groups) != 1 || len(result.Groups[0].Players) != 2 {
			t.Errorf("expected both players in the one list, got %+v", result.Groups)
		}
	})

	t.Run("named filter narrows to one division", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
			RankType: "singles", SortOrder: "points_desc", DivisionFilter: "Primera",
		})
		if len(result.Groups) != 1 || len(result.Groups[0].Players) != 1 || result.Groups[0].Players[0].ID != "2" {
			t.Errorf("expected only player 2 (Primera) in the list, got %+v", result.Groups)
		}
	})

	t.Run("unknown filter name keeps all players unfiltered", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
			RankType: "singles", SortOrder: "points_desc", DivisionFilter: "Nonexistent",
		})
		total := 0
		for _, g := range result.Groups {
			total += len(g.Players)
		}
		if total != 2 {
			t.Errorf("expected both players still present when filter matches no division, got %d", total)
		}
	})
}

func TestBuildRanking_RankDelta(t *testing.T) {
	players := []*player.Player{
		{ID: "a", FirstName: "A", SinglesElo: 1000},
		{ID: "b", FirstName: "B", SinglesElo: 900},
		{ID: "c", FirstName: "C", SinglesElo: 800},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		SortOrder: "points_desc",
		PreviousElo: map[string]int16{
			"b": 700,  // was lowest by own-Elo (rank 3), now 900 -> up 1
			"c": 1100, // was highest by own-Elo (rank 1), now 800 -> down 2
			"a": 1000, // unchanged -> no movement
		},
	})

	byID := map[string]leaderboard.RankedPlayer{}
	for _, p := range result.Groups[0].Players {
		byID[p.ID] = p
	}

	if got := byID["b"].RankDelta; got == nil || *got != 1 {
		t.Errorf("expected b to have moved up 1 place, got %v", got)
	}
	if got := byID["c"].RankDelta; got == nil || *got != -2 {
		t.Errorf("expected c to have moved down 2 places, got %v", got)
	}
	if got := byID["a"].RankDelta; got == nil || *got != 0 {
		t.Errorf("expected a to show unchanged (0), got %v", got)
	}
}

func TestBuildRanking_RankDelta_NilWhenNoPreviousSnapshot(t *testing.T) {
	players := []*player.Player{
		{ID: "a", FirstName: "A", SinglesElo: 1000},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		SortOrder: "points_desc",
	})

	if result.Groups[0].Players[0].RankDelta != nil {
		t.Errorf("expected nil RankDelta when the player has no previous Elo snapshot, got %v", *result.Groups[0].Players[0].RankDelta)
	}
}

// TestBuildRanking_RankDeltaIsGenderScoped is a regression test: even in
// BuildRanking's combined-pool "overall" view (one shared 1..N rank number
// across genders), a player's RankDelta must be computed against their own
// gender's Elo pool only -- M and F Elo are separate rating scales, so a
// male player's movement must not be measured against female opponents'
// current Elo, and vice versa.
func TestBuildRanking_RankDeltaIsGenderScoped(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "M1", Gender: "M", SinglesElo: 2000},
		{ID: "m2", FirstName: "M2", Gender: "M", SinglesElo: 1000},
		// f1 sits between m1 and m2 in the combined pool, but is the ONLY
		// female player -- within her own gender pool she is unopposed, so
		// her rank movement must always be 0, regardless of what happens in
		// the men's pool around her combined-pool position.
		{ID: "f1", FirstName: "F1", Gender: "F", SinglesElo: 1500},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		SortOrder: "points_desc",
		PreviousElo: map[string]int16{
			"f1": 500, // previously far below current Elo, but no other female player exists
		},
	})

	byID := map[string]leaderboard.RankedPlayer{}
	for _, p := range result.Groups[0].Players {
		byID[p.ID] = p
	}

	if got := byID["f1"].RankDelta; got == nil || *got != 0 {
		t.Errorf("expected f1's movement to be 0 (unopposed within her own gender pool), got %v", got)
	}
}

func TestBuildRanking_DoublesUsesDoublesElo(t *testing.T) {
	players := []*player.Player{
		{ID: "1", FirstName: "A", SinglesElo: 2000, DoublesElo: 1000},
		{ID: "2", FirstName: "B", SinglesElo: 1000, DoublesElo: 2000},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType: "doubles", SortOrder: "points_desc",
	})

	got := result.Groups[0].Players
	if got[0].ID != "2" {
		t.Errorf("expected player 2 (higher doubles Elo) ranked first, got %s", got[0].ID)
	}
}

func TestBuildRanking_InactivePlayersHiddenByDefault(t *testing.T) {
	players := []*player.Player{
		{ID: "1", FirstName: "Active", SinglesElo: 2000},
		{ID: "2", FirstName: "Benched", SinglesElo: 1800, Inactive: true},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc",
	})
	got := result.Groups[0].Players
	if len(got) != 1 || got[0].ID != "1" {
		t.Fatalf("expected only the active player by default, got %+v", got)
	}

	shown := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc", ShowInactive: true,
	})
	gotShown := shown.Groups[0].Players
	if len(gotShown) != 2 {
		t.Fatalf("expected both players when ShowInactive is set, got %+v", gotShown)
	}
}

func TestBuildRanking_PlacementBonus(t *testing.T) {
	players := []*player.Player{
		{ID: "champ", FirstName: "Champ", SinglesElo: 2000},
		{ID: "nobody", FirstName: "Nobody", SinglesElo: 1800},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc",
		PlacementBonus: map[string]float64{"champ": 64},
	})

	got := result.Groups[0].Players
	var champ, nobody *leaderboard.RankedPlayer
	for i := range got {
		switch got[i].ID {
		case "champ":
			champ = &got[i]
		case "nobody":
			nobody = &got[i]
		}
	}
	if champ == nil || champ.PlacementBonus == nil || *champ.PlacementBonus != 64 {
		t.Errorf("expected champ's placement bonus to be 64, got %+v", champ)
	}
	if nobody == nil || nobody.PlacementBonus != nil {
		t.Errorf("expected nobody to have no placement bonus, got %+v", nobody)
	}
}

func genderDivisionFixture() []*division.Division {
	return []*division.Division{
		{ID: "none", Name: "No Division"},
		// Gender-agnostic bands (today's default) must be ignored entirely by
		// BuildGenderRanking, even though their Elo ranges overlap the
		// gender-specific ones below.
		{ID: "div-first", Name: "First Division", MinElo: 1600, MaxElo: nil, Category: "both", Gender: "both"},
		{ID: "div-first-male", Name: "1st Division (Men)", DisplayOrder: 10, MinElo: 2000, MaxElo: nil, Category: "both", Gender: "M"},
		{ID: "div-second-male", Name: "2nd Division (Men)", DisplayOrder: 11, MinElo: 0, MaxElo: eloPtr(2000), Category: "both", Gender: "M"},
		{ID: "div-first-female", Name: "1st Division (Women)", DisplayOrder: 12, MinElo: 1300, MaxElo: nil, Category: "both", Gender: "F"},
		{ID: "div-second-female", Name: "2nd Division (Women)", DisplayOrder: 13, MinElo: 0, MaxElo: eloPtr(1300), Category: "both", Gender: "F"},
	}
}

func TestBuildGenderRanking_DefaultsToMaleWithOwnEnumeration(t *testing.T) {
	// A female player outranks every male by absolute Elo, but the dropdown
	// defaults to Male when no GenderFilter is given, and that gender's
	// ranks must start fresh at 1 -- not reflect its slice of the combined
	// pool (which would start at 2, since the female player is #1 overall).
	players := []*player.Player{
		{ID: "f1", FirstName: "Female", LastName: "Top", Gender: "F", SinglesElo: 2500},
		{ID: "m1", FirstName: "Male", LastName: "High", Gender: "M", SinglesElo: 2100},
		{ID: "m2", FirstName: "Male", LastName: "Low", Gender: "M", SinglesElo: 1800},
	}

	result := leaderboard.BuildGenderRanking(players, genderDivisionFixture(), leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc",
	})

	if !result.IsDivisional {
		t.Fatalf("expected BuildGenderRanking to return a divisional result")
	}
	if len(result.Groups) != 2 {
		t.Fatalf("expected only the two men's groups (1st/2nd) when defaulting to Male, got %d: %+v", len(result.Groups), result.Groups)
	}

	firstDiv, secondDiv := result.Groups[0], result.Groups[1]
	if firstDiv.Division.Name != "1st Division (Men)" || len(firstDiv.Players) != 1 || firstDiv.Players[0].ID != "m1" {
		t.Fatalf("expected 1st Division (Men) to contain only m1, got %+v", firstDiv)
	}
	if firstDiv.Players[0].Rank != 1 {
		t.Errorf("expected m1's own enumeration to start at rank 1, got %d", firstDiv.Players[0].Rank)
	}
	if secondDiv.Division.Name != "2nd Division (Men)" || len(secondDiv.Players) != 1 || secondDiv.Players[0].ID != "m2" {
		t.Fatalf("expected 2nd Division (Men) to contain only m2, got %+v", secondDiv)
	}
	if secondDiv.Players[0].Rank != 2 {
		t.Errorf("expected m2's own enumeration to be rank 2 (not 3, its combined-pool rank), got %d", secondDiv.Players[0].Rank)
	}
}

func TestBuildGenderRanking_FemaleFilter(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "Male", Gender: "M", SinglesElo: 2100},
		{ID: "f1", FirstName: "Female", LastName: "High", Gender: "F", SinglesElo: 1400},
		{ID: "f2", FirstName: "Female", LastName: "Low", Gender: "F", SinglesElo: 1200},
	}

	result := leaderboard.BuildGenderRanking(players, genderDivisionFixture(), leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc", GenderFilter: "F",
	})

	if len(result.Groups) != 2 {
		t.Fatalf("expected only the two women's groups (1st/2nd), got %d: %+v", len(result.Groups), result.Groups)
	}
	if result.Groups[0].Division.Name != "1st Division (Women)" || result.Groups[0].Players[0].ID != "f1" {
		t.Fatalf("expected 1st Division (Women) to contain only f1, got %+v", result.Groups[0])
	}
	if result.Groups[1].Division.Name != "2nd Division (Women)" || result.Groups[1].Players[0].ID != "f2" {
		t.Fatalf("expected 2nd Division (Women) to contain only f2, got %+v", result.Groups[1])
	}
}

func TestBuildGenderRanking_OmitsEmptyGroups(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "Male", Gender: "M", SinglesElo: 2100},
	}

	result := leaderboard.BuildGenderRanking(players, genderDivisionFixture(), leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc", GenderFilter: "M",
	})

	if len(result.Groups) != 1 {
		t.Fatalf("expected only the one non-empty group, got %d: %+v", len(result.Groups), result.Groups)
	}
	if result.Groups[0].Division.Name != "1st Division (Men)" {
		t.Errorf("expected the sole group to be 1st Division (Men), got %+v", result.Groups[0].Division)
	}
}

func TestBuildGenderRanking_SearchAndSortStillApply(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "Alice", LastName: "Smith", Country: "NIC", Gender: "M", SinglesElo: 2100},
		{ID: "m2", FirstName: "Bob", LastName: "Jones", Country: "CUB", Gender: "M", SinglesElo: 2050},
	}

	result := leaderboard.BuildGenderRanking(players, genderDivisionFixture(), leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc", Query: "cub", GenderFilter: "M",
	})

	if len(result.Groups) != 1 || len(result.Groups[0].Players) != 1 || result.Groups[0].Players[0].ID != "m2" {
		t.Fatalf("expected search to narrow to only Bob (country CUB), got %+v", result.Groups)
	}
}
