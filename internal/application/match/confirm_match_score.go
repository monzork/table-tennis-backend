package match

import (
	"context"
	"errors"
	"fmt"

	"table-tennis-backend/internal/domain/event"
)

// ErrNoScoreProposal is returned when there's nothing staged to confirm/reject.
var ErrNoScoreProposal = errors.New("match has no pending score proposal")

// ErrNotOpposingParticipant is returned when the confirming player isn't on
// the opposing team from whoever proposed the score (or is the proposer
// themselves trying to confirm their own submission).
var ErrNotOpposingParticipant = errors.New("only the opposing player can confirm this score")

// ConfirmMatchScoreUseCase finalizes a staged score proposal by calling the
// existing, unchanged UpdateMatchScoreUseCase — the exact same code path
// admin/QR already use (bracket advancement etc. all reused, zero
// duplication) — then clears the proposal.
type ConfirmMatchScoreUseCase struct {
	matchRepo      event.MatchRepository
	tournamentRepo event.Repository
	updateScoreUC  *UpdateMatchScoreUseCase
}

func NewConfirmMatchScoreUseCase(matchRepo event.MatchRepository, tournamentRepo event.Repository, updateScoreUC *UpdateMatchScoreUseCase) *ConfirmMatchScoreUseCase {
	return &ConfirmMatchScoreUseCase{matchRepo: matchRepo, tournamentRepo: tournamentRepo, updateScoreUC: updateScoreUC}
}

// Execute finalizes matchID's staged proposal. If isAdmin, the opposing-
// participant check is skipped (an admin verifying the result directly).
// Otherwise confirmedByPlayerID must be a match participant on the opposing
// team from Match.ProposedByPlayerID, and not the proposer themselves.
func (uc *ConfirmMatchScoreUseCase) Execute(ctx context.Context, matchID string, confirmedByPlayerID *string, isAdmin bool) error {
	m, err := uc.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}
	if m.ProposedByPlayerID == nil || len(m.ProposedSets) == 0 {
		return ErrNoScoreProposal
	}

	if !isAdmin {
		if confirmedByPlayerID == nil {
			return ErrNotOpposingParticipant
		}
		if *confirmedByPlayerID == *m.ProposedByPlayerID {
			return ErrNotOpposingParticipant
		}
		proposerOnA := event.TeamContains(m.TeamA, *m.ProposedByPlayerID)
		confirmerOnA := event.TeamContains(m.TeamA, *confirmedByPlayerID)
		confirmerOnB := event.TeamContains(m.TeamB, *confirmedByPlayerID)
		if !confirmerOnA && !confirmerOnB {
			return ErrNotOpposingParticipant
		}
		if proposerOnA == confirmerOnA {
			// Same side as the proposer (or neither side actually matched).
			return ErrNotOpposingParticipant
		}
	}

	rawScores := make([]string, len(m.ProposedSets))
	for i, s := range m.ProposedSets {
		rawScores[i] = fmt.Sprintf("%d-%d", s.ScoreA, s.ScoreB)
	}

	if err := uc.updateScoreUC.Execute(ctx, matchID, rawScores, m.EventID, m.Stage); err != nil {
		return err
	}

	return uc.matchRepo.ClearScoreProposal(ctx, matchID)
}
