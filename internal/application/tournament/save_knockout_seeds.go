package tournament

import (
	"context"
	"encoding/json"
	
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"strings"
)

type SaveKnockoutSeedsUseCase struct {
	repo    tournamentDomain.Repository
	divRepo divisionDomain.Repository
}

func NewSaveKnockoutSeedsUseCase(repo tournamentDomain.Repository, divRepo divisionDomain.Repository) *SaveKnockoutSeedsUseCase {
	return &SaveKnockoutSeedsUseCase{repo: repo, divRepo: divRepo}
}

func (uc *SaveKnockoutSeedsUseCase) Execute(ctx context.Context, tournamentID, divID, playerIDsJSON string) error {
	var pids []string
	if err := json.Unmarshal([]byte(playerIDsJSON), &pids); err != nil {
		return err
	}

	t, err := uc.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	divs, err := uc.divRepo.GetAll(ctx)
	if err != nil {
		return err
	}
	divName := ""
	if divID == "" {
		if t.SkipElo {
			divName = "Open Bracket"
		} else {
			divName = "Unclassified"
		}
	} else {
		for _, d := range divs {
			if d.ID == divID {
				divName = d.Name
				break
			}
		}
		if strings.HasSuffix(strings.ToLower(divName), " division") {
			divName = divName[:len(divName)-9]
		}
	}

	groupName := divName + " - Knockout Seeds"
	var knockoutGroup *tournamentDomain.Group
	for i := range t.Groups {
		if t.Groups[i].Name == groupName {
			knockoutGroup = &t.Groups[i]
			break
		}
	}

	var players []*player.Player
	for _, pid := range pids {
		found := false
		for _, p := range t.Participants {
			if p.ID == pid {
				players = append(players, p)
				found = true
				break
			}
		}
		if !found && len(t.Teams) > 0 {
			for _, team := range t.Teams {
				if team.ID == pid {
					avgElo := team.AverageElo(t.Type)
					players = append(players, &player.Player{
						ID:         team.ID,
						FirstName:  team.Name,
						LastName:   " (Team)",
						SinglesElo: avgElo,
						DoublesElo: avgElo,
					})
					break
				}
			}
		}
	}

	if knockoutGroup != nil {
		knockoutGroup.Players = players
	} else {
		newGroup := tournamentDomain.Group{
			ID:      idgen.Generate(),
			Name:    groupName,
			Players: players,
		}
		t.Groups = append(t.Groups, newGroup)
	}

	return uc.repo.UpdateGroups(ctx, t)
}
