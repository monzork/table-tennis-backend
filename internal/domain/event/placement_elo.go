package event

import (
	"math"
	"strings"

	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
)

// Flat Elo bonuses awarded on top of a player's normal match-based Elo swing
// for the event, based on their final tournament placement: the champion
// gets an extra bonus of PlacementEloMultiplier x K-factor (2x by default,
// see division.DefaultPlacementEloMultiplier), the runner-up half of that,
// and whoever is eliminated at the 3rd-place tier (both semifinal losers in
// a bracket, or the single 3rd-rank finisher in a round robin) half of the
// runner-up's bonus (rounded up). The multiplier is configured per division
// (Division.PlacementEloMultiplier) so e.g. an elite division can award a
// smaller champion bonus than a beginner division. Anyone else is absent
// from the map returned by PlacementEloBonus and gets no bonus.
const (
	FirstPlaceEloBonus  = match.DefaultKFactor * division.DefaultPlacementEloMultiplier
	SecondPlaceEloBonus = FirstPlaceEloBonus / 2
	ThirdPlaceEloBonus  = SecondPlaceEloBonus / 2
)

// placementBonusAmounts returns the champion/runner-up/3rd-place flat Elo
// bonuses for a division, scaled off its own PlacementEloMultiplier (or
// division.DefaultPlacementEloMultiplier when divisionID matches no known
// division, e.g. an event with no divisions at all).
func placementBonusAmounts(divisions []*division.Division, divisionID string) (first, second, third float64) {
	multiplier := division.DefaultPlacementEloMultiplier
	for _, d := range divisions {
		if d.ID == divisionID {
			if d.PlacementEloMultiplier > 0 {
				multiplier = d.PlacementEloMultiplier
			}
			break
		}
	}
	first = multiplier * match.DefaultKFactor
	second = first / 2
	third = math.Ceil(second / 2)
	return
}

// resolveDivisionID returns divisionID unchanged if non-empty (the match
// was already tagged, e.g. an event whose bracket splits multiple divisions
// internally). Some tournaments instead create one whole event per division
// with no per-match tagging at all (e.g. "II Ranking Nacional por
// Divisiones" makes a separate "...(1st Division (Women))" event per
// division) -- for those, resolve the division from a representative
// participant's Elo, so the right PlacementEloMultiplier is used instead of
// silently falling back to the default for every such event.
func resolveDivisionID(t *Event, divisions []*division.Division, divisionID string, sample *player.Player) string {
	if divisionID != "" || sample == nil {
		return divisionID
	}
	elo := sample.SinglesElo
	if t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams" {
		elo = sample.DoublesElo
	}
	for _, d := range divisions {
		if d.MinElo == 0 && d.MaxElo == nil {
			continue // skip the catch-all "No Division" band
		}
		if d.Category != "both" && d.Category != t.Type {
			continue
		}
		if !divisionMatchesEventCategoryForBonus(d, t) {
			continue
		}
		if d.ContainsElo(elo) {
			return d.ID
		}
	}
	return divisionID
}

// divisionMatchesEventCategoryForBonus mirrors
// bracket.divisionMatchesEventCategory -- domain/event can't import
// domain/bracket (which itself imports domain/event) without a cycle.
func divisionMatchesEventCategoryForBonus(d *division.Division, t *Event) bool {
	isGenderSpecific := d.Gender != "" && !strings.EqualFold(d.Gender, "both")
	if t.UseGenderDivisions {
		if !isGenderSpecific {
			return false
		}
		switch t.EventCategory {
		case "men":
			return strings.EqualFold(d.Gender, "M")
		case "women":
			return strings.EqualFold(d.Gender, "F")
		default:
			return false
		}
	}
	return !isGenderSpecific
}

// firstPlayer returns the first player in a team slot, or nil for an empty
// one.
func firstPlayer(team []*player.Player) *player.Player {
	if len(team) == 0 {
		return nil
	}
	return team[0]
}

// PlacementEloBonus computes, for every player whose final placement in this
// event lands on the podium, the flat Elo bonus (in rating points) they earn
// besides whatever they gained or lost from the matches they actually
// played. It only needs the relevant matches to be finished (Stage
// "final"/"semifinal" for elimination formats, or a full round robin), not
// for t.Status to already be "finished" -- the Elo application loop that
// calls this runs before the event is marked finished. divisions supplies
// the placement-bonus multiplier for each match's DivisionID; pass nil (or
// an event with no divisions) to use the default multiplier everywhere.
func PlacementEloBonus(t *Event, divisions []*division.Division) map[string]float64 {
	switch t.Format {
	case "elimination", "groups_elimination":
		return knockoutPlacementBonus(t, divisions)
	case "round_robin":
		return roundRobinPlacementBonus(t, divisions)
	default:
		return nil
	}
}

// resolveTeamSlot resolves a team-level slot to the real players it should
// apply to. "teams" format events route the final/semifinal aggregate match
// (and round-robin standings) through a single pseudo player wrapping the
// Team entity (ID == team.ID, see BuildBracket) -- same for doubles/
// mixed_doubles round-robin standings -- so any element whose ID matches a
// registered team is expanded to that team's actual roster. Elements that
// don't match a team (elimination-format doubles/mixed_doubles matches,
// which already carry the two real players directly, or singles) are kept
// as-is.
func resolveTeamSlot(t *Event, slot []*player.Player) []*player.Player {
	var resolved []*player.Player
	for _, p := range slot {
		matched := false
		for _, team := range t.Teams {
			if team.ID == p.ID {
				resolved = append(resolved, team.Players...)
				matched = true
				break
			}
		}
		if !matched {
			resolved = append(resolved, p)
		}
	}
	return resolved
}

// knockoutPlacementBonus walks every finished final/semifinal, keyed by its
// own DivisionID -- an event split into multiple divisions runs one
// independent bracket (and podium) per division, so each division's final
// and semifinals are scored with that division's own bonus multiplier.
func knockoutPlacementBonus(t *Event, divisions []*division.Division) map[string]float64 {
	bonus := make(map[string]float64)

	for i := range t.Matches {
		m := &t.Matches[i]
		if m.Stage != "final" || m.Status != "finished" || m.TeamMatchID != nil || m.WinnerTeam == "" {
			continue
		}
		divID := resolveDivisionID(t, divisions, m.DivisionID, firstPlayer(m.TeamA))
		first, second, _ := placementBonusAmounts(divisions, divID)
		winners, losers := m.TeamA, m.TeamB
		if m.WinnerTeam == "B" {
			winners, losers = m.TeamB, m.TeamA
		}
		for _, p := range resolveTeamSlot(t, winners) {
			bonus[p.ID] = first
		}
		for _, p := range resolveTeamSlot(t, losers) {
			bonus[p.ID] = second
		}
	}

	for i := range t.Matches {
		m := &t.Matches[i]
		if m.Stage != "semifinal" || m.Status != "finished" || m.TeamMatchID != nil || m.WinnerTeam == "" {
			continue
		}
		divID := resolveDivisionID(t, divisions, m.DivisionID, firstPlayer(m.TeamA))
		_, _, third := placementBonusAmounts(divisions, divID)
		loser := m.TeamA
		if m.WinnerTeam == "A" {
			loser = m.TeamB
		}
		for _, p := range resolveTeamSlot(t, loser) {
			bonus[p.ID] = third
		}
	}

	return bonus
}

// roundRobinPlacementBonus builds one standings table over every
// participant and awards its top 3 the bonus for the event's single
// division. Round-robin-format events in this codebase are always scoped to
// exactly one division (e.g. "II Ranking Nacional por Divisiones" creates a
// separate whole event per division, see resolveDivisionID) -- multi-
// division brackets go through groups_elimination/knockoutPlacementBonus
// instead, where per-match DivisionID is reliably stamped. Deliberately
// does NOT bucket by each match's own DivisionID: a defaulting player's
// forfeited matches have been observed to carry a stale/incorrect
// DivisionID (e.g. a legacy gender-agnostic division ID) while every other
// match in the same round-robin group carries none, which would otherwise
// split one real group's matches into a bogus second "division" whose
// standings are computed from only that handful of forfeits.
func roundRobinPlacementBonus(t *Event, divisions []*division.Division) map[string]float64 {
	pool := roundRobinParticipantPool(t)
	if len(pool) == 0 {
		return nil
	}

	var matches []Match
	for _, m := range t.Matches {
		if m.TeamMatchID != nil {
			continue
		}
		matches = append(matches, m)
	}

	standings := BuildStandings(pool, matches)
	divID := resolveDivisionID(t, divisions, "", firstPlayer(pool))
	first, second, third := placementBonusAmounts(divisions, divID)

	bonus := make(map[string]float64)
	assign := func(rank int, amount float64) {
		if len(standings) <= rank {
			return
		}
		slot := []*player.Player{standings[rank].Player}
		for _, p := range resolveTeamSlot(t, slot) {
			bonus[p.ID] = amount
		}
	}
	assign(0, first)
	assign(1, second)
	assign(2, third)

	return bonus
}

// roundRobinParticipantPool returns the full candidate pool for round-robin
// standings: pseudo players wrapping each Team for teams/doubles/
// mixed_doubles events (see resolveTeamSlot), or the event's real
// participants for singles.
func roundRobinParticipantPool(t *Event) []*player.Player {
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		pool := make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			pool[i] = &player.Player{ID: team.ID}
		}
		return pool
	}
	return t.Participants
}
