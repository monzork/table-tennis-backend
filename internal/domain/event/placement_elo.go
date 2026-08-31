package event

import (
	"math"

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
		first, second, _ := placementBonusAmounts(divisions, m.DivisionID)
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
		_, _, third := placementBonusAmounts(divisions, m.DivisionID)
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

// roundRobinPlacementBonus builds one standings table per DivisionID present
// in the event's matches (an event with no divisions has every match share
// the same empty DivisionID, i.e. one table over every participant, matching
// the pre-division behavior) and awards each table's own top 3 the bonus for
// that division's multiplier.
func roundRobinPlacementBonus(t *Event, divisions []*division.Division) map[string]float64 {
	pool := roundRobinParticipantPool(t)
	if len(pool) == 0 {
		return nil
	}

	byDivision := make(map[string][]Match)
	for _, m := range t.Matches {
		if m.TeamMatchID != nil {
			continue
		}
		byDivision[m.DivisionID] = append(byDivision[m.DivisionID], m)
	}

	bonus := make(map[string]float64)
	for divID, matches := range byDivision {
		ids := make(map[string]bool)
		for _, m := range matches {
			for _, p := range m.TeamA {
				ids[p.ID] = true
			}
			for _, p := range m.TeamB {
				ids[p.ID] = true
			}
		}

		var participants []*player.Player
		for _, p := range pool {
			if ids[p.ID] {
				participants = append(participants, p)
			}
		}
		if len(participants) == 0 {
			continue
		}

		standings := BuildStandings(participants, matches)
		first, second, third := placementBonusAmounts(divisions, divID)
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
	}

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
