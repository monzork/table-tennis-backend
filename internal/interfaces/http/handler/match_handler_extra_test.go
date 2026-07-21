package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
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

func TestMatchHandlerExtra(t *testing.T) {
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

	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewEventRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)

	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Alice", "Smith", time.Now(), "F", "", "", "")
	p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Bob", "Jones", time.Now(), "M", "", "", "")
	p3, _ := playerDomain.NewPlayer(uuid.New().String(), "Charlie", "Brown", time.Now(), "M", "", "", "")
	p4, _ := playerDomain.NewPlayer(uuid.New().String(), "Diana", "Prince", time.Now(), "F", "", "", "")
	playerRepo.Save(ctx, p1)
	playerRepo.Save(ctx, p2)
	playerRepo.Save(ctx, p3)
	playerRepo.Save(ctx, p4)

	tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Test Tourney", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2}, false)
	tourney.EventID = new(string)
	*tourney.EventID = uuid.New().String()
	tournamentRepo.Save(ctx, tourney)

	// Another tourney for teams
	tourneyTeams, _ := tournamentDomain.NewTournament(uuid.New().String(), "Teams Tourney", "teams", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2, p3, p4}, false)
	tournamentRepo.Save(ctx, tourneyTeams)

	m := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourney.ID, MatchType: "singles", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}, Status: "scheduled"}
	matchRepo.Save(ctx, m)

	t.Run("Create Match", func(t *testing.T) {
		body := map[string]interface{}{
			"tournamentId":   tourney.ID,
			"matchType":      "singles",
			"teamAPlayerIds": []string{p3.ID},
			"teamBPlayerIds": []string{p4.ID},
		}
		b, _ := json.Marshal(body)
		req := httptest.NewRequest("POST", "/matches/create", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("ShowScoreForm", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/matches/%s/score", m.ID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("ShowPublicScoreForm", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/public/matches/score/form?matchId=%s&tournamentId=%s", m.ID, tourney.ID), nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Reset Match", func(t *testing.T) {
		// Finish it first
		data := url.Values{}
		data.Set("matchId", m.ID)
		data.Set("winnerTeam", "A")
		reqFinish := httptest.NewRequest("POST", "/matches/finish", strings.NewReader(data.Encode()))
		reqFinish.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqFinish.Header.Set("Cookie", sessionCookie)
		app.Test(reqFinish)

		// Now reset
		req := httptest.NewRequest("POST", fmt.Sprintf("/matches/%s/reset", m.ID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify status
		mUUID, _ := uuid.Parse(m.ID)
		mModel, _ := matchRepo.GetModelByID(ctx, mUUID)
		if mModel.Status != "scheduled" {
			t.Errorf("expected scheduled, got %s", mModel.Status)
		}
	})

	t.Run("ShowMatchScorePage", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/public/score/%s", m.ID), nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("ShowTableScorePage No Match", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/public/score/table/1/tournament/%s", tourney.ID), nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("ShowTableScorePage With Match", func(t *testing.T) {
		// Set match to in_progress and table 999
		mUUID, _ := uuid.Parse(m.ID)
		mModel, _ := matchRepo.GetModelByID(ctx, mUUID)
		mModel.Status = "in_progress"
		tblNum := 999
		mModel.TableNumber = &tblNum
		matchRepo.DB().NewUpdate().Model(mModel).WherePK().Exec(ctx)

		req := httptest.NewRequest("GET", fmt.Sprintf("/public/score/table/999/tournament/%s", tourney.ID), nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("ValidateMatchPIN Invalid Match", func(t *testing.T) {
		data := url.Values{}
		data.Set("pin", "1234")
		req := httptest.NewRequest("POST", "/public/score/invalid-id/verify", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 400 {
			t.Errorf("expected 400 Bad Request, got %v", resp.StatusCode)
		}
	})

	t.Run("ValidateMatchPIN Valid", func(t *testing.T) {
		officialModel := &bunRepo.EventOfficialModel{
			TournamentID: uuid.MustParse(tourney.ID),
			PlayerID:     uuid.MustParse(p1.ID),
			Pin:          "1234",
		}
		matchRepo.DB().NewInsert().Model(officialModel).Exec(ctx)

		data := url.Values{}
		data.Set("pin", "1234")
		req := httptest.NewRequest("POST", fmt.Sprintf("/public/score/%s/verify", m.ID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		// Assuming status 200 means success
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Team Form Render", func(t *testing.T) {
		teamMatch := &tournamentDomain.Match{ID: uuid.New().String(), TournamentID: tourneyTeams.ID, MatchType: "teams", TeamA: []*playerDomain.Player{p1, p3}, TeamB: []*playerDomain.Player{p2, p4}, Status: "scheduled"}
		matchRepo.Save(ctx, teamMatch)

		// Admin form
		reqAdmin := httptest.NewRequest("GET", fmt.Sprintf("/matches/%s/score", teamMatch.ID), nil)
		reqAdmin.Header.Set("Cookie", sessionCookie)
		respAdmin, err := app.Test(reqAdmin)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if respAdmin.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", respAdmin.StatusCode)
		}

		// Public form
		reqPublic := httptest.NewRequest("GET", fmt.Sprintf("/public/matches/score/form?matchId=%s&tournamentId=%s", teamMatch.ID, tourneyTeams.ID), nil)
		respPublic, err := app.Test(reqPublic)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if respPublic.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", respPublic.StatusCode)
		}
	})
}
