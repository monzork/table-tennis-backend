package handler

import (
	"fmt"
	"math"
	"sort"
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

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
}

type GroupView struct {
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

	participants := make([]*player.Player, len(t.Participants))
	copy(participants, t.Participants)

	// Sort participants by correct Elo
	sort.Slice(participants, func(i, j int) bool {
		ei := participants[i].SinglesElo
		ej := participants[j].SinglesElo
		if t.Type == "doubles" {
			ei = participants[i].DoublesElo
			ej = participants[j].DoublesElo
		}
		return ei > ej
	})

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
		for _, p := range participants {
			if assignedMap[p.ID] {
				continue
			}
			elo := p.SinglesElo
			if t.Type == "doubles" {
				elo = p.DoublesElo
			}
			if elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
				dPlayers = append(dPlayers, p)
				assignedMap[p.ID] = true
			}
		}

		if len(dPlayers) > 0 {
			vm.Divisions = append(vm.Divisions, buildDivisionView(t, d.Name, d.Color, d.MinElo, d.MaxElo, false, dPlayers))
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

func buildStandings(players []*player.Player, matches []tournament.Match) []PlayerStanding {
	stats := make([]PlayerStanding, len(players))
	for i, p := range players {
		wins := 0
		played := 0
		for _, m := range matches {
			if m.Status != "finished" {
				continue
			}
			var isA, isB bool
			if len(m.TeamA) > 0 { isA = m.TeamA[0].ID == p.ID }
			if len(m.TeamB) > 0 { isB = m.TeamB[0].ID == p.ID }
			if isA || isB {
				played++
				if (isA && m.WinnerTeam == "A") || (isB && m.WinnerTeam == "B") {
					wins++
				}
			}
		}
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
			Losses:        played - wins,
			WinRate:       winRate,
			WinPercentage: winPercentage,
		}
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Wins != stats[j].Wins {
			return stats[i].Wins > stats[j].Wins
		}
		if stats[i].WinRate != stats[j].WinRate {
			return stats[i].WinRate > stats[j].WinRate
		}
		return stats[i].Losses < stats[j].Losses
	})

	return stats
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
		// Since each match is counted twice in the above loop:
		finished = finished / 2
		
		isFinished := expectedMatches > 0 && finished >= expectedMatches
		if !isFinished {
			allFinished = false
		}

		gv := GroupView{
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
				v1 := m.P1 != nil && m.P1.Player != nil
				v2 := m.P2 != nil && m.P2.Player != nil

				if (!v1 && !v2) { return nil }
				if (v1 && !v2) { return m.P1 }
				if (!v1 && v2) { return m.P2 }

				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.Status == "finished" && len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if (tm.TeamA[0].ID == m.P1.Player.ID && tm.TeamB[0].ID == m.P2.Player.ID) {
							if tm.WinnerTeam == "A" { return m.P1 } else { return m.P2 }
						}
						if (tm.TeamA[0].ID == m.P2.Player.ID && tm.TeamB[0].ID == m.P1.Player.ID) {
							if tm.WinnerTeam == "A" { return m.P2 } else { return m.P1 }
						}
					}
				}
				return nil
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
		
		if p1 != nil && p2 != nil && p1.Player != nil && p2.Player != nil {
			for k := range t.Matches {
				tm := t.Matches[k]
				if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
					if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
						finalMatch = &t.Matches[k]
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
		} else if p1 != nil && p1.Player != nil {
			champion = p1
		} else if p2 != nil && p2.Player != nil {
			champion = p2
		}

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
		
		rounds = append(rounds, RoundView{
			Name: "Champion",
			Matches: []BracketMatchView{
				{ Player1: champion, Player2: nil },
			},
		})
	}

	return rounds
}
