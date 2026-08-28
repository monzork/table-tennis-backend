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

func newBareEvent(t *testing.T, name string, participants []*player.Player) *event.Event {
	t.Helper()
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	e, err := event.NewEvent(uuid.NewString(), name, "singles", "elimination", "open", start, end, nil, 2, participants, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	return e
}

func savePlayer(t *testing.T, repo *bunRepo.PlayerRepository, first, last, gender string) *player.Player {
	t.Helper()
	ctx := context.Background()
	p := newTestPlayer(t, first, last, gender)
	if err := repo.Save(ctx, p); err != nil {
		t.Fatalf("Save player: %v", err)
	}
	return p
}

func TestEventRepository_SaveAndGetByID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "P", "One", "M")
	p2 := savePlayer(t, playerRepo, "P", "Two", "F")

	e := newBareEvent(t, "Club Championship", []*player.Player{p1, p2})
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Club Championship" || got.Type != "singles" {
		t.Fatalf("unexpected event: %+v", got)
	}
	if len(got.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(got.Participants))
	}
	if len(got.StageRules) == 0 {
		t.Fatalf("expected default stage rules to be persisted")
	}
}

func TestEventRepository_SaveAndGetByID_UseGenderDivisions(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "P", "One", "M")

	e := newBareEvent(t, "Gendered Event", []*player.Player{p1})
	e.UseGenderDivisions = true
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if !got.UseGenderDivisions {
		t.Fatalf("expected UseGenderDivisions to round-trip as true, got false")
	}
}

func TestEventRepository_GetByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetByID(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing event, got nil")
	}
}

func TestEventRepository_GetByID_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetByID(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestEventRepository_GetByIDLite(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "Lite", "One", "M")
	e := newBareEvent(t, "Lite Event", []*player.Player{p1})

	team, err := event.NewTeam(uuid.NewString(), e.ID, "Team A")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	team.Players = append(team.Players, p1)
	e.Teams = append(e.Teams, team)

	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := eventRepo.GetByIDLite(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByIDLite: %v", err)
	}
	if len(got.Participants) != 1 {
		t.Fatalf("expected 1 participant, got %d", len(got.Participants))
	}
	if len(got.Teams) != 1 || len(got.Teams[0].Players) != 1 {
		t.Fatalf("expected 1 team with 1 player, got %+v", got.Teams)
	}
}

func TestEventRepository_GetByIDLite_NotFound(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetByIDLite(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing event, got nil")
	}
}

func TestEventRepository_GetAll(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "All", "One", "M")
	e1 := newBareEvent(t, "Event One", []*player.Player{p1})
	e2 := newBareEvent(t, "Event Two", nil)

	if err := eventRepo.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1: %v", err)
	}
	if err := eventRepo.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2: %v", err)
	}

	all, err := eventRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 events, got %d", len(all))
	}
	for _, e := range all {
		if e.ID == e1.ID && len(e.Participants) != 1 {
			t.Fatalf("expected event one to have 1 participant placeholder, got %d", len(e.Participants))
		}
	}
}

func TestEventRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "Upd", "One", "M")
	p2 := savePlayer(t, playerRepo, "Upd", "Two", "F")

	e := newBareEvent(t, "Original Name", []*player.Player{p1})
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	e.Name = "Renamed Event"
	e.Participants = []*player.Player{p1, p2}
	e.NumTables = 5
	if err := eventRepo.Update(ctx, e); err != nil {
		t.Fatalf("Update: %v", err)
	}

	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Renamed Event" || got.NumTables != 5 {
		t.Fatalf("unexpected updated event: %+v", got)
	}
	if len(got.Participants) != 2 {
		t.Fatalf("expected 2 participants after update, got %d", len(got.Participants))
	}
}

func TestEventRepository_Update_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	e := &event.Event{ID: "bad-id"}
	if err := eventRepo.Update(ctx, e); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestEventRepository_UpdateGroups(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "Grp", "One", "M")
	e := newBareEvent(t, "Groups Event", []*player.Player{p1})
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	e.Groups = []event.Group{
		{ID: uuid.NewString(), EventID: e.ID, Name: "Group A", Players: []*player.Player{p1}},
	}
	if err := eventRepo.UpdateGroups(ctx, e); err != nil {
		t.Fatalf("UpdateGroups: %v", err)
	}

	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Groups) != 1 || got.Groups[0].Name != "Group A" {
		t.Fatalf("expected 1 group, got %+v", got.Groups)
	}
}

func TestEventRepository_UpdateGroups_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if err := eventRepo.UpdateGroups(ctx, &event.Event{ID: "bad-id"}); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestEventRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "Del", "One", "M")
	e := newBareEvent(t, "To Delete", []*player.Player{p1})
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := eventRepo.Delete(ctx, e.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := eventRepo.GetByID(ctx, e.ID); err == nil {
		t.Fatal("expected error after delete, got nil")
	}
}

func TestEventRepository_Delete_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if err := eventRepo.Delete(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestEventRepository_GetByTournamentID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	tournamentID := uuid.New()
	tIDStr := tournamentID.String()

	e1 := newBareEvent(t, "Sub Event 1", nil)
	e1.TournamentID = &tIDStr
	e2 := newBareEvent(t, "Sub Event 2", nil)
	e2.TournamentID = &tIDStr

	if err := eventRepo.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1: %v", err)
	}
	if err := eventRepo.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2: %v", err)
	}

	got, err := eventRepo.GetByTournamentID(ctx, tournamentID, false)
	if err != nil {
		t.Fatalf("GetByTournamentID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sub-events, got %d", len(got))
	}

	deep, err := eventRepo.GetByTournamentID(ctx, tournamentID, true)
	if err != nil {
		t.Fatalf("GetByTournamentID (deep): %v", err)
	}
	if len(deep) != 2 {
		t.Fatalf("expected 2 sub-events (deep), got %d", len(deep))
	}

	none, err := eventRepo.GetByTournamentID(ctx, uuid.New(), false)
	if err != nil {
		t.Fatalf("GetByTournamentID (none): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected 0 sub-events for unrelated tournament id, got %d", len(none))
	}
}

func TestEventRepository_GetByParticipantID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	e1 := newBareEvent(t, "Event With Player", nil)
	if err := eventRepo.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1: %v", err)
	}
	e2 := newBareEvent(t, "Event Without Player", nil)
	if err := eventRepo.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2: %v", err)
	}

	p1 := savePlayer(t, playerRepo, "Part", "Icipant", "M")
	if err := eventRepo.AddParticipant(ctx, e1.ID, p1.ID, 1000, 1000); err != nil {
		t.Fatalf("AddParticipant: %v", err)
	}

	got, err := eventRepo.GetByParticipantID(ctx, p1.ID)
	if err != nil {
		t.Fatalf("GetByParticipantID: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 event for participant, got %d", len(got))
	}
	if got[0].ID != e1.ID {
		t.Fatalf("expected event %s, got %s", e1.ID, got[0].ID)
	}

	p2 := savePlayer(t, playerRepo, "No", "Events", "F")
	none, err := eventRepo.GetByParticipantID(ctx, p2.ID)
	if err != nil {
		t.Fatalf("GetByParticipantID (none): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected 0 events for player with no participation, got %d", len(none))
	}
}

// TestEventRepository_GetByParticipantID_HydratesMatchPlayerElo guards
// against a real regression: match.TeamA/TeamB players hydrated by this path
// were only ever given ID/FirstName/LastName, never Elo -- silently
// defaulting to 0 for anything (like the player-stats page's Elo preview)
// that reads TeamA[0].SinglesElo/DoublesElo off a match loaded this way.
func TestEventRepository_GetByParticipantID_HydratesMatchPlayerElo(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	ctx := context.Background()

	p1 := savePlayer(t, playerRepo, "Kevin", "Munoz", "M")
	p1.SinglesElo = 2163
	p1.DoublesElo = 1300
	if err := playerRepo.Save(ctx, p1); err != nil {
		t.Fatalf("Save p1: %v", err)
	}
	p2 := savePlayer(t, playerRepo, "Nelson", "Reyes", "M")
	p2.SinglesElo = 1907
	if err := playerRepo.Save(ctx, p2); err != nil {
		t.Fatalf("Save p2: %v", err)
	}

	e1 := newBareEvent(t, "Elo Hydration Event", []*player.Player{p1, p2})
	if err := eventRepo.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1: %v", err)
	}

	m := &event.Match{
		ID:        uuid.NewString(),
		EventID:   e1.ID,
		MatchType: "singles",
		TeamA:     []*player.Player{p1},
		TeamB:     []*player.Player{p2},
		Status:    "finished",
	}
	winner := "B"
	if err := matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save match: %v", err)
	}
	if err := matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: m.ID, WinnerTeam: winner}); err != nil {
		t.Fatalf("FinishMatch: %v", err)
	}

	got, err := eventRepo.GetByParticipantID(ctx, p1.ID)
	if err != nil {
		t.Fatalf("GetByParticipantID: %v", err)
	}
	if len(got) != 1 || len(got[0].Matches) != 1 {
		t.Fatalf("expected 1 event with 1 match, got %+v", got)
	}
	hydrated := got[0].Matches[0]
	if len(hydrated.TeamA) != 1 || hydrated.TeamA[0].SinglesElo != 2163 {
		t.Errorf("expected TeamA[0].SinglesElo 2163, got %+v", hydrated.TeamA)
	}
	if len(hydrated.TeamB) != 1 || hydrated.TeamB[0].SinglesElo != 1907 {
		t.Errorf("expected TeamB[0].SinglesElo 1907, got %+v", hydrated.TeamB)
	}
}

func TestEventRepository_TeamLifecycle(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	e, err := event.NewEvent(uuid.NewString(), "Doubles Cup", "doubles", "elimination", "open", time.Now(), time.Now().Add(time.Hour), nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	p1 := savePlayer(t, playerRepo, "Team", "One", "M")
	p2 := savePlayer(t, playerRepo, "Team", "Two", "M")

	team, err := event.NewTeam(uuid.NewString(), e.ID, "Dream Team")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	if err := eventRepo.SaveTeam(ctx, team); err != nil {
		t.Fatalf("SaveTeam: %v", err)
	}

	if err := eventRepo.AddPlayerToTeam(ctx, team.ID, p1.ID); err != nil {
		t.Fatalf("AddPlayerToTeam p1: %v", err)
	}
	if err := eventRepo.AddPlayerToTeam(ctx, team.ID, p2.ID); err != nil {
		t.Fatalf("AddPlayerToTeam p2: %v", err)
	}

	// Adding p1 again should fail: already registered in this event's team.
	if err := eventRepo.AddPlayerToTeam(ctx, team.ID, p1.ID); err == nil {
		t.Fatal("expected error adding player already registered in a team")
	}

	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Teams) != 1 || len(got.Teams[0].Players) != 2 {
		t.Fatalf("expected 1 team with 2 players, got %+v", got.Teams)
	}

	if err := eventRepo.RemovePlayerFromTeam(ctx, team.ID, p2.ID); err != nil {
		t.Fatalf("RemovePlayerFromTeam: %v", err)
	}
	got, err = eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID after remove: %v", err)
	}
	if len(got.Teams[0].Players) != 1 {
		t.Fatalf("expected 1 player left on team, got %d", len(got.Teams[0].Players))
	}

	if err := eventRepo.DeleteTeam(ctx, team.ID); err != nil {
		t.Fatalf("DeleteTeam: %v", err)
	}
	got, err = eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID after delete team: %v", err)
	}
	if len(got.Teams) != 0 {
		t.Fatalf("expected 0 teams after delete, got %d", len(got.Teams))
	}
}

func TestEventRepository_AddPlayerToTeam_GenderRestrictions(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	e, err := event.NewEvent(uuid.NewString(), "Women's Doubles", "doubles", "elimination", "women", time.Now(), time.Now().Add(time.Hour), nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	team, err := event.NewTeam(uuid.NewString(), e.ID, "Team")
	if err != nil {
		t.Fatalf("NewTeam: %v", err)
	}
	if err := eventRepo.SaveTeam(ctx, team); err != nil {
		t.Fatalf("SaveTeam: %v", err)
	}

	male := savePlayer(t, playerRepo, "Male", "Player", "M")
	if err := eventRepo.AddPlayerToTeam(ctx, team.ID, male.ID); err == nil {
		t.Fatal("expected error adding male player to women's event")
	}
}

func TestEventRepository_ParticipantLifecycle(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	e := newBareEvent(t, "Participant Event", nil)
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	p1 := savePlayer(t, playerRepo, "Part", "One", "M")
	if err := eventRepo.AddParticipant(ctx, e.ID, p1.ID, 1000, 1000); err != nil {
		t.Fatalf("AddParticipant: %v", err)
	}

	snapshots, err := eventRepo.GetParticipantSnapshots(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetParticipantSnapshots: %v", err)
	}
	if len(snapshots) != 1 || snapshots[0].PlayerID != p1.ID {
		t.Fatalf("unexpected snapshots: %+v", snapshots)
	}

	pin, err := eventRepo.GetParticipantPIN(ctx, e.ID, p1.ID)
	if err != nil {
		t.Fatalf("GetParticipantPIN: %v", err)
	}
	if pin == "" {
		t.Fatal("expected non-empty PIN")
	}

	pins, err := eventRepo.GetParticipantPINsByTournament(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetParticipantPINsByTournament: %v", err)
	}
	if pins[p1.ID] != pin {
		t.Fatalf("expected pin map to contain %s -> %s, got %+v", p1.ID, pin, pins)
	}

	foundID, err := eventRepo.GetParticipantOrOfficialByPIN(ctx, e.ID, pin)
	if err != nil {
		t.Fatalf("GetParticipantOrOfficialByPIN: %v", err)
	}
	if foundID != p1.ID {
		t.Fatalf("expected to resolve participant by PIN, got %q", foundID)
	}

	if err := eventRepo.UpdateParticipantElo(ctx, e.ID, p1.ID, 1050, 1010); err != nil {
		t.Fatalf("UpdateParticipantElo: %v", err)
	}
	if err := eventRepo.UpdateParticipantEloBefore(ctx, e.ID, p1.ID, 900, 950); err != nil {
		t.Fatalf("UpdateParticipantEloBefore: %v", err)
	}
	snapshots, err = eventRepo.GetParticipantSnapshots(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetParticipantSnapshots (after update): %v", err)
	}
	if *snapshots[0].EloAfterSingles != 1050 || *snapshots[0].EloBeforeSingles != 900 {
		t.Fatalf("unexpected snapshot after elo updates: %+v", snapshots[0])
	}

	if err := eventRepo.UpdateParticipantsElo(ctx, e.ID, []*player.Player{{ID: p1.ID, SinglesElo: 1100, DoublesElo: 1080}}); err != nil {
		t.Fatalf("UpdateParticipantsElo: %v", err)
	}
	snapshots, err = eventRepo.GetParticipantSnapshots(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetParticipantSnapshots (after bulk update): %v", err)
	}
	if *snapshots[0].EloAfterSingles != 1100 {
		t.Fatalf("expected bulk elo update to apply, got %+v", snapshots[0])
	}

	if err := eventRepo.RemoveParticipant(ctx, e.ID, p1.ID); err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	snapshots, err = eventRepo.GetParticipantSnapshots(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetParticipantSnapshots (after remove): %v", err)
	}
	if len(snapshots) != 0 {
		t.Fatalf("expected 0 participants after remove, got %d", len(snapshots))
	}
}

func TestEventRepository_UpdateParticipantsElo_Empty(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if err := eventRepo.UpdateParticipantsElo(ctx, uuid.NewString(), nil); err != nil {
		t.Fatalf("expected no-op for empty players, got %v", err)
	}
}

func TestEventRepository_GetParticipantOrOfficialByPIN_EmptyPin(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetParticipantOrOfficialByPIN(ctx, uuid.NewString(), ""); err == nil {
		t.Fatal("expected error for empty pin, got nil")
	}
}

func TestEventRepository_GetParticipantOrOfficialByPIN_NotFound(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetParticipantOrOfficialByPIN(ctx, uuid.NewString(), "9999"); err == nil {
		t.Fatal("expected error for no matching pin, got nil")
	}
}

func TestEventRepository_OfficialLifecycle(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	e := newBareEvent(t, "Officials Event", nil)
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save: %v", err)
	}

	ref := savePlayer(t, playerRepo, "Ref", "Eree", "M")
	if err := eventRepo.AddOfficial(ctx, e.ID, ref.ID, "1234"); err != nil {
		t.Fatalf("AddOfficial: %v", err)
	}

	officials, err := eventRepo.GetOfficials(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetOfficials: %v", err)
	}
	if len(officials) != 1 || officials[0].PlayerID != ref.ID || officials[0].Pin != "1234" {
		t.Fatalf("unexpected officials: %+v", officials)
	}

	foundID, err := eventRepo.GetParticipantOrOfficialByPIN(ctx, e.ID, "1234")
	if err != nil {
		t.Fatalf("GetParticipantOrOfficialByPIN: %v", err)
	}
	if foundID != ref.ID {
		t.Fatalf("expected to resolve official by PIN, got %q", foundID)
	}

	// AddOfficial again with a new pin should upsert.
	if err := eventRepo.AddOfficial(ctx, e.ID, ref.ID, "5678"); err != nil {
		t.Fatalf("AddOfficial (upsert): %v", err)
	}
	officials, err = eventRepo.GetOfficials(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetOfficials (after upsert): %v", err)
	}
	if len(officials) != 1 || officials[0].Pin != "5678" {
		t.Fatalf("expected upserted pin, got %+v", officials)
	}

	if err := eventRepo.RemoveOfficial(ctx, e.ID, ref.ID); err != nil {
		t.Fatalf("RemoveOfficial: %v", err)
	}
	officials, err = eventRepo.GetOfficials(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetOfficials (after remove): %v", err)
	}
	if len(officials) != 0 {
		t.Fatalf("expected 0 officials after remove, got %d", len(officials))
	}
}

func TestEventRepository_OfficialLifecycle_SharedAcrossTournament(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	ctx := context.Background()

	tournamentID := uuid.NewString()

	e1 := newBareEvent(t, "Men's Singles", nil)
	e1.TournamentID = &tournamentID
	if err := eventRepo.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1: %v", err)
	}

	e2 := newBareEvent(t, "Women's Singles", nil)
	e2.TournamentID = &tournamentID
	if err := eventRepo.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2: %v", err)
	}

	ref := savePlayer(t, playerRepo, "Ref", "Eree", "M")
	if err := eventRepo.AddOfficial(ctx, e1.ID, ref.ID, "1234"); err != nil {
		t.Fatalf("AddOfficial: %v", err)
	}

	// The official was added via e1 but should be visible from sibling event e2,
	// since both share the same parent tournament.
	officials, err := eventRepo.GetOfficials(ctx, e2.ID)
	if err != nil {
		t.Fatalf("GetOfficials: %v", err)
	}
	if len(officials) != 1 || officials[0].PlayerID != ref.ID {
		t.Fatalf("expected official shared across sibling events, got %+v", officials)
	}

	foundID, err := eventRepo.GetParticipantOrOfficialByPIN(ctx, e2.ID, "1234")
	if err != nil {
		t.Fatalf("GetParticipantOrOfficialByPIN: %v", err)
	}
	if foundID != ref.ID {
		t.Fatalf("expected to resolve official by PIN via sibling event, got %q", foundID)
	}

	if err := eventRepo.RemoveOfficial(ctx, e2.ID, ref.ID); err != nil {
		t.Fatalf("RemoveOfficial: %v", err)
	}
	officials, err = eventRepo.GetOfficials(ctx, e1.ID)
	if err != nil {
		t.Fatalf("GetOfficials (after remove via sibling): %v", err)
	}
	if len(officials) != 0 {
		t.Fatalf("expected removal via sibling event to clear official for both, got %d", len(officials))
	}
}

func TestEventRepository_GetTournamentNumTables(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	tr := newTestTournament(t, "Tables Tournament")
	tr.NumTables = 12
	if err := tournamentRepo.Save(ctx, tr); err != nil {
		t.Fatalf("Save tournament: %v", err)
	}

	n, err := eventRepo.GetTournamentNumTables(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetTournamentNumTables: %v", err)
	}
	if n != 12 {
		t.Fatalf("expected 12 tables, got %d", n)
	}
}

func TestEventRepository_GetTournamentNumTables_InvalidID(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetTournamentNumTables(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestEventRepository_GetTournamentNumTables_NotFound(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if _, err := eventRepo.GetTournamentNumTables(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing tournament, got nil")
	}
}

func TestEventRepository_UpdateEventIDBulk(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	e1 := newBareEvent(t, "Bulk 1", nil)
	e2 := newBareEvent(t, "Bulk 2", nil)
	if err := eventRepo.Save(ctx, e1); err != nil {
		t.Fatalf("Save e1: %v", err)
	}
	if err := eventRepo.Save(ctx, e2); err != nil {
		t.Fatalf("Save e2: %v", err)
	}

	tournamentID := uuid.NewString()
	if err := eventRepo.UpdateEventIDBulk(ctx, []string{e1.ID, e2.ID}, tournamentID); err != nil {
		t.Fatalf("UpdateEventIDBulk: %v", err)
	}

	tUUID, err := uuid.Parse(tournamentID)
	if err != nil {
		t.Fatalf("parse tournamentID: %v", err)
	}
	got, err := eventRepo.GetByTournamentID(ctx, tUUID, false)
	if err != nil {
		t.Fatalf("GetByTournamentID: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events linked to tournament, got %d", len(got))
	}
}

func TestEventRepository_UpdateEventIDBulk_EmptyAndInvalid(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	if err := eventRepo.UpdateEventIDBulk(ctx, nil, uuid.NewString()); err != nil {
		t.Fatalf("expected no-op for empty ids, got %v", err)
	}
	if err := eventRepo.UpdateEventIDBulk(ctx, []string{"not-a-uuid"}, uuid.NewString()); err != nil {
		t.Fatalf("expected no-op when no valid ids, got %v", err)
	}
	if err := eventRepo.UpdateEventIDBulk(ctx, []string{uuid.NewString()}, "bad-event-id"); err == nil {
		t.Fatal("expected error for invalid eventID, got nil")
	}
}

func TestEventRepository_DB(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	if eventRepo.DB() != db {
		t.Fatal("expected DB() to return the underlying bun.DB")
	}
}
