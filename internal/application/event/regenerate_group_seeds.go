package event

import (
	"context"
	"errors"

	"golang.org/x/sync/errgroup"

	tournamentDomain "table-tennis-backend/internal/domain/event"
)

type RegenerateGroupSeedsUseCase struct {
	tournamentRepo tournamentDomain.Repository
	matchRepo      tournamentDomain.MatchRepository
}

func NewRegenerateGroupSeedsUseCase(
	tournamentRepo tournamentDomain.Repository,
	matchRepo tournamentDomain.MatchRepository,
) *RegenerateGroupSeedsUseCase {
	return &RegenerateGroupSeedsUseCase{
		tournamentRepo: tournamentRepo,
		matchRepo:      matchRepo,
	}
}

func (uc *RegenerateGroupSeedsUseCase) Execute(ctx context.Context, eventID string) error {
	var (
		t           *tournamentDomain.Event
		hasActivity bool
	)

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		var err error
		t, err = uc.tournamentRepo.GetByID(gCtx, eventID)
		return err
	})

	g.Go(func() error {
		var err error
		hasActivity, err = uc.matchRepo.HasStartedOrFinishedMatches(gCtx, eventID)
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

	if err := (&tournamentDomain.OpenBracketSnakeSeeder{}).AssignGroups(t); err != nil {
		return err
	}

	if err := uc.matchRepo.DeleteByEvent(ctx, eventID); err != nil {
		return err
	}

	return uc.tournamentRepo.UpdateGroups(ctx, t)
}
