package handler_test

import (
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

	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player1", time.Now(), "M", "", "", "")
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
		data.Add("participant_ids[]", p1.ID)

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

	t.Run("Export Tournament PDF", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/export/pdf", createdTournamentID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		if !strings.Contains(resp.Header.Get("Content-Type"), "application/pdf") {
			t.Errorf("expected application/pdf Content-Type, got %v", resp.Header.Get("Content-Type"))
		}
	})

	t.Run("Delete Tournament", func(t *testing.T) {
		tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Temp", "singles", "elimination", "open", time.Now(), time.Now(), []tournamentDomain.Rule{}, 2, nil, false)
		tournamentRepo.Save(ctx, tourney)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/tournaments/%s", tourney.ID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK for delete, got %v", resp.StatusCode)
		}
	})

	t.Run("Move Player Between Groups", func(t *testing.T) {
		p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player2", time.Now(), "M", "", "", "")
		playerRepo.Save(ctx, p2)
		p3, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player3", time.Now(), "M", "", "", "")
		playerRepo.Save(ctx, p3)
		p4, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player4", time.Now(), "M", "", "", "")
		playerRepo.Save(ctx, p4)
		p5, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player5", time.Now(), "M", "", "", "")
		playerRepo.Save(ctx, p5)

		data := url.Values{}
		data.Set("name", "Move Players Tourney")
		data.Set("type", "singles")
		data.Set("format", "groups_elimination")
		data.Set("startDate", time.Now().Format("2006-01-02"))
		data.Set("endDate", time.Now().Add(48*time.Hour).Format("2006-01-02"))
		data.Set("groupPassCount", "2")
		data.Set("skipElo", "true")
		data.Add("participant_ids[]", p1.ID)
		data.Add("participant_ids[]", p2.ID)
		data.Add("participant_ids[]", p3.ID)
		data.Add("participant_ids[]", p4.ID)
		data.Add("participant_ids[]", p5.ID)

		req := httptest.NewRequest("POST", "/tournaments", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to create tournament: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200 OK, got %v", resp.StatusCode)
		}

		var tm bunRepo.TournamentModel
		err = db.NewSelect().Model(&tm).Where("name = ?", "Move Players Tourney").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find tournament: %v", err)
		}

		tourney, err := tournamentRepo.GetByID(ctx, tm.ID.String())
		if err != nil {
			t.Fatalf("failed to load tourney domain: %v", err)
		}

		if len(tourney.Groups) < 2 {
			t.Fatalf("expected at least 2 groups, got %d", len(tourney.Groups))
		}

		groupA := tourney.Groups[0]
		groupB := tourney.Groups[1]
		if len(groupA.Players) == 0 {
			t.Fatalf("group A has no players")
		}
		movingPlayerID := groupA.Players[0].ID

		moveData := url.Values{}
		moveData.Set("playerId", movingPlayerID)
		moveData.Set("targetGroupId", groupB.ID)

		moveReq := httptest.NewRequest("POST", fmt.Sprintf("/admin/tournaments/%s/move-player", tourney.ID), strings.NewReader(moveData.Encode()))
		moveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		moveReq.Header.Set("Cookie", sessionCookie)

		moveResp, err := app.Test(moveReq)
		if err != nil {
			t.Fatalf("failed to post move: %v", err)
		}
		if moveResp.StatusCode != 200 {
			t.Fatalf("expected 200 OK for move, got %v", moveResp.StatusCode)
		}

		tourneyReloaded, err := tournamentRepo.GetByID(ctx, tm.ID.String())
		if err != nil {
			t.Fatalf("failed to reload tourney: %v", err)
		}

		var foundInA, foundInB bool
		for _, p := range tourneyReloaded.Groups[0].Players {
			if p.ID == movingPlayerID {
				foundInA = true
			}
		}
		for _, p := range tourneyReloaded.Groups[1].Players {
			if p.ID == movingPlayerID {
				foundInB = true
			}
		}

		if foundInA {
			t.Errorf("player was not removed from group A")
		}
		if !foundInB {
			t.Errorf("player was not added to group B")
		}
	})
}
