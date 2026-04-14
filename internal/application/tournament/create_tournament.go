package tournament

import (
	"context"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"time"

	"github.com/google/uuid"
)

type CreateTournamentUseCase struct {
	repo       *bun.TournamentRepository
	playerRepo *bun.PlayerRepository
}

func NewCreateTournamentUseCase(repo *bun.TournamentRepository, playerRepo *bun.PlayerRepository) *CreateTournamentUseCase {
	return &CreateTournamentUseCase{repo: repo, playerRepo: playerRepo}
}

type NewPlayerData struct {
	FirstName string
	LastName  string
	Gender    string
}

func (uc *CreateTournamentUseCase) Execute(
	ctx context.Context,
	name string,
	tournamentType string,
	format string,
	category string,
	startStr, endStr string,
	participantIDs []string,
	newPlayers []NewPlayerData,
	groupPassCount int,
	stageRuleOverrides []StageRuleOverride,
) (*tournamentDomain.Tournament, error) {
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		return nil, err
	}

	var participants []*playerDomain.Player

	// Handle existing players
	for _, idStr := range participantIDs {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		p, err := uc.playerRepo.GetById(ctx, id)
		if err == nil {
			participants = append(participants, p)
		}
	}

	// Handle new players
	for _, np := range newPlayers {
		p, err := playerDomain.NewPlayer(np.FirstName, np.LastName, time.Now(), np.Gender, "")
		if err != nil {
			return nil, err
		}
		if err := uc.playerRepo.Save(ctx, p); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	// Filter participants by category
	var filteredParticipants []*playerDomain.Player
	for _, p := range participants {
		switch category {
		case "men":
			if p.Gender == "M" {
				filteredParticipants = append(filteredParticipants, p)
			}
		case "women":
			if p.Gender == "F" {
				filteredParticipants = append(filteredParticipants, p)
			}
		default:
			filteredParticipants = append(filteredParticipants, p)
		}
	}

	t, err := tournamentDomain.NewTournament(name, tournamentType, format, category, start, end, []tournamentDomain.Rule{}, groupPassCount, filteredParticipants)
	if err != nil {
		return nil, err
	}

	// Apply any stage rule overrides submitted by the admin
	for i := range t.StageRules {
		for _, ov := range stageRuleOverrides {
			if t.StageRules[i].Stage == ov.Stage {
				t.StageRules[i].BestOf = ov.BestOf
				t.StageRules[i].PointsToWin = ov.PointsToWin
				t.StageRules[i].PointsMargin = ov.PointsMargin
			}
		}
	}

	if err := uc.repo.Save(ctx, t); err != nil {
		return nil, err
	}

	// Auto-create a paired tournament for the opposite gender
	if category == "men" || category == "women" {
		pairCategory, pairGender, pairSuffix := "women", "F", "Women's"
		if category == "women" {
			pairCategory, pairGender, pairSuffix = "men", "M", "Men's"
		}

		var pairParticipants []*playerDomain.Player
		for _, p := range participants {
			if p.Gender == pairGender {
				pairParticipants = append(pairParticipants, p)
			}
		}

		pairName := pairSuffix + " " + name
		pairT, err := tournamentDomain.NewTournament(pairName, tournamentType, format, pairCategory, start, end, []tournamentDomain.Rule{}, groupPassCount, pairParticipants)
		if err == nil {
			uc.repo.Save(ctx, pairT)
		}
	}

	return t, nil
}
