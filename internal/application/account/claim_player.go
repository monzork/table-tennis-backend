package account

import (
	"context"
	"errors"

	"table-tennis-backend/internal/domain/player"
)

var (
	// ErrPlayerAlreadyLinked is returned when an account tries to claim a
	// player that already has a guardian account.
	ErrPlayerAlreadyLinked = errors.New("player is already linked to an account")
	// ErrClaimAlreadyPending is returned when an account tries to claim a
	// player that another (or the same) account already has a pending claim on.
	ErrClaimAlreadyPending = errors.New("a claim is already pending for this player")
)

// ClaimPlayerUseCase lets a guardian account request to be linked to an
// existing, unlinked Player record. The claim stays pending until an admin
// approves or rejects it — see ApprovePlayerClaimUseCase and
// RejectPlayerClaimUseCase — since there's no other identity verification
// (same rationale as AssignPlayerToAccountUseCase).
type ClaimPlayerUseCase struct {
	playerRepo player.Repository
}

func NewClaimPlayerUseCase(playerRepo player.Repository) *ClaimPlayerUseCase {
	return &ClaimPlayerUseCase{playerRepo: playerRepo}
}

func (uc *ClaimPlayerUseCase) Execute(ctx context.Context, accountID, playerID string) error {
	p, err := uc.playerRepo.GetById(ctx, playerID)
	if err != nil {
		return err
	}
	if p.GuardianAccountID != nil {
		return ErrPlayerAlreadyLinked
	}
	if p.ClaimedByAccountID != nil {
		return ErrClaimAlreadyPending
	}
	p.ClaimedByAccountID = &accountID
	return uc.playerRepo.Save(ctx, p)
}

// SearchClaimablePlayersUseCase searches players a guardian account could
// claim: name match, not yet linked to any account, no pending claim.
type SearchClaimablePlayersUseCase struct {
	playerRepo player.Repository
}

func NewSearchClaimablePlayersUseCase(playerRepo player.Repository) *SearchClaimablePlayersUseCase {
	return &SearchClaimablePlayersUseCase{playerRepo: playerRepo}
}

func (uc *SearchClaimablePlayersUseCase) Execute(ctx context.Context, query string) ([]*player.Player, error) {
	results, err := uc.playerRepo.SearchForSelection(ctx, query, "")
	if err != nil {
		return nil, err
	}
	claimable := make([]*player.Player, 0, len(results))
	for _, p := range results {
		if p.GuardianAccountID == nil && p.ClaimedByAccountID == nil {
			claimable = append(claimable, p)
		}
	}
	return claimable, nil
}
