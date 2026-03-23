package match

import "table-tennis-backend/internal/domain/player"

// CalculateAndApplyElo computes the new Elo ratings for players after a match
// and applies them to the player structs.
func CalculateAndApplyElo(matchType string, teamA, teamB []*player.Player, winnerTeam string) {
	if len(teamA) == 0 || len(teamB) == 0 {
		return
	}

	K := 32.0

	var r1, r2 float64
	if matchType == "doubles" && len(teamA) == 2 && len(teamB) == 2 {
		r1 = float64(teamA[0].DoublesElo+teamA[1].DoublesElo) / 2.0
		r2 = float64(teamB[0].DoublesElo+teamB[1].DoublesElo) / 2.0
	} else {
		// Defaults to singles or team play (using singles elo)
		r1 = float64(teamA[0].SinglesElo)
		r2 = float64(teamB[0].SinglesElo)
	}

	e1 := 1.0 / (1 + pow10((r2-r1)/400))
	e2 := 1.0 / (1 + pow10((r1-r2)/400))

	var s1, s2 float64
	if winnerTeam == "A" {
		s1, s2 = 1.0, 0.0
	} else if winnerTeam == "B" {
		s1, s2 = 0.0, 1.0
	} else {
		return // Invalid winner team
	}

	delta1 := K * (s1 - e1)
	delta2 := K * (s2 - e2)

	// Apply to players
	if matchType == "doubles" && len(teamA) == 2 && len(teamB) == 2 {
		teamA[0].UpdateDoublesElo(int16(float64(teamA[0].DoublesElo) + delta1))
		teamA[1].UpdateDoublesElo(int16(float64(teamA[1].DoublesElo) + delta1))
		teamB[0].UpdateDoublesElo(int16(float64(teamB[0].DoublesElo) + delta2))
		teamB[1].UpdateDoublesElo(int16(float64(teamB[1].DoublesElo) + delta2))
	} else {
		// Singles mode
		teamA[0].UpdateSinglesElo(int16(float64(teamA[0].SinglesElo) + delta1))
		teamB[0].UpdateSinglesElo(int16(float64(teamB[0].SinglesElo) + delta2))
	}
}

func pow10(x float64) float64 {
	res := 1.0
	for i := 0; i < int(x*0.43429); i++ {
		res *= 10
	}
	// Fallback for an actual math.Pow replacement if needed, but keeping original app's logic
	return res
}
