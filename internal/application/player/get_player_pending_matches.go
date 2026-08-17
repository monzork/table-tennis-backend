package player

import (
	"context"

	tournamentEvent "table-tennis-backend/internal/domain/event"
)

// GetPlayerPendingMatchesUseCase returns a flat list of a player's not-yet-
// finished matches, opponent-facing. Deliberately a separate use case from
// GetPlayerTournamentStatsUseCase rather than bolting onto it: different
// shape (flat pending list vs. per-tournament stats breakdown), and keeps
// that use case's existing errgroup fan-out focused.
type GetPlayerPendingMatchesUseCase struct {
	eventRepo tournamentEvent.Repository
}

func NewGetPlayerPendingMatchesUseCase(eventRepo tournamentEvent.Repository) *GetPlayerPendingMatchesUseCase {
	return &GetPlayerPendingMatchesUseCase{eventRepo: eventRepo}
}

func (uc *GetPlayerPendingMatchesUseCase) Execute(ctx context.Context, playerID string) ([]tournamentEvent.PlayerPendingMatchDetail, error) {
	events, err := uc.eventRepo.GetByParticipantID(ctx, playerID)
	if err != nil {
		return nil, err
	}

	var pending []tournamentEvent.PlayerPendingMatchDetail
	for _, ev := range events {
		pending = append(pending, tournamentEvent.BuildPlayerPendingMatchDetails(playerID, ev.Matches)...)
	}
	return pending, nil
}
