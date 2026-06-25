package tournament

import (
	"context"
	"errors"
	"sort"
	"time"

	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type FinishTournamentUseCase struct {
	tournamentRepo tournamentDomain.Repository
	matchRepo      tournamentDomain.MatchRepository
	playerRepo     player.Repository
}

func NewFinishTournamentUseCase(
	tournamentRepo tournamentDomain.Repository,
	matchRepo tournamentDomain.MatchRepository,
	playerRepo player.Repository,
) *FinishTournamentUseCase {
	return &FinishTournamentUseCase{
		tournamentRepo: tournamentRepo,
		matchRepo:      matchRepo,
		playerRepo:     playerRepo,
	}
}

func (uc *FinishTournamentUseCase) Execute(ctx context.Context, tournamentID string) error {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}
	if t.Status == "finished" {
		return errors.New("tournament is already finished")
	}

	// Check if all matches are finished
	unfinishedCount, err := uc.matchRepo.CountUnfinishedMatches(ctx, tournamentID)
	if err != nil {
		return err
	}
	if unfinishedCount > 0 {
		return errors.New("cannot finish tournament: there are still matches in progress or scheduled")
	}

	// Verify enough matches have been played for the format
	finishedCount, err := uc.matchRepo.CountFinishedMatches(ctx, tournamentID)
	if err != nil {
		return err
	}

	participantCount := len(t.Participants)
	if t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams" {
		participantCount = len(t.Teams)
	}

	if participantCount > 1 && finishedCount < participantCount-1 {
		return errors.New("cannot finish tournament: not all rounds have been played")
	}

	// Apply Elo changes match by match in order of UpdatedAt if NOT skip_elo
	if !t.SkipElo {
		// Sort the tournament matches in-memory by UpdatedAt
		sort.SliceStable(t.Matches, func(i, j int) bool {
			tI := time.Time{}
			if t.Matches[i].UpdatedAt != nil {
				tI = *t.Matches[i].UpdatedAt
			}
			tJ := time.Time{}
			if t.Matches[j].UpdatedAt != nil {
				tJ = *t.Matches[j].UpdatedAt
			}
			return tI.Before(tJ)
		})

		var allUpdatedPlayers []*player.Player
		for _, m := range t.Matches {
			if m.WinnerTeam == "" {
				continue
			}

			if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
				match.CalculateAndApplyElo(m.MatchType, m.TeamA, m.TeamB, m.WinnerTeam)
				allUpdatedPlayers = append(allUpdatedPlayers, m.TeamA...)
				allUpdatedPlayers = append(allUpdatedPlayers, m.TeamB...)
			}
		}

		if len(allUpdatedPlayers) > 0 {
			// deduplicate players by ID if needed, though SaveMultiple with ON CONFLICT handles it.
			// however, we've updated their Elo in memory, so the latest instance in the slice 
			// has the most recent Elo. Saving them all works if ordered correctly, but better to dedup.
			latestPlayers := make(map[string]*player.Player)
			for _, p := range allUpdatedPlayers {
				latestPlayers[p.ID] = p
			}
			var deduplicated []*player.Player
			for _, p := range latestPlayers {
				deduplicated = append(deduplicated, p)
			}
			_ = uc.playerRepo.SaveMultiple(ctx, deduplicated)
		}
	}

	// Finalize EloAfter snapshots using domain repository method
	var pids []string
	for _, p := range t.Participants {
		pids = append(pids, p.ID)
	}
	if updatedPlayers, err := uc.playerRepo.GetByIDs(ctx, pids); err == nil {
		_ = uc.tournamentRepo.UpdateParticipantsElo(ctx, tournamentID, updatedPlayers)
	}

	// Calculate and set the tournament winner name
	t.WinnerName = uc.determineWinner(t)

	// Mark tournament as finished
	t.Status = "finished"
	return uc.tournamentRepo.Update(ctx, t)
}

func (uc *FinishTournamentUseCase) determineWinner(t *tournamentDomain.Tournament) string {
	if t.Format == "elimination" || t.Format == "groups_elimination" {
		// Find the finished final match
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
				return getTeamDisplayName(finalMatch.TeamA, t.Type)
			} else if finalMatch.WinnerTeam == "B" {
				return getTeamDisplayName(finalMatch.TeamB, t.Type)
			}
		}
	} else if t.Format == "round_robin" {
		// Calculate round robin standings with ITTF-compliant domain logic
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

		if len(participants) == 0 {
			return ""
		}

		standings := tournamentDomain.BuildStandings(participants, t.Matches)
		if len(standings) == 0 {
			return ""
		}
		return getTeamDisplayName([]*player.Player{standings[0].Player}, t.Type)
	}

	return ""
}

func getTeamDisplayName(team []*player.Player, tournamentType string) string {
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
