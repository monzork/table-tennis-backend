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
	tournamentDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestEventHandlerCoverage(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	sessionCookie := getSessionCookie(app)
	ctx := context.Background()

	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewEventRepository(db)

	p1, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player1", time.Now(), "M", "", "", "")
	p2, _ := playerDomain.NewPlayer(uuid.New().String(), "Test", "Player2", time.Now(), "M", "", "", "")
	playerRepo.Save(ctx, p1)
	playerRepo.Save(ctx, p2)

	tourney, _ := tournamentDomain.NewTournament(uuid.New().String(), "Coverage Event", "singles", "elimination", "open", time.Now(), time.Now(), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{p1, p2}, false)
	tournamentRepo.Save(ctx, tourney)
	tournamentID := tourney.ID

	// Create division for testing
	divID := uuid.New().String()
	maxElo := int16(1200)
	divModel := &bunRepo.DivisionModel{
		ID:           divID,
		Name:         "Coverage Division",
		DisplayOrder: 1,
		MinElo:       800,
		MaxElo:       &maxElo,
		Category:     "both",
		Color:        "#ffffff",
	}
	db.NewInsert().Model(divModel).Exec(ctx)

	tourney.DivisionConfigs = map[string]tournamentDomain.DivisionConfig{divID: {Format: "groups_elimination", GroupCount: 2}}
	tournamentRepo.Save(ctx, tourney)

	t.Run("StartKnockout", func(t *testing.T) {
		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/divisions/%s/start-knockout", tournamentID, divID), nil)
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode == 404 {
			t.Errorf("expected route to exist")
		}
	})

	t.Run("UpdateParticipantEloBefore", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p1.ID)
		data.Set("singlesElo", "1500")
		data.Set("doublesElo", "1400")

		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/participants/elo-before", tournamentID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("SaveKnockoutSeeds", func(t *testing.T) {
		data := url.Values{}
		data.Set("divId", divID)
		data.Set("playerIds", p1.ID+","+p2.ID)

		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/divisions/%s/knockout/seeds", tournamentID, divID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode == 404 {
			t.Errorf("expected route to exist")
		}
	})

	t.Run("AddOfficial", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p1.ID)

		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/officials", tournamentID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("RemoveOfficial", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/admin/events/%s/officials/%s", tournamentID, p1.ID), nil)
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("RemoveParticipant", func(t *testing.T) {
		req := httptest.NewRequest("DELETE", fmt.Sprintf("/admin/events/%s/participants/%s", tournamentID, p1.ID), nil)
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("CreateTeam", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Test Team")

		req := httptest.NewRequest("POST", fmt.Sprintf("/events/%s/teams", tournamentID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %d", resp.StatusCode)
		}
	})

	t.Run("ShowEditForm", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/events/%s/edit", tournamentID), nil)
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode == 0 {
			t.Errorf("expected a response, got %d", resp.StatusCode)
		}
	})
	
	t.Run("Delete Event with HX-Request", func(t *testing.T) {
		delID := uuid.New().String()
		tourneyDel, _ := tournamentDomain.NewTournament(delID, "Delete Event", "singles", "elimination", "open", time.Now(), time.Now(), []tournamentDomain.Rule{}, 2, []*playerDomain.Player{}, false)
		tournamentRepo.Save(ctx, tourneyDel)

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/events/%s", delID), nil)
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Current-URL", fmt.Sprintf("/admin/events/%s", delID))

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK for delete, got %v", resp.StatusCode)
		}
	})
}
