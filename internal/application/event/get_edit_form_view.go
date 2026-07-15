package event

import (
	"context"
	"sync"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

// EditFormView holds all data required to render the event edit form.
type EditFormView struct {
	Event     *tournamentDomain.Event
	Players   any
	Divisions []*divisionDomain.Division
}

// GetEditFormViewUseCase orchestrates parallel fetching of event, players, and divisions
// for rendering the admin event edit form.
type GetEditFormViewUseCase struct {
	getByID       *GetTournamentByIDUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
	divisionUC    *division.DivisionUseCase
}

func NewGetEditFormViewUseCase(
	getByID *GetTournamentByIDUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	divisionUC *division.DivisionUseCase,
) *GetEditFormViewUseCase {
	return &GetEditFormViewUseCase{
		getByID:       getByID,
		leaderboardUC: leaderboardUC,
		divisionUC:    divisionUC,
	}
}

func (uc *GetEditFormViewUseCase) Execute(ctx context.Context, id string) (*EditFormView, error) {
	type result struct {
		event     *tournamentDomain.Event
		err       error
		players   any
		divisions []*divisionDomain.Division
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
		res.players, _ = uc.leaderboardUC.ExecuteSingles(ctx)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = uc.divisionUC.GetAll(ctx)
	}()
	wg.Wait()

	if res.err != nil {
		return nil, res.err
	}

	return &EditFormView{
		Event:     res.event,
		Players:   res.players,
		Divisions: res.divisions,
	}, nil
}
