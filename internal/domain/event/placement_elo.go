package event

import (
	"table-tennis-backend/internal/domain/match"
	"table-tennis-backend/internal/domain/player"
)

// Flat Elo bonuses awarded on top of a player's normal match-based Elo swing
// for the event, based on their final tournament placement: the champion
// gets an extra 2x K-factor, the runner-up an extra 1x K-factor, and
// whoever is eliminated at the 3rd-place tier (both semifinal losers in a
// bracket, or the single 3rd-rank finisher in a round robin) an extra 0.5x
// K-factor. Anyone else is absent from the map returned by
// PlacementEloBonus and gets no bonus.
const (
	FirstPlaceEloBonus  = match.DefaultKFactor * 2
	SecondPlaceEloBonus = match.DefaultKFactor
	ThirdPlaceEloBonus  = match.DefaultKFactor / 2
)

// PlacementEloBonus computes, for every player whose final placement in this
// event lands on the podium, the flat Elo bonus (in rating points) they earn
// besides whatever they gained or lost from the matches they actually
// played. It only needs the relevant matches to be finished (Stage
// "final"/"semifinal" for elimination formats, or a full round robin), not
// for t.Status to already be "finished" -- the Elo application loop that
// calls this runs before the event is marked finished.
func PlacementEloBonus(t *Event) map[string]float64 {
	switch t.Format {
	case "elimination", "groups_elimination":
		return knockoutPlacementBonus(t)
	case "round_robin":
		return roundRobinPlacementBonus(t)
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

func knockoutPlacementBonus(t *Event) map[string]float64 {
	bonus := make(map[string]float64)

	var finalMatch *Match
	for i := range t.Matches {
		m := &t.Matches[i]
		if m.Stage == "final" && m.Status == "finished" && m.TeamMatchID == nil && m.WinnerTeam != "" {
			finalMatch = m
			break
		}
	}
	if finalMatch != nil {
		winners, losers := finalMatch.TeamA, finalMatch.TeamB
		if finalMatch.WinnerTeam == "B" {
			winners, losers = finalMatch.TeamB, finalMatch.TeamA
		}
		for _, p := range resolveTeamSlot(t, winners) {
			bonus[p.ID] = FirstPlaceEloBonus
		}
		for _, p := range resolveTeamSlot(t, losers) {
			bonus[p.ID] = SecondPlaceEloBonus
		}
	}

	for i := range t.Matches {
		m := &t.Matches[i]
		if m.Stage != "semifinal" || m.Status != "finished" || m.TeamMatchID != nil || m.WinnerTeam == "" {
			continue
		}
		loser := m.TeamA
		if m.WinnerTeam == "A" {
			loser = m.TeamB
		}
		for _, p := range resolveTeamSlot(t, loser) {
			bonus[p.ID] = ThirdPlaceEloBonus
		}
	}

	return bonus
}

func roundRobinPlacementBonus(t *Event) map[string]float64 {
	var participants []*player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		participants = make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			participants[i] = &player.Player{ID: team.ID}
		}
	} else {
		participants = t.Participants
	}
	if len(participants) == 0 {
		return nil
	}

	standings := BuildStandings(participants, t.Matches)
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
	assign(0, FirstPlaceEloBonus)
	assign(1, SecondPlaceEloBonus)
	assign(2, ThirdPlaceEloBonus)

	return bonus
}
