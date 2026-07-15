package event

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type PublicTVDashboardView struct {
	Event            *tournamentDomain.Event
	Divisions        []*divisionDomain.Division
	BracketViewModel *bracket.Bracket
	Scheduled        []BoardCard
	InProgress       []BoardCard
	Finished         []BoardCard
	Tables           any // Will be casted in handler
}

type GetPublicTVDashboardViewUseCase struct {
	getByID       *GetTournamentByIDUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
	divisionUC    *division.DivisionUseCase
}

func NewGetPublicTVDashboardViewUseCase(getByID *GetTournamentByIDUseCase, leaderboardUC *leaderboard.GetLeaderboardUseCase, divisionUC *division.DivisionUseCase) *GetPublicTVDashboardViewUseCase {
	return &GetPublicTVDashboardViewUseCase{
		getByID:       getByID,
		leaderboardUC: leaderboardUC,
		divisionUC:    divisionUC,
	}
}

func (uc *GetPublicTVDashboardViewUseCase) Execute(
	ctx context.Context, id, playerSearch string,
	buildBoardCards func(*tournamentDomain.Event, []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard),
	tmap map[string]string,
) (*PublicTVDashboardView, error) {
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
		res.event, res.err = uc.getByID.Execute(ctx, id)
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

	vm := bracket.BuildBracket(res.event, res.divisions, tmap)
	vm.IsPublic = true

	scheduled, inProgress, finished := buildBoardCards(res.event, res.divisions)

	playerSearch = strings.ToLower(playerSearch)
	if playerSearch != "" {
		searchTerms := strings.Fields(playerSearch)
		filterCards := func(cards []BoardCard) []BoardCard {
			var result []BoardCard
			for _, card := range cards {
				fullMatchString := strings.ToLower(fmt.Sprintf("%s %s", card.PlayerAName, card.PlayerBName))
				match := true
				for _, term := range searchTerms {
					if !strings.Contains(fullMatchString, term) {
						match = false
						break
					}
				}
				if match {
					result = append(result, card)
				}
			}
			return result
		}
		scheduled = filterCards(scheduled)
		inProgress = filterCards(inProgress)
		finished = filterCards(finished)
	}

	return &PublicTVDashboardView{
		Event:            res.event,
		Divisions:        res.divisions,
		BracketViewModel: vm,
		Scheduled:        scheduled,
		InProgress:       inProgress,
		Finished:         finished,
	}, nil
}
