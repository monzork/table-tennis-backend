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
	IsPublic   bool
	T          map[string]string
}

type DivisionView struct {
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
	RoundRobinMatches  []MatchView
	RoundRobinFinished bool

	Groups            []GroupView
	AllGroupsFinished bool

	KnockoutRounds       []RoundView
	KnockoutRoundsLeft   []RoundView
	KnockoutRoundsRight  []RoundView
	KnockoutRoundsCenter []RoundView
	
	KnockoutGroupID   string
	KnockoutAdvancing []*player.Player
}

type PlayerStanding = tournament.PlayerStanding

type GroupView struct {
	ID        string
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

func BuildTournamentViewModel(t *tournament.Tournament, divs []*division.Division, tmap map[string]string) *TournamentViewModel {
	vm := &TournamentViewModel{
		Tournament: t,
		Type:       t.Type,
		Format:     t.Format,
		Divisions:  []DivisionView{},
		T:          tmap,
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

	// Valid divisions for tournament type
	var validDivs []*division.Division
	for _, d := range divs {
		if !t.SkipElo && d.MinElo == 0 && d.MaxElo == nil {
			// Skip "0-infinite" divisions (like 'No Division') for Elo tournaments
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

// findGroupByPlayers returns the first tournament group that contains any of the given players.
// Returns nil if no match is found.
func findGroupByPlayers(t *tournament.Tournament, players []*player.Player) *tournament.Group {
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

func getBestOfForStage(t *tournament.Tournament, stage string, divID string) int {
	return t.GetEffectiveStageRule(stage, divID).BestOf
}

func buildDivisionView(t *tournament.Tournament, divID, name, color string, minElo int16, maxElo *int16, unclassified bool, players []*player.Player) DivisionView {
	dv := DivisionView{
		ID:             divID,
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
		dv.RoundRobinMatches = buildRRMatches(t, divID, players, "group")
		expectedMatches := len(players) * (len(players) - 1) / 2
		finishedMatches := 0
		for _, m := range t.Matches {
			if m.Stage == "group" && m.Status == "finished" {
				finishedMatches++
			}
		}
		dv.RoundRobinFinished = expectedMatches > 0 && finishedMatches >= expectedMatches
	} else if t.Format == "groups_elimination" {
		dv.Groups, dv.AllGroupsFinished = buildGroupEliminationGroups(t, divID, players)

		if dv.AllGroupsFinished {
			var advancing []*player.Player
			var knockoutGroup *tournament.Group
			for i := range t.Groups {
				if t.Groups[i].Name == name+" - Knockout Seeds" {
					knockoutGroup = &t.Groups[i]
					break
				}
			}

			if knockoutGroup != nil {
				advancing = knockoutGroup.Players
				dv.KnockoutGroupID = knockoutGroup.ID
			} else {
				for _, g := range dv.Groups {
					take := t.GetGroupPassCount(divID)
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
				dv.KnockoutGroupID = "virtual-knockout-" + divID
			}
			dv.KnockoutAdvancing = advancing
			dv.KnockoutRounds = buildBracketRounds(t, divID, advancing)
		}
	} else {
		dv.KnockoutRounds = buildBracketRounds(t, divID, players)
	}

	dv.KnockoutRoundsLeft, dv.KnockoutRoundsRight, dv.KnockoutRoundsCenter = splitKnockoutRounds(dv.KnockoutRounds)

	return dv
}

func splitKnockoutRounds(rounds []RoundView) (left, right, center []RoundView) {
	for _, r := range rounds {
		if r.Name == "🏆 Final" || r.Name == "Champion" || r.Name == "🥉 3rd Place" {
			center = append(center, r)
		} else {
			half := len(r.Matches) / 2

			leftRound := RoundView{Name: r.Name, Matches: r.Matches[:half]}
			rightRound := RoundView{Name: r.Name, Matches: r.Matches[half:]}

			left = append(left, leftRound)
			right = append(right, rightRound)
		}
	}

	for i, j := 0, len(right)-1; i < j; i, j = i+1, j-1 {
		right[i], right[j] = right[j], right[i]
	}

	return left, right, center
}

func buildStandings(players []*player.Player, matches []tournament.Match) []PlayerStanding {
	return tournament.BuildStandings(players, matches)
}
func buildRRMatches(t *tournament.Tournament, divID string, players []*player.Player, stage string) []MatchView {
	var results []MatchView
	bestOf := getBestOfForStage(t, stage, divID)
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

func buildGroupEliminationGroups(t *tournament.Tournament, divID string, players []*player.Player) ([]GroupView, bool) {
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

			views = append(views, GroupView{
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

		gv := GroupView{
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

func buildBracketRounds(t *tournament.Tournament, divID string, players []*player.Player) []RoundView {
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

	var thirdPlaceP1, thirdPlaceP2 *MatchSlot

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

		rounds = append(rounds, RoundView{
			Name: "🏆 Final",
			Matches: []BracketMatchView{
				{
					Player1: p1,
					Player2: p2,
					Match:   finalMatch,
					Stage:   "final",
					BestOf:  getBestOfForStage(t, "final", divID),
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

	if t.HasThirdPlaceMatch && (thirdPlaceP1 != nil || thirdPlaceP2 != nil) {
		var thirdPlaceMatch *tournament.Match
		if thirdPlaceP1 != unresolvedSlot && thirdPlaceP2 != unresolvedSlot && thirdPlaceP1 != nil && thirdPlaceP2 != nil && thirdPlaceP1.Player != nil && thirdPlaceP2.Player != nil {
			for k := range t.Matches {
				tm := t.Matches[k]
				if tm.TeamMatchID != nil {
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

		rounds = append(rounds, RoundView{
			Name: "🥉 3rd Place",
			Matches: []BracketMatchView{
				{
					Player1: thirdPlaceP1,
					Player2: thirdPlaceP2,
					Match:   thirdPlaceMatch,
					Stage:   "3rd_place",
					BestOf:  getBestOfForStage(t, "3rd_place", divID),
				},
			},
		})
	}

	return rounds
}
