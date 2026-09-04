package player

import (
	"context"
	"sort"
	"sync"

	tournamentEvent "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"golang.org/x/sync/errgroup"
)

// PlayerEventStatsView pairs a child event (category) with the requested
// player's match record and Elo snapshot within it.
type PlayerEventStatsView struct {
	Event            *tournamentEvent.Event
	Stats            tournamentEvent.PlayerEventStats
	Matches          []tournamentEvent.PlayerMatchDetail
	Participated     bool // false if the player was registered but never actually played a match
	EloBeforeSingles *int16
	EloAfterSingles  *int16
	EloBeforeDoubles *int16
	EloAfterDoubles  *int16
	// PendingProposals previews the Elo effect of any not-yet-confirmed
	// score proposal on this player's matches within the event.
	PendingProposals []tournamentEvent.PendingProposalPreview
	// Placement/PlacementBonus are this player's durable podium result for
	// this specific event, if any -- see tournamentEvent.SavePlacementResults.
	Placement      *string
	PlacementBonus *float64
}

// PlayerTournamentView groups a player's per-event stats under the parent
// tournament they belong to.
type PlayerTournamentView struct {
	Tournament *tournament.Tournament
	Events     []PlayerEventStatsView
}

// TotalMatchesPlayed sums matches played across all events within this tournament.
func (v PlayerTournamentView) TotalMatchesPlayed() int {
	total := 0
	for _, ev := range v.Events {
		total += ev.Stats.Played
	}
	return total
}

type GetPlayerTournamentStatsUseCase struct {
	playerRepo     player.Repository
	eventRepo      tournamentEvent.Repository
	tournamentRepo tournament.Repository
}

func NewGetPlayerTournamentStatsUseCase(playerRepo player.Repository, eventRepo tournamentEvent.Repository, tournamentRepo tournament.Repository) *GetPlayerTournamentStatsUseCase {
	return &GetPlayerTournamentStatsUseCase{
		playerRepo:     playerRepo,
		eventRepo:      eventRepo,
		tournamentRepo: tournamentRepo,
	}
}

func (uc *GetPlayerTournamentStatsUseCase) Execute(ctx context.Context, playerID string) (*player.Player, []PlayerTournamentView, error) {
	var p *player.Player
	var events []*tournamentEvent.Event

	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		var err error
		p, err = uc.playerRepo.GetById(egCtx, playerID)
		return err
	})
	eg.Go(func() error {
		var err error
		events, err = uc.eventRepo.GetByParticipantID(egCtx, playerID)
		return err
	})
	if err := eg.Wait(); err != nil {
		return nil, nil, err
	}

	// Fetch each event's Elo snapshot and each distinct parent tournament
	// concurrently, since these are independent per-event/per-tournament reads.
	tournamentIDs := make(map[string]struct{})
	for _, ev := range events {
		if ev.TournamentID != nil && *ev.TournamentID != "" {
			tournamentIDs[*ev.TournamentID] = struct{}{}
		}
	}

	var mu sync.Mutex
	snapshotsByEvent := make(map[string]tournamentEvent.ParticipantSnapshot, len(events))
	tournamentsByID := make(map[string]*tournament.Tournament, len(tournamentIDs))

	fg, fgCtx := errgroup.WithContext(ctx)
	for _, ev := range events {
		fg.Go(func() error {
			snapshots, err := uc.eventRepo.GetParticipantSnapshots(fgCtx, ev.ID)
			if err != nil {
				return nil // best-effort: missing snapshot just omits Elo info
			}
			for _, snap := range snapshots {
				if snap.PlayerID == playerID {
					mu.Lock()
					snapshotsByEvent[ev.ID] = snap
					mu.Unlock()
					break
				}
			}
			return nil
		})
	}
	for tid := range tournamentIDs {
		fg.Go(func() error {
			t, err := uc.tournamentRepo.GetByID(fgCtx, tid)
			if err != nil {
				return nil // best-effort: skip tournaments that fail to load
			}
			mu.Lock()
			tournamentsByID[tid] = t
			mu.Unlock()
			return nil
		})
	}
	_ = fg.Wait()

	grouped := make(map[string][]PlayerEventStatsView)
	var order []string
	for _, ev := range events {
		if ev.TournamentID == nil || *ev.TournamentID == "" {
			continue
		}
		tid := *ev.TournamentID
		if _, ok := tournamentsByID[tid]; !ok {
			continue
		}
		if _, seen := grouped[tid]; !seen {
			order = append(order, tid)
		}

		stats := tournamentEvent.BuildPlayerEventStats(playerID, ev.Matches)
		view := PlayerEventStatsView{
			Event:            ev,
			Stats:            stats,
			Matches:          tournamentEvent.BuildPlayerMatchDetails(playerID, ev.Matches, ev.Status == "finished"),
			Participated:     stats.Played > 0,
			PendingProposals: tournamentEvent.BuildPendingProposalPreviews(playerID, ev.Name, ev.Matches),
		}
		if snap, ok := snapshotsByEvent[ev.ID]; ok {
			view.EloBeforeSingles = snap.EloBeforeSingles
			view.EloAfterSingles = snap.EloAfterSingles
			view.EloBeforeDoubles = snap.EloBeforeDoubles
			view.EloAfterDoubles = snap.EloAfterDoubles
			view.Placement = snap.Placement
			view.PlacementBonus = snap.PlacementBonusElo
		}
		grouped[tid] = append(grouped[tid], view)
	}

	sort.Slice(order, func(i, j int) bool {
		return tournamentsByID[order[i]].StartDate.After(tournamentsByID[order[j]].StartDate)
	})

	result := make([]PlayerTournamentView, 0, len(order))
	for _, tid := range order {
		result = append(result, PlayerTournamentView{
			Tournament: tournamentsByID[tid],
			Events:     grouped[tid],
		})
	}

	return p, result, nil
}
