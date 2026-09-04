package inactivity_test

import (
	"testing"

	"table-tennis-backend/internal/domain/inactivity"
)

func TestBandFloor(t *testing.T) {
	cases := []struct {
		elo  int16
		want int16
	}{
		{2101, 2000},
		{2436, 2300},
		{1859, 1700},
		{2000, 1900}, // exactly on a band boundary
		{50, 0},      // clamped at zero rather than going negative
	}
	for _, tc := range cases {
		if got := inactivity.BandFloor(tc.elo); got != tc.want {
			t.Errorf("BandFloor(%d) = %d, want %d", tc.elo, got, tc.want)
		}
	}
}
