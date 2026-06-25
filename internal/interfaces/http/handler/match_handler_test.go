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
	
	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Alice", "Smith", time.Now(), "F", "", "", "")
	p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Bob", "Jones", time.Now(), "M", "", "", "")
	playerRepo.Save(ctx, p1)
	playerRepo.Save(ctx, p2)

	tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Tourney", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2})
	tournamentRepo.Save(ctx, tourney)

	m := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
	matchRepo.Save(ctx, m)

	t.Run("Update Score", func(t *testing.T) {
		data := url.Values{}
		data.Set("tournamentId", tourney.ID)
		data.Set("stage", "final")
		data.Add("scores[]", "11-9")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/matches/%s/score", m.ID), strings.NewReader(data.Encode()))
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
		data.Set("matchId", m.ID)
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

	t.Run("Public Score Update - Invalid PIN", func(t *testing.T) {
		mUUID, _ := uuid.Parse(m.ID)
		mModel, err := matchRepo.GetByID(ctx, mUUID)
		if err != nil {
			t.Fatalf("failed to get match: %v", err)
		}
		mModel.Pin = "5555"
		_, err = matchRepo.DB().NewUpdate().Model(mModel).WherePK().Column("pin").Exec(ctx)
		if err != nil {
			t.Fatalf("failed to update match pin: %v", err)
		}

		data := url.Values{}
		data.Set("matchId", m.ID)
		data.Set("tournamentId", tourney.ID)
		data.Set("stage", "final")
		data.Set("pin", "9999") // Invalid PIN
		data.Add("scores[]_a", "11")
		data.Add("scores[]_b", "7")

		req := httptest.NewRequest("POST", "/public/matches/score/update", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		bodyStr := buf.String()
		if !strings.Contains(bodyStr, "Invalid Verification PIN") {
			t.Errorf("expected PIN error message, got: %s", bodyStr)
		}
	})

	t.Run("Public Score Update - Valid Match PIN", func(t *testing.T) {
		data := url.Values{}
		data.Set("matchId", m.ID)
		data.Set("tournamentId", tourney.ID)
		data.Set("stage", "final")
		data.Set("pin", "5555") // Valid match PIN
		data.Add("scores[]_a", "11")
		data.Add("scores[]_b", "7")

		req := httptest.NewRequest("POST", "/public/matches/score/update", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Table Exclusivity and Override", func(t *testing.T) {
		// Create two scheduled matches
		m1 := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
		matchRepo.Save(ctx, m1)

		m2 := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
		matchRepo.Save(ctx, m2)

		// Start first match on Table 1 manually
		startData1 := url.Values{}
		startData1.Set("tableNumber", "1")
		req1 := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m1.ID), strings.NewReader(startData1.Encode()))
		req1.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req1.Header.Set("Cookie", sessionCookie)
		resp1, err := app.Test(req1)
		if err != nil || resp1.StatusCode != 200 {
			buf := new(bytes.Buffer)
			if resp1 != nil && resp1.Body != nil {
				buf.ReadFrom(resp1.Body)
			}
			t.Fatalf("failed to start first match (status %d): %s, err: %v", resp1.StatusCode, buf.String(), err)
		}

		// Verify table 1 is occupied
		mUUID1, _ := uuid.Parse(m1.ID)
		mModel1, _ := matchRepo.GetByID(ctx, mUUID1)
		if mModel1.TableNumber == nil || *mModel1.TableNumber != 1 {
			t.Errorf("expected table 1, got %v", mModel1.TableNumber)
		}

		// Try starting second match on Table 1 (occupied) -> should fail
		startData2 := url.Values{}
		startData2.Set("tableNumber", "1")
		req2 := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m2.ID), strings.NewReader(startData2.Encode()))
		req2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req2.Header.Set("Cookie", sessionCookie)
		resp2, err := app.Test(req2)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		hxTrigger := resp2.Header.Get("HX-Trigger")
		if !strings.Contains(hxTrigger, "Table 1 is currently occupied by another match!") {
			t.Errorf("expected occupied table toast in HX-Trigger, got: %s", hxTrigger)
		}

		mUUID2, _ := uuid.Parse(m2.ID)
		mModel2, _ := matchRepo.GetByID(ctx, mUUID2)
		if mModel2.Status != "scheduled" {
			t.Errorf("expected match 2 to remain scheduled, got status: %s", mModel2.Status)
		}

		// Try starting second match on Table 2 (free) -> should succeed
		startData3 := url.Values{}
		startData3.Set("tableNumber", "2")
		req3 := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", m2.ID), strings.NewReader(startData3.Encode()))
		req3.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req3.Header.Set("Cookie", sessionCookie)
		resp3, err := app.Test(req3)
		if err != nil || resp3.StatusCode != 200 {
			t.Errorf("expected start on table 2 to succeed, got %v", resp3.StatusCode)
		}
	})
}
