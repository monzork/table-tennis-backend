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
	// RankDelta is how many positions this player has moved since the Elo
	// snapshot taken before their most recently finished event, holding the
	// rest of the field at its current Elo. Positive means moved up
	// (improved rank number went down); nil means no finished event yet, so
	// movement can't be shown.
	RankDelta *int
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
	GenderFilter   string // "M" | "F" -- only consulted by BuildGenderRanking
	// PreviousElo maps playerID -> Elo held before that player's most
	// recently finished event (see PreviousEloRepository). Optional: a nil
	// map simply means no rank-movement indicator is shown for anyone.
	PreviousElo map[string]int16
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

// previousRankFor estimates the rank a player would hold if they still held
// prevElo, with every other player fixed at their current Elo. sortedElos
// must be all players' current Elo values, descending. This isolates
// movement caused by the player's own last tournament from movement caused
// by the rest of the field -- it is not a reconstruction of the actual
// historical leaderboard (other players' Elo has moved too since then).
func previousRankFor(sortedElos []int16, prevElo int16, currentElo int16) int {
	// Count of elements strictly greater than prevElo: sortedElos is
	// descending, so this is the index of the first element <= prevElo.
	greaterCount := sort.Search(len(sortedElos), func(i int) bool { return sortedElos[i] <= prevElo })
	if currentElo > prevElo {
		// The player's own current entry is in that count; exclude it since
		// we only want how many *other* players are above prevElo.
		greaterCount--
	}
	return greaterCount + 1
}

// rankAndFilter runs the shared pipeline both BuildRanking and
// BuildGenderRanking need: pre-rank the given players by absolute Elo, then
// apply search filtering, optional division filtering, and the requested
// sort order. Rank numbers are 1..N over whatever slice is passed in --
// BuildRanking passes the full combined pool (one shared ranking regardless
// of gender, since Elo is a single shared pool), while BuildGenderRanking
// pre-filters to one gender first so that gender gets its own enumeration.
func rankAndFilter(players []*player.Player, divisions []*division.Division, params RankingParams) []RankedPlayer {
	// 0. Pre-rank all players by absolute Elo.
	var preRanked []RankedPlayer
	sorted := append([]*player.Player{}, players...)
	sort.Slice(sorted, func(i, j int) bool { return eloOf(sorted[i], params.RankType) > eloOf(sorted[j], params.RankType) })

	sortedElos := make([]int16, len(sorted))
	for i, p := range sorted {
		sortedElos[i] = eloOf(p, params.RankType)
	}

	for i, p := range sorted {
		rp := RankedPlayer{Player: p, Rank: i + 1}
		if prevElo, ok := params.PreviousElo[p.ID]; ok {
			prevRank := previousRankFor(sortedElos, prevElo, eloOf(p, params.RankType))
			delta := prevRank - rp.Rank
			rp.RankDelta = &delta
		}
		preRanked = append(preRanked, rp)
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

	return final
}

// BuildRanking applies search filtering, division filtering, Elo-based
// ranking/sorting, and division grouping to a player list.
func BuildRanking(players []*player.Player, divisions []*division.Division, params RankingParams) RankingResult {
	divisions = filterRankableDivisions(divisions)
	final := rankAndFilter(players, divisions, params)

	// The public ranking page shows one flat list -- no division grouping.
	return RankingResult{
		IsDivisional: false,
		Groups:       []DivisionGroupView{{Division: nil, Players: final}},
	}
}

// BuildGenderRanking narrows the pool to one gender (params.GenderFilter,
// defaulting to "M") *before* ranking, so each gender gets its own 1..N
// enumeration instead of keeping its slice of the combined pool's rank
// numbers. The result is grouped into that gender's own division bands
// (Division.Gender matching the filter) -- e.g. "1st Division (Men)" /
// "2nd Division (Men)" -- with empty groups omitted.
func BuildGenderRanking(players []*player.Player, divisions []*division.Division, params RankingParams) RankingResult {
	divisions = filterRankableDivisions(divisions)

	gender := strings.ToUpper(params.GenderFilter)
	if gender != "M" && gender != "F" {
		gender = "M"
	}

	var genderPlayers []*player.Player
	for _, p := range players {
		if strings.EqualFold(p.Gender, gender) {
			genderPlayers = append(genderPlayers, p)
		}
	}
	final := rankAndFilter(genderPlayers, divisions, params)

	var genderDivs []*division.Division
	for _, d := range divisions {
		if strings.EqualFold(d.Gender, gender) {
			genderDivs = append(genderDivs, d)
		}
	}
	sort.Slice(genderDivs, func(i, j int) bool { return genderDivs[i].DisplayOrder < genderDivs[j].DisplayOrder })

	var groups []DivisionGroupView
	for _, d := range genderDivs {
		var divPlayers []RankedPlayer
		for _, rp := range final {
			if d.ContainsElo(eloOf(rp.Player, params.RankType)) {
				divPlayers = append(divPlayers, rp)
			}
		}
		if len(divPlayers) > 0 {
			groups = append(groups, DivisionGroupView{Division: d, Players: divPlayers})
		}
	}

	return RankingResult{
		IsDivisional: true,
		Groups:       groups,
	}
}
