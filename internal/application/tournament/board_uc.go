package tournament

import (
	"context"
	"fmt"
	"sort"
	"time"

	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/tournament"
)

type GetBoardDataUseCase struct {
	eventRepo    eventDomain.Repository
	divisionRepo divisionDomain.Repository
}

func NewGetBoardDataUseCase(eventRepo eventDomain.Repository, divisionRepo divisionDomain.Repository) *GetBoardDataUseCase {
	return &GetBoardDataUseCase{
		eventRepo:    eventRepo,
		divisionRepo: divisionRepo,
	}
}

func (uc *GetBoardDataUseCase) Execute(ctx context.Context, idStr string) (*eventDomain.Tournament, []*divisionDomain.Division, []event.BoardCard, []event.BoardCard, []event.BoardCard, error) {
	e, err := uc.eventRepo.GetByIDDeep(ctx, idStr)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	divs, _ := uc.divisionRepo.GetAll(ctx)

	var inProgress []event.BoardCard
	var finished []event.BoardCard

	type activePool struct {
		id             string
		tournamentID   string
		divisionName   string
		tournamentName string
		pool           []event.BoardCard
	}
	var activePools []activePool

	divPlayerCounts := make(map[string]int)
	lastActivity := make(map[string]time.Time)

	for _, t := range e.Events {
		s, i, f := event.BuildBoardCards(t, divs)

		vm := bracket.BuildBracket(t, divs, nil)
		for _, dv := range vm.Divisions {
			divPlayerCounts[dv.Name] += len(dv.Players)
		}

		for idx := range s {
			s[idx].TournamentName = t.Name
			s[idx].TournamentID = t.ID
		}
		for idx := range i {
			i[idx].TournamentName = t.Name
			i[idx].TournamentID = t.ID
		}
		for idx := range f {
			f[idx].TournamentName = t.Name
			f[idx].TournamentID = t.ID
		}

		scheduledByDiv := make(map[string][]event.BoardCard)
		for _, card := range s {
			scheduledByDiv[card.DivisionName] = append(scheduledByDiv[card.DivisionName], card)
		}

		allDivNamesMap := make(map[string]bool)
		for _, card := range s {
			allDivNamesMap[card.DivisionName] = true
		}
		for _, card := range i {
			allDivNamesMap[card.DivisionName] = true
		}

		for divName := range allDivNamesMap {
			var u []event.BoardCard
			var v []event.BoardCard
			for _, card := range scheduledByDiv[divName] {
				if card.MatchID == "" {
					v = append(v, card)
				} else {
					u = append(u, card)
				}
			}

			pool := append(u, v...)
			poolKey := fmt.Sprintf("%s:%s", t.ID, divName)

			activePools = append(activePools, activePool{
				id:             poolKey,
				tournamentID:   t.ID,
				divisionName:   divName,
				tournamentName: t.Name,
				pool:           pool,
			})
		}

		inProgress = append(inProgress, i...)
		finished = append(finished, f...)

		for _, m := range t.Matches {
			if m.Status == "finished" {
				tAct := time.Time{}
				if m.UpdatedAt != nil {
					tAct = *m.UpdatedAt
				}
				for _, p := range m.TeamA {
					if lastActivity[p.ID].Before(tAct) {
						lastActivity[p.ID] = tAct
					}
				}
				for _, p := range m.TeamB {
					if lastActivity[p.ID].Before(tAct) {
						lastActivity[p.ID] = tAct
					}
				}
			}
		}
	}

	activeDivisionsMap := make(map[string]bool)
	for _, ap := range activePools {
		activeDivisionsMap[ap.divisionName] = true
	}

	type activeDivInfo struct {
		name        string
		playerCount int
	}
	var activeDivs []activeDivInfo
	totalPlayers := 0

	for divName := range activeDivisionsMap {
		pCount := divPlayerCounts[divName]
		if pCount <= 0 {
			pCount = 1
		}
		activeDivs = append(activeDivs, activeDivInfo{
			name:        divName,
			playerCount: pCount,
		})
		totalPlayers += pCount
	}

	maxTables := e.NumTables
	if maxTables <= 0 {
		maxTables = 6
	}

	allocated := make(map[string]int)
	remainingTables := maxTables
	numActiveDivs := len(activeDivs)

	if numActiveDivs > 0 {
		if remainingTables >= numActiveDivs {
			for _, ad := range activeDivs {
				allocated[ad.name] = 1
				remainingTables--
			}
		} else {
			sort.Slice(activeDivs, func(i, j int) bool {
				return activeDivs[i].playerCount > activeDivs[j].playerCount
			})
			for _, ad := range activeDivs {
				if remainingTables > 0 {
					allocated[ad.name] = 1
					remainingTables--
				} else {
					allocated[ad.name] = 0
				}
			}
		}

		if remainingTables > 0 && totalPlayers > 0 {
			type remainderInfo struct {
				name      string
				remainder float64
			}
			var remainders []remainderInfo
			allocatedSum := 0

			for _, ad := range activeDivs {
				share := float64(ad.playerCount) / float64(totalPlayers) * float64(remainingTables)
				intShare := int(share)
				allocated[ad.name] += intShare
				allocatedSum += intShare
				remainders = append(remainders, remainderInfo{
					name:      ad.name,
					remainder: share - float64(intShare),
				})
			}

			leftover := remainingTables - allocatedSum
			sort.Slice(remainders, func(i, j int) bool {
				return remainders[i].remainder > remainders[j].remainder
			})
			for i := 0; i < leftover && i < len(remainders); i++ {
				allocated[remainders[i].name]++
			}
		}
	}

	pools := make(map[string][]event.BoardCard)
	for _, ap := range activePools {
		pools[ap.id] = ap.pool
	}

	var globalScheduled []event.BoardCard
	type runningMatch struct {
		divisionName string
		endTime      time.Time
	}
	var runningMatches []runningMatch

	simClock := time.Now()
	availableTime := make(map[string]time.Time)
	scheduledCounts := make(map[string]int)

	for pID, tAct := range lastActivity {
		availableTime[pID] = tAct.Add(10 * time.Minute)
	}

	for _, c := range inProgress {
		avail := time.Now().Add(25 * time.Minute)
		if c.P1Id != "" {
			availableTime[c.P1Id] = avail
		}
		if c.P2Id != "" {
			availableTime[c.P2Id] = avail
		}
		runningMatches = append(runningMatches, runningMatch{
			divisionName: c.DivisionName,
			endTime:      time.Now().Add(15 * time.Minute),
		})
	}

	getPlayerAvail := func(playerID string) time.Time {
		if playerID == "" {
			return time.Time{}
		}
		if t, ok := availableTime[playerID]; ok {
			return t
		}
		return time.Time{}
	}

	for {
		var nextRunning []runningMatch
		activeCount := make(map[string]int)
		for _, rm := range runningMatches {
			if rm.endTime.After(simClock) {
				nextRunning = append(nextRunning, rm)
				activeCount[rm.divisionName]++
			}
		}
		runningMatches = nextRunning

		type candidate struct {
			card         event.BoardCard
			maxAvail     time.Time
			ratio        float64
			poolIdx      string
			divisionName string
		}
		var candidates []candidate

		for id, pool := range pools {
			if len(pool) == 0 {
				continue
			}
			c := pool[0]
			t1 := getPlayerAvail(c.P1Id)
			t2 := getPlayerAvail(c.P2Id)
			maxAvail := t1
			if t2.After(t1) {
				maxAvail = t2
			}

			divName := c.DivisionName
			alloc := allocated[divName]
			var ratio float64
			if alloc > 0 {
				ratio = float64(activeCount[divName]) / float64(alloc)
			} else {
				ratio = float64(activeCount[divName]) / 0.5
			}

			candidates = append(candidates, candidate{
				card:         c,
				maxAvail:     maxAvail,
				ratio:        ratio,
				poolIdx:      id,
				divisionName: divName,
			})
		}

		if len(candidates) == 0 {
			break
		}

		var availableCandidates []candidate
		for _, cand := range candidates {
			if !cand.maxAvail.After(simClock) {
				availableCandidates = append(availableCandidates, cand)
			}
		}

		var chosen candidate
		if len(availableCandidates) > 0 {
			sort.Slice(availableCandidates, func(i, j int) bool {
				c1, c2 := availableCandidates[i], availableCandidates[j]
				u1 := c1.ratio < 1.0
				u2 := c2.ratio < 1.0
				if u1 != u2 {
					return u1
				}
				if c1.ratio != c2.ratio {
					return c1.ratio < c2.ratio
				}
				alloc1 := allocated[c1.divisionName]
				alloc2 := allocated[c2.divisionName]
				if alloc1 != alloc2 {
					return alloc1 > alloc2
				}
				count1 := scheduledCounts[c1.poolIdx]
				count2 := scheduledCounts[c2.poolIdx]
				if count1 != count2 {
					return count1 < count2
				}
				if !c1.maxAvail.Equal(c2.maxAvail) {
					return c1.maxAvail.Before(c2.maxAvail)
				}
				return c1.poolIdx < c2.poolIdx
			})
			chosen = availableCandidates[0]
		} else {
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].maxAvail.Before(candidates[j].maxAvail)
			})
			simClock = candidates[0].maxAvail
			continue
		}

		globalScheduled = append(globalScheduled, chosen.card)
		scheduledCounts[chosen.poolIdx]++

		runningMatches = append(runningMatches, runningMatch{
			divisionName: chosen.divisionName,
			endTime:      simClock.Add(20 * time.Minute),
		})

		avail := simClock.Add(30 * time.Minute)
		if chosen.card.P1Id != "" {
			availableTime[chosen.card.P1Id] = avail
		}
		if chosen.card.P2Id != "" {
			availableTime[chosen.card.P2Id] = avail
		}

		pools[chosen.poolIdx] = pools[chosen.poolIdx][1:]
	}

	inMatchPlayers := make(map[string]bool)
	for _, c := range inProgress {
		if c.P1Id != "" {
			inMatchPlayers[c.P1Id] = true
		}
		if c.P2Id != "" {
			inMatchPlayers[c.P2Id] = true
		}
	}
	for i := range globalScheduled {
		if globalScheduled[i].P1Id != "" && inMatchPlayers[globalScheduled[i].P1Id] {
			globalScheduled[i].P1InMatch = true
		} else {
			globalScheduled[i].P1InMatch = false
		}
		if globalScheduled[i].P2Id != "" && inMatchPlayers[globalScheduled[i].P2Id] {
			globalScheduled[i].P2InMatch = true
		} else {
			globalScheduled[i].P2InMatch = false
		}
		globalScheduled[i].QueuePosition = i + 1
	}

	return e, divs, globalScheduled, inProgress, finished, nil
}
