package event

import "table-tennis-backend/internal/domain/player"

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
}

// BuildPlayerMatchDetails returns the per-match breakdown (opponent, sets,
// points) for every finished match the given player took part in.
func BuildPlayerMatchDetails(playerID string, matches []Match) []PlayerMatchDetail {
	details := make([]PlayerMatchDetail, 0, len(matches))
	for _, m := range matches {
		if m.Status != "finished" {
			continue
		}
		isA := teamContains(m.TeamA, playerID)
		if !isA && !teamContains(m.TeamB, playerID) {
			continue
		}

		opponentTeam := m.TeamB
		setsWon, setsLost := m.ScoreA(), m.ScoreB()
		won := m.WinnerTeam == "A"
		if !isA {
			opponentTeam = m.TeamA
			setsWon, setsLost = m.ScoreB(), m.ScoreA()
			won = m.WinnerTeam == "B"
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
		})
	}
	return details
}

func teamContains(team []*player.Player, playerID string) bool {
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
