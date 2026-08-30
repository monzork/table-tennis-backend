package event

import (
	"context"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	"time"
)

type CreateTournamentUseCase struct {
	repo       tournamentDomain.Repository
	playerRepo playerDomain.Repository
}

func NewCreateTournamentUseCase(repo tournamentDomain.Repository, playerRepo playerDomain.Repository) *CreateTournamentUseCase {
	return &CreateTournamentUseCase{repo: repo, playerRepo: playerRepo}
}

type NewPlayerData struct {
	FirstName string
	LastName  string
	Gender    string
}

type CreateEventCommand struct {
	Name                 string
	Type                 string
	Format               string
	Category             string
	StartDate            string
	EndDate              string
	ParticipantIDs       []string
	NewPlayers           []NewPlayerData
	GroupCount           int
	GroupPassCount       int
	LosersGroupPassCount int
	StageRuleOverrides   []StageRuleOverride
	SkipElo              bool
	TournamentID         *string
	TeamFormat           string
	NumTables            int
	HasThirdPlaceMatch   bool

	KnockoutBracketsCount int
}

func (uc *CreateTournamentUseCase) Execute(ctx context.Context, cmd CreateEventCommand) (*tournamentDomain.Event, error) {
	start, err := time.Parse("2006-01-02", cmd.StartDate)
	if err != nil {
		return nil, err
	}
	end, err := time.Parse("2006-01-02", cmd.EndDate)
	if err != nil {
		return nil, err
	}

	var participants []*playerDomain.Player

	// Handle existing players
	var validIDs []string
	for _, idStr := range cmd.ParticipantIDs {
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
	for _, np := range cmd.NewPlayers {
		p, err := playerDomain.NewPlayer(idgen.Generate(), np.FirstName, np.LastName, time.Now(), np.Gender, "", "", "")
		if err != nil {
			return nil, err
		}
		if err := uc.playerRepo.Save(ctx, p); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	// Filter participants by cmd.Category
	var filteredParticipants []*playerDomain.Player
	for _, p := range participants {
		switch cmd.Category {
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

	t, err := tournamentDomain.NewEvent(idgen.Generate(), cmd.Name, cmd.Type, cmd.Format, cmd.Category, start, end, []tournamentDomain.Rule{}, cmd.GroupPassCount, filteredParticipants, cmd.HasThirdPlaceMatch)
	if err != nil {
		return nil, err
	}
	t.SkipElo = cmd.SkipElo
	t.SkipDivisionSplit = true
	t.TournamentID = cmd.TournamentID

	t.LosersGroupPassCount = cmd.LosersGroupPassCount

	t.TeamFormat = cmd.TeamFormat
	t.NumTables = cmd.NumTables
	t.KnockoutBracketsCount = cmd.KnockoutBracketsCount
	t.GroupCount = cmd.GroupCount

	if t.Format == "groups_elimination" || t.Format == "round_robin" || t.Format == "elimination" || t.Format == "single_division_multiple_brackets" {
		if err := (&tournamentDomain.OpenBracketSnakeSeeder{}).AssignGroups(t); err != nil {
			return nil, err
		}
	}

	// Apply any stage rule overrides submitted by the admin
	for i := range t.StageRules {
		for _, ov := range cmd.StageRuleOverrides {
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

	// Auto-create a paired event for the opposite gender
	if cmd.Category == "men" || cmd.Category == "women" {
		pairCategory, pairGender, pairSuffix := "women", "F", "Women's"
		if cmd.Category == "women" {
			pairCategory, pairGender, pairSuffix = "men", "M", "Men's"
		}

		var pairParticipants []*playerDomain.Player
		for _, p := range participants {
			if p.Gender == pairGender {
				pairParticipants = append(pairParticipants, p)
			}
		}

		pairName := pairSuffix + " " + cmd.Name
		pairT, err := tournamentDomain.NewEvent(idgen.Generate(), pairName, cmd.Type, cmd.Format, pairCategory, start, end, []tournamentDomain.Rule{}, cmd.GroupPassCount, pairParticipants, cmd.HasThirdPlaceMatch)
		if err == nil {
			pairT.SkipElo = cmd.SkipElo
			pairT.TournamentID = cmd.TournamentID

			pairT.LosersGroupPassCount = cmd.LosersGroupPassCount
			pairT.KnockoutBracketsCount = cmd.KnockoutBracketsCount
			pairT.GroupCount = cmd.GroupCount
			if pairT.Format == "groups_elimination" || pairT.Format == "round_robin" || pairT.Format == "elimination" {
				(&tournamentDomain.OpenBracketSnakeSeeder{}).AssignGroups(pairT)
			}
			uc.repo.Save(ctx, pairT)
		}
	}

	return t, nil
}
