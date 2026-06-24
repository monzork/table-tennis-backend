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

	t.Run("Create Event", func(t *testing.T) {
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

		req := httptest.NewRequest("POST", "/events", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify event was saved and tournaments were generated in DB
		var eventModels []bunRepo.EventModel
		_ = db.NewSelect().Model(&eventModels).Scan(ctx)
		if len(eventModels) != 1 {
			t.Fatalf("expected 1 event in db, got %d", len(eventModels))
		}
		if eventModels[0].Name != "Spring Cup 2026" {
			t.Errorf("expected event name Spring Cup 2026, got %s", eventModels[0].Name)
		}
		if !eventModels[0].SkipElo {
			t.Errorf("expected event skip_elo to be true")
		}

		eventID = eventModels[0].ID.String()

		// Verify tournaments generated
		var tourneyModels []bunRepo.TournamentModel
		_ = db.NewSelect().Model(&tourneyModels).Scan(ctx)
		if len(tourneyModels) < 2 {
			t.Errorf("expected at least 2 categories auto-provisioned, got %d", len(tourneyModels))
		}

		for _, tm := range tourneyModels {
			if !tm.SkipElo {
				t.Errorf("expected child tournament %s skip_elo to be true", tm.Name)
			}
			if tm.EventID == nil || tm.EventID.String() != eventID {
				t.Errorf("expected child tournament %s to reference event %s", tm.Name, eventID)
			}
		}
	})

	t.Run("Get Event Detail", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/events/%s", eventID), nil)
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

	t.Run("Delete Event", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/events/%s", eventID), bytes.NewReader([]byte{}))
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify event is deleted in DB
		var eventModels []bunRepo.EventModel
		_ = db.NewSelect().Model(&eventModels).Scan(ctx)
		if len(eventModels) != 0 {
			t.Errorf("expected 0 events in db, got %d", len(eventModels))
		}

		// Verify child tournaments are also cascade-deleted
		var tourneyModels []bunRepo.TournamentModel
		_ = db.NewSelect().Model(&tourneyModels).Scan(ctx)
		if len(tourneyModels) != 0 {
			t.Errorf("expected child tournaments to be cascade-deleted, got %d remaining", len(tourneyModels))
		}
	})
}
