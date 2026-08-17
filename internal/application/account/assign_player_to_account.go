package account

import (
	"context"
	"errors"

	"table-tennis-backend/internal/domain/account"
	"table-tennis-backend/internal/domain/player"
)

// ErrAccountNotFound is returned when an admin tries to link a player to an
// account email that doesn't correspond to any existing Account.
var ErrAccountNotFound = errors.New("no account found for that email")

// AssignPlayerToAccountUseCase lets an admin manually link a Player to an
// Account by the account's email — covers both "attach an existing adult
// player to their own Google login" and "fix a mis-linked/duplicate child
// player," since there's no real identity verification to let a player
// self-claim a record.
type AssignPlayerToAccountUseCase struct {
	playerRepo  player.Repository
	accountRepo account.Repository
}

func NewAssignPlayerToAccountUseCase(playerRepo player.Repository, accountRepo account.Repository) *AssignPlayerToAccountUseCase {
	return &AssignPlayerToAccountUseCase{playerRepo: playerRepo, accountRepo: accountRepo}
}

// Execute links playerID to the Account owning accountEmail.
func (uc *AssignPlayerToAccountUseCase) Execute(ctx context.Context, playerID, accountEmail string) error {
	a, err := uc.accountRepo.GetByEmail(ctx, accountEmail)
	if err != nil {
		return err
	}
	if a == nil {
		return ErrAccountNotFound
	}

	p, err := uc.playerRepo.GetById(ctx, playerID)
	if err != nil {
		return err
	}

	id := a.ID
	p.GuardianAccountID = &id
	return uc.playerRepo.Save(ctx, p)
}

// Unlink clears a player's guardian account link, correcting a mistaken link.
func (uc *AssignPlayerToAccountUseCase) Unlink(ctx context.Context, playerID string) error {
	p, err := uc.playerRepo.GetById(ctx, playerID)
	if err != nil {
		return err
	}
	p.GuardianAccountID = nil
	return uc.playerRepo.Save(ctx, p)
}
