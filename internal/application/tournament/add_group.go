package tournament

import (
	"context"
	"errors"
	"fmt"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"
)

type AddGroupUseCase struct {
	repo tournament.Repository
}

func NewAddGroupUseCase(repo tournament.Repository) *AddGroupUseCase {
	return &AddGroupUseCase{repo: repo}
}

func (uc *AddGroupUseCase) Execute(ctx context.Context, tournamentID string, divisionName string) error {
	t, err := uc.repo.GetByID(ctx, tournamentID)
	if err != nil {
		return err
	}

	if t.Status == "finished" {
		return errors.New("cannot add groups to a finished tournament")
	}

	// Calculate the next group letter
	count := 0
	prefix := divisionName + " - Group "
	for _, g := range t.Groups {
		if len(g.Name) > len(prefix) && g.Name[:len(prefix)] == prefix {
			count++
		}
	}

	newLetter := 'A' + count
	newName := fmt.Sprintf("%s - Group %c", divisionName, newLetter)

	newGroup := tournament.Group{
		ID:           idgen.Generate(),
		TournamentID: t.ID,
		Name:         newName,
		Players:      []*player.Player{},
	}
	t.Groups = append(t.Groups, newGroup)

	return uc.repo.UpdateGroups(ctx, t)
}
