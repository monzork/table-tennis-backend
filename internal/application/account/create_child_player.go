package account

import (
	"context"
	"time"

	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
)

// CreateChildPlayerCommand is the guardian-facing "add a child" form. This is
// the explicit-by-ID replacement pattern the guardian flow uses instead of
// internal/application/event/self_register.go's fuzzy name matching — it is
// deliberately separate and does not reuse that heuristic.
type CreateChildPlayerCommand struct {
	GuardianAccountID string
	FirstName         string
	LastName          string
	Birthdate         time.Time
	Gender            string
	Country           string
	Department        string
}

type CreateChildPlayerUseCase struct {
	playerRepo player.Repository
}

func NewCreateChildPlayerUseCase(playerRepo player.Repository) *CreateChildPlayerUseCase {
	return &CreateChildPlayerUseCase{playerRepo: playerRepo}
}

func (uc *CreateChildPlayerUseCase) Execute(ctx context.Context, cmd CreateChildPlayerCommand) (*player.Player, error) {
	p, err := player.NewGuardianChildPlayer(
		idgen.Generate(),
		cmd.GuardianAccountID,
		cmd.FirstName,
		cmd.LastName,
		cmd.Birthdate,
		cmd.Gender,
		cmd.Country,
		cmd.Department,
	)
	if err != nil {
		return nil, err
	}
	if err := uc.playerRepo.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}
