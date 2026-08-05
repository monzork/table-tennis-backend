package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func newTestPlayerWithCountry(t *testing.T, first, last, country string) *player.Player {
	t.Helper()
	p, err := player.NewPlayer(uuid.NewString(), first, last, time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), "M", country, "Managua", "")
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	return p
}

func TestDashboardRepository_GetStats(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	eventRepo := bunRepo.NewEventRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db, eventRepo)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	dashRepo := bunRepo.NewDashboardRepository(db)

	p1 := newTestPlayerWithCountry(t, "A", "One", "NIC")
	p2 := newTestPlayerWithCountry(t, "B", "Two", "NIC")
	if err := playerRepo.Save(ctx, p1); err != nil {
		t.Fatalf("save p1: %v", err)
	}
	if err := playerRepo.Save(ctx, p2); err != nil {
		t.Fatalf("save p2: %v", err)
	}

	tr, err := tournament.NewTournament(uuid.NewString(), "Cup", nil, true, time.Now(), time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	if err := tournamentRepo.Save(ctx, tr); err != nil {
		t.Fatalf("save tournament: %v", err)
	}

	ev := newBareEvent(t, "Event", []*player.Player{p1, p2})
	if err := eventRepo.Save(ctx, ev); err != nil {
		t.Fatalf("save event: %v", err)
	}

	m := event.Match{
		ID: uuid.NewString(), EventID: ev.ID, MatchType: "singles", Status: "finished", WinnerTeam: "A",
		TeamA: []*player.Player{p1}, TeamB: []*player.Player{p2},
		Sets: []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}},
	}
	if err := matchRepo.Save(ctx, &m); err != nil {
		t.Fatalf("save match: %v", err)
	}

	stats, err := dashRepo.GetStats(ctx)
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.TotalPlayers != 2 {
		t.Errorf("expected 2 players, got %d", stats.TotalPlayers)
	}
	if stats.TotalTournaments != 1 {
		t.Errorf("expected 1 tournament, got %d", stats.TotalTournaments)
	}
	if stats.TotalMatches != 1 {
		t.Errorf("expected 1 finished match, got %d", stats.TotalMatches)
	}
}

func TestDashboardRepository_GetPlayersByCountry(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	dashRepo := bunRepo.NewDashboardRepository(db)

	for _, c := range []string{"NIC", "NIC", "USA"} {
		p := newTestPlayerWithCountry(t, "P", "Layer", c)
		if err := playerRepo.Save(ctx, p); err != nil {
			t.Fatalf("save player: %v", err)
		}
	}

	items, err := dashRepo.GetPlayersByCountry(ctx)
	if err != nil {
		t.Fatalf("GetPlayersByCountry: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 country buckets, got %d: %+v", len(items), items)
	}
	if items[0].Label != "NIC" || items[0].Value != 2 {
		t.Errorf("expected NIC=2 first (highest count), got %+v", items[0])
	}
}

func TestDashboardRepository_GetEventsByFormat(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	eventRepo := bunRepo.NewEventRepository(db)

	singles1 := newBareEvent(t, "S1", nil)
	singles2 := newBareEvent(t, "S2", nil)
	doubles1, err := event.NewEvent(uuid.NewString(), "D1", "doubles", "elimination", "open", time.Now(), time.Now().Add(time.Hour), nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	for _, e := range []*event.Event{singles1, singles2, doubles1} {
		if err := eventRepo.Save(ctx, e); err != nil {
			t.Fatalf("save event: %v", err)
		}
	}

	dashRepo := bunRepo.NewDashboardRepository(db)
	items, err := dashRepo.GetEventsByFormat(ctx)
	if err != nil {
		t.Fatalf("GetEventsByFormat: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 format buckets, got %d: %+v", len(items), items)
	}
	if items[0].Label != "singles" || items[0].Value != 2 {
		t.Errorf("expected singles=2 first, got %+v", items[0])
	}
}

func TestDashboardRepository_GetTournamentActivityByMonth(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	eventRepo := bunRepo.NewEventRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db, eventRepo)

	jan := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2026, 2, 3, 0, 0, 0, 0, time.UTC)
	for _, start := range []time.Time{jan, jan, feb} {
		tr, err := tournament.NewTournament(uuid.NewString(), "Cup", nil, true, start, start.Add(time.Hour))
		if err != nil {
			t.Fatalf("NewTournament: %v", err)
		}
		if err := tournamentRepo.Save(ctx, tr); err != nil {
			t.Fatalf("save tournament: %v", err)
		}
	}

	dashRepo := bunRepo.NewDashboardRepository(db)
	items, err := dashRepo.GetTournamentActivityByMonth(ctx)
	if err != nil {
		t.Fatalf("GetTournamentActivityByMonth: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 month buckets, got %d: %+v", len(items), items)
	}
	if items[0].Label != "Jan 2026" || items[0].Value != 2 {
		t.Errorf("expected Jan 2026=2 first (chronological), got %+v", items[0])
	}
	if items[1].Label != "Feb 2026" || items[1].Value != 1 {
		t.Errorf("expected Feb 2026=1 second, got %+v", items[1])
	}
}

func TestDashboardRepository_GetTopEloGainers(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	eventRepo := bunRepo.NewEventRepository(db)

	gainer := savePlayer(t, playerRepo, "Gainer", "Player", "M")
	loser := savePlayer(t, playerRepo, "Loser", "Player", "M")

	ev := newBareEvent(t, "Event", []*player.Player{gainer, loser})
	if err := eventRepo.Save(ctx, ev); err != nil {
		t.Fatalf("save event: %v", err)
	}

	if err := eventRepo.AddParticipant(ctx, ev.ID, gainer.ID, 1000, 1000); err != nil {
		t.Fatalf("AddParticipant gainer: %v", err)
	}
	if err := eventRepo.UpdateParticipantElo(ctx, ev.ID, gainer.ID, 1100, 1000); err != nil {
		t.Fatalf("UpdateParticipantElo gainer: %v", err)
	}
	if err := eventRepo.AddParticipant(ctx, ev.ID, loser.ID, 1000, 1000); err != nil {
		t.Fatalf("AddParticipant loser: %v", err)
	}
	if err := eventRepo.UpdateParticipantElo(ctx, ev.ID, loser.ID, 950, 1000); err != nil {
		t.Fatalf("UpdateParticipantElo loser: %v", err)
	}

	dashRepo := bunRepo.NewDashboardRepository(db)
	items, err := dashRepo.GetTopEloGainers(ctx, 10)
	if err != nil {
		t.Fatalf("GetTopEloGainers: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected only the positive gainer, got %d: %+v", len(items), items)
	}
	if items[0].Label != "Gainer Player" || items[0].Value != 100 {
		t.Errorf("expected Gainer Player +100, got %+v", items[0])
	}
}
