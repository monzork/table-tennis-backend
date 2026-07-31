package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// TestEventRepository_SaveAndUpdate_WithGroupsAndTeams exercises the branches
// of saveTx/Update that persist Groups and Teams together, and verifies that
// re-registering the same participant on Update preserves their existing PIN
// and Elo-before/after snapshot rather than generating a new one.
func TestEventRepository_SaveAndUpdate_WithGroupsAndTeams(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "Grp", "One", "M")
	p2 := savePlayer(t, playerRepo, "Grp", "Two", "M")

	e := newBareEvent(t, "Full Save Event", []*player.Player{p1, p2})
	e.Groups = []event.Group{
		{ID: uuid.NewString(), EventID: e.ID, Name: "Group A", Players: []*player.Player{p1, p2}},
	}
	team, err := event.NewTeam(uuid.NewString(), e.ID, "Team One")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	team.Players = []*player.Player{p1}
	e.Teams = []*event.Team{team}

	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Groups) != 1 || len(got.Groups[0].Players) != 2 {
		t.Fatalf("expected 1 group with 2 players, got %+v", got.Groups)
	}
	if len(got.Teams) != 1 || len(got.Teams[0].Players) != 1 {
		t.Fatalf("expected 1 team with 1 player, got %+v", got.Teams)
	}

	originalPIN, err := eventRepo.GetParticipantPIN(ctx, e.ID, p1.ID)
	if err != nil {
		t.Fatalf("GetParticipantPIN: %v", err)
	}
	if err := eventRepo.UpdateParticipantElo(ctx, e.ID, p1.ID, 1234, 1235); err != nil {
		t.Fatalf("UpdateParticipantElo: %v", err)
	}

	// Update the event, re-registering the same 2 participants plus a new group/team.
	e.Name = "Renamed Full Save Event"
	e.Groups = []event.Group{
		{ID: uuid.NewString(), EventID: e.ID, Name: "Group B", Players: []*player.Player{p2}},
	}
	team2, err := event.NewTeam(uuid.NewString(), e.ID, "Team Two")
	if err != nil {
		t.Fatalf("NewTeam2: %v", err)
	}
	team2.Players = []*player.Player{p2}
	e.Teams = []*event.Team{team2}

	if err := eventRepo.Update(ctx, e); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err = eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID (after update): %v", err)
	}
	if got.Name != "Renamed Full Save Event" {
		t.Fatalf("expected renamed event, got %q", got.Name)
	}
	if len(got.Groups) != 1 || got.Groups[0].Name != "Group B" {
		t.Fatalf("expected groups replaced, got %+v", got.Groups)
	}
	if len(got.Teams) != 1 || got.Teams[0].Name != "Team Two" {
		t.Fatalf("expected teams replaced, got %+v", got.Teams)
	}

	// The re-registered participant's PIN and prior Elo-after snapshot must be preserved.
	newPIN, err := eventRepo.GetParticipantPIN(ctx, e.ID, p1.ID)
	if err != nil {
		t.Fatalf("GetParticipantPIN (after update): %v", err)
	}
	if newPIN != originalPIN {
		t.Fatalf("expected PIN to be preserved across update, got %q want %q", newPIN, originalPIN)
	}
	snapshots, err := eventRepo.GetParticipantSnapshots(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetParticipantSnapshots: %v", err)
	}
	found := false
	for _, s := range snapshots {
		if s.PlayerID == p1.ID {
			found = true
			if s.EloAfterSingles == nil || *s.EloAfterSingles != 1234 {
				t.Fatalf("expected preserved elo_after_singles snapshot, got %+v", s)
			}
		}
	}
	if !found {
		t.Fatal("expected p1 snapshot to be present after update")
	}
}
