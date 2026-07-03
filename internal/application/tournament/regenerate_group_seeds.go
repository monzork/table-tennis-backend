package tournament

import (
	"context"
	"errors"

	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
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
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}
	if t.Status == "finished" {
		return errors.New("cannot regenerate seeds: tournament is already finished")
	}

	hasActivity, err := uc.matchRepo.HasStartedOrFinishedMatches(ctx, tournamentID)
	if err != nil {
		return err
	}
	if hasActivity {
		return errors.New("cannot regenerate seeds: matches have already been started or finished")
	}

	var divsList []tournamentDomain.DivisionSeeding
	if !t.SkipElo {
		divs, err := uc.divisionRepo.GetAll(ctx)
		if err == nil {
			for _, d := range divs {
				if d.Category == "both" || d.Category == t.Type {
					divsList = append(divsList, tournamentDomain.DivisionSeeding{
						Name:   d.Name,
						MinElo: d.MinElo,
						MaxElo: d.MaxElo,
					})
				}
			}
		}
	}

	if err := t.AssignGroupsByDivisions(divsList); err != nil {
		return err
	}

	if err := uc.matchRepo.DeleteByTournament(ctx, tournamentID); err != nil {
		return err
	}

	return uc.tournamentRepo.UpdateGroups(ctx, t)
}
