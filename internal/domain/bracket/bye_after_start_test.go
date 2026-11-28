package bracket

import (
	"fmt"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

// Regression: once round 1 was created, players with a bye disappeared and the
// remaining matches were relabelled as a later round.
func TestBuildBracketRounds_ByesSurviveStartingTheKnockout(t *testing.T) {
	for n := 3; n <= 32; n++ {
		var players []*player.Player
		for i := range n {
			id := fmt.Sprintf("p%02d", i)
			players = append(players, &player.Player{ID: id, FirstName: id})
		}
		ev := &event.Event{}
		describe := func() (out string) {
			for _, r := range buildBracketRounds(ev, "d", players, 0) {
				out += r.Name + ":"
				for _, m := range r.Matches {
					for _, s := range []*MatchSlot{m.Player1, m.Player2} {
						if s != nil && s.Player != nil {
							out += s.Player.ID
						}
						out += ","
					}
					out += m.Stage + ";"
				}
			}
			return out
		}
		before := describe()
		for _, m := range buildBracketRounds(ev, "d", players, 0)[0].Matches {
			if m.Player1 == nil || m.Player1.Player == nil || m.Player2 == nil || m.Player2.Player == nil {
				continue
			}
			ev.Matches = append(ev.Matches, event.Match{
				ID: m.Player1.Player.ID, DivisionID: "d", Stage: m.Stage, Status: "scheduled",
				TeamA: []*player.Player{m.Player1.Player}, TeamB: []*player.Player{m.Player2.Player},
			})
		}
		if after := describe(); after != before {
			t.Errorf("%d players: bracket changed on start\nbefore %s\nafter  %s", n, before, after)
		}
	}
}

// Regression: the pass count was re-inferred from who had a match, so starting
// a knockout with byes also dropped players from the qualifiers.
func TestBuildBracket_StartedKnockoutKeepsByeQualifiers(t *testing.T) {
	for passCount := 1; passCount <= 2; passCount++ {
		for n := 2; n <= 12; n++ {
			ev := &event.Event{Type: "singles", Format: "groups_elimination", SkipElo: true, GroupPassCount: passCount}
			for gi := range n {
				g := event.Group{ID: fmt.Sprint(gi), Name: fmt.Sprintf("Group %d", gi)}
				for i := range 3 {
					id := fmt.Sprintf("g%02d-%d", gi, i)
					p := &player.Player{ID: id, FirstName: id, SinglesElo: int16(2000 - 100*i - gi)}
					g.Players = append(g.Players, p)
					ev.Participants = append(ev.Participants, p)
				}
				for a := range 3 {
					for b := a + 1; b < 3; b++ {
						ev.Matches = append(ev.Matches, event.Match{
							Stage: "group", Status: "finished", WinnerTeam: "A",
							TeamA: []*player.Player{g.Players[a]}, TeamB: []*player.Player{g.Players[b]},
						})
					}
				}
				ev.Groups = append(ev.Groups, g)
			}
			knockout := func() KnockoutBracket {
				return BuildBracket(ev, nil, nil).Divisions[0].KnockoutBrackets[0]
			}
			before := knockout()
			for _, m := range before.Rounds[0].Matches {
				if m.Player1 == nil || m.Player1.Player == nil || m.Player2 == nil || m.Player2.Player == nil {
					continue
				}
				ev.Matches = append(ev.Matches, event.Match{
					ID: m.Player1.Player.ID, DivisionID: "open", Stage: m.Stage, Status: "scheduled",
					TeamA: []*player.Player{m.Player1.Player}, TeamB: []*player.Player{m.Player2.Player},
				})
			}
			ev.Status = "in_progress"
			after := knockout()
			if len(after.Advancing) != len(before.Advancing) || len(after.Rounds) != len(before.Rounds) {
				t.Errorf("pass=%d groups=%d: qualifiers %d -> %d, rounds %d -> %d", passCount, n,
					len(before.Advancing), len(after.Advancing), len(before.Rounds), len(after.Rounds))
			}
		}
	}
}
