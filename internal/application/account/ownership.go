package account

import (
	"errors"

	"table-tennis-backend/internal/domain/player"
)

// ErrPlayerNotOwnedByAccount is returned when an account tries to view or
// mutate a player it isn't the guardian of. AccountProtected middleware only
// proves *who* is logged in, not *which* players they may touch — that check
// belongs here, at the use-case layer, so every caller gets it for free.
var ErrPlayerNotOwnedByAccount = errors.New("player is not linked to this account")

// EnsurePlayerBelongsToAccount verifies p is guarded by the given account.
func EnsurePlayerBelongsToAccount(p *player.Player, accountID string) error {
	if p == nil || p.GuardianAccountID == nil || *p.GuardianAccountID != accountID {
		return ErrPlayerNotOwnedByAccount
	}
	return nil
}
