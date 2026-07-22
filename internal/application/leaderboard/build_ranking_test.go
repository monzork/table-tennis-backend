package leaderboard_test

import (
	"testing"

	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
)

func eloPtr(v int16) *int16 { return &v }

func TestBuildRanking_DivisionGrouping(t *testing.T) {
	divisions := []*division.Division{
		{ID: "none", Name: "No Division"}, // must be excluded
		{ID: "low", Name: "Segunda", MinElo: 0, MaxElo: eloPtr(1000), Category: "both"},
		{ID: "high", Name: "Primera", MinElo: 1000, MaxElo: nil, Category: "both"},
	}
	players := []*player.Player{
		{ID: "1", FirstName: "A", Gender: "M", SinglesElo: 900},
		{ID: "2", FirstName: "B", Gender: "M", SinglesElo: 1000}, // boundary: belongs to Primera, not Segunda
		{ID: "3", FirstName: "C", Gender: "M", SinglesElo: 1500},
	}

	result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
		RankType:  "singles",
		Gender:    "M",
		SortOrder: "points_desc",
	})

	if !result.IsDivisional {
		t.Fatalf("expected divisional grouping to be active")
	}
	if len(result.Groups) != 2 {
		t.Fatalf("expected 2 non-empty groups (No Division excluded), got %d", len(result.Groups))
	}
	segunda := result.Groups[0]
	if segunda.Division.Name != "Segunda" || len(segunda.Players) != 1 || segunda.Players[0].ID != "1" {
		t.Errorf("expected Segunda to contain only player 1 (elo 900), got %+v", segunda)
	}
	primera := result.Groups[1]
	if primera.Division.Name != "Primera" || len(primera.Players) != 2 {
		t.Errorf("expected Primera to contain players 2 and 3 (elo >= 1000), got %+v", primera)
	}
}

func TestBuildRanking_SearchQuery(t *testing.T) {
	players := []*player.Player{
		{ID: "1", FirstName: "Alice", LastName: "Smith", Country: "NIC", Gender: "F", SinglesElo: 1200},
		{ID: "2", FirstName: "Bob", LastName: "Jones", Country: "CUB", Gender: "F", SinglesElo: 1100},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		Gender:    "F",
		Query:     "cub",
		SortOrder: "points_desc",
	})

	if len(result.Groups) != 1 || len(result.Groups[0].Players) != 1 || result.Groups[0].Players[0].ID != "2" {
		t.Errorf("expected only Bob (country CUB) to match query, got %+v", result.Groups)
	}
}

func TestBuildRanking_SortOrders(t *testing.T) {
	players := []*player.Player{
		{ID: "1", FirstName: "Zed", Gender: "M", SinglesElo: 1000},
		{ID: "2", FirstName: "Amy", Gender: "M", SinglesElo: 1500},
	}

	t.Run("points_asc", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
			RankType: "singles", Gender: "M", SortOrder: "points_asc",
		})
		got := result.Groups[0].Players
		if got[0].ID != "1" || got[1].ID != "2" {
			t.Errorf("expected ascending Elo order [1,2], got [%s,%s]", got[0].ID, got[1].ID)
		}
	})

	t.Run("name_asc", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
			RankType: "singles", Gender: "M", SortOrder: "name_asc",
		})
		got := result.Groups[0].Players
		if got[0].ID != "2" || got[1].ID != "1" {
			t.Errorf("expected name order [Amy, Zed], got [%s,%s]", got[0].FirstName, got[1].FirstName)
		}
	})
}

func TestBuildRanking_MixedGenderSplitsByRankWithinGender(t *testing.T) {
	players := []*player.Player{
		{ID: "m1", FirstName: "M1", Gender: "M", SinglesElo: 2000},
		{ID: "f1", FirstName: "F1", Gender: "F", SinglesElo: 1000},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType:  "singles",
		Gender:    "",
		SortOrder: "points_desc",
	})

	if !result.IsMixed {
		t.Fatalf("expected mixed result for empty gender filter")
	}
	if len(result.MenGroups) != 1 || result.MenGroups[0].Players[0].Rank != 1 {
		t.Errorf("expected men ranked independently starting at 1, got %+v", result.MenGroups)
	}
	if len(result.WomenGroups) != 1 || result.WomenGroups[0].Players[0].Rank != 1 {
		t.Errorf("expected women ranked independently starting at 1, got %+v", result.WomenGroups)
	}
}

func TestBuildRanking_DivisionFilter(t *testing.T) {
	divisions := []*division.Division{
		{ID: "low", Name: "Segunda", MinElo: 0, MaxElo: eloPtr(1000), Category: "both"},
		{ID: "high", Name: "Primera", MinElo: 1000, Category: "both"},
	}
	players := []*player.Player{
		{ID: "1", FirstName: "A", Gender: "M", SinglesElo: 900},
		{ID: "2", FirstName: "B", Gender: "M", SinglesElo: 1500},
	}

	t.Run("all keeps every player", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
			RankType: "singles", Gender: "M", SortOrder: "points_desc", DivisionFilter: "all",
		})
		if len(result.Groups) != 2 {
			t.Errorf("expected both divisions populated, got %d groups", len(result.Groups))
		}
	})

	t.Run("named filter narrows to one division", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
			RankType: "singles", Gender: "M", SortOrder: "points_desc", DivisionFilter: "Primera",
		})
		if len(result.Groups) != 1 || result.Groups[0].Players[0].ID != "2" {
			t.Errorf("expected only Primera group with player 2, got %+v", result.Groups)
		}
	})

	t.Run("unknown filter name keeps all players unfiltered", func(t *testing.T) {
		result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
			RankType: "singles", Gender: "M", SortOrder: "points_desc", DivisionFilter: "Nonexistent",
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
		{ID: "1", FirstName: "A", Gender: "M", SinglesElo: 2000, DoublesElo: 1000},
		{ID: "2", FirstName: "B", Gender: "M", SinglesElo: 1000, DoublesElo: 2000},
	}

	result := leaderboard.BuildRanking(players, nil, leaderboard.RankingParams{
		RankType: "doubles", Gender: "M", SortOrder: "points_desc",
	})

	got := result.Groups[0].Players
	if got[0].ID != "2" {
		t.Errorf("expected player 2 (higher doubles Elo) ranked first, got %s", got[0].ID)
	}
}
