package handler_test

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestAccountHandler_ClaimFlow(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	repo := bunRepo.NewPlayerRepository(db)
	p, err := player.NewPlayer(idgen.Generate(), "Unclaimed", "Player", time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "", "")
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("Save player: %v", err)
	}

	claimantID := seedTestAccount(t, db, "sub-claim", "claimant@x.com", "Claimant")
	claimantCookie := accountLogin(t, app, claimantID)

	t.Run("show claim form", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/claim", nil)
		req.Header.Set("Cookie", claimantCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("search finds the unclaimed player", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/claim/search?q=Unclaimed", nil)
		req.Header.Set("Cookie", claimantCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("claim submits and requires admin approval", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/account/players/"+p.ID+"/claim", nil)
		req.Header.Set("Cookie", claimantCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %v", resp.StatusCode)
		}

		got, err := repo.GetById(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if got.ClaimedByAccountID == nil || *got.ClaimedByAccountID != claimantID {
			t.Fatalf("expected pending claim by %q, got %+v", claimantID, got.ClaimedByAccountID)
		}
		if got.GuardianAccountID != nil {
			t.Fatalf("expected player NOT yet linked before admin approval, got %+v", got.GuardianAccountID)
		}
	})

	t.Run("claiming an already-pending player fails", func(t *testing.T) {
		other := seedTestAccount(t, db, "sub-other-claimant", "other-claimant@x.com", "Other")
		otherCookie := accountLogin(t, app, other)
		req := httptest.NewRequest("POST", "/account/players/"+p.ID+"/claim", nil)
		req.Header.Set("Cookie", otherCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Fatalf("expected 400, got %v", resp.StatusCode)
		}
	})

	adminCookie := adminLogin(t, app)

	t.Run("admin sees the pending claim on the players page", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/admin/players", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %v", resp.StatusCode)
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		if !strings.Contains(string(body), "Claimant") || !strings.Contains(string(body), "claimant@x.com") {
			t.Errorf("expected pending claim to show claimant name and email, got: %s", body)
		}
	})

	t.Run("admin approves the claim, linking the player", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/approve-claim", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %v", resp.StatusCode)
		}

		got, err := repo.GetById(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if got.ClaimedByAccountID != nil {
			t.Fatalf("expected claim cleared, got %+v", got.ClaimedByAccountID)
		}
		if got.GuardianAccountID == nil || *got.GuardianAccountID != claimantID {
			t.Fatalf("expected player linked to %q, got %+v", claimantID, got.GuardianAccountID)
		}
	})

	t.Run("approving a player with no pending claim fails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/approve-claim", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Fatalf("expected 400, got %v", resp.StatusCode)
		}
	})
}

func TestAccountHandler_RejectClaim(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	repo := bunRepo.NewPlayerRepository(db)
	p, err := player.NewPlayer(idgen.Generate(), "Rejectable", "Player", time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "", "")
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("Save player: %v", err)
	}

	claimantID := seedTestAccount(t, db, "sub-reject", "reject@x.com", "Claimant")
	claimantCookie := accountLogin(t, app, claimantID)

	req := httptest.NewRequest("POST", "/account/players/"+p.ID+"/claim", nil)
	req.Header.Set("Cookie", claimantCookie)
	if resp, err := app.Test(req); err != nil || resp.StatusCode != 200 {
		t.Fatalf("claim setup failed: err=%v status=%v", err, resp)
	}

	adminCookie := adminLogin(t, app)

	t.Run("admin rejects the claim, leaving the player unlinked", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/reject-claim", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %v", resp.StatusCode)
		}

		got, err := repo.GetById(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if got.ClaimedByAccountID != nil || got.GuardianAccountID != nil {
			t.Fatalf("expected player left unlinked, got %+v", got)
		}
	})

	t.Run("rejecting a player with no pending claim fails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/reject-claim", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Fatalf("expected 400, got %v", resp.StatusCode)
		}
	})
}
