package match_test

import (
	"testing"

	matchDomain "table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
)

func TestFFTTPoints(t *testing.T) {
	cases := []struct {
		rA, rB  int
		won     bool
		coeff   float64
		expect  float64
		label   string
	}{
		// Same-ish rating (gap 0-24) — normal win
		{1000, 1010, true, 1.0, 6.0, "equal ratings normal win"},
		// Same-ish rating — normal loss
		{1000, 1010, false, 1.0, -5.0, "equal ratings normal loss"},
		// Upset win: 800 beats 1000 (gap 200, 800 is lower-rated, 800 wins)
		{800, 1000, true, 1.0, 17.0, "upset win gap 200"},
		// Upset loss: 1000 loses to 800 (gap 200, 1000 is higher-rated)
		{1000, 800, false, 1.0, -12.5, "upset loss gap 200"},
		// Gap 400-499, normal win — 0.5
		{1500, 1050, true, 1.0, 0.5, "gap 450 normal win"},
		// Gap 500+, upset win — 40 points
		{500, 1100, true, 1.0, 40.0, "gap 600 upset win"},
		// Coefficient halves all points
		{1000, 1010, true, 0.5, 3.0, "local tournament coeff 0.5"},
	}

	for _, c := range cases {
		got := matchDomain.FFTTPoints(c.rA, c.rB, c.won, c.coeff)
		if got != c.expect {
			t.Errorf("[%s] FFTTPoints(%d, %d, won=%v, coeff=%.2f) = %.1f, want %.1f",
				c.label, c.rA, c.rB, c.won, c.coeff, got, c.expect)
		}
	}
}

func TestCalculateAndApplyEloSingles(t *testing.T) {
	// Player A (1000) beats Player B (800) — normal win, gap 200
	pA := &player.Player{SinglesElo: 1000}
	pB := &player.Player{SinglesElo: 800}

	matchDomain.CalculateAndApplyElo("singles", []*player.Player{pA}, []*player.Player{pB}, "A")

	// A gains 2 points (normal win, gap 200-299)
	if pA.SinglesElo != 1002 {
		t.Errorf("expected A to have 1002, got %d", pA.SinglesElo)
	}
	// B loses 1 point (normal loss, gap 200-299)
	if pB.SinglesElo != 799 {
		t.Errorf("expected B to have 799, got %d", pB.SinglesElo)
	}
}

func TestCalculateAndApplyEloUpset(t *testing.T) {
	// Player A (800) upsets Player B (1000) — upset win, gap 200
	pA := &player.Player{SinglesElo: 800}
	pB := &player.Player{SinglesElo: 1000}

	matchDomain.CalculateAndApplyElo("singles", []*player.Player{pA}, []*player.Player{pB}, "A")

	// A gains 17 points (upset win)
	if pA.SinglesElo != 817 {
		t.Errorf("expected A to have 817, got %d", pA.SinglesElo)
	}
	// B loses 12.5 -> rounds to -12 or -13
	if pB.SinglesElo != 988 && pB.SinglesElo != 987 {
		t.Errorf("expected B to have ~988, got %d", pB.SinglesElo)
	}
}
