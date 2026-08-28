package account_test

import (
	"context"
	"testing"
	"time"

	accountApp "table-tennis-backend/internal/application/account"
	playerApp "table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/infrastructure/identity"

	tournamentEvent "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

func TestLoginWithGoogleUseCase_Execute(t *testing.T) {
	repo := newFakeAccountRepo()
	uc := accountApp.NewLoginWithGoogleUseCase(repo)

	cmd := accountApp.LoginWithGoogleCommand{GoogleSub: "sub1", Email: "a@b.com", Name: "Alice", PictureURL: "pic1"}

	t.Run("creates a new account on first login", func(t *testing.T) {
		a, err := uc.Execute(context.Background(), cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.GoogleSub != "sub1" || a.Email != "a@b.com" {
			t.Fatalf("unexpected account: %+v", a)
		}
	})

	t.Run("repeat login is idempotent and refreshes profile fields", func(t *testing.T) {
		first, _ := uc.Execute(context.Background(), cmd)

		cmd2 := accountApp.LoginWithGoogleCommand{GoogleSub: "sub1", Email: "a@b.com", Name: "Alice Updated", PictureURL: "pic2"}
		second, err := uc.Execute(context.Background(), cmd2)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if second.ID != first.ID {
			t.Fatalf("expected same account ID on repeat login, got %q vs %q", second.ID, first.ID)
		}
		if second.Name != "Alice Updated" || second.PictureURL != "pic2" {
			t.Errorf("expected refreshed profile fields, got %+v", second)
		}
	})

	t.Run("invalid command surfaces domain validation error", func(t *testing.T) {
		if _, err := uc.Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "", Email: "x@y.com"}); err == nil {
			t.Fatal("expected error for empty google sub")
		}
	})
}

func TestGetAccountByIDUseCase(t *testing.T) {
	repo := newFakeAccountRepo()
	uc := accountApp.NewGetAccountByIDUseCase(repo)

	loginUC := accountApp.NewLoginWithGoogleUseCase(repo)
	created, _ := loginUC.Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s", Email: "e@x.com"})

	got, err := uc.Execute(context.Background(), created.ID)
	if err != nil || got.ID != created.ID {
		t.Fatalf("unexpected result: %+v, err=%v", got, err)
	}

	if _, err := uc.Execute(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestUpdateAccountUseCase(t *testing.T) {
	repo := newFakeAccountRepo()
	loginUC := accountApp.NewLoginWithGoogleUseCase(repo)
	created, _ := loginUC.Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s", Email: "e@x.com", Name: "Old"})

	uc := accountApp.NewUpdateAccountUseCase(repo)
	updated, err := uc.Execute(context.Background(), created.ID, "New Name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Name != "New Name" {
		t.Errorf("expected updated name, got %q", updated.Name)
	}

	if _, err := uc.Execute(context.Background(), "missing", "x"); err == nil {
		t.Fatal("expected error for missing account")
	}
}

func TestCreateChildPlayerUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	uc := accountApp.NewCreateChildPlayerUseCase(playerRepo)

	p, err := uc.Execute(context.Background(), accountApp.CreateChildPlayerCommand{
		GuardianAccountID: "acc-1",
		FirstName:         "Kid",
		LastName:          "Smith",
		Birthdate:         time.Now(),
		Gender:            "M",
		Country:           "USA",
		Department:        "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.GuardianAccountID == nil || *p.GuardianAccountID != "acc-1" {
		t.Fatalf("expected linked GuardianAccountID, got %+v", p)
	}
	if _, ok := playerRepo.players[p.ID]; !ok {
		t.Fatal("expected player to be persisted")
	}

	t.Run("validation error propagates", func(t *testing.T) {
		if _, err := uc.Execute(context.Background(), accountApp.CreateChildPlayerCommand{GuardianAccountID: "acc-1"}); err == nil {
			t.Fatal("expected validation error for missing names")
		}
	})
}

func TestGetLinkedPlayersUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	gid := "acc-1"
	other := "acc-2"
	playerRepo.players["p1"] = &player.Player{ID: "p1", GuardianAccountID: &gid}
	playerRepo.players["p2"] = &player.Player{ID: "p2", GuardianAccountID: &other}

	uc := accountApp.NewGetLinkedPlayersUseCase(playerRepo)
	players, err := uc.Execute(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(players) != 1 || players[0].ID != "p1" {
		t.Fatalf("expected only p1, got %+v", players)
	}
}

func TestEnsurePlayerBelongsToAccount(t *testing.T) {
	gid := "acc-1"
	owned := &player.Player{ID: "p1", GuardianAccountID: &gid}
	unowned := &player.Player{ID: "p2"}

	if err := accountApp.EnsurePlayerBelongsToAccount(owned, "acc-1"); err != nil {
		t.Errorf("expected no error for owned player, got %v", err)
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(owned, "acc-2"); err != accountApp.ErrPlayerNotOwnedByAccount {
		t.Errorf("expected ErrPlayerNotOwnedByAccount for mismatched account, got %v", err)
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(unowned, "acc-1"); err != accountApp.ErrPlayerNotOwnedByAccount {
		t.Errorf("expected ErrPlayerNotOwnedByAccount for unlinked player, got %v", err)
	}
	if err := accountApp.EnsurePlayerBelongsToAccount(nil, "acc-1"); err != accountApp.ErrPlayerNotOwnedByAccount {
		t.Errorf("expected ErrPlayerNotOwnedByAccount for nil player, got %v", err)
	}
}

func TestAssignPlayerToAccountUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	accountRepo := newFakeAccountRepo()
	playerRepo.players["p1"] = &player.Player{ID: "p1", FirstName: "Kid", LastName: "Smith"}
	acc, _ := accountApp.NewLoginWithGoogleUseCase(accountRepo).Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s", Email: "guardian@x.com"})

	uc := accountApp.NewAssignPlayerToAccountUseCase(playerRepo, accountRepo)

	t.Run("links player by account email", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "p1", "guardian@x.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := playerRepo.players["p1"]
		if p.GuardianAccountID == nil || *p.GuardianAccountID != acc.ID {
			t.Fatalf("expected player linked to account %q, got %+v", acc.ID, p.GuardianAccountID)
		}
	})

	t.Run("unlink clears the guardian account", func(t *testing.T) {
		if err := uc.Unlink(context.Background(), "p1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if playerRepo.players["p1"].GuardianAccountID != nil {
			t.Fatalf("expected GuardianAccountID cleared")
		}
	})

	t.Run("unknown email returns ErrAccountNotFound", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "p1", "nobody@x.com"); err != accountApp.ErrAccountNotFound {
			t.Fatalf("expected ErrAccountNotFound, got %v", err)
		}
	})

	t.Run("unknown player propagates error", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "missing", "guardian@x.com"); err == nil {
			t.Fatal("expected error for missing player")
		}
	})
}

func TestGetGuardianPendingMatchesUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	eventRepo := newFakeEventRepo()

	gid := "acc-1"
	p1 := &player.Player{ID: "p1", FirstName: "Kid1", GuardianAccountID: &gid}
	p2 := &player.Player{ID: "p2", FirstName: "Kid2", GuardianAccountID: &gid}
	playerRepo.players["p1"] = p1
	playerRepo.players["p2"] = p2

	opponent := &player.Player{ID: "opp", FirstName: "Opp"}
	eventRepo.eventsByPlayer["p1"] = []*tournamentEvent.Event{
		{ID: "e1", Matches: []tournamentEvent.Match{
			{ID: "m1", Status: "scheduled", TeamA: []*player.Player{p1}, TeamB: []*player.Player{opponent}},
		}},
	}
	// p2 has no pending matches.

	getLinkedUC := accountApp.NewGetLinkedPlayersUseCase(playerRepo)
	getPendingUC := playerApp.NewGetPlayerPendingMatchesUseCase(eventRepo, &fakeDivisionRepo{})
	uc := accountApp.NewGetGuardianPendingMatchesUseCase(getLinkedUC, getPendingUC)

	results, err := uc.Execute(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 players in result, got %d", len(results))
	}

	var found bool
	for _, r := range results {
		if r.Player.ID == "p1" {
			found = true
			if len(r.Matches) != 1 || r.Matches[0].Opponent != "Opp " {
				t.Errorf("unexpected pending matches for p1: %+v", r.Matches)
			}
		}
	}
	if !found {
		t.Fatal("expected p1 in results")
	}
}

func TestLoginWithGoogleUseCase_SaveError(t *testing.T) {
	repo := newFakeAccountRepo()
	repo.saveErr = context.DeadlineExceeded
	uc := accountApp.NewLoginWithGoogleUseCase(repo)

	if _, err := uc.Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s", Email: "e@x.com"}); err == nil {
		t.Fatal("expected error when Save fails")
	}
}

func TestUpdateAccountUseCase_SaveError(t *testing.T) {
	repo := newFakeAccountRepo()
	created, _ := accountApp.NewLoginWithGoogleUseCase(repo).Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s", Email: "e@x.com"})

	repo.saveErr = context.DeadlineExceeded
	uc := accountApp.NewUpdateAccountUseCase(repo)
	if _, err := uc.Execute(context.Background(), created.ID, "New"); err == nil {
		t.Fatal("expected error when Save fails")
	}
}

func TestCreateChildPlayerUseCase_SaveError(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	playerRepo.saveErr = context.DeadlineExceeded
	uc := accountApp.NewCreateChildPlayerUseCase(playerRepo)

	_, err := uc.Execute(context.Background(), accountApp.CreateChildPlayerCommand{
		GuardianAccountID: "acc-1", FirstName: "Kid", LastName: "Smith",
	})
	if err == nil {
		t.Fatal("expected error when Save fails")
	}
}

func TestAssignPlayerToAccountUseCase_ErrorPaths(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	accountRepo := newFakeAccountRepo()
	uc := accountApp.NewAssignPlayerToAccountUseCase(playerRepo, accountRepo)

	t.Run("GetByEmail error propagates", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "p1", "x@y.com"); err != accountApp.ErrAccountNotFound {
			t.Fatalf("expected ErrAccountNotFound, got %v", err)
		}
	})

	t.Run("Unlink on unknown player propagates error", func(t *testing.T) {
		if err := uc.Unlink(context.Background(), "missing"); err == nil {
			t.Fatal("expected error for missing player")
		}
	})

	t.Run("Unlink Save error propagates", func(t *testing.T) {
		playerRepo.players["p1"] = &player.Player{ID: "p1"}
		playerRepo.saveErr = context.DeadlineExceeded
		if err := uc.Unlink(context.Background(), "p1"); err == nil {
			t.Fatal("expected Save error to propagate")
		}
		playerRepo.saveErr = nil
	})

	t.Run("Execute Save error propagates", func(t *testing.T) {
		playerRepo.players["p1"] = &player.Player{ID: "p1"}
		acc, _ := accountApp.NewLoginWithGoogleUseCase(accountRepo).Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s2", Email: "s2@x.com"})
		_ = acc
		playerRepo.saveErr = context.DeadlineExceeded
		if err := uc.Execute(context.Background(), "p1", "s2@x.com"); err == nil {
			t.Fatal("expected Save error to propagate")
		}
		playerRepo.saveErr = nil
	})
}

func TestAssignPlayerToAccountUseCase_GetLinkedAccount(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	accountRepo := newFakeAccountRepo()
	playerRepo.players["p1"] = &player.Player{ID: "p1", FirstName: "Kid", LastName: "Smith"}
	acc, _ := accountApp.NewLoginWithGoogleUseCase(accountRepo).Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s3", Email: "guardian3@x.com", Name: "Guardian Three"})

	uc := accountApp.NewAssignPlayerToAccountUseCase(playerRepo, accountRepo)

	t.Run("unlinked player returns nil account", func(t *testing.T) {
		got, err := uc.GetLinkedAccount(context.Background(), "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != nil {
			t.Fatalf("expected nil account, got %+v", got)
		}
	})

	t.Run("linked player returns its account", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "p1", "guardian3@x.com"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		got, err := uc.GetLinkedAccount(context.Background(), "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == nil || got.ID != acc.ID {
			t.Fatalf("expected account %q, got %+v", acc.ID, got)
		}
	})

	t.Run("unknown player propagates error", func(t *testing.T) {
		if _, err := uc.GetLinkedAccount(context.Background(), "missing"); err == nil {
			t.Fatal("expected error for missing player")
		}
	})
}

func TestGetGuardianPendingMatchesUseCase_EventRepoError(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	eventRepo := newFakeEventRepo()
	eventRepo.err = context.DeadlineExceeded

	gid := "acc-1"
	playerRepo.players["p1"] = &player.Player{ID: "p1", GuardianAccountID: &gid}

	getLinkedUC := accountApp.NewGetLinkedPlayersUseCase(playerRepo)
	getPendingUC := playerApp.NewGetPlayerPendingMatchesUseCase(eventRepo, &fakeDivisionRepo{})
	uc := accountApp.NewGetGuardianPendingMatchesUseCase(getLinkedUC, getPendingUC)

	// best-effort: per-player errors don't fail the whole call, just leave
	// that player's Matches empty.
	results, err := uc.Execute(context.Background(), "acc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].Matches != nil {
		t.Fatalf("expected 1 result with no matches, got %+v", results)
	}
}
