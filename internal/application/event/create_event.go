package event

import (
	"context"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
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

type CreateEventCommand struct {
	Name                    string
	Type                    string
	Format                  string
	Category                string
	StartDate               string
	EndDate                 string
	ParticipantIDs          []string
	NewPlayers              []NewPlayerData
	GroupPassCount          int
	StageRuleOverrides      []StageRuleOverride
	DivisionRules           []tournamentDomain.DivisionRule
	SkipElo                 bool
	EventID                 *string
	TeamFormat              string
	NumTables               int
	HasThirdPlaceMatch      bool
	DivisionFormats         map[string]string
	DivisionGroupPassCounts map[string]int
	DivisionGroupCounts     map[string]int
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

	t, err := tournamentDomain.NewTournament(idgen.Generate(), cmd.Name, cmd.Type, cmd.Format, cmd.Category, start, end, []tournamentDomain.Rule{}, cmd.GroupPassCount, filteredParticipants, cmd.HasThirdPlaceMatch)
	if err != nil {
		return nil, err
	}
	t.SkipElo = cmd.SkipElo
	t.DivisionFormats = cmd.DivisionFormats
	t.DivisionGroupPassCounts = cmd.DivisionGroupPassCounts
	t.DivisionGroupCounts = cmd.DivisionGroupCounts

	// Save the event to DB
	t.TeamFormat = cmd.TeamFormat
	t.NumTables = cmd.NumTables

	// Fetch divisions list to seed groups per-division
	var divsList []tournamentDomain.DivisionSeeding
	if !cmd.SkipElo {
		divs, err := uc.divisionRepo.GetAll(ctx)
		if err == nil {
			for _, d := range divs {
				if d.Category == "both" || d.Category == cmd.Type {
					divsList = append(divsList, tournamentDomain.DivisionSeeding{
						ID:     d.ID,
						Name:   d.Name,
						MinElo: d.MinElo,
						MaxElo: d.MaxElo,
					})
				}
			}
		}
	}

	if t.Format == "groups_elimination" || t.Format == "round_robin" || t.Format == "elimination" {
		if err := (&tournamentDomain.DivisionSeeder{Divisions: divsList}).AssignGroups(t); err != nil {
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

	// Apply division-specific rules
	t.DivisionRules = cmd.DivisionRules

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
		pairT, err := tournamentDomain.NewTournament(idgen.Generate(), pairName, cmd.Type, cmd.Format, pairCategory, start, end, []tournamentDomain.Rule{}, cmd.GroupPassCount, pairParticipants, cmd.HasThirdPlaceMatch)
		if err == nil {
			pairT.SkipElo = cmd.SkipElo
			pairT.EventID = cmd.EventID
			pairT.DivisionFormats = cmd.DivisionFormats
			pairT.DivisionGroupPassCounts = cmd.DivisionGroupPassCounts
			pairT.DivisionGroupCounts = cmd.DivisionGroupCounts
			if pairT.Format == "groups_elimination" || pairT.Format == "round_robin" || pairT.Format == "elimination" {
				(&tournamentDomain.DivisionSeeder{Divisions: divsList}).AssignGroups(pairT)
			}
			uc.repo.Save(ctx, pairT)
		}
	}

	return t, nil
}
