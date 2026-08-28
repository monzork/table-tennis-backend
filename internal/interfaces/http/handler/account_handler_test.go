package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/account"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
)

// seedTestAccount creates and persists an Account directly (bypassing OAuth,
// which is exercised end-to-end in internal/infrastructure/oauth), returning
// its ID for use with accountLogin.
func seedTestAccount(t *testing.T, db *bun.DB, googleSub, email, name string) string {
	t.Helper()
	repo := bunRepo.NewAccountRepository(db)
	a, err := account.NewAccount(idgen.Generate(), googleSub, email, name, "")
	if err != nil {
		t.Fatalf("NewAccount: %v", err)
	}
	if err := repo.Save(context.Background(), a); err != nil {
		t.Fatalf("Save account: %v", err)
	}
	return a.ID
}

// seedLinkedPlayer creates and persists a Player linked to accountID.
func seedLinkedPlayer(t *testing.T, db *bun.DB, accountID, firstName string) *player.Player {
	t.Helper()
	repo := bunRepo.NewPlayerRepository(db)
	p, err := player.NewGuardianChildPlayer(idgen.Generate(), accountID, firstName, "Test", time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "")
	if err != nil {
		t.Fatalf("NewGuardianChildPlayer: %v", err)
	}
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("Save player: %v", err)
	}
	return p
}

// seedScheduledMatch creates a minimal open-category event and a scheduled
// singles match between p1 and p2, returning the match ID.
func seedScheduledMatch(t *testing.T, db *bun.DB, p1, p2 *player.Player) string {
	t.Helper()
	ctx := context.Background()

	eventRepo := bunRepo.NewEventRepository(db)
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	ev, err := event.NewEvent(uuid.NewString(), "Guardian Flow Event", "singles", "elimination", "open", start, start.Add(24*time.Hour), nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewEvent: %v", err)
	}
	if err := eventRepo.Save(ctx, ev); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	m := &event.Match{
		ID:        uuid.NewString(),
		EventID:   ev.ID,
		MatchType: "singles",
		TeamA:     []*player.Player{p1},
		TeamB:     []*player.Player{p2},
		Status:    "scheduled",
		// "group" stage defaults to Bo5 (event.DefaultStageRules), so 3
		// won sets is enough to decide the match — unlike knockout stages
		// (quarterfinal/semifinal/final/3rd_place), which default to Bo7.
		Stage: "group",
	}
	if err := matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save match: %v", err)
	}
	return m.ID
}

// adminLogin logs into the seeded default admin account (see SetupTestDB)
// and returns the session cookie, for exercising admin-protected routes.
func adminLogin(t *testing.T, app interface {
	Test(*http.Request, ...int) (*http.Response, error)
}) string {
	t.Helper()
	data := url.Values{}
	data.Set("username", "admin")
	data.Set("password", "password")
	req := httptest.NewRequest("POST", "/admin/login", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("admin login failed: %v", err)
	}
	var cookie string
	for _, v := range resp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			cookie = strings.Split(v, ";")[0]
		}
	}
	if cookie == "" {
		t.Fatal("expected session cookie from admin login")
	}
	return cookie
}

func findFirstPlayerIDForAccount(t *testing.T, db *bun.DB, accountID string) string {
	t.Helper()
	repo := bunRepo.NewPlayerRepository(db)
	players, err := repo.GetByGuardianAccountID(context.Background(), accountID)
	if err != nil {
		t.Fatalf("GetByGuardianAccountID: %v", err)
	}
	if len(players) == 0 {
		t.Fatal("expected at least one linked player")
	}
	return players[0].ID
}

// accountLogin establishes an account session for accountID via the test-only
// bypass route (real Google OAuth is exercised end-to-end in
// internal/infrastructure/oauth, not here) and returns the session cookie.
func accountLogin(t *testing.T, app interface {
	Test(*http.Request, ...int) (*http.Response, error)
}, accountID string) string {
	t.Helper()
	data := url.Values{}
	data.Set("accountId", accountID)
	req := httptest.NewRequest("POST", "/test/account-login", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("account login failed: %v", err)
	}
	var cookie string
	for _, v := range resp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			cookie = strings.Split(v, ";")[0]
		}
	}
	if cookie == "" {
		t.Fatal("expected session cookie from account login")
	}
	return cookie
}

func TestAccountHandler_ShowLogin(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest("GET", "/account/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %v", resp.StatusCode)
	}
}

func TestAccountHandler_GoogleLogin_RedirectsToGoogle(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest("GET", "/account/google/login", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected 302, got %v", resp.StatusCode)
	}
	loc := resp.Header.Get("Location")
	if !strings.Contains(loc, "accounts.google.com") {
		t.Errorf("expected redirect to Google, got %q", loc)
	}
}

func TestAccountHandler_GoogleCallback_MissingState(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest("GET", "/account/google/callback?state=bogus&code=abc", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	// No prior oauth_state in session -> rendered login page with error, 200.
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 (error re-render), got %v", resp.StatusCode)
	}
}

func TestAccountHandler_GoogleCallback_StateMismatch(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	loginReq := httptest.NewRequest("GET", "/account/google/login", nil)
	loginResp, err := app.Test(loginReq)
	if err != nil {
		t.Fatalf("google login failed: %v", err)
	}
	var cookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			cookie = strings.Split(v, ";")[0]
		}
	}

	req := httptest.NewRequest("GET", "/account/google/callback?state=wrong-state&code=abc", nil)
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 (error re-render), got %v", resp.StatusCode)
	}
}

func TestAccountHandler_Logout(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest("POST", "/account/logout", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected 302, got %v", resp.StatusCode)
	}
	if loc := resp.Header.Get("Location"); loc != "/account/login" {
		t.Errorf("expected redirect to /account/login, got %v", loc)
	}
}

func TestAccountHandler_ProtectedRoutes_RequireSession(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	for _, path := range []string{"/account/", "/account/me", "/account/players/new", "/account/pending-matches"} {
		req := httptest.NewRequest("GET", path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed for %s: %v", path, err)
		}
		if resp.StatusCode != 302 {
			t.Errorf("expected 302 for unauthenticated %s, got %v", path, resp.StatusCode)
		}
	}
}

func TestAccountHandler_DashboardAndMyInfo(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountID := seedTestAccount(t, db, "sub-dash", "dash@x.com", "Dash User")
	cookie := accountLogin(t, app, accountID)

	t.Run("dashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/", nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("my info show", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/me", nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("my info update", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "New Name")
		req := httptest.NewRequest("PUT", "/account/me", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	t.Run("pending matches", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/pending-matches", nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})
}

func TestAccountHandler_ChildPlayerLifecycle(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountID := seedTestAccount(t, db, "sub-child", "child@x.com", "Parent")
	cookie := accountLogin(t, app, accountID)

	t.Run("show add child form", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/new", nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	var playerID string
	t.Run("create child", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "Kid")
		data.Set("lastName", "Smith")
		data.Set("birthdate", "2012-01-01")
		data.Set("gender", "M")
		data.Set("country", "NIC")
		req := httptest.NewRequest("POST", "/account/players", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 {
			t.Errorf("expected 302 redirect, got %v", resp.StatusCode)
		}
	})

	t.Run("dashboard now lists the child", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/", nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}
	})

	// Look up the created child directly via a second account to test ownership.
	t.Run("another account cannot view this player", func(t *testing.T) {
		playerID = findFirstPlayerIDForAccount(t, db, accountID)
		otherAccountID := seedTestAccount(t, db, "sub-other", "other@x.com", "Other")
		otherCookie := accountLogin(t, app, otherAccountID)

		req := httptest.NewRequest("GET", "/account/players/"+playerID, nil)
		req.Header.Set("Cookie", otherCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Errorf("expected 403 for non-owner, got %v", resp.StatusCode)
		}
	})

	t.Run("owner can view player detail and edit form", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/"+playerID, nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}

		req2 := httptest.NewRequest("GET", "/account/players/"+playerID+"/edit", nil)
		req2.Header.Set("Cookie", cookie)
		resp2, err := app.Test(req2)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp2.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp2.StatusCode)
		}
	})

	t.Run("owner can update player", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "KidUpdated")
		data.Set("lastName", "Smith")
		data.Set("birthdate", "2012-01-01")
		data.Set("gender", "M")
		data.Set("country", "NIC")
		req := httptest.NewRequest("PUT", "/account/players/"+playerID, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 {
			t.Errorf("expected 302, got %v", resp.StatusCode)
		}
	})

	t.Run("nonexistent player returns 404", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/does-not-exist", nil)
		req.Header.Set("Cookie", cookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})
}

func TestAccountHandler_CreateChild_InvalidBody(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountID := seedTestAccount(t, db, "sub-bad", "bad@x.com", "")
	cookie := accountLogin(t, app, accountID)

	// Missing required names -> use case validation error -> 400.
	data := url.Values{}
	data.Set("birthdate", "2012-01-01")
	req := httptest.NewRequest("POST", "/account/players", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %v", resp.StatusCode)
	}
}

func TestAccountHandler_ScoreProposalFlow(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountA := seedTestAccount(t, db, "sub-flow-a", "a@x.com", "Guardian A")
	accountB := seedTestAccount(t, db, "sub-flow-b", "b@x.com", "Guardian B")
	p1 := seedLinkedPlayer(t, db, accountA, "PlayerOne")
	p2 := seedLinkedPlayer(t, db, accountB, "PlayerTwo")
	matchID := seedScheduledMatch(t, db, p1, p2)

	cookieA := accountLogin(t, app, accountA)
	cookieB := accountLogin(t, app, accountB)

	t.Run("wrong account cannot propose for someone else's player", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p2.ID)
		data.Add("sets[]", "11-5")
		data.Add("sets[]", "11-7")
		data.Add("sets[]", "11-9")
		req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieA)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("expected 400 for ownership violation, got %v", resp.StatusCode)
		}
	})

	t.Run("owner proposes a score", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p1.ID)
		data.Add("sets[]", "11-5")
		data.Add("sets[]", "11-7")
		data.Add("sets[]", "11-9")
		req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieA)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 {
			t.Errorf("expected 302 redirect, got %v", resp.StatusCode)
		}
	})

	t.Run("proposer cannot confirm their own proposal", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p1.ID)
		req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/confirm-score", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieA)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("expected 400, got %v", resp.StatusCode)
		}
	})

	t.Run("opposing account confirms the score", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p2.ID)
		req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/confirm-score", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 {
			t.Errorf("expected 302 redirect, got %v", resp.StatusCode)
		}
	})
}

func TestAccountHandler_RejectScoreFlow(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountA := seedTestAccount(t, db, "sub-reject-a", "ra@x.com", "Guardian A")
	accountB := seedTestAccount(t, db, "sub-reject-b", "rb@x.com", "Guardian B")
	p1 := seedLinkedPlayer(t, db, accountA, "PlayerOne")
	p2 := seedLinkedPlayer(t, db, accountB, "PlayerTwo")
	matchID := seedScheduledMatch(t, db, p1, p2)

	cookieA := accountLogin(t, app, accountA)
	cookieB := accountLogin(t, app, accountB)

	data := url.Values{}
	data.Set("playerId", p1.ID)
	data.Add("sets[]", "11-5")
	data.Add("sets[]", "11-7")
	data.Add("sets[]", "11-9")
	req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieA)
	if resp, err := app.Test(req); err != nil || resp.StatusCode != 302 {
		t.Fatalf("propose failed: resp=%+v err=%v", resp, err)
	}

	t.Run("unowned player cannot reject", func(t *testing.T) {
		rejectData := url.Values{}
		rejectData.Set("playerId", "not-a-real-player-id")
		req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/reject-score", strings.NewReader(rejectData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})

	t.Run("opposing account rejects the proposal", func(t *testing.T) {
		rejectData := url.Values{}
		rejectData.Set("playerId", p2.ID)
		req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/reject-score", strings.NewReader(rejectData.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 {
			t.Errorf("expected 302, got %v", resp.StatusCode)
		}
	})
}

func TestAccountHandler_ProposeScore_SplitABFields(t *testing.T) {
	// The account UI's score form mirrors the public QR/PIN form's split
	// A/B number-input pair per set (scores[]_a / scores[]_b) rather than a
	// single "A-B" text field — verify that combining path works too.
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountA := seedTestAccount(t, db, "sub-splitab-a", "sa@x.com", "Guardian A")
	accountB := seedTestAccount(t, db, "sub-splitab-b", "sb@x.com", "Guardian B")
	p1 := seedLinkedPlayer(t, db, accountA, "PlayerSplitA")
	p2 := seedLinkedPlayer(t, db, accountB, "PlayerSplitB")
	matchID := seedScheduledMatch(t, db, p1, p2)
	cookieA := accountLogin(t, app, accountA)

	data := url.Values{}
	data.Set("playerId", p1.ID)
	data.Add("scores[]_a", "11")
	data.Add("scores[]_b", "5")
	data.Add("scores[]_a", "11")
	data.Add("scores[]_b", "7")
	data.Add("scores[]_a", "11")
	data.Add("scores[]_b", "9")
	req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieA)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 302 {
		t.Errorf("expected 302 redirect, got %v", resp.StatusCode)
	}
}

func TestAccountHandler_ProposeScore_InvalidBody(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountID := seedTestAccount(t, db, "sub-invalid", "invalid@x.com", "")
	cookie := accountLogin(t, app, accountID)

	// Malformed set score ("abc" isn't "A-B") -> ParseSetScores error -> 400.
	data := url.Values{}
	data.Set("playerId", "whatever")
	data.Add("sets[]", "abc")
	req := httptest.NewRequest("POST", "/account/matches/m1/propose-score", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %v", resp.StatusCode)
	}
}

func TestPlayerHandler_LinkAndUnlinkAccount(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	// LinkAccount/UnlinkAccount are admin-protected routes.
	adminCookie := adminLogin(t, app)

	accountID := seedTestAccount(t, db, "sub-link", "link@x.com", "Guardian")
	repo := bunRepo.NewPlayerRepository(db)
	p, _ := player.NewPlayer(idgen.Generate(), "Adult", "Player", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC), "M", "NIC", "", "")
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("Save player: %v", err)
	}

	t.Run("link by email", func(t *testing.T) {
		data := url.Values{}
		data.Set("email", "link@x.com")
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/link-account", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}

		got, err := repo.GetById(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if got.GuardianAccountID == nil || *got.GuardianAccountID != accountID {
			t.Fatalf("expected player linked, got %+v", got.GuardianAccountID)
		}
	})

	t.Run("edit form shows linked account name and email", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/players/"+p.ID+"/edit", nil)
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
		if !strings.Contains(string(body), "Guardian") || !strings.Contains(string(body), "link@x.com") {
			t.Errorf("expected edit form to show linked account name and email, got: %s", body)
		}
	})

	t.Run("link with unknown email fails", func(t *testing.T) {
		data := url.Values{}
		data.Set("email", "nobody@x.com")
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/link-account", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("expected 400, got %v", resp.StatusCode)
		}
	})

	t.Run("unlink", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/players/"+p.ID+"/unlink-account", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200, got %v", resp.StatusCode)
		}

		got, err := repo.GetById(context.Background(), p.ID)
		if err != nil {
			t.Fatalf("GetById: %v", err)
		}
		if got.GuardianAccountID != nil {
			t.Fatalf("expected player unlinked, got %+v", got.GuardianAccountID)
		}
	})

	t.Run("unlink unknown player fails", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/players/does-not-exist/unlink-account", nil)
		req.Header.Set("Cookie", adminCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("expected 400, got %v", resp.StatusCode)
		}
	})
}

func TestMatchHandler_AdminConfirmProposal(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	adminCookie := adminLogin(t, app)

	accountA := seedTestAccount(t, db, "sub-admin-confirm-a", "aca@x.com", "")
	accountB := seedTestAccount(t, db, "sub-admin-confirm-b", "acb@x.com", "")
	p1 := seedLinkedPlayer(t, db, accountA, "PlayerOne")
	p2 := seedLinkedPlayer(t, db, accountB, "PlayerTwo")
	matchID := seedScheduledMatch(t, db, p1, p2)

	cookieA := accountLogin(t, app, accountA)
	data := url.Values{}
	data.Set("playerId", p1.ID)
	data.Add("sets[]", "11-5")
	data.Add("sets[]", "11-7")
	data.Add("sets[]", "11-9")
	req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieA)
	if resp, err := app.Test(req); err != nil || resp.StatusCode != 302 {
		t.Fatalf("propose failed: resp=%+v err=%v", resp, err)
	}

	req2 := httptest.NewRequest("POST", "/matches/"+matchID+"/confirm-proposal", nil)
	req2.Header.Set("Cookie", adminCookie)
	resp2, err := app.Test(req2)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp2.StatusCode != 200 {
		t.Errorf("expected 200, got %v", resp2.StatusCode)
	}
}

func TestAccountHandler_GoogleCallback_MissingCode(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	loginResp, err := app.Test(httptest.NewRequest("GET", "/account/google/login", nil))
	if err != nil {
		t.Fatalf("google login failed: %v", err)
	}
	var cookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			cookie = strings.Split(v, ";")[0]
		}
	}
	loc := loginResp.Header.Get("Location")
	u, err := url.Parse(loc)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	state := u.Query().Get("state")

	req := httptest.NewRequest("GET", "/account/google/callback?state="+state, nil)
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 (error re-render for missing code), got %v", resp.StatusCode)
	}
}

func TestAccountHandler_ShowMyInfo_UnknownAccount(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	cookie := accountLogin(t, app, "does-not-exist")

	req := httptest.NewRequest("GET", "/account/me", nil)
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 404 {
		t.Errorf("expected 404, got %v", resp.StatusCode)
	}
}

func TestAccountHandler_UpdateMyInfo_UnknownAccount(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	cookie := accountLogin(t, app, "does-not-exist")

	data := url.Values{}
	data.Set("name", "X")
	req := httptest.NewRequest("PUT", "/account/me", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %v", resp.StatusCode)
	}
}

func TestAccountHandler_EditAndUpdatePlayer_OwnershipAndNotFound(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountA := seedTestAccount(t, db, "sub-edit-a", "edita@x.com", "")
	accountB := seedTestAccount(t, db, "sub-edit-b", "editb@x.com", "")
	p1 := seedLinkedPlayer(t, db, accountA, "Owned")
	cookieB := accountLogin(t, app, accountB)

	t.Run("EditPlayer: 404 for nonexistent player", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/does-not-exist/edit", nil)
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})

	t.Run("EditPlayer: 403 for non-owner", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/account/players/"+p1.ID+"/edit", nil)
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Errorf("expected 403, got %v", resp.StatusCode)
		}
	})

	t.Run("UpdatePlayer: 404 for nonexistent player", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "X")
		req := httptest.NewRequest("PUT", "/account/players/does-not-exist", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})

	t.Run("UpdatePlayer: 403 for non-owner", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "Hijacked")
		req := httptest.NewRequest("PUT", "/account/players/"+p1.ID, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieB)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 403 {
			t.Errorf("expected 403, got %v", resp.StatusCode)
		}
	})
}

// TestAccountHandler_UpdatePlayer_PreservesFieldsNotOnTheForm guards against
// the guardian "edit player" form -- which has no secondName, secondLastName,
// whatsAppNumber, or nationalID inputs -- silently wiping those fields when
// a guardian saves an otherwise-unrelated change (e.g. birthdate).
func TestAccountHandler_UpdatePlayer_PreservesFieldsNotOnTheForm(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountA := seedTestAccount(t, db, "sub-preserve", "preserve@x.com", "")
	p := seedLinkedPlayer(t, db, accountA, "Kid")

	repo := bunRepo.NewPlayerRepository(db)
	p.SecondName = "Middle"
	p.SecondLastName = "Second"
	p.WhatsAppNumber = "+15551234567"
	p.NationalID = "ID999"
	if err := repo.Save(context.Background(), p); err != nil {
		t.Fatalf("Save: %v", err)
	}

	cookieA := accountLogin(t, app, accountA)

	data := url.Values{}
	data.Set("firstName", "Kid")
	data.Set("lastName", "Test")
	data.Set("birthdate", "2011-02-03")
	data.Set("gender", "M")
	data.Set("country", "NIC")
	data.Set("department", "Managua")
	req := httptest.NewRequest("PUT", "/account/players/"+p.ID, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieA)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		t.Fatalf("expected 200 or 302, got %v", resp.StatusCode)
	}

	got, err := repo.GetById(context.Background(), p.ID)
	if err != nil {
		t.Fatalf("GetById: %v", err)
	}
	if got.SecondName != "Middle" || got.SecondLastName != "Second" {
		t.Errorf("expected second name/last name preserved, got %q %q", got.SecondName, got.SecondLastName)
	}
	if got.WhatsAppNumber != "+15551234567" || got.NationalID != "ID999" {
		t.Errorf("expected whatsapp/nationalID preserved, got %q %q", got.WhatsAppNumber, got.NationalID)
	}
	if got.Birthdate.Format("2006-01-02") != "2011-02-03" {
		t.Errorf("expected birthdate actually updated, got %v", got.Birthdate)
	}
}

func TestAccountHandler_ConfirmAndRejectScore_ErrorPaths(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountA := seedTestAccount(t, db, "sub-cerr-a", "cerra@x.com", "")
	cookieA := accountLogin(t, app, accountA)

	t.Run("ConfirmScore: unknown player -> 404", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", "does-not-exist")
		req := httptest.NewRequest("POST", "/account/matches/m1/confirm-score", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieA)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})

	t.Run("RejectScore: unknown player -> 404", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", "does-not-exist")
		req := httptest.NewRequest("POST", "/account/matches/m1/reject-score", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", cookieA)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 404 {
			t.Errorf("expected 404, got %v", resp.StatusCode)
		}
	})
}

func TestAccountHandler_CreateChild_HXRequest(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountID := seedTestAccount(t, db, "sub-hx", "hx@x.com", "")
	cookie := accountLogin(t, app, accountID)

	data := url.Values{}
	data.Set("firstName", "Kid")
	data.Set("lastName", "Smith")
	data.Set("birthdate", "2012-01-01")
	data.Set("gender", "M")
	data.Set("country", "NIC")
	req := httptest.NewRequest("POST", "/account/players", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookie)
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 for HX-Request, got %v", resp.StatusCode)
	}
	if resp.Header.Get("HX-Redirect") != "/account" {
		t.Errorf("expected HX-Redirect to /account, got %v", resp.Header.Get("HX-Redirect"))
	}
}

func TestAccountHandler_ScoreProposalFlow_HXRequest(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountA := seedTestAccount(t, db, "sub-hxflow-a", "hxflowa@x.com", "")
	accountB := seedTestAccount(t, db, "sub-hxflow-b", "hxflowb@x.com", "")
	p1 := seedLinkedPlayer(t, db, accountA, "PlayerOne")
	p2 := seedLinkedPlayer(t, db, accountB, "PlayerTwo")
	matchID := seedScheduledMatch(t, db, p1, p2)

	cookieA := accountLogin(t, app, accountA)
	cookieB := accountLogin(t, app, accountB)

	data := url.Values{}
	data.Set("playerId", p1.ID)
	data.Add("sets[]", "11-5")
	data.Add("sets[]", "11-7")
	data.Add("sets[]", "11-9")
	req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieA)
	req.Header.Set("HX-Request", "true")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 || resp.Header.Get("HX-Redirect") != "/account/pending-matches" {
		t.Errorf("expected HX 200 + redirect header, got status=%d hx-redirect=%q", resp.StatusCode, resp.Header.Get("HX-Redirect"))
	}

	confirmData := url.Values{}
	confirmData.Set("playerId", p2.ID)
	confirmReq := httptest.NewRequest("POST", "/account/matches/"+matchID+"/confirm-score", strings.NewReader(confirmData.Encode()))
	confirmReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	confirmReq.Header.Set("Cookie", cookieB)
	confirmReq.Header.Set("HX-Request", "true")
	confirmResp, err := app.Test(confirmReq)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if confirmResp.StatusCode != 200 || confirmResp.Header.Get("HX-Redirect") != "/account/pending-matches" {
		t.Errorf("expected HX 200 + redirect header, got status=%d hx-redirect=%q", confirmResp.StatusCode, confirmResp.Header.Get("HX-Redirect"))
	}
}

func TestAccountHandler_RejectScore_HXRequest(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	accountA := seedTestAccount(t, db, "sub-hxrej-a", "hxreja@x.com", "")
	accountB := seedTestAccount(t, db, "sub-hxrej-b", "hxrejb@x.com", "")
	p1 := seedLinkedPlayer(t, db, accountA, "PlayerOne")
	p2 := seedLinkedPlayer(t, db, accountB, "PlayerTwo")
	matchID := seedScheduledMatch(t, db, p1, p2)

	cookieA := accountLogin(t, app, accountA)
	cookieB := accountLogin(t, app, accountB)

	data := url.Values{}
	data.Set("playerId", p1.ID)
	data.Add("sets[]", "11-5")
	data.Add("sets[]", "11-7")
	data.Add("sets[]", "11-9")
	req := httptest.NewRequest("POST", "/account/matches/"+matchID+"/propose-score", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Cookie", cookieA)
	if resp, err := app.Test(req); err != nil || resp.StatusCode != 302 {
		t.Fatalf("propose failed: resp=%+v err=%v", resp, err)
	}

	rejectData := url.Values{}
	rejectData.Set("playerId", p2.ID)
	rejectReq := httptest.NewRequest("POST", "/account/matches/"+matchID+"/reject-score", strings.NewReader(rejectData.Encode()))
	rejectReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectReq.Header.Set("Cookie", cookieB)
	rejectReq.Header.Set("HX-Request", "true")
	resp, err := app.Test(rejectReq)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 200 || resp.Header.Get("HX-Redirect") != "/account/pending-matches" {
		t.Errorf("expected HX 200 + redirect header, got status=%d hx-redirect=%q", resp.StatusCode, resp.Header.Get("HX-Redirect"))
	}
}

func TestAccountHandler_ConfirmScore_InvalidBody(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountID := seedTestAccount(t, db, "sub-cbad", "cbad@x.com", "")
	cookie := accountLogin(t, app, accountID)

	req := httptest.NewRequest("POST", "/account/matches/m1/confirm-score", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %v", resp.StatusCode)
	}
}

func TestAccountHandler_RejectScore_InvalidBody(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	accountID := seedTestAccount(t, db, "sub-rbad", "rbad@x.com", "")
	cookie := accountLogin(t, app, accountID)

	req := httptest.NewRequest("POST", "/account/matches/m1/reject-score", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}
	if resp.StatusCode != 400 {
		t.Errorf("expected 400, got %v", resp.StatusCode)
	}
}
