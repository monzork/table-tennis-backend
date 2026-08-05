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
