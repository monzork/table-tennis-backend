package tournament

import (
	"context"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
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

func (uc *GetTournamentByIDUseCase) Execute(ctx context.Context, idStr string) (*tournamentDomain.Tournament, error) {
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
	// For doubles/teams tournaments, also regenerate if the group participant count
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
							Name:   d.Name,
							MinElo: d.MinElo,
							MaxElo: d.MaxElo,
						})
					}
				}
			}
		}
		if err := t.AssignGroupsByDivisions(divsList); err == nil {
			_ = uc.repo.Update(ctx, t)
		}
	}

	return t, nil
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

func (uc *UpdateTournamentUseCase) Execute(
	ctx context.Context, idStr, name, tournamentType, format, category, startStr, endStr string,
	registrationOpen bool,
	participantIDs []string, newPlayers []NewPlayerData,
	stageRuleOverrides []StageRuleOverride, groupPassCount int,
	skipElo bool, eventID *string,
	teamFormat string,
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
	for _, pidStr := range participantIDs {
		if pidStr == "" {
			continue
		}
		p, err := uc.playerRepo.GetById(ctx, pidStr)
		if err == nil {
			participants = append(participants, p)
		}
	}

	// Handle newly created players ad-hoc
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

	t, err := tournamentDomain.NewTournament(idStr, name, tournamentType, format, category, start, end, []tournamentDomain.Rule{}, groupPassCount, participants)
	if err != nil {
		return nil, err
	}
	t.RegistrationOpen = registrationOpen
	t.SkipElo = skipElo
	t.EventID = eventID
	t.TeamFormat = teamFormat

	// Preserve existing teams and conditionally preserve/regenerate groups
	if existing, err := uc.repo.GetByID(ctx, idStr); err == nil {
		t.Teams = existing.Teams

		// Check if participants, format, type, or category changed
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

		formatChanged := existing.Format != format
		typeChanged := existing.Type != tournamentType
		categoryChanged := existing.EventCategory != category

		if !participantsChanged && !formatChanged && !typeChanged && !categoryChanged && len(existing.Groups) > 0 {
			t.Groups = existing.Groups
		} else {
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
		}
	} else {
		// Fallback for new / not found
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
	}

	// Apply any stage rule overrides submitted by the admin
	for i := range t.StageRules {
		for _, ov := range stageRuleOverrides {
			if t.StageRules[i].Stage == ov.Stage {
				t.StageRules[i].TournamentID = idStr
				t.StageRules[i].BestOf = ov.BestOf
				t.StageRules[i].PointsToWin = ov.PointsToWin
				t.StageRules[i].PointsMargin = ov.PointsMargin
			}
		}
	}

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

func (uc *DeleteTournamentUseCase) Execute(ctx context.Context, idStr string) error {
	return uc.repo.Delete(ctx, idStr)
}
