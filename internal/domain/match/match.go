package match

import (
	"context"
	"math"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

// Repository defines the data access methods for the Match domain.
// Used to decouple the HTTP/Application layers from the underlying database driver (e.g. Bun).
type Repository interface {
	GetByID(ctx context.Context, id string) (*event.Match, error)
	Update(ctx context.Context, m *event.Match) error
	GetActiveMatchByTable(ctx context.Context, tableNumber int, tournamentID string, eventID string) (*event.Match, error)
	DeleteMatchSets(ctx context.Context, matchID string) error
	GetSubMatches(ctx context.Context, parentID string) ([]*event.Match, error)
}

// Standard Elo calculation constants.
const (
	DefaultKFactor = 32.0
)

// StandardEloPoints calculates the rating change using the standard logistic Elo formula.
// Suitable for any pool size (e.g. ~150 players in Nicaragua).
func StandardEloPoints(rA, rB int, wonMatch bool, kFactor float64) float64 {
	expectedA := 1.0 / (1.0 + math.Pow(10.0, float64(rB-rA)/400.0))
	actualA := 0.0
	if wonMatch {
		actualA = 1.0
	}
	return math.Round(kFactor*(actualA-expectedA)*10) / 10
}

// FFTTTier represents one row in the FFTT official point table (French System).
type FFTTTier struct {
	MinGap     int
	MaxGap     int
	WinNormal  float64
	LossNormal float64
	WinUpset   float64
	LossUpset  float64
}

// ffttTable is the official FFTT bareme
var ffttTable = []FFTTTier{
	{0, 24, 6.0, -5.0, 6.0, -5.0},
	{25, 49, 5.5, -4.5, 7.0, -6.0},
	{50, 99, 5.0, -4.0, 8.0, -7.0},
	{100, 149, 4.0, -3.0, 10.0, -8.0},
	{150, 199, 3.0, -2.0, 13.0, -10.0},
	{200, 299, 2.0, -1.0, 17.0, -12.5},
	{300, 399, 1.0, -0.5, 22.0, -16.0},
	{400, 499, 0.5, 0.0, 28.0, -20.0},
	{500, -1, 0.0, 0.0, 40.0, -29.0},
}

// ffttLookup selects the correct tier for a given absolute rating gap.
func ffttLookup(gap int) FFTTTier {
	for _, t := range ffttTable {
		if gap >= t.MinGap && (t.MaxGap == -1 || gap <= t.MaxGap) {
			return t
		}
	}
	return ffttTable[len(ffttTable)-1]
}

// FFTTPoints returns the point delta using the French FFTT table system.
func FFTTPoints(rA, rB int, wonMatch bool, coeff float64) float64 {
	gap := int(math.Abs(float64(rA - rB)))
	tier := ffttLookup(gap)

	var raw float64
	higherRated := rA >= rB

	switch {
	case wonMatch && higherRated:
		raw = tier.WinNormal
	case wonMatch && !higherRated:
		raw = tier.WinUpset
	case !wonMatch && !higherRated:
		raw = tier.LossNormal
	case !wonMatch && higherRated:
		raw = tier.LossUpset
	}

	return math.Round(raw*coeff*10) / 10
}

// CalculateAndApplyElo computes and applies rating adjustments after a match.
// By default, it uses the standard internationally recognized Elo rating system (K-factor = 32),
// which is highly recommended for standard player pools like Nicaragua.
func CalculateAndApplyElo(matchType string, teamA, teamB []*player.Player, winnerTeam string) {
	if len(teamA) == 0 || len(teamB) == 0 {
		return
	}

	wonA := winnerTeam == "A"
	wonB := winnerTeam == "B"
	if !wonA && !wonB {
		return
	}

	kFactor := DefaultKFactor

	if matchType == "doubles" && len(teamA) == 2 && len(teamB) == 2 {
		rA := (int(teamA[0].DoublesElo) + int(teamA[1].DoublesElo)) / 2
		rB := (int(teamB[0].DoublesElo) + int(teamB[1].DoublesElo)) / 2

		deltaA := StandardEloPoints(rA, rB, wonA, kFactor)
		deltaB := StandardEloPoints(rB, rA, wonB, kFactor)

		teamA[0].UpdateDoublesElo(int16(math.Round(float64(teamA[0].DoublesElo) + deltaA)))
		teamA[1].UpdateDoublesElo(int16(math.Round(float64(teamA[1].DoublesElo) + deltaA)))
		teamB[0].UpdateDoublesElo(int16(math.Round(float64(teamB[0].DoublesElo) + deltaB)))
		teamB[1].UpdateDoublesElo(int16(math.Round(float64(teamB[1].DoublesElo) + deltaB)))
	} else {
		rA := int(teamA[0].SinglesElo)
		rB := int(teamB[0].SinglesElo)

		deltaA := StandardEloPoints(rA, rB, wonA, kFactor)
		deltaB := StandardEloPoints(rB, rA, wonB, kFactor)

		teamA[0].UpdateSinglesElo(int16(math.Round(float64(teamA[0].SinglesElo) + deltaA)))
		teamB[0].UpdateSinglesElo(int16(math.Round(float64(teamB[0].SinglesElo) + deltaB)))
	}
}
