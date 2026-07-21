package event

import (
	"context"
	"errors"
	"testing"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
)

// ─── GetTournamentByIDUseCase ───────────────────────────────────────────────

func TestGetTournamentByIDUseCase_Execute(t *testing.T) {
	t.Run("returns event as-is when groups already present", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:     "t1",
			Format: "round_robin",
			Groups: []tournamentDomain.Group{{ID: "g1", Name: "Group A"}},
		}
		divRepo := &mockDivisionRepo{}
		uc := NewGetTournamentByIDUseCase(repo, divRepo)

		got, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got.Groups) != 1 {
			t.Errorf("expected groups to remain untouched, got %d", len(got.Groups))
		}
		if repo.updateCalls != 0 {
			t.Errorf("expected no update call, got %d", repo.updateCalls)
		}
	})

	t.Run("regenerates missing groups for group formats", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:           "t1",
			Format:       "round_robin",
			SkipElo:      true,
			Participants: []*playerDomain.Player{{ID: "p1", FirstName: "A", LastName: "B"}},
		}
		divRepo := &mockDivisionRepo{}
		uc := NewGetTournamentByIDUseCase(repo, divRepo)

		got, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got.Groups) == 0 {
			t.Errorf("expected groups to be regenerated")
		}
		if repo.updateCalls != 1 {
			t.Errorf("expected 1 update call after regen, got %d", repo.updateCalls)
		}
	})

	t.Run("regenerates when team count mismatches group participant count", func(t *testing.T) {
		repo := newMockRepo()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:      "t1",
			Format:  "groups_elimination",
			Type:    "doubles",
			SkipElo: true,
			Teams: []*tournamentDomain.Team{
				{ID: "team1", Name: "T1"},
				{ID: "team2", Name: "T2"},
			},
			Groups: []tournamentDomain.Group{{ID: "g1", Name: "Group A", Players: nil}},
		}
		divRepo := &mockDivisionRepo{}
		uc := NewGetTournamentByIDUseCase(repo, divRepo)

		got, err := uc.Execute(context.Background(), "t1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		total := 0
		for _, g := range got.Groups {
			total += len(g.Players)
		}
		if total != 2 {
			t.Errorf("expected 2 group participants after regen, got %d", total)
		}
	})

	t.Run("get error propagates", func(t *testing.T) {
		repo := newMockRepo()
		repo.getErr = errors.New("db error")
		uc := NewGetTournamentByIDUseCase(repo, &mockDivisionRepo{})

		_, err := uc.Execute(context.Background(), "missing")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("GetSnapshots delegates to repo", func(t *testing.T) {
		repo := newMockRepo()
		repo.snapshots = []tournamentDomain.ParticipantSnapshot{{PlayerID: "p1"}}
		uc := NewGetTournamentByIDUseCase(repo, &mockDivisionRepo{})

		snaps, err := uc.GetSnapshots(context.Background(), "t1")
		if err != nil || len(snaps) != 1 {
			t.Fatalf("expected 1 snapshot, no error, got %v, %v", snaps, err)
		}
	})

	t.Run("GetOfficials delegates to repo", func(t *testing.T) {
		repo := newMockRepo()
		repo.officials = []tournamentDomain.ParticipantSnapshot{{PlayerID: "p1"}}
		uc := NewGetTournamentByIDUseCase(repo, &mockDivisionRepo{})

		offs, err := uc.GetOfficials(context.Background(), "t1")
		if err != nil || len(offs) != 1 {
			t.Fatalf("expected 1 official, no error, got %v, %v", offs, err)
		}
	})

	t.Run("AddOfficial generates unique pin avoiding existing", func(t *testing.T) {
		repo := newMockRepo()
		repo.officials = []tournamentDomain.ParticipantSnapshot{{PlayerID: "p1", Pin: "1234"}}
		uc := NewGetTournamentByIDUseCase(repo, &mockDivisionRepo{})

		err := uc.AddOfficial(context.Background(), "t1", "p2")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("AddOfficial propagates GetOfficials error", func(t *testing.T) {
		repo := newMockRepo()
		repo.officialsErr = errors.New("boom")
		uc := NewGetTournamentByIDUseCase(repo, &mockDivisionRepo{})

		if err := uc.AddOfficial(context.Background(), "t1", "p2"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("RemoveOfficial delegates to repo", func(t *testing.T) {
		repo := newMockRepo()
		uc := NewGetTournamentByIDUseCase(repo, &mockDivisionRepo{})
		if err := uc.RemoveOfficial(context.Background(), "t1", "p1"); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		repo.removeOfficErr = errors.New("boom")
		if err := uc.RemoveOfficial(context.Background(), "t1", "p1"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

// ─── UpdateTournamentUseCase ────────────────────────────────────────────────

func TestUpdateTournamentUseCase_Execute(t *testing.T) {
	newUC := func() (*UpdateTournamentUseCase, *mockRepo, *mockPlayerRepo, *mockDivisionRepo) {
		repo := newMockRepo()
		playerRepo := newMockPlayerRepo()
		divRepo := &mockDivisionRepo{}
		uc := NewUpdateTournamentUseCase(repo, playerRepo, divRepo)
		return uc, repo, playerRepo, divRepo
	}

	baseCmd := func() UpdateEventCommand {
		return UpdateEventCommand{
			ID:        "t1",
			Name:      "Updated",
			Type:      "singles",
			Format:    "round_robin",
			Category:  "open",
			StartDate: "2026-01-01",
			EndDate:   "2026-01-02",
		}
	}

	t.Run("invalid start date", func(t *testing.T) {
		uc, _, _, _ := newUC()
		cmd := baseCmd()
		cmd.StartDate = "bad"
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid end date", func(t *testing.T) {
		uc, _, _, _ := newUC()
		cmd := baseCmd()
		cmd.EndDate = "bad"
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("fallback path for non-existent event still succeeds", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		cmd := baseCmd()
		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.Name != "Updated" {
			t.Errorf("expected name Updated, got %s", got.Name)
		}
		if repo.updateCalls != 1 {
			t.Errorf("expected 1 update call, got %d", repo.updateCalls)
		}
	})

	t.Run("preserves groups when nothing changed and manual lock is off", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		existingGroups := []tournamentDomain.Group{{ID: "g1", Name: "Group A"}}
		repo.events["t1"] = &tournamentDomain.Event{
			ID:            "t1",
			Format:        "round_robin",
			Type:          "singles",
			EventCategory: "open",
			Groups:        existingGroups,
		}
		cmd := baseCmd()

		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got.Groups) != 1 || got.Groups[0].ID != "g1" {
			t.Errorf("expected existing groups preserved, got %+v", got.Groups)
		}
	})

	t.Run("regenerates groups when format changed", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.events["t1"] = &tournamentDomain.Event{
			ID:            "t1",
			Format:        "elimination",
			Type:          "singles",
			EventCategory: "open",
			SkipElo:       true,
			Groups:        []tournamentDomain.Group{{ID: "g1", Name: "Old Group"}},
		}
		cmd := baseCmd()
		cmd.SkipElo = true

		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		// format changed from elimination -> round_robin, group should be regenerated (not "Old Group")
		for _, g := range got.Groups {
			if g.ID == "g1" {
				t.Errorf("expected old group to be replaced by regeneration")
			}
		}
	})

	t.Run("preserves groups when manual seeding locked even if participants changed", func(t *testing.T) {
		uc, repo, playerRepo, _ := newUC()
		existingGroups := []tournamentDomain.Group{{ID: "g1", Name: "Locked Group"}}
		repo.events["t1"] = &tournamentDomain.Event{
			ID:                  "t1",
			Format:              "round_robin",
			Type:                "singles",
			EventCategory:       "open",
			ManualSeedingLocked: true,
			Groups:              existingGroups,
		}
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1", FirstName: "A", LastName: "B"}
		cmd := baseCmd()
		cmd.ParticipantIDs = []string{"p1"}

		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got.Groups) != 1 || got.Groups[0].ID != "g1" {
			t.Errorf("expected locked groups preserved, got %+v", got.Groups)
		}
	})

	t.Run("new player creation error propagates", func(t *testing.T) {
		uc, _, _, _ := newUC()
		cmd := baseCmd()
		cmd.NewPlayers = []NewPlayerData{{FirstName: "", LastName: ""}}
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("player save error propagates", func(t *testing.T) {
		uc, _, playerRepo, _ := newUC()
		playerRepo.saveErr = errors.New("db error")
		cmd := baseCmd()
		cmd.NewPlayers = []NewPlayerData{{FirstName: "Bob", LastName: "B"}}
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("update error propagates", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.updateErr = errors.New("db error")
		cmd := baseCmd()
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("applies stage rule overrides", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		cmd := baseCmd()
		cmd.StageRuleOverrides = []StageRuleOverride{
			{Stage: "final", BestOf: 7, PointsToWin: 11, PointsMargin: 2},
		}
		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		found := false
		for _, sr := range got.StageRules {
			if sr.Stage == "final" {
				found = true
				if sr.BestOf != 7 {
					t.Errorf("expected BestOf 7, got %d", sr.BestOf)
				}
			}
		}
		if !found {
			t.Errorf("expected final stage rule present")
		}
		_ = repo
	})
}
