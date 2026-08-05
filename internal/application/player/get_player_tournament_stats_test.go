package player_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"table-tennis-backend/internal/application/player"
	eventDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"
)

// ─── minimal fake event.Repository ─────────────────────────────────────────

type fakeEventRepo struct {
	eventDomain.Repository // embed nil interface: unused methods panic if called
	events                 []*eventDomain.Event
	eventsErr              error
	snapshots              map[string][]eventDomain.ParticipantSnapshot
	snapshotsErr           error
}

func (f *fakeEventRepo) GetByParticipantID(ctx context.Context, playerID string) ([]*eventDomain.Event, error) {
	return f.events, f.eventsErr
}

func (f *fakeEventRepo) GetParticipantSnapshots(ctx context.Context, eventID string) ([]eventDomain.ParticipantSnapshot, error) {
	if f.snapshotsErr != nil {
		return nil, f.snapshotsErr
	}
	return f.snapshots[eventID], nil
}

// ─── minimal fake tournament.Repository ────────────────────────────────────

type fakeTournamentRepo struct {
	tournament.Repository
	byID map[string]*tournament.Tournament
}

func (f *fakeTournamentRepo) GetByID(ctx context.Context, id string) (*tournament.Tournament, error) {
	t, ok := f.byID[id]
	if !ok {
		return nil, errors.New("tournament not found")
	}
	return t, nil
}

func TestGetPlayerTournamentStatsUseCase_Execute(t *testing.T) {
	t.Run("groups events by tournament with stats and elo", func(t *testing.T) {
		playerRepo := newMockPlayerRepo()
		p := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A"}
		playerRepo.players["p1"] = p

		tid := "t1"
		before := int16(1000)
		after := int16(1020)
		ev := &eventDomain.Event{
			ID:           "e1",
			Name:         "Men's Singles",
			Status:       "finished",
			TournamentID: &tid,
			Matches: []eventDomain.Match{
				{
					Status:     "finished",
					WinnerTeam: "A",
					TeamA:      []*playerDomain.Player{{ID: "p1"}},
					TeamB:      []*playerDomain.Player{{ID: "p2"}},
					Sets:       []eventDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 7}},
				},
			},
		}

		eventRepo := &fakeEventRepo{
			events: []*eventDomain.Event{ev},
			snapshots: map[string][]eventDomain.ParticipantSnapshot{
				"e1": {{PlayerID: "p1", EloBeforeSingles: &before, EloAfterSingles: &after}},
			},
		}
		tournamentRepo := &fakeTournamentRepo{byID: map[string]*tournament.Tournament{
			tid: {ID: tid, Name: "Open Cup", StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		}}

		uc := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)

		gotPlayer, history, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if gotPlayer.ID != "p1" {
			t.Fatalf("expected player p1, got %+v", gotPlayer)
		}
		if len(history) != 1 {
			t.Fatalf("expected 1 tournament, got %d", len(history))
		}
		tv := history[0]
		if tv.Tournament.Name != "Open Cup" {
			t.Errorf("expected tournament Open Cup, got %s", tv.Tournament.Name)
		}
		if len(tv.Events) != 1 {
			t.Fatalf("expected 1 event, got %d", len(tv.Events))
		}
		stats := tv.Events[0].Stats
		if stats.Played != 1 || stats.Wins != 1 || stats.Losses != 0 {
			t.Errorf("expected 1 played/1 win, got %+v", stats)
		}
		if stats.SetsWon != 2 || stats.SetsLost != 0 {
			t.Errorf("expected 2-0 sets, got %d-%d", stats.SetsWon, stats.SetsLost)
		}
		if tv.Events[0].EloBeforeSingles == nil || *tv.Events[0].EloBeforeSingles != 1000 {
			t.Errorf("expected elo before 1000, got %v", tv.Events[0].EloBeforeSingles)
		}
		if tv.Events[0].EloAfterSingles == nil || *tv.Events[0].EloAfterSingles != 1020 {
			t.Errorf("expected elo after 1020, got %v", tv.Events[0].EloAfterSingles)
		}
	})

	t.Run("player repo error propagates", func(t *testing.T) {
		playerRepo := newMockPlayerRepo()
		eventRepo := &fakeEventRepo{}
		tournamentRepo := &fakeTournamentRepo{byID: map[string]*tournament.Tournament{}}
		uc := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)

		_, _, err := uc.Execute(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("event repo error propagates", func(t *testing.T) {
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1"}
		eventRepo := &fakeEventRepo{eventsErr: errors.New("boom")}
		tournamentRepo := &fakeTournamentRepo{byID: map[string]*tournament.Tournament{}}
		uc := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)

		_, _, err := uc.Execute(context.Background(), "p1")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("events without a parent tournament are skipped", func(t *testing.T) {
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1"}
		eventRepo := &fakeEventRepo{events: []*eventDomain.Event{{ID: "e1", TournamentID: nil}}}
		tournamentRepo := &fakeTournamentRepo{byID: map[string]*tournament.Tournament{}}
		uc := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)

		_, history, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(history) != 0 {
			t.Errorf("expected 0 tournaments, got %d", len(history))
		}
	})

	t.Run("unfinished events are excluded from the history", func(t *testing.T) {
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1"}
		tid := "t1"
		eventRepo := &fakeEventRepo{events: []*eventDomain.Event{
			{ID: "e1", Status: "in_progress", TournamentID: &tid},
		}}
		tournamentRepo := &fakeTournamentRepo{byID: map[string]*tournament.Tournament{
			tid: {ID: tid, Name: "Cup"},
		}}
		uc := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)

		_, history, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(history) != 0 {
			t.Errorf("expected 0 tournaments for an in-progress event, got %d", len(history))
		}
	})

	t.Run("missing snapshot is tolerated", func(t *testing.T) {
		playerRepo := newMockPlayerRepo()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1"}
		tid := "t1"
		eventRepo := &fakeEventRepo{
			events:       []*eventDomain.Event{{ID: "e1", Status: "finished", TournamentID: &tid}},
			snapshotsErr: errors.New("no snapshot"),
		}
		tournamentRepo := &fakeTournamentRepo{byID: map[string]*tournament.Tournament{
			tid: {ID: tid, Name: "Cup"},
		}}
		uc := player.NewGetPlayerTournamentStatsUseCase(playerRepo, eventRepo, tournamentRepo)

		_, history, err := uc.Execute(context.Background(), "p1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(history) != 1 || history[0].Events[0].EloBeforeSingles != nil {
			t.Errorf("expected tournament present with nil elo, got %+v", history)
		}
	})
}
