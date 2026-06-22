package tournament

import (
	"strings"

	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

// getTournamentPlaces determines the 1st, 2nd, and 3rd place winners of a finished tournament.
func getTournamentPlaces(t *tournamentDomain.Tournament) (first, second, third string) {
	if t.Status != "finished" {
		return "", "", ""
	}

	if t.Format == "elimination" || t.Format == "groups_elimination" {
		// 1st and 2nd Place: Final Match
		var finalMatch *tournamentDomain.Match
		for i := range t.Matches {
			m := &t.Matches[i]
			if m.Stage == "final" && m.Status == "finished" && m.TeamMatchID == nil {
				finalMatch = m
				break
			}
		}
		if finalMatch != nil && finalMatch.WinnerTeam != "" {
			if finalMatch.WinnerTeam == "A" {
				first = getTeamDisplayName(finalMatch.TeamA, t.Type)
				second = getTeamDisplayName(finalMatch.TeamB, t.Type)
			} else {
				first = getTeamDisplayName(finalMatch.TeamB, t.Type)
				second = getTeamDisplayName(finalMatch.TeamA, t.Type)
			}
		} else {
			// Fallback to WinnerName
			first = t.WinnerName
		}

		// 3rd Place: Semifinal losers
		var semiLosers []string
		for i := range t.Matches {
			m := &t.Matches[i]
			if m.Stage == "semifinal" && m.Status == "finished" && m.TeamMatchID == nil {
				if m.WinnerTeam == "A" {
					semiLosers = append(semiLosers, getTeamDisplayName(m.TeamB, t.Type))
				} else if m.WinnerTeam == "B" {
					semiLosers = append(semiLosers, getTeamDisplayName(m.TeamA, t.Type))
				}
			}
		}
		if len(semiLosers) > 0 {
			third = strings.Join(semiLosers, " & ")
		}

	} else if t.Format == "round_robin" {
		var participants []*player.Player
		if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
			participants = make([]*player.Player, len(t.Teams))
			for i, team := range t.Teams {
				participants[i] = &player.Player{
					ID:        team.ID,
					FirstName: team.Name,
					LastName:  "",
				}
			}
		} else {
			participants = t.Participants
		}

		if len(participants) > 0 {
			standings := tournamentDomain.BuildStandings(participants, t.Matches)
			if len(standings) > 0 {
				first = getTeamDisplayName([]*player.Player{standings[0].Player}, t.Type)
			}
			if len(standings) > 1 {
				second = getTeamDisplayName([]*player.Player{standings[1].Player}, t.Type)
			}
			if len(standings) > 2 {
				third = getTeamDisplayName([]*player.Player{standings[2].Player}, t.Type)
			}
		}
	}

	return first, second, third
}
