package match_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/application/match"
	eventDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func setupProposalFixtures() (*mockMatchRepo, *mockEventRepo, *mockPlayerRepo, *playerDomain.Player, *playerDomain.Player) {
	matchRepo := newMockMatchRepo()
	eventRepo := newMockEventRepo()
	playerRepo := newMockPlayerRepo()

	accountID := "acc-1"
	p1 := &playerDomain.Player{ID: "p1", FirstName: "Ana", GuardianAccountID: &accountID}
	p2 := &playerDomain.Player{ID: "p2", FirstName: "Beto"}
	playerRepo.players["p1"] = p1
	playerRepo.players["p2"] = p2

	eventRepo.events["e1"] = &eventDomain.Event{ID: "e1", Status: "in_progress"}

	m := &eventDomain.Match{
		ID:      "m1",
		EventID: "e1",
		Stage:   "final",
		Status:  "in_progress",
		TeamA:   []*playerDomain.Player{p1},
		TeamB:   []*playerDomain.Player{p2},
	}
	matchRepo.matches["m1"] = m

	return matchRepo, eventRepo, playerRepo, p1, p2
}

func sampleWinningSets() []eventDomain.MatchSet {
	return []eventDomain.MatchSet{
		{Number: 1, ScoreA: 11, ScoreB: 5},
		{Number: 2, ScoreA: 11, ScoreB: 7},
		{Number: 3, ScoreA: 11, ScoreB: 9},
	}
}

func TestProposeMatchScoreUseCase(t *testing.T) {
	t.Run("happy path stages the proposal", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		uc := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)

		err := uc.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := matchRepo.matches["m1"]
		if m.ProposedByPlayerID == nil || *m.ProposedByPlayerID != "p1" {
			t.Fatalf("expected proposal staged for p1, got %+v", m.ProposedByPlayerID)
		}
	})

	t.Run("rejects when player not owned by account", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		uc := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)

		err := uc.Execute(context.Background(), "acc-WRONG", "m1", "p1", sampleWinningSets())
		if err == nil {
			t.Fatal("expected ownership error")
		}
	})

	t.Run("rejects when player is not a match participant", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		accountID := "acc-2"
		outsider := &playerDomain.Player{ID: "p3", GuardianAccountID: &accountID}
		playerRepo.players["p3"] = outsider

		uc := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		err := uc.Execute(context.Background(), "acc-2", "m1", "p3", sampleWinningSets())
		if err != match.ErrPlayerNotInMatch {
			t.Fatalf("expected ErrPlayerNotInMatch, got %v", err)
		}
	})
}

func newConfirmUC(matchRepo *mockMatchRepo, eventRepo *mockEventRepo) *match.ConfirmMatchScoreUseCase {
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, eventRepo)
	return match.NewConfirmMatchScoreUseCase(matchRepo, eventRepo, updateScoreUC)
}

func TestConfirmMatchScoreUseCase(t *testing.T) {
	t.Run("no proposal returns ErrNoScoreProposal", func(t *testing.T) {
		matchRepo, eventRepo, _, _, _ := setupProposalFixtures()
		uc := newConfirmUC(matchRepo, eventRepo)

		p2 := "p2"
		if err := uc.Execute(context.Background(), "m1", &p2, false); err != match.ErrNoScoreProposal {
			t.Fatalf("expected ErrNoScoreProposal, got %v", err)
		}
	})

	t.Run("opposing participant confirms happy path, finalizes and clears proposal", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		if err := proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets()); err != nil {
			t.Fatalf("propose failed: %v", err)
		}

		uc := newConfirmUC(matchRepo, eventRepo)
		p2 := "p2"
		if err := uc.Execute(context.Background(), "m1", &p2, false); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		m := matchRepo.matches["m1"]
		if m.ProposedByPlayerID != nil || len(m.ProposedSets) != 0 {
			t.Errorf("expected proposal cleared after confirm, got %+v", m)
		}
		if !matchRepo.scoresUpdated {
			t.Errorf("expected UpdateScore to have been called via the existing finalize path")
		}
	})

	t.Run("proposer cannot confirm their own proposal", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		_ = proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())

		uc := newConfirmUC(matchRepo, eventRepo)
		p1 := "p1"
		if err := uc.Execute(context.Background(), "m1", &p1, false); err != match.ErrNotOpposingParticipant {
			t.Fatalf("expected ErrNotOpposingParticipant, got %v", err)
		}
	})

	t.Run("non-participant cannot confirm", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		_ = proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())

		uc := newConfirmUC(matchRepo, eventRepo)
		outsider := "p3"
		if err := uc.Execute(context.Background(), "m1", &outsider, false); err != match.ErrNotOpposingParticipant {
			t.Fatalf("expected ErrNotOpposingParticipant, got %v", err)
		}
	})

	t.Run("nil confirmedByPlayerID rejected for non-admin", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		_ = proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())

		uc := newConfirmUC(matchRepo, eventRepo)
		if err := uc.Execute(context.Background(), "m1", nil, false); err != match.ErrNotOpposingParticipant {
			t.Fatalf("expected ErrNotOpposingParticipant, got %v", err)
		}
	})

	t.Run("admin confirm bypasses the opposing-participant check", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		_ = proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())

		uc := newConfirmUC(matchRepo, eventRepo)
		if err := uc.Execute(context.Background(), "m1", nil, true); err != nil {
			t.Fatalf("unexpected error for admin confirm: %v", err)
		}
		m := matchRepo.matches["m1"]
		if m.ProposedByPlayerID != nil {
			t.Errorf("expected proposal cleared after admin confirm")
		}
	})
}

func TestRejectMatchScoreProposalUseCase(t *testing.T) {
	t.Run("no proposal returns ErrNoScoreProposal", func(t *testing.T) {
		matchRepo, _, _, _, _ := setupProposalFixtures()
		uc := match.NewRejectMatchScoreProposalUseCase(matchRepo)
		if err := uc.Execute(context.Background(), "m1", "p2"); err != match.ErrNoScoreProposal {
			t.Fatalf("expected ErrNoScoreProposal, got %v", err)
		}
	})

	t.Run("opposing participant rejects, proposal cleared, re-propose loop works", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		if err := proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets()); err != nil {
			t.Fatalf("propose failed: %v", err)
		}

		rejectUC := match.NewRejectMatchScoreProposalUseCase(matchRepo)
		if err := rejectUC.Execute(context.Background(), "m1", "p2"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		m := matchRepo.matches["m1"]
		if m.ProposedByPlayerID != nil {
			t.Fatalf("expected proposal cleared after reject, got %+v", m.ProposedByPlayerID)
		}

		// Re-propose after rejection should work again.
		if err := proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets()); err != nil {
			t.Fatalf("expected re-propose to succeed, got %v", err)
		}
		if matchRepo.matches["m1"].ProposedByPlayerID == nil {
			t.Fatal("expected proposal staged again after re-propose")
		}
	})

	t.Run("proposer cannot reject their own proposal", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		_ = proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())

		rejectUC := match.NewRejectMatchScoreProposalUseCase(matchRepo)
		if err := rejectUC.Execute(context.Background(), "m1", "p1"); err != match.ErrNotOpposingParticipant {
			t.Fatalf("expected ErrNotOpposingParticipant, got %v", err)
		}
	})

	t.Run("non-participant cannot reject", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		proposeUC := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		_ = proposeUC.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets())

		rejectUC := match.NewRejectMatchScoreProposalUseCase(matchRepo)
		if err := rejectUC.Execute(context.Background(), "m1", "p3"); err != match.ErrNotOpposingParticipant {
			t.Fatalf("expected ErrNotOpposingParticipant, got %v", err)
		}
	})
}

func TestProposeMatchScoreUseCase_ErrorPaths(t *testing.T) {
	t.Run("unknown player propagates GetById error", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		uc := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		if err := uc.Execute(context.Background(), "acc-1", "m1", "missing", sampleWinningSets()); err == nil {
			t.Fatal("expected error for unknown player")
		}
	})

	t.Run("unknown match propagates GetByID error", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		uc := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		if err := uc.Execute(context.Background(), "acc-1", "missing-match", "p1", sampleWinningSets()); err == nil {
			t.Fatal("expected error for unknown match")
		}
	})

	t.Run("unknown event propagates tournamentRepo error", func(t *testing.T) {
		matchRepo, eventRepo, playerRepo, _, _ := setupProposalFixtures()
		matchRepo.matches["m1"].EventID = "no-such-event"
		uc := match.NewProposeMatchScoreUseCase(matchRepo, eventRepo, playerRepo)
		if err := uc.Execute(context.Background(), "acc-1", "m1", "p1", sampleWinningSets()); err == nil {
			t.Fatal("expected error for unknown event")
		}
	})
}

func TestConfirmMatchScoreUseCase_ErrorPaths(t *testing.T) {
	t.Run("unknown match propagates GetByID error", func(t *testing.T) {
		matchRepo, eventRepo, _, _, _ := setupProposalFixtures()
		uc := newConfirmUC(matchRepo, eventRepo)
		if err := uc.Execute(context.Background(), "missing-match", nil, true); err == nil {
			t.Fatal("expected error for unknown match")
		}
	})
}

func TestRejectMatchScoreProposalUseCase_ErrorPaths(t *testing.T) {
	t.Run("unknown match propagates GetByID error", func(t *testing.T) {
		matchRepo, _, _, _, _ := setupProposalFixtures()
		uc := match.NewRejectMatchScoreProposalUseCase(matchRepo)
		if err := uc.Execute(context.Background(), "missing-match", "p2"); err == nil {
			t.Fatal("expected error for unknown match")
		}
	})
}
