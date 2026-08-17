package player

import (
	"context"

	eventApp "table-tennis-backend/internal/application/event"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentEvent "table-tennis-backend/internal/domain/event"
)

// GetPlayerPendingMatchesUseCase returns a flat list of a player's not-yet-
// finished matches, opponent-facing. Deliberately a separate use case from
// GetPlayerTournamentStatsUseCase rather than bolting onto it: different
// shape (flat pending list vs. per-tournament stats breakdown), and keeps
// that use case's existing errgroup fan-out focused.
//
// Includes both real Match rows AND "potential" matchups the admin board
// already shows before a match has actually been created in the database
// (round-robin/knockout pairings computed from group/bracket structure via
// eventApp.BuildBoardCards — the same logic backing the admin board/TV
// dashboard). Without this, a player registered into a group whose first
// round hasn't been "started" yet would see nothing here even though the
// admin board already displays it as an upcoming matchup.
type GetPlayerPendingMatchesUseCase struct {
	eventRepo    tournamentEvent.Repository
	divisionRepo divisionDomain.Repository
}

func NewGetPlayerPendingMatchesUseCase(eventRepo tournamentEvent.Repository, divisionRepo divisionDomain.Repository) *GetPlayerPendingMatchesUseCase {
	return &GetPlayerPendingMatchesUseCase{eventRepo: eventRepo, divisionRepo: divisionRepo}
}

func (uc *GetPlayerPendingMatchesUseCase) Execute(ctx context.Context, playerID string) ([]tournamentEvent.PlayerPendingMatchDetail, error) {
	events, err := uc.eventRepo.GetByParticipantID(ctx, playerID)
	if err != nil {
		return nil, err
	}

	divs, err := uc.divisionRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var pending []tournamentEvent.PlayerPendingMatchDetail
	for _, ev := range events {
		// Real matches (already persisted rows) — carries proposal state.
		real := tournamentEvent.BuildPlayerPendingMatchDetails(playerID, ev.Name, ev.Matches)
		for i := range real {
			real[i].BestOf = ev.GetEffectiveStageRule(real[i].Stage).BestOf
		}
		pending = append(pending, real...)

		// Potential matchups the board already projects but that don't exist
		// as a Match row yet (BoardCard.MatchID == ""). No proposal is
		// possible on these until a real match is created.
		scheduled, inProgress, _ := eventApp.BuildBoardCards(ev, divs)
		for _, card := range append(scheduled, inProgress...) {
			if card.MatchID != "" {
				continue // already covered by BuildPlayerPendingMatchDetails above
			}
			var opponent, opponentID string
			switch playerID {
			case card.P1Id:
				opponent, opponentID = card.PlayerBName, card.P2Id
			case card.P2Id:
				opponent, opponentID = card.PlayerAName, card.P1Id
			default:
				continue
			}
			pending = append(pending, tournamentEvent.PlayerPendingMatchDetail{
				EventID:    ev.ID,
				EventName:  ev.Name,
				Stage:      card.Stage,
				Opponent:   opponent,
				OpponentID: opponentID,
				Status:     card.Status,
				BestOf:     card.BestOf,
			})
		}
	}
	return pending, nil
}
