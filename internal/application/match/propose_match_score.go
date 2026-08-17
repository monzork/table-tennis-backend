package match

import (
	"context"
	"errors"

	accountApp "table-tennis-backend/internal/application/account"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

// ErrPlayerNotInMatch is returned when the proposing player isn't a
// participant on either side of the match.
var ErrPlayerNotInMatch = errors.New("player is not a participant in this match")

// ProposeMatchScoreUseCase lets a logged-in guardian's linked player stage a
// score on their own not-yet-finished match, without finalizing it — see
// event.MatchRepository.ProposeScore.
type ProposeMatchScoreUseCase struct {
	matchRepo      event.MatchRepository
	tournamentRepo event.Repository
	playerRepo     player.Repository
}

func NewProposeMatchScoreUseCase(matchRepo event.MatchRepository, tournamentRepo event.Repository, playerRepo player.Repository) *ProposeMatchScoreUseCase {
	return &ProposeMatchScoreUseCase{matchRepo: matchRepo, tournamentRepo: tournamentRepo, playerRepo: playerRepo}
}

// Execute validates that proposedByPlayerID both (a) belongs to the calling
// account and (b) is an actual participant on the match, then stages sets as
// a pending proposal.
func (uc *ProposeMatchScoreUseCase) Execute(ctx context.Context, accountID, matchID, proposedByPlayerID string, sets []event.MatchSet) error {
	p, err := uc.playerRepo.GetById(ctx, proposedByPlayerID)
	if err != nil {
		return err
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(p, accountID); err != nil {
		return err
	}

	m, err := uc.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}
	if !event.TeamContains(m.TeamA, proposedByPlayerID) && !event.TeamContains(m.TeamB, proposedByPlayerID) {
		return ErrPlayerNotInMatch
	}

	t, err := uc.tournamentRepo.GetByID(ctx, m.EventID)
	if err != nil {
		return err
	}
	stageRule := t.GetEffectiveStageRule(m.Stage)

	return uc.matchRepo.ProposeScore(ctx, matchID, sets, proposedByPlayerID, stageRule)
}
