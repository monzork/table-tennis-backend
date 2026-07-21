package event

import (
	"context"
	"testing"

	divisionDomain "table-tennis-backend/internal/domain/division"
	playerDomain "table-tennis-backend/internal/domain/player"
)

func TestCreateTournamentUseCase_Execute(t *testing.T) {
	newUC := func() (*CreateTournamentUseCase, *mockRepo, *mockPlayerRepo, *mockDivisionRepo) {
		repo := newMockRepo()
		playerRepo := newMockPlayerRepo()
		divRepo := &mockDivisionRepo{}
		uc := NewCreateTournamentUseCase(repo, playerRepo, divRepo)
		return uc, repo, playerRepo, divRepo
	}

	t.Run("happy path with existing and new players", func(t *testing.T) {
		uc, repo, playerRepo, divRepo := newUC()
		existing := &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", Gender: "F", SinglesElo: 1200}
		playerRepo.players["p1"] = existing
		divRepo.divisions = []*divisionDomain.Division{
			{ID: "d1", Name: "Open", Category: "both", MinElo: 0, MaxElo: nil},
		}

		cmd := CreateEventCommand{
			Name:           "Spring Open",
			Type:           "singles",
			Format:         "round_robin",
			Category:       "open",
			StartDate:      "2026-01-01",
			EndDate:        "2026-01-02",
			ParticipantIDs: []string{"p1", ""},
			NewPlayers: []NewPlayerData{
				{FirstName: "Bob", LastName: "B", Gender: "M"},
			},
			GroupPassCount: 2,
		}

		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got.Name != "Spring Open" {
			t.Errorf("expected name Spring Open, got %s", got.Name)
		}
		if len(got.Participants) != 2 {
			t.Fatalf("expected 2 participants, got %d", len(got.Participants))
		}
		if len(playerRepo.savedPlayers) != 1 {
			t.Errorf("expected 1 new player saved, got %d", len(playerRepo.savedPlayers))
		}
		if repo.saveCalls != 1 {
			t.Errorf("expected 1 save call (no pair event for open category), got %d", repo.saveCalls)
		}
	})

	t.Run("invalid start date returns error", func(t *testing.T) {
		uc, _, _, _ := newUC()
		cmd := CreateEventCommand{Name: "X", StartDate: "bad-date", EndDate: "2026-01-02"}
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid end date returns error", func(t *testing.T) {
		uc, _, _, _ := newUC()
		cmd := CreateEventCommand{Name: "X", StartDate: "2026-01-01", EndDate: "bad-date"}
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("end before start returns domain error", func(t *testing.T) {
		uc, _, _, _ := newUC()
		cmd := CreateEventCommand{Name: "X", StartDate: "2026-01-02", EndDate: "2026-01-01"}
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("category filters participants by gender", func(t *testing.T) {
		uc, _, playerRepo, _ := newUC()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", Gender: "F"}
		playerRepo.players["p2"] = &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", Gender: "M"}

		cmd := CreateEventCommand{
			Name:           "Mens Cup",
			Category:       "men",
			StartDate:      "2026-01-01",
			EndDate:        "2026-01-02",
			ParticipantIDs: []string{"p1", "p2"},
		}
		got, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got.Participants) != 1 || got.Participants[0].ID != "p2" {
			t.Fatalf("expected only p2 (male) in men's event, got %+v", got.Participants)
		}
	})

	t.Run("creates paired event for opposite gender", func(t *testing.T) {
		uc, repo, playerRepo, _ := newUC()
		playerRepo.players["p1"] = &playerDomain.Player{ID: "p1", FirstName: "Alice", LastName: "A", Gender: "F"}
		playerRepo.players["p2"] = &playerDomain.Player{ID: "p2", FirstName: "Bob", LastName: "B", Gender: "M"}

		cmd := CreateEventCommand{
			Name:           "Cup",
			Category:       "men",
			StartDate:      "2026-01-01",
			EndDate:        "2026-01-02",
			ParticipantIDs: []string{"p1", "p2"},
		}
		_, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if repo.saveCalls != 2 {
			t.Errorf("expected 2 save calls (main + paired event), got %d", repo.saveCalls)
		}
		foundPaired := false
		for _, ev := range repo.events {
			if ev.Name == "Women's Cup" {
				foundPaired = true
				if len(ev.Participants) != 1 || ev.Participants[0].ID != "p1" {
					t.Errorf("expected paired event to contain only female participant p1")
				}
			}
		}
		if !foundPaired {
			t.Errorf("expected a paired Women's Cup event to be created")
		}
	})

	t.Run("save error propagates", func(t *testing.T) {
		uc, repo, _, _ := newUC()
		repo.saveErr = context.DeadlineExceeded
		cmd := CreateEventCommand{Name: "X", StartDate: "2026-01-01", EndDate: "2026-01-02"}
		_, err := uc.Execute(context.Background(), cmd)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}
