package tournament

import (
	"context"
	"errors"
	"sort"

	"github.com/google/uuid"
	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type FinishTournamentUseCase struct {
	tournamentRepo *bun.TournamentRepository
	matchRepo      *bun.MatchRepository
	playerRepo     *bun.PlayerRepository
}

func NewFinishTournamentUseCase(
	tournamentRepo *bun.TournamentRepository,
	matchRepo *bun.MatchRepository,
	playerRepo *bun.PlayerRepository,
) *FinishTournamentUseCase {
	return &FinishTournamentUseCase{
		tournamentRepo: tournamentRepo,
		matchRepo:      matchRepo,
		playerRepo:     playerRepo,
	}
}

func (uc *FinishTournamentUseCase) Execute(ctx context.Context, tournamentID uuid.UUID) error {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}
	if t.Status == "finished" {
		return errors.New("tournament is already finished")
	}

	// Check if all matches are finished
	unfinishedCount, _ := uc.matchRepo.DB().NewSelect().
		Model((*bun.MatchModel)(nil)).
		Where("tournament_id = ?", tournamentID).
		Where("status != ?", "finished").
		Where("team_match_id IS NULL").
		Count(ctx)

	if unfinishedCount > 0 {
		return errors.New("cannot finish tournament: there are still matches in progress or scheduled")
	}

	// Verify enough matches have been played for the format
	finishedCount, _ := uc.matchRepo.DB().NewSelect().
		Model((*bun.MatchModel)(nil)).
		Where("tournament_id = ?", tournamentID).
		Where("status = ?", "finished").
		Where("team_match_id IS NULL").
		Count(ctx)

	participantCount := len(t.Participants)
	if t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams" {
		participantCount = len(t.Teams)
	}

	if participantCount > 1 && finishedCount < participantCount-1 {
		return errors.New("cannot finish tournament: not all rounds have been played")
	}

	// Fetch all matches for the tournament in chronological order
	var matchModels []bun.MatchModel
	err = uc.matchRepo.DB().NewSelect().
		Model(&matchModels).
		Where("tournament_id = ?", tournamentID).
		Order("updated_at ASC").
		Scan(ctx)
	if err != nil {
		return err
	}

	// Apply Elo changes match by match in order if NOT skip_elo
	if !t.SkipElo {
		for _, m := range matchModels {
			if m.WinnerTeam == nil || *m.WinnerTeam == "" {
				continue
			}

			var teamA, teamB []*player.Player
			
			p1a, err1 := uc.playerRepo.GetById(ctx, m.TeamAPlayer1ID)
			if err1 == nil && p1a != nil {
				teamA = append(teamA, p1a)
			}
			if m.TeamAPlayer2ID != nil {
				p2a, err2 := uc.playerRepo.GetById(ctx, *m.TeamAPlayer2ID)
				if err2 == nil && p2a != nil {
					teamA = append(teamA, p2a)
				}
			}

			p1b, err3 := uc.playerRepo.GetById(ctx, m.TeamBPlayer1ID)
			if err3 == nil && p1b != nil {
				teamB = append(teamB, p1b)
			}
			if m.TeamBPlayer2ID != nil {
				p2b, err4 := uc.playerRepo.GetById(ctx, *m.TeamBPlayer2ID)
				if err4 == nil && p2b != nil {
					teamB = append(teamB, p2b)
				}
			}

			if len(teamA) > 0 && len(teamB) > 0 {
				match.CalculateAndApplyElo(m.MatchType, teamA, teamB, *m.WinnerTeam)
				for _, p := range teamA {
					_ = uc.playerRepo.Save(ctx, p)
				}
				for _, p := range teamB {
					_ = uc.playerRepo.Save(ctx, p)
				}
			}
		}
	}

	// Finalize EloAfter snapshots
	for _, p := range t.Participants {
		if updatedPlayer, err := uc.playerRepo.GetById(ctx, p.ID); err == nil {
			_, _ = uc.matchRepo.DB().NewUpdate().
				TableExpr("tournament_participants").
				Set("elo_after_singles = ?, elo_after_doubles = ?", updatedPlayer.SinglesElo, updatedPlayer.DoublesElo).
				Where("tournament_id = ? AND player_id = ?", tournamentID, p.ID).
				Exec(ctx)
		}
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
		// Calculate round robin standings
		type standing struct {
			id         uuid.UUID
			name       string
			wins       int
			played     int
			setsWon    int
			setsLost   int
			pointsWon  int
			pointsLost int
		}

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

		standings := make([]standing, len(participants))
		standingsMap := make(map[uuid.UUID]int)
		for i, p := range participants {
			standings[i] = standing{
				id:   p.ID,
				name: getPlayerName(p, t.Type),
			}
			standingsMap[p.ID] = i
		}

		for _, m := range t.Matches {
			if m.Status != "finished" || m.TeamMatchID != nil {
				continue
			}
			if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
				continue
			}
			idA := m.TeamA[0].ID
			idB := m.TeamB[0].ID

			idxA, okA := standingsMap[idA]
			idxB, okB := standingsMap[idB]
			if !okA || !okB {
				continue
			}

			standings[idxA].played++
			standings[idxB].played++

			scoreA := m.ScoreA()
			scoreB := m.ScoreB()

			standings[idxA].setsWon += scoreA
			standings[idxA].setsLost += scoreB
			standings[idxB].setsWon += scoreB
			standings[idxB].setsLost += scoreA

			for _, s := range m.Sets {
				standings[idxA].pointsWon += s.ScoreA
				standings[idxA].pointsLost += s.ScoreB
				standings[idxB].pointsWon += s.ScoreB
				standings[idxB].pointsLost += s.ScoreA
			}

			if m.WinnerTeam == "A" {
				standings[idxA].wins++
			} else if m.WinnerTeam == "B" {
				standings[idxB].wins++
			}
		}

		sort.Slice(standings, func(i, j int) bool {
			si, sj := standings[i], standings[j]
			if si.wins != sj.wins {
				return si.wins > sj.wins
			}

			// Tiebreaker: Head-to-head
			for _, m := range t.Matches {
				if m.Status != "finished" || m.TeamMatchID != nil {
					continue
				}
				if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
					continue
				}
				idA := m.TeamA[0].ID
				idB := m.TeamB[0].ID
				if (idA == si.id && idB == sj.id) || (idA == sj.id && idB == si.id) {
					if m.WinnerTeam == "A" {
						if idA == si.id {
							return true
						}
						return false
					} else if m.WinnerTeam == "B" {
						if idB == si.id {
							return true
						}
						return false
					}
				}
			}

			// Set ratio
			ratioI := float64(si.setsWon)
			if si.setsLost > 0 {
				ratioI = float64(si.setsWon) / float64(si.setsLost)
			}
			ratioJ := float64(sj.setsWon)
			if sj.setsLost > 0 {
				ratioJ = float64(sj.setsWon) / float64(sj.setsLost)
			}
			if ratioI != ratioJ {
				return ratioI > ratioJ
			}

			// Point ratio
			ptRatioI := float64(si.pointsWon)
			if si.pointsLost > 0 {
				ptRatioI = float64(si.pointsWon) / float64(si.pointsLost)
			}
			ptRatioJ := float64(sj.pointsWon)
			if sj.pointsLost > 0 {
				ptRatioJ = float64(sj.pointsWon) / float64(sj.pointsLost)
			}
			return ptRatioI > ptRatioJ
		})

		return standings[0].name
	}

	return ""
}

func getPlayerName(p *player.Player, tournamentType string) string {
	if tournamentType == "teams" {
		return p.FirstName
	}
	if p.LastName == "" {
		return p.FirstName
	}
	return p.FirstName + " " + p.LastName
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
