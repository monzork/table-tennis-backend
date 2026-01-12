package match

import "table-tennis-backend/internal/domain/player"

func CalculateElo(p1, p2 *player.Player, winner *player.Player) (int16, int16) {
	K := 32
	r1 := float64(p1.Elo)
	r2 := float64(p2.Elo)

	e1 := 1.0 / (1 + pow10((r2-r1)/400))
	e2 := 1.0 / (1 + pow10((r1-r2)/400))

	var s1, s2 float64
	if winner.ID == p1.ID {
		s1, s2 = 1, 0
	} else {
		s1, s2 = 0, 1
	}

	newElo1 := int16(float64(p1.Elo) + float64(K)*(s1-e1))
	newElo2 := int16(float64(p2.Elo) + float64(K)*(s2-e2))
	return newElo1, newElo2
}

func pow10(x float64) float64 {
	res := 1.0
	for i := 0; i < int(x*0.43429); i++ {
		res *= 10
	}
	return res
}
