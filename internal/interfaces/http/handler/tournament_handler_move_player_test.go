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

func TestTournamentHandler_MovePlayer(t *testing.T) {
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

	// Seed player
	p, _ := playerDomain.NewPlayer(uuid.New().String(), "Jose", "Mena", time.Now(), "M", "", "", "")
	err = playerRepo.Save(ctx, p)
	if err != nil {
		t.Fatalf("failed to save player: %v", err)
	}

	// Seed event
	tourneyID := uuid.New().String()
	tourney := &tournamentDomain.Event{
		ID:           tourneyID,
		Name:         "Move Player Test Tourney",
		Status:       "scheduled",
		Format:       "groups_elimination",
		Participants: []*playerDomain.Player{p},
	}
	err = tournamentRepo.Save(ctx, tourney)
	if err != nil {
		t.Fatalf("failed to save event: %v", err)
	}

	// Seed Groups
	g1ID := uuid.New().String()
	g1 := tournamentDomain.Group{
		ID:      g1ID,
		Name:    "First Division - Group A",
		Players: []*playerDomain.Player{p},
	}
	g2ID := uuid.New().String()
	g2 := tournamentDomain.Group{
		ID:      g2ID,
		Name:    "First Division - Group B",
		Players: []*playerDomain.Player{},
	}
	tourney.Groups = []tournamentDomain.Group{g1, g2}
	err = tournamentRepo.UpdateGroups(ctx, tourney)
	if err != nil {
		t.Fatalf("failed to update groups: %v", err)
	}

	t.Run("Standard move-player request", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p.ID)
		data.Set("targetGroupId", g2ID)
		data.Set("targetIndex", "0")

		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/move-player", tourneyID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to test request: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("HTMX move-player request sets headers", func(t *testing.T) {
		data := url.Values{}
		data.Set("playerId", p.ID)
		data.Set("targetGroupId", g1ID) // move back to Group A

		req := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/move-player", tourneyID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to test request: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		triggerHeader := resp.Header.Get("HX-Trigger")
		if !strings.Contains(triggerHeader, "reload-bracket") || !strings.Contains(triggerHeader, "reload-matches") {
			t.Errorf("expected HX-Trigger to contain reload-bracket and reload-matches, got '%s'", triggerHeader)
		}
	})
}
