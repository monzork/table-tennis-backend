package tournament

import (
	"strings"

	"table-tennis-backend/internal/domain/player"
)

// GetTeamDisplayName returns the formatted name for a team or singles player.
func GetTeamDisplayName(team []*player.Player, tournamentType string) string {
	if len(team) == 0 {
		return "N/A"
	}
	if tournamentType == "teams" {
		return team[0].FirstName
	}
	if len(team) == 1 {
		if team[0].LastName == "" {
			return team[0].FirstName
		}
		return team[0].FirstName + " " + team[0].LastName
	}
	if len(team) == 2 {
		return team[0].FirstName + " " + team[0].LastName + " / " + team[1].FirstName + " " + team[1].LastName
	}
	return team[0].FirstName + " " + team[0].LastName
}

// GetTournamentPlaces determines the 1st, 2nd, and 3rd place winners of a finished tournament.
func GetTournamentPlaces(t *Tournament) (first, second, third string) {
	if t.Status != "finished" {
		return "", "", ""
	}

	if t.Format == "elimination" || t.Format == "groups_elimination" {
		// 1st and 2nd Place: Final Match
		var finalMatch *Match
		for i := range t.Matches {
			m := &t.Matches[i]
			if m.Stage == "final" && m.Status == "finished" && m.TeamMatchID == nil {
				finalMatch = m
				break
			}
		}
		if finalMatch != nil && finalMatch.WinnerTeam != "" {
			if finalMatch.WinnerTeam == "A" {
				first = GetTeamDisplayName(finalMatch.TeamA, t.Type)
				second = GetTeamDisplayName(finalMatch.TeamB, t.Type)
			} else {
				first = GetTeamDisplayName(finalMatch.TeamB, t.Type)
				second = GetTeamDisplayName(finalMatch.TeamA, t.Type)
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
					semiLosers = append(semiLosers, GetTeamDisplayName(m.TeamB, t.Type))
				} else if m.WinnerTeam == "B" {
					semiLosers = append(semiLosers, GetTeamDisplayName(m.TeamA, t.Type))
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
			standings := BuildStandings(participants, t.Matches)
			if len(standings) > 0 {
				first = GetTeamDisplayName([]*player.Player{standings[0].Player}, t.Type)
			}
			if len(standings) > 1 {
				second = GetTeamDisplayName([]*player.Player{standings[1].Player}, t.Type)
			}
			if len(standings) > 2 {
				third = GetTeamDisplayName([]*player.Player{standings[2].Player}, t.Type)
			}
		}
	}

	return first, second, third
}
