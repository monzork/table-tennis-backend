package bracket

import (
	"fmt"
	"math"
	"sort"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"

	"strings"

	"github.com/google/uuid"
)

type Bracket struct {
	Event     *event.Event
	Type      string
	Format    string
	Divisions []Division
	IsPublic  bool
	T         map[string]string
}

type Division struct {
	ID             string
	Name           string
	Color          string
	MinElo         int16
	MaxElo         *int16
	IsUnclassified bool
	Players        []*player.Player

	// GroupID is the DB group ID for elimination-format seeding draw.
	// Empty for rounds-robin / groups-elimination divisions.
	GroupID string

	Format             string
	Standings          []PlayerStanding
	RoundRobinMatches  []Match
	RoundRobinFinished bool

	Groups            []Group
	AllGroupsFinished bool

	KnockoutBrackets []KnockoutBracket
}

type KnockoutBracket struct {
	Tier         int
	Name         string
	Advancing    []*player.Player
	GroupID      string
	Rounds       []Round
	RoundsLeft   []Round
	RoundsRight  []Round
	RoundsCenter []Round
}

type PlayerStanding = event.PlayerStanding

type Group struct {
	ID        string
	Name      string
	Players   []*player.Player
	Standings []PlayerStanding
	Matches   []Match
	Finished  bool
}

type Match struct {
	Player1 *player.Player
	Player2 *player.Player
	Match   *event.Match
	Stage   string
	BestOf  int
}

type Round struct {
	Name    string
	Matches []BracketMatch
}

type BracketMatch struct {
	Player1 *MatchSlot
	Player2 *MatchSlot
	Match   *event.Match
	Stage   string
	BestOf  int
}

type MatchSlot struct {
	Seed   int
	Player *player.Player
}

func BuildBracket(t *event.Event, divs []*division.Division, tmap map[string]string) *Bracket {
	vm := &Bracket{
		Event:     t,
		Type:      t.Type,
		Format:    t.Format,
		Divisions: []Division{},
		T:         tmap,
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

	if t.SkipElo || t.Format == "single_division_multiple_brackets" {
		openPlayers := participants
		var openGroupID string
		if t.Format == "elimination" {
			if g := findGroupByPlayers(t, openPlayers); g != nil {
				openGroupID = g.ID
				if len(g.Players) > 0 {
					openPlayers = g.Players
				}
			}
		}
		dv := buildDivisionView(t, "", "Open Bracket", "", 0, nil, false, openPlayers)
		dv.GroupID = openGroupID
		vm.Divisions = append(vm.Divisions, dv)
		return vm
	}

	assignedMap := make(map[string]bool)

	// Valid divisions for event type
	var validDivs []*division.Division
	for _, d := range divs {
		if !t.SkipElo && d.MinElo == 0 && d.MaxElo == nil {
			// Skip "0-infinite" divisions (like 'No Division') for Elo events
			continue
		}
		if d.Category == "both" || d.Category == t.Type {
			validDivs = append(validDivs, d)
		}
	}

	for _, d := range validDivs {
		var dPlayers []*player.Player
		var divGroupID string
		name := d.Name
		if strings.HasSuffix(strings.ToLower(name), " division") {
			name = name[:len(name)-9]
		}

		if t.Format == "elimination" {
			// Find the bracket-draw group generated for this division. Groups are
			// named "<division name> - Bracket Draw" at generation time, so match
			// on that stable name instead of re-deriving membership from each
			// player's current Elo — a player's Elo can be edited after the group
			// was generated, which would otherwise let the same group satisfy more
			// than one division's Elo range and be shown under both.
			expectedGroupName := d.Name + " - Bracket Draw"
			for i := range t.Groups {
				if t.Groups[i].Name != expectedGroupName {
					continue
				}
				divGroupID = t.Groups[i].ID
				dPlayers = t.Groups[i].Players
				for _, p := range dPlayers {
					assignedMap[p.ID] = true
				}
				break
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
			dv := buildDivisionView(t, d.ID, name, d.Color, d.MinElo, d.MaxElo, false, dPlayers)
			dv.GroupID = divGroupID
			vm.Divisions = append(vm.Divisions, dv)
		}
	}

	var unassigned []*player.Player
	for _, p := range participants {
		if !assignedMap[p.ID] {
			unassigned = append(unassigned, p)
		}
	}

	if len(unassigned) > 0 {
		var unclassifiedGroupID string
		if t.Format == "elimination" {
			if g := findGroupByPlayers(t, unassigned); g != nil {
				unclassifiedGroupID = g.ID
			}
		}
		dv := buildDivisionView(t, "", "Unclassified", "", 0, nil, true, unassigned)
		dv.GroupID = unclassifiedGroupID
		vm.Divisions = append(vm.Divisions, dv)
	}

	return vm
}

// findGroupByPlayers returns the first event group that contains any of the given players.
// Returns nil if no match is found.
func findGroupByPlayers(t *event.Event, players []*player.Player) *event.Group {
	if len(players) == 0 {
		return nil
	}
	want := make(map[string]bool, len(players))
	for _, p := range players {
		want[p.ID] = true
	}
	for i := range t.Groups {
		for _, gp := range t.Groups[i].Players {
			if want[gp.ID] {
				return &t.Groups[i]
			}
		}
	}
	return nil
}

func getBestOfForStage(t *event.Event, stage string, divID string) int {
	return t.GetEffectiveStageRule(stage, divID).BestOf
}

func buildDivisionView(t *event.Event, divID, name, color string, minElo int16, maxElo *int16, unclassified bool, players []*player.Player) Division {
	divFormat := t.GetDivisionFormat(divID)
	dv := Division{
		ID:             divID,
		Name:           name,
		Color:          color,
		MinElo:         minElo,
		MaxElo:         maxElo,
		IsUnclassified: unclassified,
		Players:        players,
		Format:         divFormat,
	}

	if divFormat == "round_robin" {
		dv.Standings = buildStandings(players, t.Matches)
		dv.RoundRobinMatches = buildRRMatches(t, divID, players, "group")
		expectedMatches := len(players) * (len(players) - 1) / 2
		finishedMatches := 0
		for _, m := range t.Matches {
			if m.Stage == "group" && m.Status == "finished" {
				finishedMatches++
			}
		}
		dv.RoundRobinFinished = expectedMatches > 0 && finishedMatches >= expectedMatches
	} else if divFormat == "groups_elimination" || divFormat == "single_division_multiple_brackets" {
		dv.Groups, dv.AllGroupsFinished = buildGroupEliminationGroups(t, divID, name, players)

		if dv.AllGroupsFinished {
			bracketsCount := t.KnockoutBracketsCount
			if bracketsCount <= 0 {
				bracketsCount = 1
			}

			passCount := t.GetGroupPassCount(divID)
			if passCount == 0 {
				passCount = 2
			}

			// Pre-collect any knockout groups created by manual draw
			var knockoutGroups []*event.Group
			for i := range t.Groups {
				if strings.Contains(t.Groups[i].Name, name+" -") && strings.Contains(t.Groups[i].Name, "Knockout Seeds") {
					knockoutGroups = append(knockoutGroups, &t.Groups[i])
				}
			}

			for tier := 0; tier < bracketsCount; tier++ {
				var tierAdvancing []*player.Player
				var tierGroupID string

				// Attempt to load from DB
				var kg *event.Group
				expectedName := fmt.Sprintf("%s - Tier %d Knockout Seeds", name, tier)
				if bracketsCount == 1 {
					expectedName = name + " - Knockout Seeds"
				}
				for _, g := range knockoutGroups {
					if g.Name == expectedName {
						kg = g
						break
					}
				}

				if kg != nil {
					tierAdvancing = kg.Players
					tierGroupID = kg.ID
				} else {
					// Build virtually from group standings
					tierGroups := make([]Group, len(dv.Groups))
					for gi, g := range dv.Groups {
						tierGroups[gi] = Group{
							ID:        g.ID,
							Name:      g.Name,
							Standings: []PlayerStanding{},
						}
						startIdx := tier * passCount
						endIdx := startIdx + passCount
						for idx := startIdx; idx < endIdx && idx < len(g.Standings); idx++ {
							tierGroups[gi].Standings = append(tierGroups[gi].Standings, g.Standings[idx])
						}
					}
					tierAdvancing = buildITTFKnockoutSeeds(tierGroups, passCount)
					tierGroupID = fmt.Sprintf("virtual-knockout-%s-tier%d", divID, tier)
				}

				tierName := "Main Bracket"
				if tier > 0 {
					tierName = fmt.Sprintf("Tier %d Bracket", tier+1)
				}
				if bracketsCount == 1 {
					tierName = ""
				}

				tierRounds := buildBracketRounds(t, divID, tierAdvancing, tier)
				left, right, center := splitKnockoutRounds(tierRounds)

				dv.KnockoutBrackets = append(dv.KnockoutBrackets, KnockoutBracket{
					Tier:         tier,
					Name:         tierName,
					Advancing:    tierAdvancing,
					GroupID:      tierGroupID,
					Rounds:       tierRounds,
					RoundsLeft:   left,
					RoundsRight:  right,
					RoundsCenter: center,
				})
			}
		}
	} else {
		tierRounds := buildBracketRounds(t, divID, players, 0)
		left, right, center := splitKnockoutRounds(tierRounds)
		dv.KnockoutBrackets = append(dv.KnockoutBrackets, KnockoutBracket{
			Tier:         0,
			Name:         "",
			Advancing:    players,
			GroupID:      dv.GroupID,
			Rounds:       tierRounds,
			RoundsLeft:   left,
			RoundsRight:  right,
			RoundsCenter: center,
		})
	}

	return dv
}

func splitKnockoutRounds(rounds []Round) (left, right, center []Round) {
	for _, r := range rounds {
		if r.Name == "🏆 Final" || r.Name == "Champion" || r.Name == "🥉 3rd Place" {
			center = append(center, r)
		} else {
			half := len(r.Matches) / 2

			leftRound := Round{Name: r.Name, Matches: r.Matches[:half]}
			rightRound := Round{Name: r.Name, Matches: r.Matches[half:]}

			left = append(left, leftRound)
			right = append(right, rightRound)
		}
	}

	for i, j := 0, len(right)-1; i < j; i, j = i+1, j-1 {
		right[i], right[j] = right[j], right[i]
	}

	return left, right, center
}

func buildStandings(players []*player.Player, matches []event.Match) []PlayerStanding {
	return event.BuildStandings(players, matches)
}
func buildRRMatches(t *event.Event, divID string, players []*player.Player, stage string) []Match {
	var results []Match
	bestOf := getBestOfForStage(t, stage, divID)
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			p1 := players[i]
			p2 := players[j]
			var found *event.Match
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
			results = append(results, Match{
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

func buildGroupEliminationGroups(t *event.Event, divID string, divisionName string, players []*player.Player) ([]Group, bool) {
	// Try to load saved groups containing any players in this division first
	var divisionGroups []event.Group
	for _, g := range t.Groups {
		if strings.Contains(g.Name, "- Knockout Seeds") {
			continue
		}

		belongsToDiv := false
		prefix := divisionName + " - "
		if strings.HasPrefix(g.Name, prefix) {
			belongsToDiv = true
		} else if divisionName == "Open Bracket" && strings.HasPrefix(g.Name, "Group ") {
			belongsToDiv = true
		} else {
			for _, gp := range g.Players {
				for _, dp := range players {
					if gp.ID == dp.ID {
						belongsToDiv = true
						break
					}
				}
				if belongsToDiv {
					break
				}
			}
		}
		if belongsToDiv {
			divisionGroups = append(divisionGroups, g)
		}
	}

	if len(divisionGroups) > 0 {
		allFinished := true
		var views []Group
		for _, g := range divisionGroups {
			expectedMatches := len(g.Players) * (len(g.Players) - 1) / 2
			finished := 0
			for _, m := range t.Matches {
				if m.TeamMatchID != nil {
					continue
				}
				if m.Stage != "group" {
					continue
				}
				if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
					continue
				}
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

			views = append(views, Group{
				ID:        g.ID,
				Name:      displayName,
				Players:   g.Players,
				Standings: buildStandings(g.Players, t.Matches),
				Matches:   buildRRMatches(t, divID, g.Players, "group"),
				Finished:  isFinished,
			})
		}
		return views, allFinished
	}

	// Fallback to snake seeding
	groupSize := 4
	numGroups := int(math.Ceil(float64(len(players)) / float64(groupSize)))
	if numGroups == 0 {
		return []Group{}, true
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
	var views []Group
	for i, gp := range groups {
		expectedMatches := len(gp) * (len(gp) - 1) / 2
		finished := 0
		for _, m := range t.Matches {
			if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
				continue
			}
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

		gv := Group{
			ID:        uuid.New().String(),
			Name:      fmt.Sprintf("Group %c", 'A'+i),
			Players:   gp,
			Standings: buildStandings(gp, t.Matches),
			Matches:   buildRRMatches(t, divID, gp, "group"),
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

// buildITTFKnockoutSeeds arranges advancing players per ITTF rules:
//   - Group winners occupy seeds 1..numGroups (in group order).
//   - Each subsequent layer (runners-up, etc.) is placed into the OPPOSITE
//     bracket half from that group's winner, ensuring same-group players
//     cannot meet before the final/semi-final.
func buildITTFKnockoutSeeds(groups []Group, passCount int) []*player.Player {
	numGroups := len(groups)
	if numGroups == 0 || passCount == 0 {
		return nil
	}

	totalAdvancing := 0
	for _, g := range groups {
		take := passCount
		if take > len(g.Standings) {
			take = len(g.Standings)
		}
		totalAdvancing += take
	}
	if totalAdvancing == 0 {
		return nil
	}

	bracketSize := nextPow2(totalAdvancing)
	arrangement := getSeedingArrangement(bracketSize)

	// Determine which seed numbers fall in the top half of the bracket.
	halfSize := len(arrangement) / 2
	topHalfSeeds := make(map[int]bool, halfSize)
	for _, s := range arrangement[:halfSize] {
		topHalfSeeds[s] = true
	}

	// Result slice: index i → player with seed (i+1).
	result := make([]*player.Player, totalAdvancing)

	// Layer 0: place group winners at seeds 1..numGroups.
	winnerInTop := make([]bool, numGroups)
	for gi, g := range groups {
		if len(g.Standings) == 0 {
			continue
		}
		result[gi] = g.Standings[0].Player
		winnerInTop[gi] = topHalfSeeds[gi+1]
	}

	// Layers 1+: runners-up, 3rd-place, etc.
	// For each layer, groups whose winner is in the top half send their
	// player to a bottom-half slot, and vice versa.
	nextSlot := numGroups // first available seed index after layer 0

	for layer := 1; layer < passCount; layer++ {
		// Collect open top and bottom slots for this layer.
		layerSize := numGroups
		var topSlots, bottomSlots []int
		for i := nextSlot; i < nextSlot+layerSize && i < totalAdvancing; i++ {
			seedNum := i + 1
			if topHalfSeeds[seedNum] {
				topSlots = append(topSlots, i)
			} else {
				bottomSlots = append(bottomSlots, i)
			}
		}

		tsi, bsi := 0, 0
		for gi, g := range groups {
			if layer >= len(g.Standings) {
				continue
			}
			p := g.Standings[layer].Player
			if winnerInTop[gi] {
				// Winner is in top half → this layer player goes to bottom.
				if bsi < len(bottomSlots) {
					result[bottomSlots[bsi]] = p
					bsi++
				} else if tsi < len(topSlots) {
					result[topSlots[tsi]] = p
					tsi++
				}
			} else {
				// Winner is in bottom half → this layer player goes to top.
				if tsi < len(topSlots) {
					result[topSlots[tsi]] = p
					tsi++
				} else if bsi < len(bottomSlots) {
					result[bottomSlots[bsi]] = p
					bsi++
				}
			}
		}

		nextSlot += layerSize
	}

	// Compact: remove any nil gaps (shouldn't happen, but guard anyway).
	out := result[:0]
	for _, p := range result {
		if p != nil {
			out = append(out, p)
		}
	}
	return out
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

func buildBracketRounds(t *event.Event, divID string, players []*player.Player, tier int) []Round {
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

	var rounds []Round

	var thirdPlaceP1, thirdPlaceP2 *MatchSlot

	for len(current) > 1 {
		var next []Pair
		var rvMatches []BracketMatch

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
		if tier > 0 {
			stageNameCurrent = fmt.Sprintf("tier%d_%s", tier, stageNameCurrent)
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

				if !v1 && !v2 {
					return nil
				}
				if v1 && !v2 {
					return m.P1
				}
				if !v1 && v2 {
					return m.P2
				}

				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Stage != stageNameCurrent {
						continue
					}
					if tm.Status == "finished" && len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if tm.TeamA[0].ID == m.P1.Player.ID && tm.TeamB[0].ID == m.P2.Player.ID {
							if tm.WinnerTeam == "A" {
								return m.P1
							} else {
								return m.P2
							}
						}
						if tm.TeamA[0].ID == m.P2.Player.ID && tm.TeamB[0].ID == m.P1.Player.ID {
							if tm.WinnerTeam == "A" {
								return m.P2
							} else {
								return m.P1
							}
						}
					}
				}
				return unresolvedSlot
			}

			getLoser := func(m Pair) *MatchSlot {
				if m.P1 == unresolvedSlot || m.P2 == unresolvedSlot {
					return unresolvedSlot
				}

				v1 := m.P1 != nil && m.P1.Player != nil
				v2 := m.P2 != nil && m.P2.Player != nil

				if !v1 && !v2 {
					return nil
				}
				if v1 && !v2 {
					return nil
				}
				if !v1 && v2 {
					return nil
				}

				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Stage != stageNameCurrent {
						continue
					}
					if tm.Status == "finished" && len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if tm.TeamA[0].ID == m.P1.Player.ID && tm.TeamB[0].ID == m.P2.Player.ID {
							if tm.WinnerTeam == "A" {
								return m.P2
							} else {
								return m.P1
							}
						}
						if tm.TeamA[0].ID == m.P2.Player.ID && tm.TeamB[0].ID == m.P1.Player.ID {
							if tm.WinnerTeam == "A" {
								return m.P1
							} else {
								return m.P2
							}
						}
					}
				}
				return unresolvedSlot
			}

			if rem == 2 && t.HasThirdPlaceMatch {
				thirdPlaceP1 = getLoser(mLeft)
				thirdPlaceP2 = getLoser(mRight)
			}

			next = append(next, Pair{P1: getWinner(mLeft), P2: getWinner(mRight)})
		}

		// Save current round
		for i := 0; i < len(current); i++ {
			p1 := current[i].P1
			p2 := current[i].P2
			var foundMatch *event.Match
			if p1 != nil && p2 != nil && p1.Player != nil && p2.Player != nil {
				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Stage != stageNameCurrent {
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

			rvMatches = append(rvMatches, BracketMatch{
				Player1: p1,
				Player2: p2,
				Match:   foundMatch,
				Stage:   stageNameCurrent,
				BestOf:  getBestOfForStage(t, stageNameCurrent, divID),
			})
		}

		name := fmt.Sprintf("Round of %d", len(current)*2)
		if len(current) == 4 {
			name = "Quarter-Finals"
		} else if len(current) == 2 {
			name = "Semi-Finals"
		} else if len(current) == 1 {
			name = "Final"
		}

		rounds = append(rounds, Round{Name: name, Matches: rvMatches})

		current = next
	}

	// Final match block
	if len(current) > 0 {
		var finalMatch *event.Match
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
				finalStage := "final"
				if tier > 0 {
					finalStage = fmt.Sprintf("tier%d_%s", tier, finalStage)
				}
				if tm.Stage != finalStage {
					continue
				}
				if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
					if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
						finalMatch = &t.Matches[k]
						// Only crown champion when the final match is actually finished
						if tm.Status == "finished" {
							if tm.WinnerTeam == "A" {
								if tm.TeamA[0].ID == p1.Player.ID {
									champion = p1
								} else {
									champion = p2
								}
							} else {
								if tm.TeamB[0].ID == p1.Player.ID {
									champion = p1
								} else {
									champion = p2
								}
							}
						}
						break
					}
				}
			}
		}
		// If only one finalist is present due to a genuine bye (size-1 bracket), allow that.
		// But do NOT auto-crown when the other side is merely unresolved.

		finalStageStr := "final"
		if tier > 0 {
			finalStageStr = fmt.Sprintf("tier%d_final", tier)
		}
		rounds = append(rounds, Round{
			Name: "🏆 Final",
			Matches: []BracketMatch{
				{
					Player1: p1,
					Player2: p2,
					Match:   finalMatch,
					Stage:   finalStageStr,
					BestOf:  getBestOfForStage(t, "final", divID),
				},
			},
		})

		// Only append the Champion row when we actually have a champion
		if champion != nil {
			rounds = append(rounds, Round{
				Name: "Champion",
				Matches: []BracketMatch{
					{Player1: champion, Player2: nil},
				},
			})
		}
	}

	if t.HasThirdPlaceMatch && (thirdPlaceP1 != nil || thirdPlaceP2 != nil) {
		var thirdPlaceMatch *event.Match
		if thirdPlaceP1 != unresolvedSlot && thirdPlaceP2 != unresolvedSlot && thirdPlaceP1 != nil && thirdPlaceP2 != nil && thirdPlaceP1.Player != nil && thirdPlaceP2.Player != nil {
			for k := range t.Matches {
				tm := t.Matches[k]
				if tm.TeamMatchID != nil {
					continue
				}
				thirdStage := "3rd_place"
				if tier > 0 {
					thirdStage = fmt.Sprintf("tier%d_%s", tier, thirdStage)
				}
				if tm.Stage != thirdStage {
					continue
				}
				if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
					if (tm.TeamA[0].ID == thirdPlaceP1.Player.ID && tm.TeamB[0].ID == thirdPlaceP2.Player.ID) || (tm.TeamA[0].ID == thirdPlaceP2.Player.ID && tm.TeamB[0].ID == thirdPlaceP1.Player.ID) {
						thirdPlaceMatch = &t.Matches[k]
						break
					}
				}
			}
		}

		thirdStageStr := "3rd_place"
		if tier > 0 {
			thirdStageStr = fmt.Sprintf("tier%d_3rd_place", tier)
		}
		rounds = append(rounds, Round{
			Name: "🥉 3rd Place",
			Matches: []BracketMatch{
				{
					Player1: thirdPlaceP1,
					Player2: thirdPlaceP2,
					Match:   thirdPlaceMatch,
					Stage:   thirdStageStr,
					BestOf:  getBestOfForStage(t, "3rd_place", divID),
				},
			},
		})
	}

	return rounds
}

// ValidateSameGroupSeparation checks whether a proposed seed ordering (players[0] = seed 1,
// players[1] = seed 2, …) ensures that no two players from the same source group are
// placed in the same bracket half.
//
// groups is the list of group-stage groups (each with its Players field).
// players is the ordered list of advancing players in the proposed seed order.
//
// Returns an error that names every conflicting pair, or nil if the arrangement is valid.
func ValidateSameGroupSeparation(groups []Group, players []*player.Player) error {
	if len(players) < 2 {
		return nil
	}

	// Build a map: player ID → group name, so we can detect same-group pairs quickly.
	playerGroup := make(map[string]string, len(players))
	for _, g := range groups {
		for _, p := range g.Players {
			playerGroup[p.ID] = g.Name
		}
	}

	bracketSize := nextPow2(len(players))
	if bracketSize < 2 {
		return nil
	}
	arrangement := getSeedingArrangement(bracketSize)

	// Build match-up tree for the first round to determine which seeds land in each half.
	// The top half is slots 0 .. halfSize-1 of the arrangement; bottom half is the rest.
	halfSize := len(arrangement) / 2

	// Map seed number → bracket half (0 = top, 1 = bottom).
	seedHalf := make(map[int]int, len(arrangement))
	for i, s := range arrangement {
		if i < halfSize {
			seedHalf[s] = 0
		} else {
			seedHalf[s] = 1
		}
	}

	// For each advancing player, record their proposed seed (1-indexed) and the half they'd land in.
	type entry struct {
		name  string
		seed  int
		half  int
		group string
	}
	entries := make([]entry, 0, len(players))
	for i, p := range players {
		if p == nil {
			continue
		}
		seed := i + 1
		half, ok := seedHalf[seed]
		if !ok {
			// Seeds beyond the bracket size go to the bottom half (they are byes).
			half = 1
		}
		entries = append(entries, entry{
			name:  p.FirstNameWithSecond() + " " + p.LastNameWithSecond(),
			seed:  seed,
			half:  half,
			group: playerGroup[p.ID],
		})
	}

	// Detect same-group players landing in the same half.
	type halfGroup struct {
		group string
		half  int
	}
	seen := make(map[halfGroup]entry)
	var conflicts []string

	for _, e := range entries {
		if e.group == "" {
			continue // player not in any tracked group — skip
		}
		key := halfGroup{group: e.group, half: e.half}
		if prev, exists := seen[key]; exists {
			halfName := "top"
			if e.half == 1 {
				halfName = "bottom"
			}
			conflicts = append(conflicts,
				fmt.Sprintf("'%s' (seed %d) and '%s' (seed %d) are from the same group '%s' and are both in the %s half of the bracket",
					prev.name, prev.seed, e.name, e.seed, e.group, halfName,
				),
			)
		} else {
			seen[key] = e
		}
	}

	if len(conflicts) > 0 {
		return fmt.Errorf(
			"ITTF rule violation: same-group players must be separated into opposite bracket halves.\n%s",
			strings.Join(conflicts, "\n"),
		)
	}
	return nil
}
