package event

import "table-tennis-backend/internal/domain/player"

// OrderedAgeCategories lists every age category from youngest to oldest,
// "open" (no age ceiling) first. Exported for the admin tournament-creation
// UI and the public ranking page's age filter to enumerate valid options.
var OrderedAgeCategories = []string{"open", "u11", "u13", "u15", "u19"}

// ageCategoryMaxAge is the ITTF-style birth-year-cutoff ceiling for each
// youth bracket -- a player qualifies for a bracket if their age (in the
// event's season) is at or under this ceiling. "open" has no entry and is
// handled separately (unrestricted).
var ageCategoryMaxAge = map[string]int{
	"u11": 11,
	"u13": 13,
	"u15": 15,
	"u19": 19,
}

// IsAgeEligible reports whether a player born on p.Birthdate may enter an
// event in ageCategory during the given season (the event's own StartDate
// year, not real time -- so eligibility never drifts after the tournament
// is created). Age is computed ITTF-style, by birth year only:
// age = seasonYear - birthYear.
//
// "" and "open" are always eligible (no ceiling). Play-up is allowed: a
// player qualifies for their own bracket and every older one, never a
// younger one -- e.g. a 10-year-old (U11-eligible) may also enter U13, U15,
// U19, or Open, but a 14-year-old may not enter U11 or U13.
func IsAgeEligible(p *player.Player, ageCategory string, seasonYear int) bool {
	if ageCategory == "" || ageCategory == "open" {
		return true
	}
	maxAge, ok := ageCategoryMaxAge[ageCategory]
	if !ok {
		return true
	}
	age := seasonYear - p.Birthdate.Year()
	return age <= maxAge
}
