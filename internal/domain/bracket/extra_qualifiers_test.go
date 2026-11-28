package bracket

import (
	"fmt"
	"testing"

	"table-tennis-backend/internal/domain/player"
)

// extraTestGroups builds n groups; in each, the player right after the
// automatic qualifiers has a better record the later the group is.
func extraTestGroups(n, size, passCount int) []Group {
	groups := make([]Group, n)
	for gi := range groups {
		groups[gi].Name = fmt.Sprintf("Group %d", gi)
		for rank := range size {
			p := &player.Player{ID: fmt.Sprintf("g%d-%d", gi, rank+1)}
			st := PlayerStanding{Player: p, Wins: size - 1 - rank, Losses: rank}
			if rank == passCount {
				st.SetsWon, st.SetsLost = 9+gi, 9
			}
			groups[gi].Players = append(groups[gi].Players, p)
			groups[gi].Standings = append(groups[gi].Standings, st)
		}
	}
	return groups
}

func TestKnockoutSeeds_BestNextPlacedAdvanceAndStaySeparated(t *testing.T) {
	for passCount := 1; passCount <= 4; passCount++ {
		for n := 2; n <= 24; n++ {
			for extra := 1; extra <= n; extra++ {
				name := fmt.Sprintf("pass=%d groups=%d extra=%d", passCount, n, extra)
				groups := extraTestGroups(n, passCount+2, passCount)
				seeds := KnockoutSeeds(groups, passCount, extra, 0)

				if len(seeds) != passCount*n+extra {
					t.Fatalf("%s: got %d qualifiers, want %d", name, len(seeds), passCount*n+extra)
				}
				in := map[string]bool{}
				for _, p := range seeds {
					in[p.ID] = true
				}
				for gi := range n {
					if want := gi >= n-extra; in[fmt.Sprintf("g%d-%d", gi, passCount+1)] != want {
						t.Fatalf("%s: group %d next-placed qualified=%v, want %v", name, gi, !want, want)
					}
					for rank := 1; rank <= passCount; rank++ {
						if !in[fmt.Sprintf("g%d-%d", gi, rank)] {
							t.Fatalf("%s: group %d place %d missing", name, gi, rank)
						}
					}
				}
				if err := ValidateSameGroupSeparation(groups, seeds); err != nil {
					t.Errorf("%s: %v", name, err)
				}
			}
		}
	}
}

func TestQualifierPlaces_FillSize(t *testing.T) {
	six := extraTestGroups(6, 4, 2)
	for _, tc := range []struct {
		name                               string
		passCount, extra, fill, auto, best int
	}{
		{"byes keeps the configured counts", 2, 1, 0, 2, 1},
		{"round of 16 takes the 4 best 3rds", 2, 0, 16, 2, 4},
		{"round of 32 takes 3rds and 4ths, rest byes", 2, 0, 32, 4, 8},
		{"too small a round removes nobody", 2, 3, 8, 2, 0},
	} {
		auto, best := QualifierPlaces(six, tc.passCount, tc.extra, tc.fill)
		if auto != tc.auto || best != tc.best {
			t.Errorf("%s: got auto=%d best=%d, want %d and %d", tc.name, auto, best, tc.auto, tc.best)
		}
	}
}

func TestKnockoutSeeds_FilledBracketsStaySeparated(t *testing.T) {
	for _, fill := range []int{4, 8, 16, 32, 64} {
		for size := 3; size <= 5; size++ {
			for n := 2; n <= 16; n++ {
				groups := extraTestGroups(n, size, 2)
				seeds := KnockoutSeeds(groups, 2, 0, fill)
				if want := max(2*n, min(fill, size*n)); len(seeds) != want {
					t.Fatalf("fill=%d size=%d groups=%d: got %d qualifiers, want %d", fill, size, n, len(seeds), want)
				}
				if err := ValidateSameGroupSeparation(groups, seeds); err != nil {
					t.Errorf("fill=%d size=%d groups=%d: %v", fill, size, n, err)
				}
			}
		}
	}
}
