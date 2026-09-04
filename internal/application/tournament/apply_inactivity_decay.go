package tournament

import (
	"context"

	"table-tennis-backend/internal/domain/inactivity"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

// ApplyInactivityDecayUseCase penalizes players who stop enrolling in
// federation-endorsed tournaments: every TournamentThreshold consecutive
// such tournaments missed costs EloPenalty points, down to a per-rating
// floor computed from the elo it had the moment it first went inactive
// (inactivity.BandFloor) -- so a 2101-rated player floors at 2000 while a
// 1859-rated one floors at 1700, and neither erodes further than that once
// reached. The player is flagged Inactive once the threshold is first
// reached. Enrolling in a tournament again resets the streak, the flag, and
// both floors, so the next inactive streak computes fresh ones.
type ApplyInactivityDecayUseCase struct {
	tournamentRepo tournamentDomain.Repository
	playerRepo     playerDomain.Repository
	settingsRepo   inactivity.Repository
}

func NewApplyInactivityDecayUseCase(
	tournamentRepo tournamentDomain.Repository,
	playerRepo playerDomain.Repository,
	settingsRepo inactivity.Repository,
) *ApplyInactivityDecayUseCase {
	return &ApplyInactivityDecayUseCase{
		tournamentRepo: tournamentRepo,
		playerRepo:     playerRepo,
		settingsRepo:   settingsRepo,
	}
}

// ExecuteForEvent runs the decay pass for the parent tournament of the given
// child event, once that whole tournament has concluded (every one of its
// child events is finished). tournamentID nil (an event with no parent
// tournament) is a no-op.
//
// A tournament that isn't federation-endorsed (a local/friendly event)
// never counts toward or increments anyone's missed-tournament streak --
// but a player who shows up and plays one is demonstrably active, so their
// own streak/flag/floor still resets, PROVIDED the tournament actually
// awards Elo (SkipElo false); an exhibition with Elo turned off isn't a
// scored competitive result, so it resets nothing. That reset only ever
// touches players enrolled in *this* tournament: it never loads or affects
// anyone else, so a local tournament run at one club can't reset (or
// otherwise touch) players from elsewhere who simply didn't attend it.
func (uc *ApplyInactivityDecayUseCase) ExecuteForEvent(ctx context.Context, tournamentID *string) error {
	if tournamentID == nil {
		return nil
	}
	t, err := uc.tournamentRepo.GetByIDDeep(ctx, *tournamentID)
	if err != nil {
		return err
	}
	if len(t.Events) == 0 || t.SkipElo {
		return nil
	}
	for _, e := range t.Events {
		if e.Status != "finished" {
			return nil
		}
	}

	enrolled := make(map[string]bool)
	for _, e := range t.Events {
		for _, p := range e.Participants {
			enrolled[p.ID] = true
		}
	}

	if !t.FederationEndorsed {
		return uc.resetEnrolled(ctx, enrolled)
	}

	settings, err := uc.settingsRepo.Get(ctx)
	if err != nil {
		return err
	}

	allPlayers, err := uc.playerRepo.GetAll(ctx)
	if err != nil {
		return err
	}

	var toUpdate []*playerDomain.Player
	for _, p := range allPlayers {
		if enrolled[p.ID] {
			if reset(p) {
				toUpdate = append(toUpdate, p)
			}
			continue
		}

		p.MissedFederatedTournaments++
		if int(p.MissedFederatedTournaments) >= settings.TournamentThreshold {
			p.Inactive = true
		}
		if settings.TournamentThreshold > 0 && int(p.MissedFederatedTournaments)%settings.TournamentThreshold == 0 {
			p.SinglesElo, p.FloorSingles = decayElo(p.SinglesElo, p.FloorSingles, settings.EloPenalty)
			p.DoublesElo, p.FloorDoubles = decayElo(p.DoublesElo, p.FloorDoubles, settings.EloPenalty)
		}
		toUpdate = append(toUpdate, p)
	}

	return uc.playerRepo.UpdateInactivity(ctx, toUpdate)
}

// resetEnrolled clears the missed-tournament streak/flag/floor for exactly
// the given player IDs -- fetched by ID rather than via GetAll, so a local
// tournament's roster can never touch a player who wasn't in it.
func (uc *ApplyInactivityDecayUseCase) resetEnrolled(ctx context.Context, enrolled map[string]bool) error {
	if len(enrolled) == 0 {
		return nil
	}
	ids := make([]string, 0, len(enrolled))
	for id := range enrolled {
		ids = append(ids, id)
	}
	players, err := uc.playerRepo.GetByIDs(ctx, ids)
	if err != nil {
		return err
	}

	var toUpdate []*playerDomain.Player
	for _, p := range players {
		if reset(p) {
			toUpdate = append(toUpdate, p)
		}
	}
	return uc.playerRepo.UpdateInactivity(ctx, toUpdate)
}

// reset clears a player's missed-tournament streak/flag/floor and reports
// whether anything actually changed.
func reset(p *playerDomain.Player) bool {
	if p.MissedFederatedTournaments == 0 && !p.Inactive && p.FloorSingles == nil && p.FloorDoubles == nil {
		return false
	}
	p.MissedFederatedTournaments = 0
	p.Inactive = false
	p.FloorSingles = nil
	p.FloorDoubles = nil
	return true
}

// decayElo applies one inactivity penalty to elo, never pushing it below
// floor. floor is computed from elo the first time this rating goes
// inactive (floor == nil) and then reused on every later decay in the same
// streak, so it doesn't keep sliding down band by band on each further
// missed tournament -- see inactivity.BandFloor.
func decayElo(elo int16, floor *int16, penalty int) (int16, *int16) {
	if floor == nil {
		f := inactivity.BandFloor(elo)
		floor = &f
	}
	if elo <= *floor {
		return elo, floor
	}
	newElo := int(elo) - penalty
	if newElo < int(*floor) {
		newElo = int(*floor)
	}
	return int16(newElo), floor
}
