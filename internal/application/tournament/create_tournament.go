package tournament

import (
	"context"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"time"
)

type CreateTournamentUseCase struct {
	repo         tournamentDomain.Repository
	playerRepo   playerDomain.Repository
	divisionRepo divisionDomain.Repository
}

func NewCreateTournamentUseCase(repo tournamentDomain.Repository, playerRepo playerDomain.Repository, divisionRepo divisionDomain.Repository) *CreateTournamentUseCase {
	return &CreateTournamentUseCase{repo: repo, playerRepo: playerRepo, divisionRepo: divisionRepo}
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
	divisionRules []tournamentDomain.DivisionRule,
	skipElo bool,
	eventID *string,
	teamFormat string,
	numTables int,
	hasThirdPlaceMatch bool,
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
	var validIDs []string
	for _, idStr := range participantIDs {
		if idStr != "" {
			validIDs = append(validIDs, idStr)
		}
	}
	if len(validIDs) > 0 {
		if ps, err := uc.playerRepo.GetByIDs(ctx, validIDs); err == nil {
			participants = append(participants, ps...)
		}
	}

	// Handle new players
	for _, np := range newPlayers {
		p, err := playerDomain.NewPlayer(idgen.Generate(), np.FirstName, np.LastName, time.Now(), np.Gender, "", "", "")
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

	t, err := tournamentDomain.NewTournament(idgen.Generate(), name, tournamentType, format, category, start, end, []tournamentDomain.Rule{}, groupPassCount, filteredParticipants, hasThirdPlaceMatch)
	if err != nil {
		return nil, err
	}
	t.SkipElo = skipElo
	t.EventID = eventID
	t.TeamFormat = teamFormat
	t.NumTables = numTables

	// Fetch divisions list to seed groups per-division
	var divsList []tournamentDomain.DivisionSeeding
	if !skipElo {
		divs, err := uc.divisionRepo.GetAll(ctx)
		if err == nil {
			for _, d := range divs {
				if d.Category == "both" || d.Category == tournamentType {
					divsList = append(divsList, tournamentDomain.DivisionSeeding{
						Name:   d.Name,
						MinElo: d.MinElo,
						MaxElo: d.MaxElo,
					})
				}
			}
		}
	}

	if t.Format == "groups_elimination" || t.Format == "round_robin" || t.Format == "elimination" {
		if err := t.AssignGroupsByDivisions(divsList); err != nil {
			return nil, err
		}
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

	// Apply division-specific rules
	t.DivisionRules = divisionRules

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
		pairT, err := tournamentDomain.NewTournament(idgen.Generate(), pairName, tournamentType, format, pairCategory, start, end, []tournamentDomain.Rule{}, groupPassCount, pairParticipants, hasThirdPlaceMatch)
		if err == nil {
			pairT.SkipElo = skipElo
			pairT.EventID = eventID
			if pairT.Format == "groups_elimination" || pairT.Format == "round_robin" || pairT.Format == "elimination" {
				pairT.AssignGroupsByDivisions(divsList)
			}
			uc.repo.Save(ctx, pairT)
		}
	}

	return t, nil
}
