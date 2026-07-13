package tournament

import (
	"fmt"
	"math"
	"sort"
	"table-tennis-backend/internal/domain/player"
)

type PlayerStanding struct {
	Rank          int
	Player        *player.Player
	Played        int
	Wins          int
	Losses        int
	WinRate       float64
	WinPercentage string
	SetsWon       int
	SetsLost      int
	PointsWon     int
	PointsLost    int
}

// BuildStandings calculates the group standings for a list of players and matches in accordance with ITTF tiebreaker rules.
func BuildStandings(players []*player.Player, matches []Match) []PlayerStanding {
	stats := make([]PlayerStanding, len(players))
	for i, p := range players {
		wins, losses, setsWon, setsLost, ptsWon, ptsLost := buildMatchStats(p, matches)
		played := wins + losses
		winRate := 0.0
		winPercentage := "0"
		if played > 0 {
			winRate = float64(wins) / float64(played)
			winPercentage = fmt.Sprintf("%.0f", winRate*100)
		}
		stats[i] = PlayerStanding{
			Player:        p,
			Played:        played,
			Wins:          wins,
			Losses:        losses,
			WinRate:       winRate,
			WinPercentage: winPercentage,
			SetsWon:       setsWon,
			SetsLost:      setsLost,
			PointsWon:     ptsWon,
			PointsLost:    ptsLost,
		}
	}

	// Group players by overall Wins count
	groups := make(map[int][]*PlayerStanding)
	var uniqueWins []int
	for idx := range stats {
		w := stats[idx].Wins
		if _, exists := groups[w]; !exists {
			uniqueWins = append(uniqueWins, w)
		}
		groups[w] = append(groups[w], &stats[idx])
	}
	sort.Slice(uniqueWins, func(i, j int) bool {
		return uniqueWins[i] > uniqueWins[j]
	})

	var sortedStandings []PlayerStanding
	for _, w := range uniqueWins {
		resolvedGroup := ResolveITTFTies(groups[w], matches, 0)
		for _, ps := range resolvedGroup {
			sortedStandings = append(sortedStandings, *ps)
		}
	}

	for i := range sortedStandings {
		sortedStandings[i].Rank = i + 1
	}
	return sortedStandings
}

// ResolveITTFTies recursively resolves ties among players who are currently equal.
func ResolveITTFTies(tied []*PlayerStanding, allMatches []Match, depth int) []*PlayerStanding {
	if len(tied) <= 1 {
		return tied
	}

	// 1. Isolate players
	var players []*player.Player
	for _, ts := range tied {
		players = append(players, ts.Player)
	}

	// 2. Isolate matches between only these players (H2H subset)
	h2hMatches := matchesBetween(players, allMatches)

	// Calculate H2H wins for each tied player
	statsMap := make(map[interface{}]struct {
		wins     int
		setsWon  int
		setsLost int
		ptsWon   int
		ptsLost  int
	})
	for _, p := range players {
		statsMap[p.ID] = struct {
			wins     int
			setsWon  int
			setsLost int
			ptsWon   int
			ptsLost  int
		}{}
	}

	for _, m := range h2hMatches {
		if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
			continue
		}
		idA := m.TeamA[0].ID
		idB := m.TeamB[0].ID

		sA := statsMap[idA]
		sB := statsMap[idB]

		if m.WinnerTeam == "A" {
			sA.wins++
		} else {
			sB.wins++
		}

		matchScoreA := m.ScoreA()
		matchScoreB := m.ScoreB()

		sA.setsWon += matchScoreA
		sA.setsLost += matchScoreB
		sB.setsWon += matchScoreB
		sB.setsLost += matchScoreA

		for _, s := range m.Sets {
			sA.ptsWon += s.ScoreA
			sA.ptsLost += s.ScoreB

			sB.ptsWon += s.ScoreB
			sB.ptsLost += s.ScoreA
		}

		statsMap[idA] = sA
		statsMap[idB] = sB
	}

	// Criterion 1: Match Wins in H2H isolated subset
	hasWinDiff := false
	firstWins := statsMap[tied[0].Player.ID].wins
	for _, ts := range tied {
		if statsMap[ts.Player.ID].wins != firstWins {
			hasWinDiff = true
			break
		}
	}

	if hasWinDiff {
		// Group players by Wins descending
		groups := make(map[int][]*PlayerStanding)
		var uniqueWins []int
		for _, ts := range tied {
			w := statsMap[ts.Player.ID].wins
			if _, exists := groups[w]; !exists {
				uniqueWins = append(uniqueWins, w)
			}
			groups[w] = append(groups[w], ts)
		}
		sort.Slice(uniqueWins, func(i, j int) bool {
			return uniqueWins[i] > uniqueWins[j]
		})

		var result []*PlayerStanding
		for _, w := range uniqueWins {
			resolvedGroup := ResolveITTFTies(groups[w], allMatches, depth+1)
			result = append(result, resolvedGroup...)
		}
		return result
	}

	// Criterion 2: Set Ratio in H2H isolated subset
	getSetRatio := func(pID interface{}) float64 {
		s := statsMap[pID]
		if s.setsLost == 0 {
			if s.setsWon > 0 {
				return float64(s.setsWon) + 1000.0 // representation of infinity for sorting
			}
			return 0.0
		}
		return float64(s.setsWon) / float64(s.setsLost)
	}

	hasSetRatioDiff := false
	firstSetRatio := getSetRatio(tied[0].Player.ID)
	for _, ts := range tied {
		if math.Abs(getSetRatio(ts.Player.ID)-firstSetRatio) > 1e-9 {
			hasSetRatioDiff = true
			break
		}
	}

	if hasSetRatioDiff {
		// Group players by Set Ratio descending
		type ratioGroup struct {
			ratio float64
			items []*PlayerStanding
		}
		var rGroups []ratioGroup
		for _, ts := range tied {
			r := getSetRatio(ts.Player.ID)
			found := false
			for idx := range rGroups {
				if math.Abs(rGroups[idx].ratio-r) < 1e-9 {
					rGroups[idx].items = append(rGroups[idx].items, ts)
					found = true
					break
				}
			}
			if !found {
				rGroups = append(rGroups, ratioGroup{ratio: r, items: []*PlayerStanding{ts}})
			}
		}
		sort.Slice(rGroups, func(i, j int) bool {
			return rGroups[i].ratio > rGroups[j].ratio
		})

		var result []*PlayerStanding
		for _, rg := range rGroups {
			resolvedGroup := ResolveITTFTies(rg.items, allMatches, depth+1)
			result = append(result, resolvedGroup...)
		}
		return result
	}

	// Criterion 3: Point Ratio in H2H subset
	getPtRatio := func(pID interface{}) float64 {
		s := statsMap[pID]
		if s.ptsLost == 0 {
			if s.ptsWon > 0 {
				return float64(s.ptsWon) + 1000.0
			}
			return 0.0
		}
		return float64(s.ptsWon) / float64(s.ptsLost)
	}

	hasPtRatioDiff := false
	firstPtRatio := getPtRatio(tied[0].Player.ID)
	for _, ts := range tied {
		if math.Abs(getPtRatio(ts.Player.ID)-firstPtRatio) > 1e-9 {
			hasPtRatioDiff = true
			break
		}
	}

	if hasPtRatioDiff {
		// Group players by Point Ratio descending
		type ratioGroup struct {
			ratio float64
			items []*PlayerStanding
		}
		var rGroups []ratioGroup
		for _, ts := range tied {
			r := getPtRatio(ts.Player.ID)
			found := false
			for idx := range rGroups {
				if math.Abs(rGroups[idx].ratio-r) < 1e-9 {
					rGroups[idx].items = append(rGroups[idx].items, ts)
					found = true
					break
				}
			}
			if !found {
				rGroups = append(rGroups, ratioGroup{ratio: r, items: []*PlayerStanding{ts}})
			}
		}
		sort.Slice(rGroups, func(i, j int) bool {
			return rGroups[i].ratio > rGroups[j].ratio
		})

		var result []*PlayerStanding
		for _, rg := range rGroups {
			resolvedGroup := ResolveITTFTies(rg.items, allMatches, depth+1)
			result = append(result, resolvedGroup...)
		}
		return result
	}

	// If completely tied across everything, resolve stably by original Elo
	sort.SliceStable(tied, func(i, j int) bool {
		return tied[i].Player.SinglesElo > tied[j].Player.SinglesElo
	})

	return tied
}

// buildMatchStats computes wins, sets won/lost, and points won/lost for a player
func buildMatchStats(p *player.Player, matches []Match) (wins, losses, setsWon, setsLost, ptsWon, ptsLost int) {
	for _, m := range matches {
		if m.Status != "finished" {
			continue
		}
		if m.Stage != "group" {
			continue
		}
		var isA, isB bool
		if len(m.TeamA) > 0 {
			isA = m.TeamA[0].ID == p.ID
		}
		if len(m.TeamB) > 0 {
			isB = m.TeamB[0].ID == p.ID
		}
		if !isA && !isB {
			continue
		}
		if (isA && m.WinnerTeam == "A") || (isB && m.WinnerTeam == "B") {
			wins++
		} else {
			losses++
		}
		matchScoreA := m.ScoreA()
		matchScoreB := m.ScoreB()

		if isA {
			setsWon += matchScoreA
			setsLost += matchScoreB
		} else {
			setsWon += matchScoreB
			setsLost += matchScoreA
		}

		for _, s := range m.Sets {
			if isA {
				ptsWon += s.ScoreA
				ptsLost += s.ScoreB
			} else {
				ptsWon += s.ScoreB
				ptsLost += s.ScoreA
			}
		}
	}
	return
}

func matchesBetween(players []*player.Player, matches []Match) []Match {
	idSet := make(map[interface{}]bool)
	for _, p := range players {
		idSet[p.ID] = true
	}
	var result []Match
	for _, m := range matches {
		if m.Status != "finished" {
			continue
		}
		if m.Stage != "group" {
			continue
		}
		if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
			continue
		}
		if idSet[m.TeamA[0].ID] && idSet[m.TeamB[0].ID] {
			result = append(result, m)
		}
	}
	return result
}
