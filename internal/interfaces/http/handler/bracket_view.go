package handler

import (
	"fmt"
	"math"
	"sort"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"strings"

	"github.com/google/uuid"
)

type TournamentViewModel struct {
	Tournament *tournament.Tournament
	Type       string
	Format     string
	Divisions  []DivisionView
}

type DivisionView struct {
	Name           string
	Color          string
	MinElo         int16
	MaxElo         *int16
	IsUnclassified bool
	Players        []*player.Player

	Format            string
	Standings         []PlayerStanding
	RoundRobinMatches []MatchView

	Groups            []GroupView
	AllGroupsFinished bool

	KnockoutRounds []RoundView
}

type PlayerStanding struct {
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

type GroupView struct {
	ID        uuid.UUID
	Name      string
	Players   []*player.Player
	Standings []PlayerStanding
	Matches   []MatchView
	Finished  bool
}

type MatchView struct {
	Player1 *player.Player
	Player2 *player.Player
	Match   *tournament.Match
	Stage   string
	BestOf  int
}

type RoundView struct {
	Name    string
	Matches []BracketMatchView
}

type BracketMatchView struct {
	Player1 *MatchSlot
	Player2 *MatchSlot
	Match   *tournament.Match
	Stage   string
	BestOf  int
}

type MatchSlot struct {
	Seed   int
	Player *player.Player
}

func BuildTournamentViewModel(t *tournament.Tournament, divs []*division.Division) *TournamentViewModel {
	vm := &TournamentViewModel{
		Tournament: t,
		Type:       t.Type,
		Format:     t.Format,
		Divisions:  []DivisionView{},
	}

	var participants []*player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		participants = make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			avgElo := int16(1000)
			if len(team.Players) > 0 {
				sum := int32(0)
				for _, p := range team.Players {
					if t.Type == "doubles" || t.Type == "mixed_doubles" {
						sum += int32(p.DoublesElo)
					} else {
						sum += int32(p.SinglesElo)
					}
				}
				avgElo = int16(sum / int32(len(team.Players)))
			}
			participants[i] = &player.Player{
				ID:         team.ID,
				FirstName:  team.Name,
				LastName:   " (Team)",
				SinglesElo: avgElo,
				DoublesElo: avgElo,
			}
		}
	} else {
		participants = make([]*player.Player, len(t.Participants))
		copy(participants, t.Participants)
	}

	// Sort participants by correct Elo
	sort.Slice(participants, func(i, j int) bool {
		ei := participants[i].SinglesElo
		ej := participants[j].SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			ei = participants[i].DoublesElo
			ej = participants[j].DoublesElo
		}
		return ei > ej
	})

	if t.SkipElo {
		openPlayers := participants
		if t.Format == "elimination" {
			var matchingGroup *tournament.Group
			for i := range t.Groups {
				if t.Groups[i].Name == "Open Bracket - Bracket Draw" {
					matchingGroup = &t.Groups[i]
					break
				}
			}
			if matchingGroup != nil && len(matchingGroup.Players) > 0 {
				openPlayers = matchingGroup.Players
			}
		}
		vm.Divisions = append(vm.Divisions, buildDivisionView(t, "Open Bracket", "", 0, nil, false, openPlayers))
		return vm
	}

	assignedMap := make(map[uuid.UUID]bool)

	// Valid divisions for tournament type
	var validDivs []*division.Division
	for _, d := range divs {
		if d.Category == "both" || d.Category == t.Type {
			validDivs = append(validDivs, d)
		}
	}

	for _, d := range validDivs {
		var dPlayers []*player.Player
		name := d.Name
		if strings.HasSuffix(strings.ToLower(name), " division") {
			name = name[:len(name)-9]
		}

		if t.Format == "elimination" {
			var matchingGroup *tournament.Group
			expectedGroupName := fmt.Sprintf("%s - Bracket Draw", name)
			for i := range t.Groups {
				if t.Groups[i].Name == expectedGroupName {
					matchingGroup = &t.Groups[i]
					break
				}
			}
			if matchingGroup != nil && len(matchingGroup.Players) > 0 {
				dPlayers = matchingGroup.Players
				for _, p := range dPlayers {
					assignedMap[p.ID] = true
				}
			}
		}

		if len(dPlayers) == 0 {
			for _, p := range participants {
				if assignedMap[p.ID] {
					continue
				}
				elo := p.SinglesElo
				if t.Type == "doubles" || t.Type == "mixed_doubles" {
					elo = p.DoublesElo
				}
				if elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
					dPlayers = append(dPlayers, p)
					assignedMap[p.ID] = true
				}
			}
		}

		if len(dPlayers) > 0 {
			vm.Divisions = append(vm.Divisions, buildDivisionView(t, name, d.Color, d.MinElo, d.MaxElo, false, dPlayers))
		}
	}

	var unassigned []*player.Player
	for _, p := range participants {
		if !assignedMap[p.ID] {
			unassigned = append(unassigned, p)
		}
	}

	if len(unassigned) > 0 {
		vm.Divisions = append(vm.Divisions, buildDivisionView(t, "Unclassified", "", 0, nil, true, unassigned))
	}

	return vm
}

func getBestOfForStage(t *tournament.Tournament, stage string) int {
	for _, r := range t.StageRules {
		if r.Stage == stage {
			return r.BestOf
		}
	}
	return 5
}

func buildDivisionView(t *tournament.Tournament, name, color string, minElo int16, maxElo *int16, unclassified bool, players []*player.Player) DivisionView {
	dv := DivisionView{
		Name:           name,
		Color:          color,
		MinElo:         minElo,
		MaxElo:         maxElo,
		IsUnclassified: unclassified,
		Players:        players,
		Format:         t.Format,
	}

	if t.Format == "round_robin" {
		dv.Standings = buildStandings(players, t.Matches)
		dv.RoundRobinMatches = buildRRMatches(t, players, "group")
	} else if t.Format == "groups_elimination" {
		dv.Groups, dv.AllGroupsFinished = buildGroupEliminationGroups(t, players)
		
		if dv.AllGroupsFinished {
			var advancing []*player.Player
			for _, g := range dv.Groups {
				take := t.GroupPassCount
				if take == 0 {
					take = 2
				}
				if take > len(g.Standings) {
					take = len(g.Standings)
				}
				for i := 0; i < take; i++ {
					advancing = append(advancing, g.Standings[i].Player)
				}
			}
			sort.Slice(advancing, func(i, j int) bool {
				ei := advancing[i].SinglesElo
				ej := advancing[j].SinglesElo
				if t.Type == "doubles" {
					ei = advancing[i].DoublesElo
					ej = advancing[j].DoublesElo
				}
				return ei > ej
			})
			dv.KnockoutRounds = buildBracketRounds(t, advancing)
		}
	} else {
		dv.KnockoutRounds = buildBracketRounds(t, players)
	}

	return dv
}

// buildMatchStats computes wins, sets won/lost, and points won/lost for a player
// across only the provided matches (used for both full-group and head-to-head tiebreakers).
func buildMatchStats(p *player.Player, matches []tournament.Match) (wins, losses, setsWon, setsLost, ptsWon, ptsLost int) {
	for _, m := range matches {
		if m.Status != "finished" {
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
		for _, s := range m.Sets {
			if isA {
				setsWon += s.ScoreA
				setsLost += s.ScoreB
				ptsWon += s.ScoreA
				ptsLost += s.ScoreB
			} else {
				setsWon += s.ScoreB
				setsLost += s.ScoreA
				ptsWon += s.ScoreB
				ptsLost += s.ScoreA
			}
		}
	}
	return
}

// matchesBetween returns matches that involve only players from the given set.
func matchesBetween(players []*player.Player, matches []tournament.Match) []tournament.Match {
	idSet := make(map[interface{}]bool)
	for _, p := range players {
		idSet[p.ID] = true
	}
	var result []tournament.Match
	for _, m := range matches {
		if m.Status != "finished" {
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

// ittfLess returns true if standing i ranks ABOVE standing j using ITTF tiebreaker criteria.
// Criteria applied in order among tied players (already filtered to head-to-head matches):
// 1. More wins
// 2. Better set ratio (won/lost)
// 3. Better point ratio (won/lost)
func ittfLess(si, sj PlayerStanding, allMatches []tournament.Match) bool {
	// Step 1: Match wins
	if si.Wins != sj.Wins {
		return si.Wins > sj.Wins
	}

	// Steps 2-3 require head-to-head matches between these two players
	h2hMatches := matchesBetween([]*player.Player{si.Player, sj.Player}, allMatches)

	var iWins, jWins int
	var iSetsWon, iSetsLost, jSetsWon, jSetsLost int
	var iPtsWon, iPtsLost, jPtsWon, jPtsLost int

	for _, m := range h2hMatches {
		iIsA := len(m.TeamA) > 0 && m.TeamA[0].ID == si.Player.ID
		if (iIsA && m.WinnerTeam == "A") || (!iIsA && m.WinnerTeam == "B") {
			iWins++
		} else {
			jWins++
		}
		for _, s := range m.Sets {
			if iIsA {
				iSetsWon += s.ScoreA; iSetsLost += s.ScoreB
				jSetsWon += s.ScoreB; jSetsLost += s.ScoreA
				iPtsWon += s.ScoreA; iPtsLost += s.ScoreB
				jPtsWon += s.ScoreB; jPtsLost += s.ScoreA
			} else {
				iSetsWon += s.ScoreB; iSetsLost += s.ScoreA
				jSetsWon += s.ScoreA; jSetsLost += s.ScoreB
				iPtsWon += s.ScoreB; iPtsLost += s.ScoreA
				jPtsWon += s.ScoreA; jPtsLost += s.ScoreB
			}
		}
	}

	// Step 2: H2H wins
	if iWins != jWins {
		return iWins > jWins
	}

	// Step 3: Set ratio
	iSetRatio := 0.0
	jSetRatio := 0.0
	if iSetsLost > 0 {
		iSetRatio = float64(iSetsWon) / float64(iSetsLost)
	} else if iSetsWon > 0 {
		iSetRatio = float64(iSetsWon) + 1.0
	}
	if jSetsLost > 0 {
		jSetRatio = float64(jSetsWon) / float64(jSetsLost)
	} else if jSetsWon > 0 {
		jSetRatio = float64(jSetsWon) + 1.0
	}
	if iSetRatio != jSetRatio {
		return iSetRatio > jSetRatio
	}

	// Step 4: Point ratio
	iPtRatio := 0.0
	jPtRatio := 0.0
	if iPtsLost > 0 {
		iPtRatio = float64(iPtsWon) / float64(iPtsLost)
	} else if iPtsWon > 0 {
		iPtRatio = float64(iPtsWon) + 1.0
	}
	if jPtsLost > 0 {
		jPtRatio = float64(jPtsWon) / float64(jPtsLost)
	} else if jPtsWon > 0 {
		jPtRatio = float64(jPtsWon) + 1.0
	}
	return iPtRatio > jPtRatio
}
// resolveITTFTies recursively resolves ties among players who are currently equal.
// It returns a sorted slice of *PlayerStanding.
func resolveITTFTies(tied []*PlayerStanding, allMatches []tournament.Match, depth int) []*PlayerStanding {
	if len(tied) <= 1 {
		return tied
	}

	// 1. Isolate players
	var players []*player.Player
	for _, ts := range tied {
		players = append(players, ts.Player)
	}

	// 2. Find matches played ONLY between these tied players
	h2hMatches := matchesBetween(players, allMatches)

	// Compute H2H subset statistics for each player in the tied slice
	type h2hStats struct {
		wins     int
		setsWon  int
		setsLost int
		ptsWon   int
		ptsLost  int
	}
	statsMap := make(map[interface{}]h2hStats)
	for _, ts := range tied {
		w, _, sw, sl, pw, pl := buildMatchStats(ts.Player, h2hMatches)
		statsMap[ts.Player.ID] = h2hStats{
			wins:     w,
			setsWon:  sw,
			setsLost: sl,
			ptsWon:   pw,
			ptsLost:  pl,
		}
	}

	// Criterion 1: Match Wins in H2H subset
	hasWinsDiff := false
	firstWins := statsMap[tied[0].Player.ID].wins
	for _, ts := range tied {
		if statsMap[ts.Player.ID].wins != firstWins {
			hasWinsDiff = true
			break
		}
	}

	if hasWinsDiff {
		// Group players by H2H wins descending
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
			resolvedGroup := resolveITTFTies(groups[w], allMatches, depth+1)
			result = append(result, resolvedGroup...)
		}
		return result
	}

	// Criterion 2: Set Ratio in H2H subset
	getSetRatio := func(pID interface{}) float64 {
		s := statsMap[pID]
		if s.setsLost == 0 {
			if s.setsWon > 0 {
				return float64(s.setsWon) + 1000.0
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
			resolvedGroup := resolveITTFTies(rg.items, allMatches, depth+1)
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
			resolvedGroup := resolveITTFTies(rg.items, allMatches, depth+1)
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

func buildStandings(players []*player.Player, matches []tournament.Match) []PlayerStanding {
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
		resolvedGroup := resolveITTFTies(groups[w], matches, 0)
		for _, ps := range resolvedGroup {
			sortedStandings = append(sortedStandings, *ps)
		}
	}

	return sortedStandings
}
func buildRRMatches(t *tournament.Tournament, players []*player.Player, stage string) []MatchView {
	var results []MatchView
	bestOf := getBestOfForStage(t, stage)
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			p1 := players[i]
			p2 := players[j]
			var found *tournament.Match
			for k := range t.Matches {
				m := t.Matches[k]
				if m.TeamMatchID != nil {
					continue
				}
				if m.Stage != stage {
					continue
				}
				if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
					if (m.TeamA[0].ID == p1.ID && m.TeamB[0].ID == p2.ID) || (m.TeamA[0].ID == p2.ID && m.TeamB[0].ID == p1.ID) {
						found = &t.Matches[k]
						break
					}
				}
			}
			results = append(results, MatchView{
				Player1: p1,
				Player2: p2,
				Match:   found,
				Stage:   stage,
				BestOf:  bestOf,
			})
		}
	}
	return results
}

func buildGroupEliminationGroups(t *tournament.Tournament, players []*player.Player) ([]GroupView, bool) {
	// Try to load saved groups containing any players in this division first
	var divisionGroups []tournament.Group
	for _, g := range t.Groups {
		hasPlayer := false
		for _, gp := range g.Players {
			for _, dp := range players {
				if gp.ID == dp.ID {
					hasPlayer = true
					break
				}
			}
			if hasPlayer {
				break
			}
		}
		if hasPlayer {
			divisionGroups = append(divisionGroups, g)
		}
	}

	if len(divisionGroups) > 0 {
		allFinished := true
		var views []GroupView
		for _, g := range divisionGroups {
			expectedMatches := len(g.Players) * (len(g.Players) - 1) / 2
			finished := 0
			for _, m := range t.Matches {
				if len(m.TeamA) == 0 || len(m.TeamB) == 0 { continue }
				for _, p1 := range g.Players {
					for _, p2 := range g.Players {
						if p1.ID != p2.ID {
							if (m.TeamA[0].ID == p1.ID && m.TeamB[0].ID == p2.ID) || (m.TeamA[0].ID == p2.ID && m.TeamB[0].ID == p1.ID) {
								if m.Status == "finished" {
									finished++
								}
							}
						}
					}
				}
			}
			finished = finished / 2
			isFinished := expectedMatches > 0 && finished >= expectedMatches
			if !isFinished {
				allFinished = false
			}

			displayName := g.Name
			if idx := strings.Index(g.Name, " - "); idx != -1 {
				displayName = g.Name[idx+3:]
			}

			views = append(views, GroupView{
				ID:        g.ID,
				Name:      displayName,
				Players:   g.Players,
				Standings: buildStandings(g.Players, t.Matches),
				Matches:   buildRRMatches(t, g.Players, "group"),
				Finished:  isFinished,
			})
		}
		return views, allFinished
	}

	// Fallback to snake seeding
	groupSize := 4
	numGroups := int(math.Ceil(float64(len(players)) / float64(groupSize)))
	if numGroups == 0 {
		return []GroupView{}, true
	}

	groups := make([][]*player.Player, numGroups)
	for i, p := range players {
		row := i / numGroups
		col := i % numGroups
		groupIdx := col
		if row%2 != 0 {
			groupIdx = numGroups - 1 - col
		}
		groups[groupIdx] = append(groups[groupIdx], p)
	}

	allFinished := true
	var views []GroupView
	for i, gp := range groups {
		expectedMatches := len(gp) * (len(gp) - 1) / 2
		finished := 0
		for _, m := range t.Matches {
            if len(m.TeamA) == 0 || len(m.TeamB) == 0 { continue }
			for _, p1 := range gp {
				for _, p2 := range gp {
					if p1.ID != p2.ID {
						if (m.TeamA[0].ID == p1.ID && m.TeamB[0].ID == p2.ID) || (m.TeamA[0].ID == p2.ID && m.TeamB[0].ID == p1.ID) {
							if m.Status == "finished" {
								finished++
							}
						}
					}
				}
			}
		}
		finished = finished / 2
		
		isFinished := expectedMatches > 0 && finished >= expectedMatches
		if !isFinished {
			allFinished = false
		}

		gv := GroupView{
			ID:        uuid.New(),
			Name:      fmt.Sprintf("Group %c", 'A'+i),
			Players:   gp,
			Standings: buildStandings(gp, t.Matches),
			Matches:   buildRRMatches(t, gp, "group"),
			Finished:  isFinished,
		}
		views = append(views, gv)
	}

	return views, allFinished
}

func nextPow2(n int) int {
	p := 1
	for p < n {
		p *= 2
	}
	return p
}

func getSeedingArrangement(size int) []int {
	rounds := int(math.Log2(float64(size)))
	if rounds == 0 {
		return []int{1}
	}
	bracket := []int{1, 2}
	for r := 2; r <= rounds; r++ {
		var newBracket []int
		sum := int(math.Pow(2, float64(r))) + 1
		for i, seed := range bracket {
			if i%2 == 0 {
				newBracket = append(newBracket, seed, sum-seed)
			} else {
				newBracket = append(newBracket, sum-seed, seed)
			}
		}
		bracket = newBracket
	}
	return bracket
}

func buildBracketRounds(t *tournament.Tournament, players []*player.Player) []RoundView {
	if len(players) == 0 {
		return nil
	}
	unresolvedSlot := &MatchSlot{Seed: 0, Player: nil}
	size := nextPow2(len(players))
	if size < 2 {
		size = 2 // Minimum bracket size
	}
	arrangement := getSeedingArrangement(size)

	type Pair struct {
		P1 *MatchSlot
		P2 *MatchSlot
	}

	var current []Pair
	for i := 0; i < len(arrangement); i += 2 {
		s1 := arrangement[i] - 1
		s2 := -1
		if i+1 < len(arrangement) {
			s2 = arrangement[i+1] - 1
		}
		
		var p1, p2 *MatchSlot
		if s1 >= 0 && s1 < len(players) {
			p1 = &MatchSlot{Seed: s1 + 1, Player: players[s1]}
		}
		if s2 >= 0 && s2 < len(players) {
			p2 = &MatchSlot{Seed: s2 + 1, Player: players[s2]}
		}
		current = append(current, Pair{P1: p1, P2: p2})
	}

	var rounds []RoundView
	
	for len(current) > 1 {
		var next []Pair
		var rvMatches []BracketMatchView
		
		stageNameCurrent := "r32"
		rem := len(current)
		if rem == 8 {
			stageNameCurrent = "r16"
		} else if rem == 4 {
			stageNameCurrent = "quarterfinal"
		} else if rem == 2 {
			stageNameCurrent = "semifinal"
		} else if rem == 1 {
			stageNameCurrent = "final"
		}
		
		for i := 0; i < len(current); i += 2 {
			mLeft := current[i]
			mRight := current[i+1]

			getWinner := func(m Pair) *MatchSlot {
				if m.P1 == unresolvedSlot || m.P2 == unresolvedSlot {
					return unresolvedSlot
				}

				v1 := m.P1 != nil && m.P1.Player != nil
				v2 := m.P2 != nil && m.P2.Player != nil

				if !v1 && !v2 { return nil }
				if v1 && !v2 { return m.P1 }
				if !v1 && v2 { return m.P2 }

				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Status == "finished" && len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if (tm.TeamA[0].ID == m.P1.Player.ID && tm.TeamB[0].ID == m.P2.Player.ID) {
							if tm.WinnerTeam == "A" { return m.P1 } else { return m.P2 }
						}
						if (tm.TeamA[0].ID == m.P2.Player.ID && tm.TeamB[0].ID == m.P1.Player.ID) {
							if tm.WinnerTeam == "A" { return m.P2 } else { return m.P1 }
						}
					}
				}
				return unresolvedSlot
			}

			next = append(next, Pair{P1: getWinner(mLeft), P2: getWinner(mRight)})
		}

		// Save current round
		for i := 0; i < len(current); i++ {
			p1 := current[i].P1
			p2 := current[i].P2
			var foundMatch *tournament.Match
			if p1 != nil && p2 != nil && p1.Player != nil && p2.Player != nil {
				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
							foundMatch = &t.Matches[k]
							break
						}
					}
				}
			}
			
			rvMatches = append(rvMatches, BracketMatchView{
				Player1: p1,
				Player2: p2,
				Match: foundMatch,
				Stage: stageNameCurrent,
				BestOf: getBestOfForStage(t, stageNameCurrent),
			})
		}

		name := fmt.Sprintf("Round of %d", len(current)*2)
		if len(current) == 4 { name = "Quarter-Finals" } else if len(current) == 2 { name = "Semi-Finals" } else if len(current) == 1 { name = "Final" }
		
		rounds = append(rounds, RoundView{Name: name, Matches: rvMatches})
		
		current = next
	}

	// Final match block
	if len(current) > 0 {
		var finalMatch *tournament.Match
		p1 := current[0].P1
		p2 := current[0].P2
		var champion *MatchSlot

		// Both finalists must be known and the final match finished before crowning a champion.
		// If p2 is nil it means the other side of the bracket hasn't resolved yet — do NOT
		// advance anyone to champion in that case.
		bothFinalistsKnown := p1 != nil && p1.Player != nil && p2 != nil && p2.Player != nil

		if bothFinalistsKnown {
			for k := range t.Matches {
				tm := t.Matches[k]
				if tm.TeamMatchID != nil {
					continue
				}
				if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
					if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
						finalMatch = &t.Matches[k]
						// Only crown champion when the final match is actually finished
						if tm.Status == "finished" {
							if tm.WinnerTeam == "A" {
								if tm.TeamA[0].ID == p1.Player.ID { champion = p1 } else { champion = p2 }
							} else {
								if tm.TeamB[0].ID == p1.Player.ID { champion = p1 } else { champion = p2 }
							}
						}
						break
					}
				}
			}
		}
		// If only one finalist is present due to a genuine bye (size-1 bracket), allow that.
		// But do NOT auto-crown when the other side is merely unresolved.

		rounds = append(rounds, RoundView{
			Name: "🏆 Final",
			Matches: []BracketMatchView{
				{
					Player1: p1,
					Player2: p2,
					Match: finalMatch,
					Stage: "final",
					BestOf: getBestOfForStage(t, "final"),
				},
			},
		})

		// Only append the Champion row when we actually have a champion
		if champion != nil {
			rounds = append(rounds, RoundView{
				Name: "Champion",
				Matches: []BracketMatchView{
					{Player1: champion, Player2: nil},
				},
			})
		}
	}

	return rounds
}
