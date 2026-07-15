package event

import (
	"context"
	"sort"
	"strings"
	"sync"

	"table-tennis-backend/internal/application/division"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type BoardView struct {
	Event      *tournamentDomain.Event
	Divisions  []*divisionDomain.Division
	Scheduled  []BoardCard
	InProgress []BoardCard
	Finished   []BoardCard
	AllDivs    []string
	Tables     any
}

type GetBoardViewUseCase struct {
	getByID    *GetTournamentByIDUseCase
	divisionUC *division.DivisionUseCase
}

func NewGetBoardViewUseCase(getByID *GetTournamentByIDUseCase, divisionUC *division.DivisionUseCase) *GetBoardViewUseCase {
	return &GetBoardViewUseCase{
		getByID:    getByID,
		divisionUC: divisionUC,
	}
}

func (uc *GetBoardViewUseCase) Execute(
	ctx context.Context, id, q string, selectedDivs []string,
	buildBoardCards func(*tournamentDomain.Event, []*divisionDomain.Division) ([]BoardCard, []BoardCard, []BoardCard),
	filterBoardCards func([]BoardCard, string, []string) []BoardCard,
) (*BoardView, error) {
	type result struct {
		event     *tournamentDomain.Event
		err       error
		divisions []*divisionDomain.Division
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.event, res.err = uc.getByID.Execute(ctx, id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = uc.divisionUC.GetAll(ctx)
	}()
	wg.Wait()

	if res.err != nil {
		return nil, res.err
	}

	scheduled, inProgress, finished := buildBoardCards(res.event, res.divisions)

	uniqueDivsMap := make(map[string]bool)
	for _, c := range scheduled {
		if c.DivisionName != "" {
			uniqueDivsMap[c.DivisionName] = true
		}
	}
	for _, c := range inProgress {
		if c.DivisionName != "" {
			uniqueDivsMap[c.DivisionName] = true
		}
	}
	for _, c := range finished {
		if c.DivisionName != "" {
			uniqueDivsMap[c.DivisionName] = true
		}
	}
	var allDivs []string
	for d := range uniqueDivsMap {
		allDivs = append(allDivs, d)
	}
	sort.Strings(allDivs)

	q = strings.ToLower(q)
	if q != "" || len(selectedDivs) > 0 {
		scheduled = filterBoardCards(scheduled, q, selectedDivs)
		inProgress = filterBoardCards(inProgress, q, selectedDivs)
		finished = filterBoardCards(finished, q, selectedDivs)
	}

	return &BoardView{
		Event:      res.event,
		Divisions:  res.divisions,
		Scheduled:  scheduled,
		InProgress: inProgress,
		Finished:   finished,
		AllDivs:    allDivs,
	}, nil
}
