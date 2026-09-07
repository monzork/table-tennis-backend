package event_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

func birthdateFor(year int) time.Time {
	return time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
}

func TestIsAgeEligible(t *testing.T) {
	seasonYear := 2026

	t.Run("open and empty category are always eligible", func(t *testing.T) {
		p := &player.Player{Birthdate: birthdateFor(1980)}
		if !event.IsAgeEligible(p, "open", seasonYear) {
			t.Error("expected open to always be eligible")
		}
		if !event.IsAgeEligible(p, "", seasonYear) {
			t.Error("expected empty category to always be eligible")
		}
	})

	t.Run("exact boundary age is eligible", func(t *testing.T) {
		// Born 2015 -> age 11 in season 2026 -- exactly at the U11 ceiling.
		p := &player.Player{Birthdate: birthdateFor(2015)}
		if !event.IsAgeEligible(p, "u11", seasonYear) {
			t.Error("expected a player at the exact age ceiling to be eligible")
		}
	})

	t.Run("one year over the ceiling is ineligible", func(t *testing.T) {
		// Born 2014 -> age 12 in season 2026 -- one year too old for U11.
		p := &player.Player{Birthdate: birthdateFor(2014)}
		if event.IsAgeEligible(p, "u11", seasonYear) {
			t.Error("expected a player one year over the ceiling to be ineligible")
		}
	})

	t.Run("play-up: a younger player is eligible for every older bracket", func(t *testing.T) {
		// Born 2018 -> age 8 in season 2026 -- U11-eligible.
		p := &player.Player{Birthdate: birthdateFor(2018)}
		for _, ac := range []string{"u11", "u13", "u15", "u19", "open"} {
			if !event.IsAgeEligible(p, ac, seasonYear) {
				t.Errorf("expected an 8-year-old to be eligible for %s (play-up), got false", ac)
			}
		}
	})

	t.Run("an older player is ineligible for every younger bracket", func(t *testing.T) {
		// Born 2010 -> age 16 in season 2026 -- U19-eligible only.
		p := &player.Player{Birthdate: birthdateFor(2010)}
		for _, ac := range []string{"u11", "u13", "u15"} {
			if event.IsAgeEligible(p, ac, seasonYear) {
				t.Errorf("expected a 16-year-old to be ineligible for %s, got true", ac)
			}
		}
		if !event.IsAgeEligible(p, "u19", seasonYear) {
			t.Error("expected a 16-year-old to be eligible for u19")
		}
	})

	t.Run("unrecognized category is treated as unrestricted", func(t *testing.T) {
		p := &player.Player{Birthdate: birthdateFor(1980)}
		if !event.IsAgeEligible(p, "not-a-real-category", seasonYear) {
			t.Error("expected an unrecognized category to be treated as unrestricted")
		}
	})

	t.Run("season year is pinned to the event, not real time", func(t *testing.T) {
		// Born 2013 -> age 13 in season 2026 (U13-eligible), but age 14 in
		// season 2027 (no longer U13-eligible) -- proving the season year
		// parameter, not time.Now(), drives the computation.
		p := &player.Player{Birthdate: birthdateFor(2013)}
		if !event.IsAgeEligible(p, "u13", 2026) {
			t.Error("expected U13-eligible in season 2026")
		}
		if event.IsAgeEligible(p, "u13", 2027) {
			t.Error("expected no longer U13-eligible in season 2027")
		}
	})
}
