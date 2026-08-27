package account_test

import (
	"context"
	"testing"

	accountApp "table-tennis-backend/internal/application/account"
	"table-tennis-backend/internal/domain/player"
)

func TestClaimPlayerUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	playerRepo.players["p1"] = &player.Player{ID: "p1", FirstName: "Kid", LastName: "Smith"}
	uc := accountApp.NewClaimPlayerUseCase(playerRepo)

	t.Run("claims an unlinked player", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "acc-1", "p1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := playerRepo.players["p1"]
		if p.ClaimedByAccountID == nil || *p.ClaimedByAccountID != "acc-1" {
			t.Fatalf("expected ClaimedByAccountID set to acc-1, got %+v", p.ClaimedByAccountID)
		}
	})

	t.Run("already-pending claim is rejected", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "acc-2", "p1"); err != accountApp.ErrClaimAlreadyPending {
			t.Fatalf("expected ErrClaimAlreadyPending, got %v", err)
		}
	})

	t.Run("already-linked player is rejected", func(t *testing.T) {
		gid := "acc-existing"
		playerRepo.players["p2"] = &player.Player{ID: "p2", GuardianAccountID: &gid}
		if err := uc.Execute(context.Background(), "acc-3", "p2"); err != accountApp.ErrPlayerAlreadyLinked {
			t.Fatalf("expected ErrPlayerAlreadyLinked, got %v", err)
		}
	})

	t.Run("unknown player propagates error", func(t *testing.T) {
		if err := uc.Execute(context.Background(), "acc-1", "missing"); err == nil {
			t.Fatal("expected error for missing player")
		}
	})
}

func TestSearchClaimablePlayersUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	gid := "acc-1"
	claimedBy := "acc-2"
	playerRepo.players["free"] = &player.Player{ID: "free", FirstName: "Free"}
	playerRepo.players["linked"] = &player.Player{ID: "linked", FirstName: "Linked", GuardianAccountID: &gid}
	playerRepo.players["claimed"] = &player.Player{ID: "claimed", FirstName: "Claimed", ClaimedByAccountID: &claimedBy}

	uc := accountApp.NewSearchClaimablePlayersUseCase(playerRepo)
	results, err := uc.Execute(context.Background(), "any")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 || results[0].ID != "free" {
		t.Fatalf("expected only the unlinked, unclaimed player, got %+v", results)
	}
}

func TestGetPendingPlayerClaimsUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	accountRepo := newFakeAccountRepo()
	acc, _ := accountApp.NewLoginWithGoogleUseCase(accountRepo).Execute(context.Background(), accountApp.LoginWithGoogleCommand{GoogleSub: "s", Email: "claimant@x.com"})

	playerRepo.players["p1"] = &player.Player{ID: "p1", FirstName: "Kid", ClaimedByAccountID: &acc.ID}
	playerRepo.players["p2"] = &player.Player{ID: "p2", FirstName: "Other"}

	uc := accountApp.NewGetPendingPlayerClaimsUseCase(playerRepo, accountRepo)
	claims, err := uc.Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(claims) != 1 {
		t.Fatalf("expected exactly 1 pending claim, got %d", len(claims))
	}
	if claims[0].Player.ID != "p1" || claims[0].AccountEmail != "claimant@x.com" {
		t.Fatalf("unexpected claim: %+v", claims[0])
	}
}

func TestApproveAndRejectPlayerClaimUseCase(t *testing.T) {
	playerRepo := newFakePlayerRepo()
	claimant := "acc-1"

	t.Run("approve links the claiming account and clears the claim", func(t *testing.T) {
		playerRepo.players["p1"] = &player.Player{ID: "p1", ClaimedByAccountID: &claimant}
		uc := accountApp.NewApprovePlayerClaimUseCase(playerRepo)
		if err := uc.Execute(context.Background(), "p1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := playerRepo.players["p1"]
		if p.ClaimedByAccountID != nil {
			t.Fatalf("expected claim cleared")
		}
		if p.GuardianAccountID == nil || *p.GuardianAccountID != claimant {
			t.Fatalf("expected GuardianAccountID set to %q, got %+v", claimant, p.GuardianAccountID)
		}
	})

	t.Run("approve with no pending claim errors", func(t *testing.T) {
		playerRepo.players["p2"] = &player.Player{ID: "p2"}
		uc := accountApp.NewApprovePlayerClaimUseCase(playerRepo)
		if err := uc.Execute(context.Background(), "p2"); err != accountApp.ErrNoPendingClaim {
			t.Fatalf("expected ErrNoPendingClaim, got %v", err)
		}
	})

	t.Run("reject clears the claim without linking", func(t *testing.T) {
		playerRepo.players["p3"] = &player.Player{ID: "p3", ClaimedByAccountID: &claimant}
		uc := accountApp.NewRejectPlayerClaimUseCase(playerRepo)
		if err := uc.Execute(context.Background(), "p3"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		p := playerRepo.players["p3"]
		if p.ClaimedByAccountID != nil || p.GuardianAccountID != nil {
			t.Fatalf("expected both fields nil, got %+v", p)
		}
	})

	t.Run("reject with no pending claim errors", func(t *testing.T) {
		playerRepo.players["p4"] = &player.Player{ID: "p4"}
		uc := accountApp.NewRejectPlayerClaimUseCase(playerRepo)
		if err := uc.Execute(context.Background(), "p4"); err != accountApp.ErrNoPendingClaim {
			t.Fatalf("expected ErrNoPendingClaim, got %v", err)
		}
	})

	t.Run("unknown player propagates error", func(t *testing.T) {
		uc := accountApp.NewApprovePlayerClaimUseCase(playerRepo)
		if err := uc.Execute(context.Background(), "missing"); err == nil {
			t.Fatal("expected error for missing player")
		}
	})
}
