package account

import (
	"context"
	"errors"

	"table-tennis-backend/internal/domain/account"
	"table-tennis-backend/internal/domain/player"
)

// ErrNoPendingClaim is returned when an admin tries to approve/reject a
// player claim that isn't actually pending.
var ErrNoPendingClaim = errors.New("player has no pending claim")

// PendingClaim pairs a claimed Player with the name and email of the
// account that claimed it, for the admin review queue.
type PendingClaim struct {
	Player       *player.Player
	AccountName  string
	AccountEmail string
}

// GetPendingPlayerClaimsUseCase lists every player with a claim awaiting an
// admin decision.
type GetPendingPlayerClaimsUseCase struct {
	playerRepo  player.Repository
	accountRepo account.Repository
}

func NewGetPendingPlayerClaimsUseCase(playerRepo player.Repository, accountRepo account.Repository) *GetPendingPlayerClaimsUseCase {
	return &GetPendingPlayerClaimsUseCase{playerRepo: playerRepo, accountRepo: accountRepo}
}

func (uc *GetPendingPlayerClaimsUseCase) Execute(ctx context.Context) ([]PendingClaim, error) {
	all, err := uc.playerRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	var claimants []*player.Player
	accountIDs := make([]string, 0, len(all))
	for _, p := range all {
		if p.ClaimedByAccountID == nil {
			continue
		}
		claimants = append(claimants, p)
		accountIDs = append(accountIDs, *p.ClaimedByAccountID)
	}

	accountsByID := make(map[string]*account.Account, len(accountIDs))
	if accounts, err := uc.accountRepo.GetByIDs(ctx, accountIDs); err == nil {
		for _, a := range accounts {
			accountsByID[a.ID] = a
		}
	}

	claims := make([]PendingClaim, 0, len(claimants))
	for _, p := range claimants {
		var name, email string
		if a, ok := accountsByID[*p.ClaimedByAccountID]; ok {
			name = a.Name
			email = a.Email
		}
		claims = append(claims, PendingClaim{Player: p, AccountName: name, AccountEmail: email})
	}
	return claims, nil
}

// ApprovePlayerClaimUseCase links the claiming account as the player's
// guardian and clears the pending claim.
type ApprovePlayerClaimUseCase struct {
	playerRepo player.Repository
}

func NewApprovePlayerClaimUseCase(playerRepo player.Repository) *ApprovePlayerClaimUseCase {
	return &ApprovePlayerClaimUseCase{playerRepo: playerRepo}
}

func (uc *ApprovePlayerClaimUseCase) Execute(ctx context.Context, playerID string) error {
	p, err := uc.playerRepo.GetById(ctx, playerID)
	if err != nil {
		return err
	}
	if p.ClaimedByAccountID == nil {
		return ErrNoPendingClaim
	}
	p.GuardianAccountID = p.ClaimedByAccountID
	p.ClaimedByAccountID = nil
	return uc.playerRepo.Save(ctx, p)
}

// RejectPlayerClaimUseCase clears a pending claim without linking the account.
type RejectPlayerClaimUseCase struct {
	playerRepo player.Repository
}

func NewRejectPlayerClaimUseCase(playerRepo player.Repository) *RejectPlayerClaimUseCase {
	return &RejectPlayerClaimUseCase{playerRepo: playerRepo}
}

func (uc *RejectPlayerClaimUseCase) Execute(ctx context.Context, playerID string) error {
	p, err := uc.playerRepo.GetById(ctx, playerID)
	if err != nil {
		return err
	}
	if p.ClaimedByAccountID == nil {
		return ErrNoPendingClaim
	}
	p.ClaimedByAccountID = nil
	return uc.playerRepo.Save(ctx, p)
}
