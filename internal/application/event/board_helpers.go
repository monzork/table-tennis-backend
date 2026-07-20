package event

import (
	"sort"
	"strings"
	"time"

	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

// BuildBoardCards builds scheduled, in-progress, and finished BoardCard slices from a
// single tournament event and its division definitions. The greedy rest-interleaving
// scheduler ensures players have adequate rest between matches.
func BuildBoardCards(t *tournamentDomain.Event, divs []*divisionDomain.Division) (scheduled, inProgress, finished []BoardCard) {
	nameOf := func(players []*playerDomain.Player) string {
		if len(players) == 0 {
			return "TBD"
		}
		p := players[0]
		if len(players) > 1 {
			return p.FirstNameWithSecond() + " / " + players[1].FirstNameWithSecond()
		}
		return p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
	}
	idOf := func(players []*playerDomain.Player) string {
		if len(players) == 0 {
			return ""
		}
		return players[0].ID
	}

	findGroupName := func(playerID string) string {
		for _, g := range t.Groups {
			for _, p := range g.Players {
				if p.ID == playerID {
					name := g.Name
					if idx := strings.Index(g.Name, " - "); idx != -1 {
						name = g.Name[idx+3:]
					}
					return name
				}
			}
		}
		return ""
	}

	findDivisionName := func(playerID string) string {
		if playerID == "" {
			return ""
		}
		var targetPlayer *playerDomain.Player
		for _, p := range t.Participants {
			if p.ID == playerID {
				targetPlayer = p
				break
			}
		}
		if targetPlayer == nil {
			return ""
		}
		elo := targetPlayer.SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			elo = targetPlayer.DoublesElo
		}
		for _, d := range divs {
			if d.MinElo == 0 && d.MaxElo == nil {
				continue
			}
			if (d.Category == "both" || d.Category == t.Type) && elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
				return d.Name
			}
		}
		return "Open Bracket"
	}

	// 1. Process actual matches in database
	for i := range t.Matches {
		m := &t.Matches[i]
		if m.TeamMatchID != nil { // skip sub-matches
			continue
		}
		card := BoardCard{
			MatchID:     m.ID,
			Status:      m.Status,
			Stage:       m.Stage,
			BestOf:      t.GetEffectiveStageRule(m.Stage, m.DivisionID).BestOf,
			PlayerAName: nameOf(m.TeamA),
			PlayerBName: nameOf(m.TeamB),
			P1Id:        idOf(m.TeamA),
			P2Id:        idOf(m.TeamB),
			TableNumber: m.TableNumber,
			ScoreA:      m.ScoreA(),
			ScoreB:      m.ScoreB(),
			Pin:         m.Pin,
			RoundNumber: m.RoundNumber,
			Category:    t.EventCategory,
			GroupName: func() string {
				if len(m.TeamA) > 0 {
					return findGroupName(m.TeamA[0].ID)
				}
				return ""
			}(),
			DivisionName: func() string {
				if len(m.TeamA) > 0 {
					return findDivisionName(m.TeamA[0].ID)
				}
				return ""
			}(),
		}
		switch m.Status {
		case "in_progress":
			inProgress = append(inProgress, card)
		case "finished":
			finished = append(finished, card)
		default:
			scheduled = append(scheduled, card)
		}
	}

	// 2. Identify virtual matches that should be scheduled based on the format
	vm := bracket.BuildBracket(t, divs, nil)
	for _, dv := range vm.Divisions {
		if vm.Format == "round_robin" {
			for _, mv := range dv.RoundRobinMatches {
				if mv.Player1 != nil && mv.Player2 != nil {
					if !matchExists(t.Matches, mv.Player1.ID, mv.Player2.ID, mv.Stage) {
						groupName := findGroupName(mv.Player1.ID)
						scheduled = append(scheduled, BoardCard{
							MatchID:      "",
							Status:       "scheduled",
							Stage:        mv.Stage,
							BestOf:       mv.BestOf,
							PlayerAName:  mv.Player1.FirstNameWithSecond() + " " + mv.Player1.LastNameWithSecond(),
							PlayerBName:  mv.Player2.FirstNameWithSecond() + " " + mv.Player2.LastNameWithSecond(),
							P1Id:         mv.Player1.ID,
							P2Id:         mv.Player2.ID,
							GroupName:    groupName,
							DivisionName: dv.Name,
							Category:     t.EventCategory,
						})
					}
				}
			}
		} else if vm.Format == "groups_elimination" {
			for _, g := range dv.Groups {
				for _, mv := range g.Matches {
					if mv.Player1 != nil && mv.Player2 != nil {
						if !matchExists(t.Matches, mv.Player1.ID, mv.Player2.ID, mv.Stage) {
							scheduled = append(scheduled, BoardCard{
								MatchID:      "",
								Status:       "scheduled",
								Stage:        mv.Stage,
								BestOf:       mv.BestOf,
								PlayerAName:  mv.Player1.FirstNameWithSecond() + " " + mv.Player1.LastNameWithSecond(),
								PlayerBName:  mv.Player2.FirstNameWithSecond() + " " + mv.Player2.LastNameWithSecond(),
								P1Id:         mv.Player1.ID,
								P2Id:         mv.Player2.ID,
								GroupName:    g.Name,
								DivisionName: dv.Name,
								Category:     t.EventCategory,
							})
						}
					}
				}
			}
			if dv.AllGroupsFinished {
				for _, bracket := range dv.KnockoutBrackets {
					for _, round := range bracket.Rounds {
						for _, bmv := range round.Matches {
							if bmv.Player1 != nil && bmv.Player2 != nil && bmv.Player1.Player != nil && bmv.Player2.Player != nil {
								if !matchExists(t.Matches, bmv.Player1.Player.ID, bmv.Player2.Player.ID, bmv.Stage) {
									scheduled = append(scheduled, BoardCard{
										MatchID:      "",
										Status:       "scheduled",
										Stage:        bmv.Stage,
										BestOf:       bmv.BestOf,
										PlayerAName:  bmv.Player1.Player.FirstNameWithSecond() + " " + bmv.Player1.Player.LastNameWithSecond(),
										PlayerBName:  bmv.Player2.Player.FirstNameWithSecond() + " " + bmv.Player2.Player.LastNameWithSecond(),
										P1Id:         bmv.Player1.Player.ID,
										P2Id:         bmv.Player2.Player.ID,
										DivisionName: dv.Name,
										Category:     t.EventCategory,
									})
								}
							}
						}
					}
				}
			}
		} else if vm.Format == "elimination" {
			for _, bracket := range dv.KnockoutBrackets {
				for _, round := range bracket.Rounds {
					for _, bmv := range round.Matches {
						if bmv.Player1 != nil && bmv.Player2 != nil && bmv.Player1.Player != nil && bmv.Player2.Player != nil {
							if !matchExists(t.Matches, bmv.Player1.Player.ID, bmv.Player2.Player.ID, bmv.Stage) {
								scheduled = append(scheduled, BoardCard{
									MatchID:      "",
									Status:       "scheduled",
									Stage:        bmv.Stage,
									BestOf:       bmv.BestOf,
									PlayerAName:  bmv.Player1.Player.FirstNameWithSecond() + " " + bmv.Player1.Player.LastNameWithSecond(),
									PlayerBName:  bmv.Player2.Player.FirstNameWithSecond() + " " + bmv.Player2.Player.LastNameWithSecond(),
									P1Id:         bmv.Player1.Player.ID,
									P2Id:         bmv.Player2.Player.ID,
									DivisionName: dv.Name,
									Category:     t.EventCategory,
								})
							}
						}
					}
				}
			}
		}
	}

	// 3. Greedy rest-interleaving scheduler
	lastActivity := make(map[string]time.Time)
	matchesPlayed := make(map[string]int)
	
	now := time.Now()

	for _, m := range t.Matches {
		if m.Status == "in_progress" || m.Status == "finished" {
			tAct := now
			if m.UpdatedAt != nil {
				tAct = *m.UpdatedAt
			}
			for _, p := range m.TeamA {
				if lastActivity[p.ID].Before(tAct) {
					lastActivity[p.ID] = tAct
				}
				matchesPlayed[p.ID]++
			}
			for _, p := range m.TeamB {
				if lastActivity[p.ID].Before(tAct) {
					lastActivity[p.ID] = tAct
				}
				matchesPlayed[p.ID]++
			}
		}
	}

	var reordered []BoardCard
	var unstarted []BoardCard
	var virtualScheduled []BoardCard

	for _, c := range scheduled {
		if c.MatchID == "" {
			virtualScheduled = append(virtualScheduled, c)
		} else {
			unstarted = append(unstarted, c)
		}
	}

	scheduleMatchGreedy := func(pool *[]BoardCard) {
		for len(*pool) > 0 {
			bestIdx := -1
			var bestPenalty time.Time
			var bestSum int64
			var bestMatchesPlayed int

			for i, c := range *pool {
				t1 := lastActivity[c.P1Id]
				t2 := lastActivity[c.P2Id]
				
				mp1 := matchesPlayed[c.P1Id]
				mp2 := matchesPlayed[c.P2Id]
				maxMp := mp1
				if mp2 > maxMp {
					maxMp = mp2
				}

				penalty := t1
				if t2.After(t1) {
					penalty = t2
				}
				sum := t1.UnixNano() + t2.UnixNano()

				// Primary sort: lowest max matches played (so everyone plays at least one game before others play their second)
				// Secondary sort: lowest penalty time (longest rested)
				// Tertiary sort: lowest sum of activity times
				
				if bestIdx == -1 || maxMp < bestMatchesPlayed {
					bestIdx = i
					bestMatchesPlayed = maxMp
					bestPenalty = penalty
					bestSum = sum
				} else if maxMp == bestMatchesPlayed {
					if penalty.Before(bestPenalty) {
						bestIdx = i
						bestPenalty = penalty
						bestSum = sum
					} else if penalty.Equal(bestPenalty) {
						if sum < bestSum {
							bestIdx = i
							bestPenalty = penalty
							bestSum = sum
						} else if sum == bestSum {
							if (*pool)[i].MatchID < (*pool)[bestIdx].MatchID {
								bestIdx = i
								bestPenalty = penalty
								bestSum = sum
							}
						}
					}
				}
			}

			picked := (*pool)[bestIdx]
			
			// Calculate estimated start time
			estStart := now
			if bestPenalty.After(now) {
				estStart = bestPenalty
			}
			
			// Give them 15 mins of rest for the next match they might play
			nextAvail := estStart.Add(15 * time.Minute)
			
			estStartCopy := estStart // Copy for pointer
			picked.EstimatedStartTime = &estStartCopy
			
			reordered = append(reordered, picked)

			if picked.P1Id != "" {
				lastActivity[picked.P1Id] = nextAvail
				matchesPlayed[picked.P1Id]++
			}
			if picked.P2Id != "" {
				lastActivity[picked.P2Id] = nextAvail
				matchesPlayed[picked.P2Id]++
			}
			*pool = append((*pool)[:bestIdx], (*pool)[bestIdx+1:]...)
		}
	}

	scheduleMatchGreedy(&unstarted)
	scheduleMatchGreedy(&virtualScheduled)
	scheduled = reordered

	inMatchPlayers := make(map[string]bool)
	for _, c := range inProgress {
		if c.P1Id != "" {
			inMatchPlayers[c.P1Id] = true
		}
		if c.P2Id != "" {
			inMatchPlayers[c.P2Id] = true
		}
	}
	for i := range scheduled {
		if scheduled[i].P1Id != "" && inMatchPlayers[scheduled[i].P1Id] {
			scheduled[i].P1InMatch = true
		}
		if scheduled[i].P2Id != "" && inMatchPlayers[scheduled[i].P2Id] {
			scheduled[i].P2InMatch = true
		}
		scheduled[i].QueuePosition = i + 1
	}

	return
}

// FilterBoardCards filters a slice of BoardCards by search query and division filters.
func FilterBoardCards(cards []BoardCard, q string, divs []string) []BoardCard {
	if q == "" && len(divs) == 0 {
		return cards
	}
	divMap := make(map[string]bool)
	for _, d := range divs {
		divMap[d] = true
	}
	var filtered []BoardCard
	for _, card := range cards {
		matchesSearch := q == "" || strings.Contains(strings.ToLower(card.PlayerAName), q) ||
			strings.Contains(strings.ToLower(card.PlayerBName), q) ||
			strings.Contains(strings.ToLower(card.GroupName), q)
		matchesDiv := len(divMap) == 0 || divMap[card.DivisionName]
		if matchesSearch && matchesDiv {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

// BuildTableVMs builds a list of TableVM values from the event's total tables,
// marking any table currently occupied by an in-progress match.
func BuildTableVMs(t *tournamentDomain.Event, excludeMatchID string, globalOccupied []int) []TableVM {
	var tables []TableVM
	if t == nil || t.NumTables <= 0 {
		return tables
	}
	used := make(map[int]bool)
	for _, m := range t.Matches {
		if m.Status == "in_progress" && m.TableNumber != nil {
			if m.ID != excludeMatchID {
				used[*m.TableNumber] = true
			}
		}
	}
	for _, occ := range globalOccupied {
		used[occ] = true
	}
	for i := 1; i <= t.NumTables; i++ {
		tables = append(tables, TableVM{
			Number: i,
			IsUsed: used[i],
		})
	}
	return tables
}

// TableVM is a view model for a single table's availability.
type TableVM struct {
	Number int
	IsUsed bool
}

// matchExists reports whether a match between two players already exists in the given stage.
func matchExists(matches []tournamentDomain.Match, p1ID, p2ID string, stage string) bool {
	for _, m := range matches {
		if m.TeamMatchID != nil {
			continue
		}
		if m.Stage != stage {
			continue
		}
		if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
			if (m.TeamA[0].ID == p1ID && m.TeamB[0].ID == p2ID) || (m.TeamA[0].ID == p2ID && m.TeamB[0].ID == p1ID) {
				return true
			}
		}
	}
	return false
}

// FilterEventBoardCards filters by query, division names, and categories.
func FilterEventBoardCards(cards []BoardCard, q string, divs []string, cats []string) []BoardCard {
	if q == "" && len(divs) == 0 && len(cats) == 0 {
		return cards
	}
	divMap := make(map[string]bool)
	for _, d := range divs {
		divMap[d] = true
	}
	catMap := make(map[string]bool)
	for _, cat := range cats {
		catMap[cat] = true
	}
	var filtered []BoardCard
	for _, card := range cards {
		matchesSearch := q == "" ||
			strings.Contains(strings.ToLower(card.PlayerAName), q) ||
			strings.Contains(strings.ToLower(card.PlayerBName), q) ||
			strings.Contains(strings.ToLower(card.GroupName), q)
		matchesDiv := len(divMap) == 0 || divMap[card.DivisionName]
		matchesCat := len(catMap) == 0 || catMap[card.Category]
		if matchesSearch && matchesDiv && matchesCat {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

// buildEventTableVMs builds table view models for a tournament (multi-event) board.
func buildEventTableVMs(numTables int, inProgress []BoardCard) []TableVM {
	if numTables <= 0 {
		return nil
	}
	used := make(map[int]bool)
	for _, c := range inProgress {
		if c.TableNumber != nil {
			used[*c.TableNumber] = true
		}
	}
	tables := make([]TableVM, numTables)
	for i := 1; i <= numTables; i++ {
		tables[i-1] = TableVM{Number: i, IsUsed: used[i]}
	}
	return tables
}

// SortBoardCards sorts a slice of BoardCards alphabetically by PlayerAName for stable ordering.
func SortBoardCards(cards []BoardCard) {
	sort.Slice(cards, func(i, j int) bool {
		return cards[i].PlayerAName < cards[j].PlayerAName
	})
}
