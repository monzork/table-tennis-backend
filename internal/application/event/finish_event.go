package event

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"time"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
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
		return errors.New("event is already finished")
	}

	// Check if all matches are finished
	unfinishedCount, err := uc.matchRepo.CountUnfinishedMatches(ctx, tournamentID)
	if err != nil {
		return err
	}
	if unfinishedCount > 0 {
		return errors.New("cannot finish event: there are still matches in progress or scheduled")
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
		return errors.New("cannot finish event: not all rounds have been played")
	}

	// Apply Elo changes match by match in order of UpdatedAt if NOT skip_elo
	if !t.SkipElo {
		// 1. Initialize player Elo map from participant snapshots (which are the true EloBefore snapshots)
		snapshots, err := uc.tournamentRepo.GetParticipantSnapshots(ctx, tournamentID)
		if err != nil {
			return err
		}

		type eloState struct {
			Singles int16
			Doubles int16
			Player  *player.Player
		}
		playerElos := make(map[string]*eloState)

		partMap := make(map[string]*player.Player)
		for _, p := range t.Participants {
			partMap[p.ID] = p
		}

		for _, snap := range snapshots {
			p, ok := partMap[snap.PlayerID]
			if !ok {
				continue
			}
			singlesElo := p.SinglesElo
			if snap.EloBeforeSingles != nil {
				singlesElo = *snap.EloBeforeSingles
			}
			doublesElo := p.DoublesElo
			if snap.EloBeforeDoubles != nil {
				doublesElo = *snap.EloBeforeDoubles
			}

			slog.Info("Participant starting Elo loaded",
				"tournamentID", tournamentID,
				"playerName", fmt.Sprintf("%s %s", p.FirstName, p.LastName),
				"singlesElo", singlesElo,
				"doublesElo", doublesElo,
			)

			playerElos[snap.PlayerID] = &eloState{
				Singles: singlesElo,
				Doubles: doublesElo,
				Player:  p,
			}
		}

		// Sort the event matches in-memory by UpdatedAt
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

		// 2. Process matches chronologically
		for _, m := range t.Matches {
			if m.WinnerTeam == "" || m.MatchType == "teams" {
				continue
			}

			var resolvedA, resolvedB []*player.Player
			for _, p := range m.TeamA {
				state, ok := playerElos[p.ID]
				if !ok {
					state = &eloState{Singles: p.SinglesElo, Doubles: p.DoublesElo, Player: p}
					playerElos[p.ID] = state
				}
				p.SinglesElo = state.Singles
				p.DoublesElo = state.Doubles
				resolvedA = append(resolvedA, p)
			}
			for _, p := range m.TeamB {
				state, ok := playerElos[p.ID]
				if !ok {
					state = &eloState{Singles: p.SinglesElo, Doubles: p.DoublesElo, Player: p}
					playerElos[p.ID] = state
				}
				p.SinglesElo = state.Singles
				p.DoublesElo = state.Doubles
				resolvedB = append(resolvedB, p)
			}

			if len(resolvedA) > 0 && len(resolvedB) > 0 {
				var beforeA, beforeB []int16
				for _, p := range resolvedA {
					if m.MatchType == "doubles" {
						beforeA = append(beforeA, p.DoublesElo)
					} else {
						beforeA = append(beforeA, p.SinglesElo)
					}
				}
				for _, p := range resolvedB {
					if m.MatchType == "doubles" {
						beforeB = append(beforeB, p.DoublesElo)
					} else {
						beforeB = append(beforeB, p.SinglesElo)
					}
				}

				match.CalculateAndApplyElo(m.MatchType, resolvedA, resolvedB, m.WinnerTeam)

				var afterA, afterB []int16
				for _, p := range resolvedA {
					if m.MatchType == "doubles" {
						afterA = append(afterA, p.DoublesElo)
					} else {
						afterA = append(afterA, p.SinglesElo)
					}
				}
				for _, p := range resolvedB {
					if m.MatchType == "doubles" {
						afterB = append(afterB, p.DoublesElo)
					} else {
						afterB = append(afterB, p.SinglesElo)
					}
				}

				descA := ""
				for i, p := range resolvedA {
					descA += fmt.Sprintf("%s %s (Elo: %d -> %d)", p.FirstName, p.LastName, beforeA[i], afterA[i])
					if i < len(resolvedA)-1 {
						descA += " & "
					}
				}
				descB := ""
				for i, p := range resolvedB {
					descB += fmt.Sprintf("%s %s (Elo: %d -> %d)", p.FirstName, p.LastName, beforeB[i], afterB[i])
					if i < len(resolvedB)-1 {
						descB += " & "
					}
				}

				slog.Info("Elo recalculation match processed",
					"matchID", m.ID,
					"matchType", m.MatchType,
					"stage", m.Stage,
					"winnerTeam", m.WinnerTeam,
					"teamA", descA,
					"teamB", descB,
				)

				for _, p := range resolvedA {
					playerElos[p.ID].Singles = p.SinglesElo
					playerElos[p.ID].Doubles = p.DoublesElo
				}
				for _, p := range resolvedB {
					playerElos[p.ID].Singles = p.SinglesElo
					playerElos[p.ID].Doubles = p.DoublesElo
				}
			}
		}

		// 3. Save final updated Elos to database
		var pids []string
		for id := range playerElos {
			pids = append(pids, id)
		}

		dbPlayers, err := uc.playerRepo.GetByIDs(ctx, pids)
		if err == nil && len(dbPlayers) > 0 {
			for _, dbP := range dbPlayers {
				if state, ok := playerElos[dbP.ID]; ok {
					dbP.UpdateSinglesElo(state.Singles)
					dbP.UpdateDoublesElo(state.Doubles)
				}
			}
			_ = uc.playerRepo.SaveMultiple(ctx, dbPlayers)
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

	snapshots, _ := uc.tournamentRepo.GetParticipantSnapshots(ctx, tournamentID)

	calculator := tournamentDomain.NewMetricsCalculator()
	t.Metrics = calculator.Calculate(t, snapshots)

	// Calculate and set the event winner name
	t.WinnerName = uc.determineWinner(t)

	// Mark event as finished
	t.Status = "finished"
	return uc.tournamentRepo.Update(ctx, t)
}

func (uc *FinishTournamentUseCase) determineWinner(t *tournamentDomain.Event) string {
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
				return tournamentDomain.GetTeamDisplayName(finalMatch.TeamA, t.Type)
			} else if finalMatch.WinnerTeam == "B" {
				return tournamentDomain.GetTeamDisplayName(finalMatch.TeamB, t.Type)
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
		return tournamentDomain.GetTeamDisplayName([]*player.Player{standings[0].Player}, t.Type)
	}

	return ""
}

// local func deleted
