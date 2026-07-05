package tournament

import (
	"context"

	"table-tennis-backend/internal/domain/events"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

// EnrollPlayerUseCase adds an existing player as a participant of a tournament,
// e.g. right after the player is created from the admin players page.
type EnrollPlayerUseCase struct {
	repo       tournamentDomain.Repository
	dispatcher events.Dispatcher
}

func NewEnrollPlayerUseCase(repo tournamentDomain.Repository, dispatcher events.Dispatcher) *EnrollPlayerUseCase {
	return &EnrollPlayerUseCase{repo: repo, dispatcher: dispatcher}
}

func (uc *EnrollPlayerUseCase) Execute(ctx context.Context, tournamentID, playerID string, singlesElo, doublesElo int16) error {
	if err := uc.repo.AddParticipant(ctx, tournamentID, playerID, singlesElo, doublesElo); err != nil {
		return err
	}
	
	if uc.dispatcher != nil {
		uc.dispatcher.DispatchAsync(ctx, events.PlayerEnrolledEvent{
			TournamentID: tournamentID,
			PlayerID:     playerID,
		})
	}
	
	return nil
}
