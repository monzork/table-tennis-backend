package event

import (
	"context"
	"errors"

	tournamentDomain "table-tennis-backend/internal/domain/event"
)

// WalkoverRecorder is satisfied by match.UpdateMatchScoreUseCase.
type WalkoverRecorder interface {
	ExecuteWalkover(ctx context.Context, matchID, eventID, stage, forfeitingSide string) error
}

// ForfeitParticipantUseCase makes a player/team lose every undecided match
// of an event by walkover: group opponents (matches are created lazily, so
// missing ones are created here) plus any already-existing knockout match
// with both sides known.
type ForfeitParticipantUseCase struct {
	repo      tournamentDomain.Repository
	matchRepo tournamentDomain.MatchRepository
	walkover  WalkoverRecorder
}

func NewForfeitParticipantUseCase(repo tournamentDomain.Repository, matchRepo tournamentDomain.MatchRepository, walkover WalkoverRecorder) *ForfeitParticipantUseCase {
	return &ForfeitParticipantUseCase{repo: repo, matchRepo: matchRepo, walkover: walkover}
}

func (uc *ForfeitParticipantUseCase) Execute(ctx context.Context, eventID, participantID string) error {
	t, err := uc.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}
	if t.Status == "finished" {
		return errors.New("cannot forfeit: event is already finished")
	}

	matchType := "singles"
	switch t.Type {
	case "doubles", "mixed_doubles":
		matchType = "doubles"
	case "teams":
		matchType = "teams"
	}

	// Match sides hold players, but a doubles/teams participant is a team:
	// collect every ID that identifies "our" side.
	ours := map[string]bool{participantID: true}
	for _, tm := range t.Teams {
		if tm.ID == participantID {
			for _, p := range tm.Players {
				ours[p.ID] = true
			}
		}
	}
	side := func(m *tournamentDomain.Match) string {
		for _, p := range m.TeamA {
			if ours[p.ID] {
				return "A"
			}
		}
		for _, p := range m.TeamB {
			if ours[p.ID] {
				return "B"
			}
		}
		return ""
	}
	decided := func(m *tournamentDomain.Match) bool {
		return m.Status == "finished" || m.Status == "double_forfeit"
	}

	forfeit := func(matchID, stage string) error {
		m, err := uc.matchRepo.GetByID(ctx, matchID)
		if err != nil || decided(m) {
			return err
		}
		s := side(m)
		if s == "" {
			return nil
		}
		return uc.walkover.ExecuteWalkover(ctx, matchID, eventID, stage, s)
	}

	for _, g := range t.Groups {
		in := false
		for _, p := range g.Players {
			in = in || p.ID == participantID
		}
		if !in {
			continue
		}
		for _, opp := range g.Players {
			if opp.ID == participantID {
				continue
			}
			id, err := uc.matchRepo.FindOrCreateMatch(ctx, eventID, participantID, opp.ID, "group", matchType)
			if err != nil {
				return err
			}
			if err := forfeit(id, "group"); err != nil {
				return err
			}
		}
	}

	// Already-created non-group matches (knockout). Re-fetching each one
	// matters: forfeiting a round can advance the opponent into the next.
	for i := range t.Matches {
		m := &t.Matches[i]
		if m.Stage == "group" || m.TeamMatchID != nil || len(m.TeamA) == 0 || len(m.TeamB) == 0 || side(m) == "" {
			continue
		}
		if err := forfeit(m.ID, m.Stage); err != nil {
			return err
		}
	}
	return nil
}
