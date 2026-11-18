package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// Board cards for doubles/teams events identify sides by team ID, while the
// stored match references the teams' players. The lookup must bridge that,
// otherwise every save from a card inserts a duplicate match.
func TestMatchRepository_LookupByTeamIDsFindsPlayerKeyedMatch(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	ctx := context.Background()

	e, err := event.NewEvent(uuid.NewString(), "Women's Doubles", "doubles", "round_robin", "women", "", time.Now(), time.Now().Add(time.Hour), nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	mkTeam := func(name string) (*event.Team, []*player.Player) {
		team, err := event.NewTeam(uuid.NewString(), e.ID, name)
		if err != nil {
			t.Fatalf("NewTeam: %v", err)
		}
		if err := eventRepo.SaveTeam(ctx, team); err != nil {
			t.Fatalf("SaveTeam: %v", err)
		}
		ps := []*player.Player{savePlayer(t, playerRepo, name, "One", "F"), savePlayer(t, playerRepo, name, "Two", "F")}
		for _, p := range ps {
			if err := eventRepo.AddPlayerToTeam(ctx, team.ID, p.ID); err != nil {
				t.Fatalf("AddPlayerToTeam: %v", err)
			}
		}
		return team, ps
	}
	teamA, psA := mkTeam("UNAN")
	teamB, psB := mkTeam("UNI")

	m := &event.Match{ID: uuid.NewString(), EventID: e.ID, MatchType: "doubles", Stage: "group", Status: "finished", TeamA: psA, TeamB: psB}
	if err := matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save match: %v", err)
	}

	for _, pair := range [][2]string{{teamA.ID, teamB.ID}, {teamB.ID, teamA.ID}} {
		got, err := matchRepo.GetMatchByParticipants(ctx, e.ID, pair[0], pair[1], "group")
		if err != nil || got == nil || got.ID != m.ID {
			t.Fatalf("GetMatchByParticipants(%v) = %v, %v; want %s", pair, got, err, m.ID)
		}
		id, err := matchRepo.FindOrCreateMatch(ctx, e.ID, pair[0], pair[1], "group", "doubles")
		if err != nil || id != m.ID {
			t.Fatalf("FindOrCreateMatch(%v) = %q, %v; want existing %s", pair, id, err, m.ID)
		}
	}

	// Plain player IDs still resolve as before.
	if got, err := matchRepo.GetMatchByParticipants(ctx, e.ID, psB[0].ID, psA[0].ID, "group"); err != nil || got.ID != m.ID {
		t.Fatalf("lookup by player IDs = %v, %v", got, err)
	}
}

// A player on a team in two events of the same tournament (e.g. Women's
// Doubles and Mixed Teams) must have each event's matches attributed to that
// event's team. A tournament-wide player→team map let the last event win, so
// the board couldn't match played pairings and showed them as pending.
func TestEventRepository_GetByTournamentIDDeep_MapsMatchesToOwnEventTeam(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	ctx := context.Background()

	tournamentID := uuid.New()
	tIDStr := tournamentID.String()
	shared := savePlayer(t, playerRepo, "Shared", "Player", "F")

	mkEvent := func(name, typ string) (*event.Event, *event.Team, *event.Team) {
		e, err := event.NewEvent(uuid.NewString(), name, typ, "round_robin", "open", "", time.Now(), time.Now().Add(time.Hour), nil, 2, nil, false)
		if err != nil {
			t.Fatalf("NewEvent: %v", err)
		}
		e.TournamentID = &tIDStr
		if err := eventRepo.Save(ctx, e); err != nil {
			t.Fatalf("Save event: %v", err)
		}
		var teams []*event.Team
		for i, n := range []string{"UNAN", "UNI"} {
			team, _ := event.NewTeam(uuid.NewString(), e.ID, n)
			if err := eventRepo.SaveTeam(ctx, team); err != nil {
				t.Fatalf("SaveTeam: %v", err)
			}
			members := []*player.Player{savePlayer(t, playerRepo, n, name, "F")}
			if i == 0 {
				members = []*player.Player{shared, members[0]}
			}
			for _, p := range members {
				if err := eventRepo.AddPlayerToTeam(ctx, team.ID, p.ID); err != nil {
					t.Fatalf("AddPlayerToTeam: %v", err)
				}
			}
			teams = append(teams, team)
		}
		return e, teams[0], teams[1]
	}

	doubles, dUNAN, dUNI := mkEvent("Women's Doubles", "doubles")
	_, _, _ = mkEvent("Mixed Teams", "teams") // loaded after: used to overwrite the shared player's team

	full, err := eventRepo.GetByID(ctx, doubles.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	var uniPlayers []*player.Player
	for _, tm := range full.Teams {
		if tm.ID == dUNI.ID {
			uniPlayers = tm.Players
		}
	}
	m := &event.Match{ID: uuid.NewString(), EventID: doubles.ID, MatchType: "doubles", Stage: "group", Status: "finished", WinnerTeam: "A",
		TeamA: []*player.Player{shared}, TeamB: uniPlayers}
	if err := matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save match: %v", err)
	}

	events, err := eventRepo.GetByTournamentID(ctx, tournamentID, true)
	if err != nil {
		t.Fatalf("GetByTournamentID: %v", err)
	}
	for _, e := range events {
		if e.ID != doubles.ID {
			continue
		}
		if len(e.Matches) != 1 {
			t.Fatalf("expected 1 match, got %d", len(e.Matches))
		}
		if a, b := e.Matches[0].TeamA[0].ID, e.Matches[0].TeamB[0].ID; a != dUNAN.ID || b != dUNI.ID {
			t.Fatalf("match mapped to teams %s vs %s, want this event's %s vs %s", a, b, dUNAN.ID, dUNI.ID)
		}
		return
	}
	t.Fatal("doubles event not loaded")
}
