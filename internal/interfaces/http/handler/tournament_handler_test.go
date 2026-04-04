package handler_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestTournamentHandler(t *testing.T) {
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
	tournamentRepo := bunRepo.NewTournamentRepository(db)
	
	p1, _ := playerDomain.NewPlayer("Test", "Player1", time.Now(), "M", "")
	playerRepo.Save(ctx, p1)

	var createdTournamentID string

	t.Run("Create Tournament", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Grand Slam")
		data.Set("type", "singles")
		data.Set("format", "elimination")
		data.Set("startDate", time.Now().Format("2006-01-02"))
		data.Set("endDate", time.Now().Add(48*time.Hour).Format("2006-01-02"))
		data.Set("groupPassCount", "2")
		data.Add("participant_ids[]", p1.ID.String())

		req := httptest.NewRequest("POST", "/tournaments", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		var tm bunRepo.TournamentModel
		err = db.NewSelect().Model(&tm).Where("name = ?", "Grand Slam").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find tournament in DB: %v", err)
		}
		createdTournamentID = tm.ID.String()
	})

	t.Run("Finish Tournament", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/tournaments/%s/finish", createdTournamentID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v\n", resp.StatusCode)
		}

		var tm bunRepo.TournamentModel
		db.NewSelect().Model(&tm).Where("id = ?", createdTournamentID).Scan(context.Background())
		if tm.Status != "finished" {
			t.Errorf("expected status 'finished', got '%s'", tm.Status)
		}
	})

	t.Run("Export Tournament", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/export", createdTournamentID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
		
		if !strings.Contains(resp.Header.Get("Content-Type"), "text/csv") {
			t.Errorf("expected text/csv Content-Type, got %v", resp.Header.Get("Content-Type"))
		}
	})
	
	t.Run("Delete Tournament", func(t *testing.T) {
		tourney, _ := tournamentDomain.NewTournament("Temp", "singles", "elimination", "open", time.Now(), time.Now(), []tournamentDomain.Rule{}, 2, nil)
		tournamentRepo.Save(ctx, tourney)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/tournaments/%s", tourney.ID.String()), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK for delete, got %v", resp.StatusCode)
		}
	})
}
