package handler_test

import (
	"context"
	"fmt"
	"io"
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

// TestEventDetail_UnassignedPlayersTray covers the full HTTP path a real
// drag-and-drop in the browser exercises: an enrolled participant who was
// never placed in any group must render in an "unassigned players" tray on
// the admin event detail page, and dropping them onto an existing group
// (POST /move-player) must succeed even while ManualSeedingLocked is true —
// since only reshuffling an *already seeded* player should be blocked by the
// lock, not placing a straggler for the first time.
func TestEventDetail_UnassignedPlayersTray(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}
	sessionCookie := getSessionCookie(app)
	ctx := context.Background()

	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewEventRepository(db)

	seated, _ := playerDomain.NewPlayer(uuid.New().String(), "Seated", "Player", time.Now(), "M", "", "", "")
	stray, _ := playerDomain.NewPlayer(uuid.New().String(), "Stray", "Gomez", time.Now(), "M", "", "", "")
	if err := playerRepo.Save(ctx, seated); err != nil {
		t.Fatalf("failed to save seated player: %v", err)
	}
	if err := playerRepo.Save(ctx, stray); err != nil {
		t.Fatalf("failed to save stray player: %v", err)
	}

	tourneyID := uuid.New().String()
	tourney := &tournamentDomain.Event{
		ID:                  tourneyID,
		Name:                "Unassigned Tray Test",
		Status:              "scheduled",
		Format:              "groups_elimination",
		ManualSeedingLocked: true,
		Participants:        []*playerDomain.Player{seated, stray},
	}
	if err := tournamentRepo.Save(ctx, tourney); err != nil {
		t.Fatalf("failed to save event: %v", err)
	}

	g1ID := uuid.New().String()
	g1 := tournamentDomain.Group{ID: g1ID, Name: "Open - Group A", Players: []*playerDomain.Player{seated}}
	tourney.Groups = []tournamentDomain.Group{g1}
	if err := tournamentRepo.UpdateGroups(ctx, tourney); err != nil {
		t.Fatalf("failed to save groups: %v", err)
	}

	// The event detail page must render the tray with the stray player,
	// draggable, even though seeding is locked.
	req := httptest.NewRequest("GET", fmt.Sprintf("/admin/events/%s", tourneyID), nil)
	req.Header.Set("Cookie", sessionCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to request event detail: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	html := string(body)
	if !strings.Contains(html, "Stray Gomez") {
		t.Errorf("expected unassigned player 'Stray Gomez' to appear in the rendered page")
	}
	if !strings.Contains(html, "onDragStart(event, '"+stray.ID+"')") {
		t.Errorf("expected the stray player row to be draggable via onDragStart")
	}

	// Dropping the stray player onto the existing group must succeed even
	// though the event has ManualSeedingLocked=true.
	data := url.Values{}
	data.Set("playerId", stray.ID)
	data.Set("targetGroupId", g1ID)
	moveReq := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/move-player", tourneyID), strings.NewReader(data.Encode()))
	moveReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	moveReq.Header.Set("Cookie", sessionCookie)
	moveResp, err := app.Test(moveReq)
	if err != nil {
		t.Fatalf("failed to request move-player: %v", err)
	}
	if moveResp.StatusCode != 200 {
		moveBody, _ := io.ReadAll(moveResp.Body)
		t.Fatalf("expected 200 assigning unassigned player while locked, got %d: %s", moveResp.StatusCode, moveBody)
	}

	reloaded, err := tournamentRepo.GetByID(ctx, tourneyID)
	if err != nil {
		t.Fatalf("failed to reload event: %v", err)
	}
	found := false
	for _, g := range reloaded.Groups {
		if g.ID != g1ID {
			continue
		}
		for _, p := range g.Players {
			if p.ID == stray.ID {
				found = true
			}
		}
	}
	if !found {
		t.Errorf("expected stray player to be assigned into group %s after the drop", g1ID)
	}

	// Now confirm the lock still blocks reshuffling an already-seated player.
	data2 := url.Values{}
	data2.Set("playerId", seated.ID)
	data2.Set("targetGroupId", g1ID)
	reshuffleReq := httptest.NewRequest("POST", fmt.Sprintf("/admin/events/%s/move-player", tourneyID), strings.NewReader(data2.Encode()))
	reshuffleReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reshuffleReq.Header.Set("Cookie", sessionCookie)
	reshuffleResp, err := app.Test(reshuffleReq)
	if err != nil {
		t.Fatalf("failed to request move-player: %v", err)
	}
	if reshuffleResp.StatusCode == 200 {
		t.Errorf("expected reshuffling an already-seeded player to be rejected while locked")
	}
}
