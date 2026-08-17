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

// ErrMissingVirtualMatchFields is returned when MatchID is empty (a
// not-yet-created "potential" matchup, per event.BuildBoardCards) but the
// fields needed to materialize it — EventID, OpponentID, Stage — weren't
// all supplied.
var ErrMissingVirtualMatchFields = errors.New("eventId, opponentId and stage are required to create a match")

// ProposeMatchScoreCommand carries the inputs for ProposeMatchScoreUseCase.
// MatchID is optional: the account UI also surfaces "potential" matchups
// the admin board already projects (round-robin/knockout pairings computed
// from group/bracket structure) before a real Match row exists for them —
// see event.BuildBoardCards. When MatchID is empty, EventID/OpponentID/Stage
// are used to find-or-create the real match first (event.MatchRepository.
// FindOrCreateMatch, the same lazy-creation path the public score-entry
// flow already uses), then the proposal is staged on it as usual.
type ProposeMatchScoreCommand struct {
	AccountID          string
	MatchID            string
	ProposedByPlayerID string
	EventID            string
	OpponentID         string
	Stage              string
	Sets               []event.MatchSet
}

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

// Execute validates that ProposedByPlayerID both (a) belongs to the calling
// account and (b) is an actual participant on the match, then stages sets as
// a pending proposal. If cmd.MatchID is empty, the match is created first
// (find-or-create, so a race with someone else creating it concurrently
// still resolves to a single row) from cmd.EventID/OpponentID/Stage.
func (uc *ProposeMatchScoreUseCase) Execute(ctx context.Context, cmd ProposeMatchScoreCommand) error {
	p, err := uc.playerRepo.GetById(ctx, cmd.ProposedByPlayerID)
	if err != nil {
		return err
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(p, cmd.AccountID); err != nil {
		return err
	}

	matchID := cmd.MatchID
	if matchID == "" {
		if cmd.EventID == "" || cmd.OpponentID == "" || cmd.Stage == "" {
			return ErrMissingVirtualMatchFields
		}
		matchID, err = uc.matchRepo.FindOrCreateMatch(ctx, cmd.EventID, cmd.ProposedByPlayerID, cmd.OpponentID, cmd.Stage, "singles")
		if err != nil {
			return err
		}
	}

	m, err := uc.matchRepo.GetByID(ctx, matchID)
	if err != nil {
		return err
	}
	if !event.TeamContains(m.TeamA, cmd.ProposedByPlayerID) && !event.TeamContains(m.TeamB, cmd.ProposedByPlayerID) {
		return ErrPlayerNotInMatch
	}

	t, err := uc.tournamentRepo.GetByID(ctx, m.EventID)
	if err != nil {
		return err
	}
	stageRule := t.GetEffectiveStageRule(m.Stage)

	return uc.matchRepo.ProposeScore(ctx, matchID, cmd.Sets, cmd.ProposedByPlayerID, stageRule)
}
