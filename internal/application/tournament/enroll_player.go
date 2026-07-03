package tournament

import (
	"context"

	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

// EnrollPlayerUseCase adds an existing player as a participant of a tournament,
// e.g. right after the player is created from the admin players page.
type EnrollPlayerUseCase struct {
	repo tournamentDomain.Repository
}

func NewEnrollPlayerUseCase(repo tournamentDomain.Repository) *EnrollPlayerUseCase {
	return &EnrollPlayerUseCase{repo: repo}
}

func (uc *EnrollPlayerUseCase) Execute(ctx context.Context, tournamentID, playerID string, singlesElo, doublesElo int16) error {
	return uc.repo.AddParticipant(ctx, tournamentID, playerID, singlesElo, doublesElo)
}
