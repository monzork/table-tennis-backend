package event

import (
	"context"
	"fmt"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournaments"
)

// EnrollPlayerUseCase adds an existing player as a participant of a event,
// e.g. right after the player is created from the admin players page.
type EnrollPlayerUseCase struct {
	repo       tournamentDomain.Repository
	playerRepo playerDomain.Repository
	dispatcher tournaments.Dispatcher
}

func NewEnrollPlayerUseCase(repo tournamentDomain.Repository, playerRepo playerDomain.Repository, dispatcher tournaments.Dispatcher) *EnrollPlayerUseCase {
	return &EnrollPlayerUseCase{repo: repo, playerRepo: playerRepo, dispatcher: dispatcher}
}

func (uc *EnrollPlayerUseCase) Execute(ctx context.Context, eventID, playerID string) error {
	ev, err := uc.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	p, err := uc.playerRepo.GetById(ctx, playerID)
	if err != nil {
		return err
	}
	if !tournamentDomain.IsAgeEligible(p, ev.AgeCategory, ev.StartDate.Year()) {
		return fmt.Errorf("restricted: %s %s does not meet the %s age requirement", p.FirstName, p.LastName, ev.AgeCategory)
	}

	// Seed elo_before from whichever Elo pool this event's age category
	// dictates (see player.Player.EloFor) -- not always the Open rating.
	singlesElo := p.EloFor(ev.AgeCategory, "singles")
	doublesElo := p.EloFor(ev.AgeCategory, "doubles")
	if err := uc.repo.AddParticipant(ctx, eventID, playerID, singlesElo, doublesElo); err != nil {
		return err
	}

	if uc.dispatcher != nil {
		uc.dispatcher.DispatchAsync(ctx, tournaments.PlayerEnrolledEvent{
			EventID:  eventID,
			PlayerID: playerID,
		})
	}

	return nil
}
