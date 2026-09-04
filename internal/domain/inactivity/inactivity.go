package inactivity

import "context"

// Settings is the single admin-configurable set of parameters driving Elo
// decay for players who stop enrolling in federation-endorsed tournaments.
type Settings struct {
	// TournamentThreshold is how many consecutive federation-endorsed
	// tournaments a player can miss before an Elo penalty applies (and the
	// player is flagged inactive). Applies again every further multiple of
	// this count.
	TournamentThreshold int
	// EloPenalty is how many Elo points are lost each time
	// TournamentThreshold is reached.
	EloPenalty int
}

// BandFloor is the elo floor decay stops at for a given rating: the start of
// the player's current full-hundred band, minus one more hundred. E.g. 2101
// and 2436 and 1859 floor at 2000, 2300, and 1700 respectively -- decay can
// erode at most one whole century band below the one a player is currently
// in, never a fixed rating shared by every player.
func BandFloor(elo int16) int16 {
	floor := (elo/100)*100 - 100
	if floor < 0 {
		floor = 0
	}
	return floor
}

type Repository interface {
	Get(ctx context.Context) (*Settings, error)
	Update(ctx context.Context, s *Settings) error
}
