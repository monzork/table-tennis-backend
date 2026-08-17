package account

import (
	"context"
	"sync"

	tournamentEvent "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"

	playerApp "table-tennis-backend/internal/application/player"

	"golang.org/x/sync/errgroup"
)

// PlayerPendingMatches pairs a linked player with their pending-match list,
// for the account dashboard's cross-player view.
type PlayerPendingMatches struct {
	Player  *player.Player
	Matches []tournamentEvent.PlayerPendingMatchDetail
}

// GetGuardianPendingMatchesUseCase composes GetLinkedPlayersUseCase with a
// per-player GetPlayerPendingMatchesUseCase fan-out (parallel via errgroup,
// same pattern as GetPlayerTournamentStatsUseCase) for the account
// dashboard's cross-player pending list.
type GetGuardianPendingMatchesUseCase struct {
	getLinkedPlayersUC  *GetLinkedPlayersUseCase
	getPendingMatchesUC *playerApp.GetPlayerPendingMatchesUseCase
}

func NewGetGuardianPendingMatchesUseCase(getLinkedPlayersUC *GetLinkedPlayersUseCase, getPendingMatchesUC *playerApp.GetPlayerPendingMatchesUseCase) *GetGuardianPendingMatchesUseCase {
	return &GetGuardianPendingMatchesUseCase{getLinkedPlayersUC: getLinkedPlayersUC, getPendingMatchesUC: getPendingMatchesUC}
}

func (uc *GetGuardianPendingMatchesUseCase) Execute(ctx context.Context, accountID string) ([]PlayerPendingMatches, error) {
	players, err := uc.getLinkedPlayersUC.Execute(ctx, accountID)
	if err != nil {
		return nil, err
	}

	results := make([]PlayerPendingMatches, len(players))
	var mu sync.Mutex

	eg, egCtx := errgroup.WithContext(ctx)
	for i, p := range players {
		i, p := i, p
		eg.Go(func() error {
			matches, err := uc.getPendingMatchesUC.Execute(egCtx, p.ID)
			if err != nil {
				return nil // best-effort: skip players whose matches fail to load
			}
			mu.Lock()
			results[i] = PlayerPendingMatches{Player: p, Matches: matches}
			mu.Unlock()
			return nil
		})
	}
	_ = eg.Wait()

	return results, nil
}
