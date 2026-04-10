package match

import (
	"math"
	"table-tennis-backend/internal/domain/player"
)

// FFTTTier represents one row in the FFTT official point table.
// Points awarded/deducted depend on the gap between the two players' ratings
// and on whether the outcome is "normal" (favourite wins) or "abnormal" (upset).
type FFTTTier struct {
	MinGap         int     // inclusive lower bound of the rating gap
	MaxGap         int     // inclusive upper bound (-1 = infinity)
	WinNormal      float64 // normal win  (higher-rated beats lower-rated)
	LossNormal     float64 // normal loss (lower-rated loses to higher-rated)
	WinUpset       float64 // performance (lower-rated upsets higher-rated)
	LossUpset      float64 // contre       (higher-rated loses to lower-rated)
}

// ffttTable is the official FFTT bareme (coefficient = 1.0 base)
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

// CompetitionCoefficient controls the weight of a match.
// Admins can later pass these in via tournament settings.
const (
	CoeffTeamChampionship  = 1.0
	CoeffFederal           = 1.0
	CoeffNationalTournament = 0.75
	CoeffLocalTournament   = 0.5
	CoeffFinalDepartmental = 1.25
)

// ffttLookup selects the correct tier for a given absolute rating gap.
func ffttLookup(gap int) FFTTTier {
	for _, t := range ffttTable {
		if gap >= t.MinGap && (t.MaxGap == -1 || gap <= t.MaxGap) {
			return t
		}
	}
	return ffttTable[len(ffttTable)-1]
}

// FFTTPoints returns the point delta for playerA given their rating (rA) vs opponent (rB).
// wonMatch = true if playerA won.  coeff is the competition coefficient (default 1.0).
func FFTTPoints(rA, rB int, wonMatch bool, coeff float64) float64 {
	gap := int(math.Abs(float64(rA - rB)))
	tier := ffttLookup(gap)

	var raw float64
	higherRated := rA >= rB

	switch {
	case wonMatch && higherRated: // normal win
		raw = tier.WinNormal
	case wonMatch && !higherRated: // upset win (perf)
		raw = tier.WinUpset
	case !wonMatch && !higherRated: // normal loss
		raw = tier.LossNormal
	case !wonMatch && higherRated: // upset loss (contre)
		raw = tier.LossUpset
	}

	return math.Round(raw*coeff*10) / 10 // round to 1 decimal
}

// CalculateAndApplyElo applies the FFTT rating system after a match.
// matchType: "singles" or "doubles"
// coeff: competition coefficient (use CoeffLocalTournament etc. or 1.0 default)
func CalculateAndApplyElo(matchType string, teamA, teamB []*player.Player, winnerTeam string) {
	CalculateAndApplyEloWithCoeff(matchType, teamA, teamB, winnerTeam, 1.0)
}

// CalculateAndApplyEloWithCoeff is the full FFTT calculation with explicit competition coefficient.
func CalculateAndApplyEloWithCoeff(matchType string, teamA, teamB []*player.Player, winnerTeam string, coeff float64) {
	if len(teamA) == 0 || len(teamB) == 0 {
		return
	}

	wonA := winnerTeam == "A"
	wonB := winnerTeam == "B"
	if !wonA && !wonB {
		return
	}

	if matchType == "doubles" && len(teamA) == 2 && len(teamB) == 2 {
		// Use averaged doubles Elo for the gap calculation
		rA := (int(teamA[0].DoublesElo) + int(teamA[1].DoublesElo)) / 2
		rB := (int(teamB[0].DoublesElo) + int(teamB[1].DoublesElo)) / 2

		deltaA := FFTTPoints(rA, rB, wonA, coeff)
		deltaB := FFTTPoints(rB, rA, wonB, coeff)

		teamA[0].UpdateDoublesElo(int16(math.Round(float64(teamA[0].DoublesElo) + deltaA)))
		teamA[1].UpdateDoublesElo(int16(math.Round(float64(teamA[1].DoublesElo) + deltaA)))
		teamB[0].UpdateDoublesElo(int16(math.Round(float64(teamB[0].DoublesElo) + deltaB)))
		teamB[1].UpdateDoublesElo(int16(math.Round(float64(teamB[1].DoublesElo) + deltaB)))
	} else {
		// Singles
		rA := int(teamA[0].SinglesElo)
		rB := int(teamB[0].SinglesElo)

		deltaA := FFTTPoints(rA, rB, wonA, coeff)
		deltaB := FFTTPoints(rB, rA, wonB, coeff)

		teamA[0].UpdateSinglesElo(int16(math.Round(float64(teamA[0].SinglesElo) + deltaA)))
		teamB[0].UpdateSinglesElo(int16(math.Round(float64(teamB[0].SinglesElo) + deltaB)))
	}
}
