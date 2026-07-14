package event

import (
	"context"

	"table-tennis-backend/internal/domain/tournaments"
	tournamentDomain "table-tennis-backend/internal/domain/event"
)

// EnrollPlayerUseCase adds an existing player as a participant of a event,
// e.g. right after the player is created from the admin players page.
type EnrollPlayerUseCase struct {
	repo       tournamentDomain.Repository
	dispatcher tournaments.Dispatcher
}

func NewEnrollPlayerUseCase(repo tournamentDomain.Repository, dispatcher tournaments.Dispatcher) *EnrollPlayerUseCase {
	return &EnrollPlayerUseCase{repo: repo, dispatcher: dispatcher}
}

func (uc *EnrollPlayerUseCase) Execute(ctx context.Context, tournamentID, playerID string, singlesElo, doublesElo int16) error {
	if err := uc.repo.AddParticipant(ctx, tournamentID, playerID, singlesElo, doublesElo); err != nil {
		return err
	}
	
	if uc.dispatcher != nil {
		uc.dispatcher.DispatchAsync(ctx, tournaments.PlayerEnrolledEvent{
			TournamentID: tournamentID,
			PlayerID:     playerID,
		})
	}
	
	return nil
}
