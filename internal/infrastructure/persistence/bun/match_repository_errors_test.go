package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"

	"github.com/google/uuid"
)

// Table-driven coverage of MatchRepository.Save's various invalid-UUID error
// paths, plus the WinnerTeam / two-player-team branches.
func TestMatchRepository_Save_InvalidIDVariants(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	base := func() *event.Match {
		return &event.Match{
			ID:           uuid.NewString(),
			TournamentID: f.tournament.ID,
			MatchType:    "singles",
			TeamA:        []*player.Player{f.players[0]},
			TeamB:        []*player.Player{f.players[1]},
			Status:       "scheduled",
			Stage:        "group",
		}
	}

	t.Run("invalid match id", func(t *testing.T) {
		m := base()
		m.ID = "bad-id"
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid tournament id", func(t *testing.T) {
		m := base()
		m.TournamentID = "bad-id"
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid teamA player id", func(t *testing.T) {
		m := base()
		m.TeamA = []*player.Player{{ID: "bad-id"}}
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid teamB player id", func(t *testing.T) {
		m := base()
		m.TeamB = []*player.Player{{ID: "bad-id"}}
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid teamA second player id", func(t *testing.T) {
		m := base()
		m.TeamA = []*player.Player{f.players[0], {ID: "bad-id"}}
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid teamB second player id", func(t *testing.T) {
		m := base()
		m.TeamB = []*player.Player{f.players[1], {ID: "bad-id"}}
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid referee id", func(t *testing.T) {
		m := base()
		bad := "bad-id"
		m.RefereeID = &bad
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("invalid team match id", func(t *testing.T) {
		m := base()
		bad := "bad-id"
		m.TeamMatchID = &bad
		if err := f.matchRepo.Save(ctx, m); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("winner team already set", func(t *testing.T) {
		m := base()
		m.WinnerTeam = "A"
		if err := f.matchRepo.Save(ctx, m); err != nil {
			t.Fatalf("Save: %v", err)
		}
		got, err := f.matchRepo.GetByID(ctx, m.ID)
		if err != nil {
			t.Fatalf("GetByID: %v", err)
		}
		if got.WinnerTeam != "A" {
			t.Fatalf("expected winner team A to persist, got %q", got.WinnerTeam)
		}
	})
}

func TestMatchRepository_FinishMatch_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	err := f.matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: "bad-id", WinnerTeam: "A"})
	if err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestMatchRepository_FinishMatch_AdvancesNextMatch(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	next := f.newMatch(t, "champion")
	if err := f.matchRepo.Save(ctx, next); err != nil {
		t.Fatalf("Save next: %v", err)
	}
	m := f.newMatch(t, "final")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := setNextMatch(ctx, f, m.ID, next.ID, "B"); err != nil {
		t.Fatalf("setNextMatch: %v", err)
	}

	if err := f.matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: m.ID, WinnerTeam: "B"}); err != nil {
		t.Fatalf("FinishMatch: %v", err)
	}

	nextGot, err := f.matchRepo.GetByID(ctx, next.ID)
	if err != nil {
		t.Fatalf("GetByID next: %v", err)
	}
	if nextGot.TeamB[0].ID != f.players[1].ID {
		t.Fatalf("expected winner to advance into slot B, got %+v", nextGot.TeamB)
	}
}

func TestMatchRepository_DeleteByTournament_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if err := f.matchRepo.DeleteByTournament(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestMatchRepository_CountMatches_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.CountUnfinishedMatches(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID (unfinished), got nil")
	}
	if _, err := f.matchRepo.CountFinishedMatches(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID (finished), got nil")
	}
	if _, err := f.matchRepo.HasStartedOrFinishedMatches(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID (started/finished), got nil")
	}
}

func TestMatchRepository_IsTableOccupiedByOtherMatch_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.IsTableOccupiedByOtherMatch(ctx, "bad-id", 1); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestMatchRepository_GetMatchByParticipants_InvalidIDs(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.GetMatchByParticipants(ctx, "bad-id", f.players[0].ID, f.players[1].ID, "group"); err == nil {
		t.Fatal("expected error for invalid tournament id, got nil")
	}
	if _, err := f.matchRepo.GetMatchByParticipants(ctx, f.tournament.ID, "bad-id", f.players[1].ID, "group"); err == nil {
		t.Fatal("expected error for invalid p1 id, got nil")
	}
	if _, err := f.matchRepo.GetMatchByParticipants(ctx, f.tournament.ID, f.players[0].ID, "bad-id", "group"); err == nil {
		t.Fatal("expected error for invalid p2 id, got nil")
	}
}

func TestMatchRepository_FindOrCreateMatch_InvalidIDs(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.FindOrCreateMatch(ctx, "bad-id", f.players[0].ID, f.players[1].ID, "group", "singles"); err == nil {
		t.Fatal("expected error for invalid tournament id, got nil")
	}
	if _, err := f.matchRepo.FindOrCreateMatch(ctx, f.tournament.ID, "bad-id", f.players[1].ID, "group", "singles"); err == nil {
		t.Fatal("expected error for invalid p1 id, got nil")
	}
	if _, err := f.matchRepo.FindOrCreateMatch(ctx, f.tournament.ID, f.players[0].ID, "bad-id", "group", "singles"); err == nil {
		t.Fatal("expected error for invalid p2 id, got nil")
	}
}

func TestMatchRepository_CreateSubMatches_InvalidIDs(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	cmd := event.CreateSubMatchesCommand{
		ParentMatchID: "bad-id",
		TournamentID:  f.tournament.ID,
		TeamAPlayers:  []string{f.players[0].ID},
		TeamBPlayers:  []string{f.players[1].ID},
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err == nil {
		t.Fatal("expected error for invalid parent match id, got nil")
	}

	cmd2 := event.CreateSubMatchesCommand{
		ParentMatchID: uuid.NewString(),
		TournamentID:  "bad-id",
		TeamAPlayers:  []string{f.players[0].ID},
		TeamBPlayers:  []string{f.players[1].ID},
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd2); err == nil {
		t.Fatal("expected error for invalid tournament id, got nil")
	}
}

func TestMatchRepository_UpdateSubMatchSquads_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	cmd := event.UpdateSubMatchSquadsCommand{
		Assignments: []event.SubMatchSquadAssignment{
			{SubMatchID: "bad-id", TeamAPlayer1ID: f.players[0].ID, TeamBPlayer1ID: f.players[1].ID},
		},
	}
	if err := f.matchRepo.UpdateSubMatchSquads(ctx, cmd); err == nil {
		t.Fatal("expected error for invalid sub-match id, got nil")
	}
}

func TestMatchRepository_GetSubMatches_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.GetSubMatches(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid parent match id, got nil")
	}
}
