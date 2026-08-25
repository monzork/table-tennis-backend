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

func TestBuildGenderRanking_SplitsByGenderAndDivision(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "Male", LastName: "High", Gender: "M", SinglesElo: 2100},
		{ID: "m2", FirstName: "Male", LastName: "Low", Gender: "M", SinglesElo: 1800},
		{ID: "f1", FirstName: "Female", LastName: "High", Gender: "F", SinglesElo: 1400},
		{ID: "f2", FirstName: "Female", LastName: "Low", Gender: "F", SinglesElo: 1200},
	}

	result := leaderboard.BuildGenderRanking(players, genderDivisionFixture(), leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc",
	})

	if !result.IsDivisional {
		t.Fatalf("expected BuildGenderRanking to return a divisional result")
	}
	if len(result.Groups) != 4 {
		t.Fatalf("expected 4 non-empty groups (1st/2nd x M/F), got %d: %+v", len(result.Groups), result.Groups)
	}

	wantOrder := []struct {
		divisionName string
		playerID     string
	}{
		{"1st Division (Men)", "m1"},
		{"2nd Division (Men)", "m2"},
		{"1st Division (Women)", "f1"},
		{"2nd Division (Women)", "f2"},
	}
	for i, want := range wantOrder {
		g := result.Groups[i]
		if g.Division == nil || g.Division.Name != want.divisionName {
			t.Fatalf("group %d: expected division %q, got %+v", i, want.divisionName, g.Division)
		}
		if len(g.Players) != 1 || g.Players[0].ID != want.playerID {
			t.Fatalf("group %d (%s): expected only player %q, got %+v", i, want.divisionName, want.playerID, g.Players)
		}
	}
}

func TestBuildGenderRanking_OmitsEmptyGroups(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "Male", Gender: "M", SinglesElo: 2100},
	}

	result := leaderboard.BuildGenderRanking(players, genderDivisionFixture(), leaderboard.RankingParams{
		RankType: "singles", SortOrder: "points_desc",
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
		RankType: "singles", SortOrder: "points_desc", Query: "cub",
	})

	if len(result.Groups) != 1 || len(result.Groups[0].Players) != 1 || result.Groups[0].Players[0].ID != "m2" {
		t.Fatalf("expected search to narrow to only Bob (country CUB), got %+v", result.Groups)
	}
}
