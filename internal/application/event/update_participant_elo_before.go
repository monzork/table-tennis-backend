package event

import (
	"context"

	tournamentDomain "table-tennis-backend/internal/domain/event"
)

// UpdateParticipantEloBeforeUseCase corrects the Elo a participant was seeded
// with for a event, then re-groups the whole event from the
// corrected values so the seeding actually reflects the fix.
type UpdateParticipantEloBeforeUseCase struct {
	repo            tournamentDomain.Repository
	regenerateSeeds *RegenerateGroupSeedsUseCase
}

func NewUpdateParticipantEloBeforeUseCase(
	repo tournamentDomain.Repository,
	regenerateSeeds *RegenerateGroupSeedsUseCase,
) *UpdateParticipantEloBeforeUseCase {
	return &UpdateParticipantEloBeforeUseCase{repo: repo, regenerateSeeds: regenerateSeeds}
}

func (uc *UpdateParticipantEloBeforeUseCase) Execute(ctx context.Context, tournamentID, playerID string, singlesElo, doublesElo int16) error {
	if err := uc.repo.UpdateParticipantEloBefore(ctx, tournamentID, playerID, singlesElo, doublesElo); err != nil {
		return err
	}
	return uc.regenerateSeeds.Execute(ctx, tournamentID)
}
