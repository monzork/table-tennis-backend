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
	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestEventHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	ctx := context.Background()

	// Seed division
	divRepo := bunRepo.NewDivisionRepository(db)
	maxEloVal := int16(3000)
	div := &division.Division{
		ID:           "div-champ",
		Name:         "Champion",
		DisplayOrder: 1,
		MinElo:       2000,
		MaxElo:       &maxEloVal,
		Category:     "both",
		Color:        "#ffffff",
	}
	_ = divRepo.Save(ctx, div)

	// Seed players
	playRepo := bunRepo.NewPlayerRepository(db)
	p1, _ := player.NewPlayer(uuid.New().String(), "John", "Doe", time.Now(), "M", "USA", "", "")
	p1.SinglesElo = 2100
	p1.DoublesElo = 2100
	_ = playRepo.Save(ctx, p1)

	p2, _ := player.NewPlayer(uuid.New().String(), "Jane", "Smith", time.Now(), "F", "CAN", "", "")
	p2.SinglesElo = 2200
	p2.DoublesElo = 2200
	_ = playRepo.Save(ctx, p2)

	// Login to get session
	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, _ := app.Test(loginReq)

	var sessionCookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			sessionCookie = strings.Split(v, ";")[0]
		}
	}

	var eventID string

	t.Run("Create Tournament", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Spring Cup 2026")
		data.Set("divisionId", "div-champ")
		data.Set("skipElo", "on")
		data.Set("startDate", "2026-05-01")
		data.Set("endDate", "2026-05-10")
		data.Set("format", "elimination")
		data.Set("autoSinglesMen", "on")
		data.Set("autoSinglesWomen", "on")
		data.Set("autoDoublesMixed", "on")
		data.Set("autoTeams", "on")
		data.Add("participant_ids[]", p1.ID)
		data.Add("participant_ids[]", p2.ID)

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

		// Verify the parent tournament was saved to the 'tournaments' table (EventModel)
		var parentModels []bunRepo.EventModel
		_ = db.NewSelect().Model(&parentModels).Scan(ctx)
		if len(parentModels) != 1 {
			t.Fatalf("expected 1 parent tournament in 'tournaments' table, got %d", len(parentModels))
		}
		if parentModels[0].Name != "Spring Cup 2026" {
			t.Errorf("expected tournament name Spring Cup 2026, got %s", parentModels[0].Name)
		}
		if !parentModels[0].SkipElo {
			t.Errorf("expected tournament skip_elo to be true")
		}

		eventID = parentModels[0].ID.String()

		// Verify child events were generated in the 'events' table (TournamentModel)
		var childModels []bunRepo.TournamentModel
		_ = db.NewSelect().Model(&childModels).Where("tournament_id = ?", eventID).Scan(ctx)
		if len(childModels) < 2 {
			t.Errorf("expected at least 2 child events in 'events' table, got %d", len(childModels))
		}

		for _, cm := range childModels {
			if !cm.SkipElo {
				t.Errorf("expected child event %s skip_elo to be true", cm.Name)
			}
			if cm.EventID == nil || cm.EventID.String() != eventID {
				t.Errorf("expected child event %s to reference parent tournament %s", cm.Name, eventID)
			}
		}
	})

	t.Run("Get Tournament Detail", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Search Selection Cards", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/players/search/cards?gender=M&selectAll=true", nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Delete Tournament", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/tournaments/%s", eventID), bytes.NewReader([]byte{}))
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify parent tournament deleted from 'tournaments' table (EventModel)
		var parentModels []bunRepo.EventModel
		_ = db.NewSelect().Model(&parentModels).Scan(ctx)
		if len(parentModels) != 0 {
			t.Errorf("expected 0 parent tournaments, got %d", len(parentModels))
		}

		// Verify child events cascade-deleted from 'events' table (TournamentModel)
		var childModels []bunRepo.TournamentModel
		_ = db.NewSelect().Model(&childModels).Where("tournament_id = ?", eventID).Scan(ctx)
		if len(childModels) != 0 {
			t.Errorf("expected child events to be cascade-deleted, got %d remaining", len(childModels))
		}
	})
}
