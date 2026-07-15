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

type RecalculateTournamentEloUseCase struct {
	tournamentRepo tournamentDomain.Repository
	playerRepo     player.Repository
}

func NewRecalculateTournamentEloUseCase(tournamentRepo tournamentDomain.Repository, playerRepo player.Repository) *RecalculateTournamentEloUseCase {
	return &RecalculateTournamentEloUseCase{
		tournamentRepo: tournamentRepo,
		playerRepo:     playerRepo,
	}
}

func (uc *RecalculateTournamentEloUseCase) Execute(ctx context.Context, tournamentID string) error {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	if t.SkipElo {
		return errors.New("cannot recalculate Elo: event has skip Elo enabled")
	}

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

	// 4. Update event_participants final Elo snapshots
	pids = nil
	for _, p := range t.Participants {
		pids = append(pids, p.ID)
	}
	if updatedPlayers, err := uc.playerRepo.GetByIDs(ctx, pids); err == nil {
		_ = uc.tournamentRepo.UpdateParticipantsElo(ctx, tournamentID, updatedPlayers)
	}

	// 5. Recalculate metrics
	snapshots, _ = uc.tournamentRepo.GetParticipantSnapshots(ctx, tournamentID)
	calculator := tournamentDomain.NewMetricsCalculator()
	t.Metrics = calculator.Calculate(t, snapshots)

	return uc.tournamentRepo.Update(ctx, t)
}
