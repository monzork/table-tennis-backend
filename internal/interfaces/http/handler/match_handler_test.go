package handler_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestMatchHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, _ := app.Test(loginReq)
	
	var sessionCookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			sessionCookie = strings.Split(v, ";")[0]
		}
	}

	// Seed players & tournament for match
	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	
	p1, _ := playerDomain.NewPlayer("Alice", "Smith", time.Now(), "F", "")
	p2, _ := playerDomain.NewPlayer("Bob", "Jones", time.Now(), "M", "")
	playerRepo.Save(ctx, p1)
	playerRepo.Save(ctx, p2)

	tourney, _ := tournamentDomain.NewTournament("Test Tourney", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2})
	tournamentRepo.Save(ctx, tourney)

	m := &tournamentDomain.Match{ID: uuid.New(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
	matchRepo.Save(ctx, m)

	t.Run("Update Score", func(t *testing.T) {
		data := url.Values{}
		data.Set("tournamentId", tourney.ID.String())
		data.Set("stage", "final")
		data.Add("scores[]", "11-9")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", m.ID.String()), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Finish Match", func(t *testing.T) {
		data := url.Values{}
		data.Set("matchId", m.ID.String())
		data.Set("winnerTeam", "A")

		req := httptest.NewRequest("POST", "/matches/finish", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})
	
	t.Run("Invalid Score Update", func(t *testing.T) {
		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", uuid.New().String()), bytes.NewReader([]byte{}))
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode == 200 {
			t.Errorf("expected error code for missing match, got 200")
		}
	})
}
