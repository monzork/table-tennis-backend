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
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
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

	// Seed players & event for match
	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)

	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Alice", "Smith", time.Now(), "F", "", "", "")
	p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Bob", "Jones", time.Now(), "M", "", "", "")
	playerRepo.Save(ctx, p1)
	playerRepo.Save(ctx, p2)

	tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Tourney", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2}, false)
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

	t.Run("Public Score Update", func(t *testing.T) {
		data := url.Values{}
		data.Set("matchId", m.ID)
		data.Set("tournamentId", tourney.ID)
		data.Set("stage", "final")
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
		mModel1, _ := matchRepo.GetModelByID(ctx, mUUID1)
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
		mModel2, _ := matchRepo.GetModelByID(ctx, mUUID2)
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

	t.Run("Priority Table Assignment Heuristic", func(t *testing.T) {
		// Clear previously occupied tables
		matchRepo.DB().NewUpdate().Table("matches").Set("status = 'scheduled'").Exec(ctx)

		// Create a tournament with 4 tables
		tourney4, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Tourney 4", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 4, []*playerDomain.Player{p1, p2}, false)
		tournamentRepo.Save(ctx, tourney4)

		// Create a low priority match (group stage, non-1st division)
		mLow := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney4.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled", Stage: "group"}
		matchRepo.Save(ctx, mLow)

		// Start low priority match (auto-assign)
		reqLow := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", mLow.ID), strings.NewReader(url.Values{}.Encode()))
		reqLow.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqLow.Header.Set("Cookie", sessionCookie)
		respLow, _ := app.Test(reqLow)
		if respLow.StatusCode != 200 {
			t.Errorf("expected start low priority match to succeed, got %v", respLow.StatusCode)
		}

		mLowUUID, _ := uuid.Parse(mLow.ID)
		mLowModel, _ := matchRepo.GetModelByID(ctx, mLowUUID)
		if mLowModel.TableNumber == nil || *mLowModel.TableNumber < 3 {
			v := 0
			if mLowModel.TableNumber != nil {
				v = *mLowModel.TableNumber
			}
			t.Errorf("expected low priority match to be assigned table >= 3, got %d", v)
		}

		// Create a high priority match (final stage)
		mHigh := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney4.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled", Stage: "final"}
		matchRepo.Save(ctx, mHigh)

		// Start high priority match
		reqHigh := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/start", mHigh.ID), strings.NewReader(url.Values{}.Encode()))
		reqHigh.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqHigh.Header.Set("Cookie", sessionCookie)
		respHigh, _ := app.Test(reqHigh)
		if respHigh.StatusCode != 200 {
			t.Errorf("expected start high priority match to succeed, got %v", respHigh.StatusCode)
		}

		mHighUUID, _ := uuid.Parse(mHigh.ID)
		mHighModel, _ := matchRepo.GetModelByID(ctx, mHighUUID)
		if mHighModel.TableNumber == nil || *mHighModel.TableNumber != 1 {
			v := 0
			if mHighModel.TableNumber != nil {
				v = *mHighModel.TableNumber
			}
			t.Errorf("expected high priority match to be assigned table 1, got %d", v)
		}
	})
}
