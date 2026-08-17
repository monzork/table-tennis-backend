package match

import (
	"context"

	"table-tennis-backend/internal/domain/event"
)

// RejectMatchScoreProposalUseCase clears a staged proposal without
// finalizing it, letting the other side re-propose (e.g. submit a
// correction) via ProposeMatchScoreUseCase again.
type RejectMatchScoreProposalUseCase struct {
	matchRepo event.MatchRepository
}

func NewRejectMatchScoreProposalUseCase(matchRepo event.MatchRepository) *RejectMatchScoreProposalUseCase {
	return &RejectMatchScoreProposalUseCase{matchRepo: matchRepo}
}

// Execute requires rejectedByPlayerID to be a match participant on the
// opposing team from Match.ProposedByPlayerID (same rule as confirm).
func (uc *RejectMatchScoreProposalUseCase) Execute(ctx context.Context, matchID, rejectedByPlayerID string) error {
	m, err := uc.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}
	if m.ProposedByPlayerID == nil {
		return ErrNoScoreProposal
	}
	if rejectedByPlayerID == *m.ProposedByPlayerID {
		return ErrNotOpposingParticipant
	}
	proposerOnA := event.TeamContains(m.TeamA, *m.ProposedByPlayerID)
	rejecterOnA := event.TeamContains(m.TeamA, rejectedByPlayerID)
	rejecterOnB := event.TeamContains(m.TeamB, rejectedByPlayerID)
	if !rejecterOnA && !rejecterOnB {
		return ErrNotOpposingParticipant
	}
	if proposerOnA == rejecterOnA {
		return ErrNotOpposingParticipant
	}

	return uc.matchRepo.ClearScoreProposal(ctx, matchID)
}
