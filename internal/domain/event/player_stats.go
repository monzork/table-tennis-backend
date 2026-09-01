package event

import (
	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
)

// PlayerEventStats is a player's full match record within a single event,
// across every stage (group and knockout).
type PlayerEventStats struct {
	Played     int
	Wins       int
	Losses     int
	SetsWon    int
	SetsLost   int
	PointsWon  int
	PointsLost int
}

// BuildAllPlayerEventStats computes the match record of every player who
// appears in matches, in a single pass, counting group and knockout stages
// alike and crediting doubles/team matches to every player on the roster.
func BuildAllPlayerEventStats(matches []Match) map[string]PlayerEventStats {
	stats := make(map[string]PlayerEventStats)
	for _, m := range matches {
		if m.Status != "finished" {
			continue
		}
		scoreA, scoreB := m.ScoreA(), m.ScoreB()
		applyTeamStats(stats, m, m.TeamA, m.WinnerTeam == "A", scoreA, scoreB, true)
		applyTeamStats(stats, m, m.TeamB, m.WinnerTeam == "B", scoreB, scoreA, false)
	}
	return stats
}

// BuildPlayerEventStats computes a single player's match record for all
// finished matches in the given slice. It is a thin convenience wrapper
// around BuildAllPlayerEventStats for callers that only need one player.
func BuildPlayerEventStats(playerID string, matches []Match) PlayerEventStats {
	return BuildAllPlayerEventStats(matches)[playerID]
}

// PlayerSetScore is one set's score from a single player's perspective.
type PlayerSetScore struct {
	Number   int
	Own      int
	Opponent int
}

// PlayerMatchDetail is a single finished match's result from one player's
// perspective: who they played, whether they won, and the set-by-set score.
type PlayerMatchDetail struct {
	Opponent   string
	OpponentID string
	Won        bool
	SetsWon  int
	SetsLost int
	Sets     []PlayerSetScore
	// EloDelta is the Elo points this player gained (positive) or lost
	// (negative) from this specific match. When the real applied delta
	// isn't available yet (event not finished/recalculated), this falls
	// back to a preview computed from current Elo and the real result --
	// see EloDeltaIsPreview. Nil only when neither is computable (e.g. the
	// owning event has SkipElo).
	EloDelta          *float64
	EloDeltaIsPreview bool
}

// BuildPlayerMatchDetails returns the per-match breakdown (opponent, sets,
// points) for every finished match the given player took part in.
// eventFinished gates the Elo preview fallback: once the owning event is
// finished, a missing EloDeltaA/B means the real value predates per-match
// Elo tracking and simply wasn't recorded -- previewing it from *current*
// Elo at that point would be actively misleading (current Elo has since
// moved through every match played since), so it's left nil rather than
// showing a plausible-looking wrong number. The preview only makes sense
// while the event is still genuinely unprocessed.
func BuildPlayerMatchDetails(playerID string, matches []Match, eventFinished bool) []PlayerMatchDetail {
	details := make([]PlayerMatchDetail, 0, len(matches))
	for _, m := range matches {
		if m.Status != "finished" {
			continue
		}
		isA := TeamContains(m.TeamA, playerID)
		if !isA && !TeamContains(m.TeamB, playerID) {
			continue
		}

		ownTeam := m.TeamA
		opponentTeam := m.TeamB
		setsWon, setsLost := m.ScoreA(), m.ScoreB()
		won := m.WinnerTeam == "A"
		eloDelta := m.EloDeltaA
		if !isA {
			ownTeam = m.TeamB
			opponentTeam = m.TeamA
			setsWon, setsLost = m.ScoreB(), m.ScoreA()
			won = m.WinnerTeam == "B"
			eloDelta = m.EloDeltaB
		}

		isPreview := false
		if eloDelta == nil && !eventFinished {
			eloDelta = finishedPreviewEloDeltaFor(m, isA, ownTeam, opponentTeam)
			isPreview = eloDelta != nil
		}

		sets := make([]PlayerSetScore, 0, len(m.Sets))
		for _, s := range m.Sets {
			own, opp := s.ScoreA, s.ScoreB
			if !isA {
				own, opp = s.ScoreB, s.ScoreA
			}
			sets = append(sets, PlayerSetScore{Number: s.Number, Own: own, Opponent: opp})
		}

		details = append(details, PlayerMatchDetail{
			Opponent:          opponentName(opponentTeam),
			OpponentID:        opponentID(opponentTeam),
			Won:               won,
			SetsWon:           setsWon,
			SetsLost:          setsLost,
			Sets:              sets,
			EloDelta:          eloDelta,
			EloDeltaIsPreview: isPreview,
		})
	}
	return details
}

// PlayerPendingMatchDetail is a single not-yet-finished match from one
// player's perspective, for account-dashboard "pending matches" views.
// Deliberately not PlayerMatchDetail, whose Won/set-score fields only make
// sense for a finished match.
type PlayerPendingMatchDetail struct {
	MatchID   string
	EventID   string // owning event — needed to materialize a virtual match
	EventName string // which event/tournament this match belongs to
	Stage     string

	Opponent   string
	OpponentID string

	Status       string // scheduled, in_progress
	TableNumber  *int
	HasProposal  bool
	ProposedByMe bool

	// BestOf is the number of sets configured for this match's stage (e.g.
	// Bo5) — not set by BuildPlayerPendingMatchDetails itself since that
	// requires the owning Event's stage rules; populated by the caller
	// (GetPlayerPendingMatchesUseCase) via Event.GetEffectiveStageRule.
	BestOf int

	// ProjectedEloWin/ProjectedEloLoss are what this player's Elo would
	// change by if they win/lose this match, computed from both sides'
	// *current* Elo -- a preview, not a guarantee (the real change at
	// finish time depends on ratings as of when the match is actually
	// played). Nil when the opponent isn't a resolved player yet (e.g. a
	// still-TBD bracket slot).
	ProjectedEloWin  *float64
	ProjectedEloLoss *float64

	// ProposedEloDelta is the actual Elo points this player would gain/lose
	// if the pending score proposal gets confirmed as-is, computed from the
	// real proposed sets (not a win/loss coin-flip like ProjectedEloWin/Loss
	// above -- the proposed result already picks an outcome). Nil when
	// there's no active proposal or its sets don't yet decide a winner.
	// ProposedWon is only meaningful when ProposedEloDelta is non-nil.
	ProposedEloDelta *float64
	ProposedWon      bool
}

// BuildPlayerPendingMatchDetails returns every not-yet-finished match the
// given player takes part in, opponent-facing, including whether a score
// proposal is currently staged on it and whether this player is the one who
// proposed it. eventName is stamped onto every detail so a guardian with a
// player in multiple simultaneous events can tell them apart.
func BuildPlayerPendingMatchDetails(playerID, eventName string, matches []Match) []PlayerPendingMatchDetail {
	details := make([]PlayerPendingMatchDetail, 0, len(matches))
	for _, m := range matches {
		if m.Status == "finished" {
			continue
		}
		isA := TeamContains(m.TeamA, playerID)
		if !isA && !TeamContains(m.TeamB, playerID) {
			continue
		}

		ownTeam := m.TeamA
		opponentTeam := m.TeamB
		if !isA {
			ownTeam = m.TeamB
			opponentTeam = m.TeamA
		}

		hasProposal := m.ProposedByPlayerID != nil
		proposedByMe := hasProposal && *m.ProposedByPlayerID == playerID

		projWin, projLoss := ProjectedEloDelta(m.MatchType, ownTeam, opponentTeam)

		proposedWinner := m.ProposedWinner()
		proposedWon := (isA && proposedWinner == "A") || (!isA && proposedWinner == "B")

		details = append(details, PlayerPendingMatchDetail{
			MatchID:          m.ID,
			EventID:          m.EventID,
			EventName:        eventName,
			Stage:            m.Stage,
			Opponent:         opponentName(opponentTeam),
			OpponentID:       opponentID(opponentTeam),
			Status:           m.Status,
			TableNumber:      m.TableNumber,
			HasProposal:      hasProposal,
			ProposedByMe:     proposedByMe,
			ProjectedEloWin:  projWin,
			ProjectedEloLoss: projLoss,
			ProposedEloDelta: proposedEloDeltaFor(m, isA, ownTeam, opponentTeam),
			ProposedWon:      proposedWon,
		})
	}
	return details
}

// PendingProposalPreview previews the Elo effect of a still-unconfirmed
// score proposal on one specific player, from the outcome implied by the
// proposed sets (not a win/loss coin-flip like ProjectedEloWin/Loss --
// the proposed result already picks one).
type PendingProposalPreview struct {
	MatchID    string
	EventName  string
	Opponent   string
	Won        bool
	CurrentElo int16
	EloDelta   float64
}

// BuildPendingProposalPreviews returns a preview for every match the given
// player is part of that currently has an unconfirmed score proposal whose
// proposed sets already decide a winner. eventName is stamped onto every
// preview the same way BuildPlayerPendingMatchDetails does.
func BuildPendingProposalPreviews(playerID, eventName string, matches []Match) []PendingProposalPreview {
	var out []PendingProposalPreview
	for _, m := range matches {
		isA := TeamContains(m.TeamA, playerID)
		if !isA && !TeamContains(m.TeamB, playerID) {
			continue
		}
		ownTeam, opponentTeam := m.TeamA, m.TeamB
		if !isA {
			ownTeam, opponentTeam = m.TeamB, m.TeamA
		}
		delta := proposedEloDeltaFor(m, isA, ownTeam, opponentTeam)
		if delta == nil {
			continue
		}
		winner := m.ProposedWinner()
		won := (isA && winner == "A") || (!isA && winner == "B")

		var currentElo int16
		if len(ownTeam) > 0 && ownTeam[0] != nil {
			if m.MatchType == "doubles" {
				currentElo = ownTeam[0].DoublesElo
			} else {
				currentElo = ownTeam[0].SinglesElo
			}
		}

		out = append(out, PendingProposalPreview{
			MatchID:    m.ID,
			EventName:  eventName,
			Opponent:   opponentName(opponentTeam),
			Won:        won,
			CurrentElo: currentElo,
			EloDelta:   *delta,
		})
	}
	return out
}

func TeamContains(team []*player.Player, playerID string) bool {
	for _, p := range team {
		if p != nil && p.ID == playerID {
			return true
		}
	}
	return false
}

func opponentName(team []*player.Player) string {
	if len(team) == 0 || team[0] == nil {
		return ""
	}
	return team[0].FullName()
}

func opponentID(team []*player.Player) string {
	if len(team) == 0 || team[0] == nil {
		return ""
	}
	return team[0].ID
}

// proposedEloDeltaFor computes the actual Elo delta a match's pending score
// proposal would produce for the player on the isA side, or nil if there's
// no active proposal, its sets don't yet decide a winner, or the opponent
// isn't a resolved player yet.
func proposedEloDeltaFor(m Match, isA bool, ownTeam, opponentTeam []*player.Player) *float64 {
	if m.ProposedByPlayerID == nil || len(m.ProposedSets) == 0 {
		return nil
	}
	return resultEloDeltaFor(m.MatchType, isA, m.ProposedWinner(), ownTeam, opponentTeam)
}

// resultEloDeltaFor previews the Elo points a specific winner side ("A" or
// "B") would be worth for the player on the isA side, from current Elo.
// Shared by proposedEloDeltaFor (a pending proposal's implied outcome) and
// finishedPreviewEloDeltaFor (a finished-but-not-yet-processed match's real
// result). Returns nil if winnerSide is "" or either side isn't resolved.
func resultEloDeltaFor(matchType string, isA bool, winnerSide string, ownTeam, opponentTeam []*player.Player) *float64 {
	if winnerSide == "" {
		return nil
	}
	won := (isA && winnerSide == "A") || (!isA && winnerSide == "B")

	projWin, projLoss := ProjectedEloDelta(matchType, ownTeam, opponentTeam)
	if projWin == nil {
		return nil
	}
	if won {
		return projWin
	}
	return projLoss
}

// finishedPreviewEloDeltaFor previews a finished match's real Elo delta from
// *current* Elo, for use only while the owning event hasn't been finished/
// recalculated yet (i.e. the real m.EloDeltaA/B aren't populated). Returns
// nil once the real value exists -- callers should prefer that.
func finishedPreviewEloDeltaFor(m Match, isA bool, ownTeam, opponentTeam []*player.Player) *float64 {
	if m.Status != "finished" || m.WinnerTeam == "" || m.IsForfeit {
		return nil
	}
	return resultEloDeltaFor(m.MatchType, isA, m.WinnerTeam, ownTeam, opponentTeam)
}

// ProjectedEloDelta previews the Elo points a win/loss would be worth right
// now, from both sides' current Elo. Doubles uses each team's average, same
// as match.CalculateAndApplyElo. Returns (nil, nil) when either side isn't a
// resolved player yet (e.g. a still-TBD bracket slot).
func ProjectedEloDelta(matchType string, ownTeam, opponentTeam []*player.Player) (*float64, *float64) {
	if len(ownTeam) == 0 || len(opponentTeam) == 0 {
		return nil, nil
	}
	for _, p := range ownTeam {
		if p == nil {
			return nil, nil
		}
	}
	for _, p := range opponentTeam {
		if p == nil {
			return nil, nil
		}
	}

	eloOf := func(team []*player.Player) int {
		sum := 0
		for _, p := range team {
			if matchType == "doubles" {
				sum += int(p.DoublesElo)
			} else {
				sum += int(p.SinglesElo)
			}
		}
		return sum / len(team)
	}

	own, opp := eloOf(ownTeam), eloOf(opponentTeam)
	win := match.StandardEloPoints(own, opp, true, match.DefaultKFactor)
	loss := match.StandardEloPoints(own, opp, false, match.DefaultKFactor)
	return &win, &loss
}

func applyTeamStats(stats map[string]PlayerEventStats, m Match, team []*player.Player, won bool, setsWon, setsLost int, isA bool) {
	for _, p := range team {
		if p == nil {
			continue
		}
		s := stats[p.ID]
		if won {
			s.Wins++
		} else {
			s.Losses++
		}
		s.SetsWon += setsWon
		s.SetsLost += setsLost
		for _, set := range m.Sets {
			if isA {
				s.PointsWon += set.ScoreA
				s.PointsLost += set.ScoreB
			} else {
				s.PointsWon += set.ScoreB
				s.PointsLost += set.ScoreA
			}
		}
		s.Played = s.Wins + s.Losses
		stats[p.ID] = s
	}
}
