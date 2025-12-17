package helpers

import "math"

func CalculateElo(playerElo int, opponentElo int, score float64, k int) int {
	expected := 1.0 / (1.0 + math.Pow(10, float64(opponentElo-playerElo)/400.0))
	return playerElo + int(float64(k)*(score-expected))
}

func TeamEloUpdate(teamA []int, teamB []int, winner string, k int) (newTeamA []int, newTeamB []int) {
	avgA := avg(teamA)
	avgB := avg(teamB)

	var scoreA float64
	switch winner {
	case "A":
		scoreA = 1
	case "B":
		scoreA = 0
	default:
		scoreA = 0.5
	}

	for _, e := range teamA {
		newTeamA = append(newTeamA, CalculateElo(e, avgB, scoreA, k))
	}
	for _, e := range teamB {
		newTeamB = append(newTeamB, CalculateElo(e, avgA, 1-scoreA, k))
	}
	return
}

func avg(arr []int) int {
	sum := 0
	for _, v := range arr {
		sum += v
	}
	return sum / len(arr)
}
