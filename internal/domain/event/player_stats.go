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
	Opponent string
	Won      bool
	SetsWon  int
	SetsLost int
	Sets     []PlayerSetScore
	// EloDelta is the Elo points this player gained (positive) or lost
	// (negative) from this specific match. Nil if Elo hasn't been applied
	// yet (e.g. the owning event has SkipElo, or Elo processing hasn't run).
	EloDelta *float64
}

// BuildPlayerMatchDetails returns the per-match breakdown (opponent, sets,
// points) for every finished match the given player took part in.
func BuildPlayerMatchDetails(playerID string, matches []Match) []PlayerMatchDetail {
	details := make([]PlayerMatchDetail, 0, len(matches))
	for _, m := range matches {
		if m.Status != "finished" {
			continue
		}
		isA := TeamContains(m.TeamA, playerID)
		if !isA && !TeamContains(m.TeamB, playerID) {
			continue
		}

		opponentTeam := m.TeamB
		setsWon, setsLost := m.ScoreA(), m.ScoreB()
		won := m.WinnerTeam == "A"
		eloDelta := m.EloDeltaA
		if !isA {
			opponentTeam = m.TeamA
			setsWon, setsLost = m.ScoreB(), m.ScoreA()
			won = m.WinnerTeam == "B"
			eloDelta = m.EloDeltaB
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
			Opponent: opponentName(opponentTeam),
			Won:      won,
			SetsWon:  setsWon,
			SetsLost: setsLost,
			Sets:     sets,
			EloDelta: eloDelta,
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
		})
	}
	return details
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
