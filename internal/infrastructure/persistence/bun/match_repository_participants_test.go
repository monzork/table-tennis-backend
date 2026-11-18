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
