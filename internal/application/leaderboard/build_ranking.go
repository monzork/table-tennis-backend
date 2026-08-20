package leaderboard

import (
	"sort"
	"strings"

	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
)

type RankedPlayer struct {
	*player.Player
	Rank int
}

type DivisionGroupView struct {
	Division *division.Division
	Players  []RankedPlayer
}

type RankingParams struct {
	RankType       string // "singles" | "doubles"
	Query          string
	DivisionFilter string
	SortOrder      string // "points_desc" | "points_asc" | "name_asc"
}

type RankingResult struct {
	IsDivisional bool
	Groups       []DivisionGroupView
}

// eloOf returns the relevant Elo rating for a player given the ranking type.
func eloOf(p *player.Player, rankType string) int16 {
	if rankType == "doubles" {
		return p.DoublesElo
	}
	return p.SinglesElo
}

// filterRankableDivisions drops the placeholder "no division" bucket, which
// never participates in ranked groupings.
func filterRankableDivisions(divisions []*division.Division) []*division.Division {
	var out []*division.Division
	for _, d := range divisions {
		if d.ID != "none" && d.Name != "No Division" {
			out = append(out, d)
		}
	}
	return out
}

// BuildRanking applies search filtering, division filtering, Elo-based
// ranking/sorting, and division grouping to a player list.
func BuildRanking(players []*player.Player, divisions []*division.Division, params RankingParams) RankingResult {
	divisions = filterRankableDivisions(divisions)

	// 0. Pre-rank all players by absolute Elo -- one combined ranking
	// regardless of gender, since both singles and doubles Elo are each a
	// single shared pool.
	var preRanked []RankedPlayer
	sorted := append([]*player.Player{}, players...)
	sort.Slice(sorted, func(i, j int) bool { return eloOf(sorted[i], params.RankType) > eloOf(sorted[j], params.RankType) })
	for i, p := range sorted {
		preRanked = append(preRanked, RankedPlayer{Player: p, Rank: i + 1})
	}

	// 1. Filter by search query (name, country, or department).
	filtered := preRanked
	if params.Query != "" {
		qUpper := strings.ToUpper(params.Query)
		filtered = nil
		for _, rp := range preRanked {
			fullName := strings.ToUpper(rp.FirstName + " " + rp.LastName)
			country := strings.ToUpper(rp.Country)
			dept := strings.ToUpper(rp.Department)
			if strings.Contains(fullName, qUpper) || strings.Contains(country, qUpper) || strings.Contains(dept, qUpper) {
				filtered = append(filtered, rp)
			}
		}
	}

	// 2. Filter by division.
	final := filtered
	if params.DivisionFilter != "" && params.DivisionFilter != "all" {
		var targetDiv *division.Division
		for _, d := range divisions {
			if d.Name == params.DivisionFilter {
				targetDiv = d
				break
			}
		}
		if targetDiv != nil {
			final = nil
			for _, rp := range filtered {
				if targetDiv.ContainsElo(eloOf(rp.Player, params.RankType)) {
					final = append(final, rp)
				}
			}
		}
	}

	// 3. Sort by the requested order.
	sort.Slice(final, func(i, j int) bool {
		a, b := final[i], final[j]
		if params.SortOrder == "name_asc" {
			return (a.FirstName + a.LastName) < (b.FirstName + b.LastName)
		}
		ptsA, ptsB := eloOf(a.Player, params.RankType), eloOf(b.Player, params.RankType)
		if ptsA == ptsB {
			if params.SortOrder == "points_asc" {
				return a.Rank > b.Rank
			}
			return a.Rank < b.Rank
		}
		if params.SortOrder == "points_asc" {
			return ptsA < ptsB
		}
		return ptsA > ptsB // default points_desc
	})

	// The public ranking page shows one flat list -- no division grouping.
	return RankingResult{
		IsDivisional: false,
		Groups:       []DivisionGroupView{{Division: nil, Players: final}},
	}
}
