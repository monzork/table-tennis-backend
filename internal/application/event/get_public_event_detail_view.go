package event

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

type PublicEventDetailView struct {
	Event            *tournamentDomain.Event
	Divisions        []*divisionDomain.Division
	BracketViewModel *bracket.Bracket
	RefereeNames     map[string]string
	JSONLD           string
}

type GetPublicEventDetailViewUseCase struct {
	getByID          *GetTournamentByIDUseCase
	leaderboardUC    *leaderboard.GetLeaderboardUseCase
	divisionUC       *division.DivisionUseCase
	buildBoardCards  func(*tournamentDomain.Event, []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard) // We'll inject or use a domain service for BoardCards if needed, wait.
}

func NewGetPublicEventDetailViewUseCase(getByID *GetTournamentByIDUseCase, leaderboardUC *leaderboard.GetLeaderboardUseCase, divisionUC *division.DivisionUseCase) *GetPublicEventDetailViewUseCase {
	return &GetPublicEventDetailViewUseCase{
		getByID:       getByID,
		leaderboardUC: leaderboardUC,
		divisionUC:    divisionUC,
	}
}

// Execute performs the complex orchestration and view-model construction.
func (uc *GetPublicEventDetailViewUseCase) Execute(ctx context.Context, tournamentID string, statusFilter, playerSearch string, canonicalURL string, buildBoardCards func(*tournamentDomain.Event, []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard), tmap map[string]string) (*PublicEventDetailView, error) {
	type result struct {
		event     *tournamentDomain.Event
		err       error
		divisions []*divisionDomain.Division
		players   any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		res.event, res.err = uc.getByID.Execute(ctx, tournamentID)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = uc.divisionUC.GetAll(ctx)
	}()
	go func() {
		defer wg.Done()
		res.players, _ = uc.leaderboardUC.ExecuteSingles(ctx)
	}()
	wg.Wait()

	if res.err != nil {
		return nil, res.err
	}
	t := res.event
	divisions := res.divisions

	playerSearch = strings.ToLower(playerSearch)

	// 1. Generate virtual scheduled matches and append to allMatches
	allMatches := t.Matches
	scheduledCards, _, _ := buildBoardCards(t, divisions)

	queuePosMap := make(map[string]int)
	for i, sc := range scheduledCards {
		key := sc.MatchID
		if key == "" {
			key = fmt.Sprintf("virtual_%s_%s", sc.P1Id, sc.P2Id)
		}
		queuePosMap[key] = i + 1
	}

	for i := range allMatches {
		if allMatches[i].ID != "" {
			if pos, ok := queuePosMap[allMatches[i].ID]; ok {
				allMatches[i].QueuePosition = pos
			}
		}
	}

	for _, sc := range scheduledCards {
		if sc.MatchID == "" {
			var teamA, teamB []*player.Player

			if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
				for _, tm := range t.Teams {
					if tm.ID == sc.P1Id {
						avgElo := tm.AverageElo(t.Type)
						teamA = append(teamA, &player.Player{
							ID: tm.ID, FirstName: tm.Name, LastName: " (Team)", SinglesElo: avgElo, DoublesElo: avgElo,
						})
					}
					if tm.ID == sc.P2Id {
						avgElo := tm.AverageElo(t.Type)
						teamB = append(teamB, &player.Player{
							ID: tm.ID, FirstName: tm.Name, LastName: " (Team)", SinglesElo: avgElo, DoublesElo: avgElo,
						})
					}
				}
			} else {
				for _, p := range t.Participants {
					if p.ID == sc.P1Id {
						teamA = append(teamA, p)
					}
					if p.ID == sc.P2Id {
						teamB = append(teamB, p)
					}
				}
			}

			if len(teamA) > 0 && len(teamB) > 0 {
				allMatches = append(allMatches, tournamentDomain.Match{
					ID:            "",
					TournamentID:  t.ID,
					MatchType:     t.Type,
					TeamA:         teamA,
					TeamB:         teamB,
					Status:        "scheduled",
					Stage:         sc.Stage,
					QueuePosition: queuePosMap[fmt.Sprintf("virtual_%s_%s", sc.P1Id, sc.P2Id)],
				})
			}
		}
	}

	// 2. Build the BracketViewModel using ALL matches
	tForVM := *t
	tForVM.Matches = allMatches
	vm := bracket.BuildBracket(&tForVM, divisions, tmap)
	vm.IsPublic = true

	// 3. Filter matches for the list view below the bracket
	var displayMatches []tournamentDomain.Match
	for _, m := range allMatches {
		matchStatus := statusFilter == "all" || m.Status == statusFilter
		matchPlayer := true

		if playerSearch != "" {
			searchTerms := strings.Fields(playerSearch)
			var names []string
			for _, p := range m.TeamA {
				names = append(names, strings.ToLower(fmt.Sprintf("%s %s %s %s", p.FirstName, p.SecondName, p.LastName, p.SecondLastName)))
			}
			for _, p := range m.TeamB {
				names = append(names, strings.ToLower(fmt.Sprintf("%s %s %s %s", p.FirstName, p.SecondName, p.LastName, p.SecondLastName)))
			}
			fullMatchString := strings.Join(names, " ")

			matchPlayer = true
			for _, term := range searchTerms {
				if !strings.Contains(fullMatchString, term) {
					matchPlayer = false
					break
				}
			}
		}

		if matchStatus && matchPlayer {
			displayMatches = append(displayMatches, m)
		}
	}
	t.Matches = displayMatches

	// 4. Build a map of Referee IDs to Names
	refereeNames := make(map[string]string)
	if allPlayers, ok := res.players.([]*player.Player); ok {
		for _, m := range t.Matches {
			if m.RefereeID != nil {
				for _, p := range allPlayers {
					if p.ID == *m.RefereeID {
						refereeNames[*m.RefereeID] = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
						break
					}
				}
			}
		}
	}

	// 5. SEO additions
	safeName := strings.ReplaceAll(t.Name, `"`, `\"`)
	jsonLD := fmt.Sprintf(`{
  "@context": "https://schema.org",
  "@type": "SportsEvent",
  "name": "%s",
  "startDate": "%s",
  "endDate": "%s",
  "sport": "Table Tennis",
  "url": "%s",
  "location": {
    "@type": "Place",
    "name": "Nicaragua"
  }
}`, safeName, t.StartDate.Format(time.RFC3339), t.EndDate.Format(time.RFC3339), canonicalURL)

	return &PublicEventDetailView{
		Event:            t,
		Divisions:        divisions,
		BracketViewModel: vm,
		RefereeNames:     refereeNames,
		JSONLD:           jsonLD,
	}, nil
}


