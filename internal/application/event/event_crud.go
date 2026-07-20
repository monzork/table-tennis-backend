package event

import (
	"context"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/pin"
	"time"
)

// ─── Get By ID ───────────────────────────────────────────────────────────────

type GetTournamentByIDUseCase struct {
	repo         tournamentDomain.Repository
	divisionRepo divisionDomain.Repository
}

func NewGetTournamentByIDUseCase(repo tournamentDomain.Repository, divisionRepo divisionDomain.Repository) *GetTournamentByIDUseCase {
	return &GetTournamentByIDUseCase{repo: repo, divisionRepo: divisionRepo}
}

func (uc *GetTournamentByIDUseCase) Execute(ctx context.Context, idStr string) (*tournamentDomain.Event, error) {
	t, err := uc.repo.GetByID(ctx, idStr)
	if err != nil {
		return nil, err
	}

	// Self-healing: regenerate seeding groups when they are missing or stale.
	needsGroupRegen := false
	needsGroups := t.Format == "elimination" || t.Format == "groups_elimination" || t.Format == "round_robin"
	if needsGroups && len(t.Groups) == 0 {
		needsGroupRegen = true
	}
	// For doubles/teams events, also regenerate if the group participant count
	// doesn't match the number of teams (teams were added/removed after initial seeding).
	if !needsGroupRegen && needsGroups && (t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams") && len(t.Teams) > 0 {
		totalGroupParticipants := 0
		for _, g := range t.Groups {
			totalGroupParticipants += len(g.Players)
		}
		if totalGroupParticipants != len(t.Teams) {
			needsGroupRegen = true
		}
	}
	if needsGroupRegen {
		var divsList []tournamentDomain.DivisionSeeding
		if !t.SkipElo && uc.divisionRepo != nil {
			divs, err := uc.divisionRepo.GetAll(ctx)
			if err == nil {
				for _, d := range divs {
					if d.Category == "both" || d.Category == t.Type {
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
		if err := (&tournamentDomain.DivisionSeeder{Divisions: divsList}).AssignGroups(t); err == nil {
			_ = uc.repo.Update(ctx, t)
		}
	}

	return t, nil
}

func (uc *GetTournamentByIDUseCase) GetSnapshots(ctx context.Context, idStr string) ([]tournamentDomain.ParticipantSnapshot, error) {
	return uc.repo.GetParticipantSnapshots(ctx, idStr)
}

func (uc *GetTournamentByIDUseCase) GetOfficials(ctx context.Context, id string) ([]tournamentDomain.ParticipantSnapshot, error) {
	return uc.repo.GetOfficials(ctx, id)
}

func (uc *GetTournamentByIDUseCase) AddOfficial(ctx context.Context, tournamentID string, playerID string) error {
	officials, err := uc.repo.GetOfficials(ctx, tournamentID)
	if err != nil {
		return err
	}

	usedPINs := make(map[string]bool)
	for _, o := range officials {
		usedPINs[o.Pin] = true
	}

	pinStr := pin.GenerateUniqueInBatch(usedPINs)
	return uc.repo.AddOfficial(ctx, tournamentID, playerID, pinStr)
}

func (uc *GetTournamentByIDUseCase) RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error {
	return uc.repo.RemoveOfficial(ctx, tournamentID, playerID)
}

// ─── Update ──────────────────────────────────────────────────────────────────

type UpdateTournamentUseCase struct {
	repo         tournamentDomain.Repository
	playerRepo   playerDomain.Repository
	divisionRepo divisionDomain.Repository
}

func NewUpdateTournamentUseCase(repo tournamentDomain.Repository, playerRepo playerDomain.Repository, divisionRepo divisionDomain.Repository) *UpdateTournamentUseCase {
	return &UpdateTournamentUseCase{repo: repo, playerRepo: playerRepo, divisionRepo: divisionRepo}
}

// StageRuleOverride carries user-submitted rule changes for a single stage.
type StageRuleOverride struct {
	Stage        string
	BestOf       int
	PointsToWin  int
	PointsMargin int
}

type UpdateEventCommand struct {
	ID                   string
	Name                 string
	Type                 string
	Format               string
	Category             string
	StartDate            string
	EndDate              string
	RegistrationOpen     bool
	ParticipantIDs       []string
	NewPlayers           []NewPlayerData
	GroupPassCount       int
	LosersGroupPassCount int
	StageRuleOverrides   []StageRuleOverride
	DivisionRules        []tournamentDomain.DivisionRule
	SkipElo              bool
	EventID              *string
	TeamFormat           string
	NumTables            int
	HasThirdPlaceMatch   bool

	DivisionConfigs       map[string]tournamentDomain.DivisionConfig
	KnockoutBracketsCount int
}

func (uc *UpdateTournamentUseCase) Execute(ctx context.Context, cmd UpdateEventCommand) (*tournamentDomain.Event, error) {
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
	for _, pidStr := range cmd.ParticipantIDs {
		if pidStr != "" {
			validIDs = append(validIDs, pidStr)
		}
	}
	if len(validIDs) > 0 {
		if ps, err := uc.playerRepo.GetByIDs(ctx, validIDs); err == nil {
			participants = append(participants, ps...)
		}
	}

	// Handle newly created players ad-hoc
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

	t, err := tournamentDomain.NewTournament(cmd.ID, cmd.Name, cmd.Type, cmd.Format, cmd.Category, start, end, []tournamentDomain.Rule{}, cmd.GroupPassCount, participants, cmd.HasThirdPlaceMatch)
	if err != nil {
		return nil, err
	}
	t.RegistrationOpen = cmd.RegistrationOpen
	t.SkipElo = cmd.SkipElo

	t.LosersGroupPassCount = cmd.LosersGroupPassCount
	t.DivisionConfigs = cmd.DivisionConfigs
	t.EventID = cmd.EventID
	t.TeamFormat = cmd.TeamFormat
	t.NumTables = cmd.NumTables
	t.HasThirdPlaceMatch = cmd.HasThirdPlaceMatch
	t.KnockoutBracketsCount = cmd.KnockoutBracketsCount

	// Preserve existing teams and conditionally preserve/regenerate groups
	if existing, err := uc.repo.GetByID(ctx, cmd.ID); err == nil {
		t.Teams = existing.Teams

		// Check if participants, cmd.Format, type, or cmd.Category changed
		participantsChanged := false
		if len(existing.Participants) != len(participants) {
			participantsChanged = true
		} else {
			existingMap := make(map[string]bool)
			for _, p := range existing.Participants {
				existingMap[p.ID] = true
			}
			for _, p := range participants {
				if !existingMap[p.ID] {
					participantsChanged = true
					break
				}
			}
		}

		formatChanged := existing.Format != cmd.Format
		typeChanged := existing.Type != cmd.Type
		categoryChanged := existing.EventCategory != cmd.Category

		divConfigsChanged := cmd.DivisionConfigs != nil

		preserveGroups := !participantsChanged && !formatChanged && !typeChanged && !categoryChanged && !divConfigsChanged && len(existing.Groups) > 0
		if existing.ManualSeedingLocked {
			preserveGroups = true
		}

		if preserveGroups {
			t.Groups = existing.Groups
		} else {
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

			if t.Format == "groups_elimination" || t.Format == "round_robin" || t.Format == "elimination" || t.Format == "single_division_multiple_brackets" {
				if err := (&tournamentDomain.DivisionSeeder{Divisions: divsList}).AssignGroups(t); err != nil {
					return nil, err
				}
			}
		}
	} else {
		// Fallback for new / not found
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

		if t.Format == "groups_elimination" || t.Format == "round_robin" || t.Format == "elimination" || t.Format == "single_division_multiple_brackets" {
			if err := (&tournamentDomain.DivisionSeeder{Divisions: divsList}).AssignGroups(t); err != nil {
				return nil, err
			}
		}
	}

	// Apply any stage rule overrides submitted by the admin
	for i := range t.StageRules {
		for _, ov := range cmd.StageRuleOverrides {
			if t.StageRules[i].Stage == ov.Stage {
				t.StageRules[i].TournamentID = cmd.ID
				t.StageRules[i].BestOf = ov.BestOf
				t.StageRules[i].PointsToWin = ov.PointsToWin
				t.StageRules[i].PointsMargin = ov.PointsMargin
			}
		}
	}

	// Apply division-specific rules
	t.DivisionRules = cmd.DivisionRules

	if err := uc.repo.Update(ctx, t); err != nil {
		return nil, err
	}
	return t, nil
}

// ─── Delete ──────────────────────────────────────────────────────────────────

type DeleteTournamentUseCase struct {
	repo tournamentDomain.Repository
}

func NewDeleteTournamentUseCase(repo tournamentDomain.Repository) *DeleteTournamentUseCase {
	return &DeleteTournamentUseCase{repo: repo}
}

func (uc *DeleteTournamentUseCase) Execute(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

// ── Remove Participant ──────────────────────────────────────────────────────

type RemoveParticipantUseCase struct {
	repo tournamentDomain.Repository
}

func NewRemoveParticipantUseCase(repo tournamentDomain.Repository) *RemoveParticipantUseCase {
	return &RemoveParticipantUseCase{repo: repo}
}

func (uc *RemoveParticipantUseCase) Execute(ctx context.Context, tournamentID, playerID string) error {
	return uc.repo.RemoveParticipant(ctx, tournamentID, playerID)
}
