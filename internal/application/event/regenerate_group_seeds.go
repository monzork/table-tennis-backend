package event

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"

	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type RegenerateGroupSeedsUseCase struct {
	tournamentRepo tournamentDomain.Repository
	matchRepo      tournamentDomain.MatchRepository
	divisionRepo   divisionDomain.Repository
}

func NewRegenerateGroupSeedsUseCase(
	tournamentRepo tournamentDomain.Repository,
	matchRepo tournamentDomain.MatchRepository,
	divisionRepo divisionDomain.Repository,
) *RegenerateGroupSeedsUseCase {
	return &RegenerateGroupSeedsUseCase{
		tournamentRepo: tournamentRepo,
		matchRepo:      matchRepo,
		divisionRepo:   divisionRepo,
	}
}

func (uc *RegenerateGroupSeedsUseCase) Execute(ctx context.Context, tournamentID string) error {
	var (
		t           *tournamentDomain.Event
		hasActivity bool
		divs        []*divisionDomain.Division
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		t, err = uc.tournamentRepo.GetByID(gCtx, tournamentID)
		return err
	})

	g.Go(func() error {
		var err error
		hasActivity, err = uc.matchRepo.HasStartedOrFinishedMatches(gCtx, tournamentID)
		return err
	})

	g.Go(func() error {
		var err error
		divs, err = uc.divisionRepo.GetAll(gCtx)
		return err
	})

	if err := g.Wait(); err != nil {
		return err
	}

	if t.Status == "finished" {
		return errors.New("cannot regenerate seeds: event is already finished")
	}

	if t.ManualSeedingLocked {
		return errors.New("cannot regenerate seeds: event seeding is locked")
	}

	if hasActivity {
		return errors.New("cannot regenerate seeds: matches have already been started or finished")
	}

	var divsList []tournamentDomain.DivisionSeeding
	if !t.SkipElo {
		for _, d := range divs {
			if d.Category == "both" || d.Category == t.Type {
				divsList = append(divsList, tournamentDomain.DivisionSeeding{
					ID:     d.ID,
					Name:   d.Name,
					MinElo: d.MinElo,
					MaxElo: d.MaxElo,
				})
			}
		}
	}

	if err := (&tournamentDomain.DivisionSeeder{Divisions: divsList}).AssignGroups(t); err != nil {
		return err
	}

	if err := uc.matchRepo.DeleteByTournament(ctx, tournamentID); err != nil {
		return err
	}

	return uc.tournamentRepo.UpdateGroups(ctx, t)
}
