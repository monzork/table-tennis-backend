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
		var parentModels []bunRepo.TournamentModel
		_ = db.NewSelect().Model(&parentModels).Where("name = ?", "Spring Cup 2026").Scan(ctx)
		if len(parentModels) == 0 {
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
		var childModels []bunRepo.EventModel
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

	t.Run("Update Tournament", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Updated Cup")
		data.Set("startDate", "2026-05-02")
		data.Set("endDate", "2026-05-11")
		data.Set("numTables", "5")
		data.Set("priority_div-champ", "1,2")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/tournaments/%s", eventID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 && resp.StatusCode != 200 {
			t.Errorf("expected redirect or 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Public Detail", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/tournaments/%s/public", eventID), nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Export Event PDF", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/tournaments/%s/pdf", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Admin Board", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/board", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Public TV Dashboard", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/tournaments/%s/tv", eventID), nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Board Columns", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/tournaments/%s/board-columns", eventID), nil)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Tournament Health", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/health", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Tournament Health Metrics", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/health/metrics", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Show Edit Form", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/edit", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Delete Bulk", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Temp Cup")
		data.Set("divisionId", "div-champ")
		data.Set("skipElo", "on")
		data.Set("startDate", "2026-05-01")
		data.Set("endDate", "2026-05-10")
		data.Set("format", "elimination")
		reqCreate := httptest.NewRequest("POST", "/tournaments", strings.NewReader(data.Encode()))
		reqCreate.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		reqCreate.Header.Set("Cookie", sessionCookie)
		app.Test(reqCreate)
		
		var parentModels []bunRepo.TournamentModel
		db.NewSelect().Model(&parentModels).Where("name = ?", "Temp Cup").Scan(ctx)
		tempID := parentModels[0].ID.String()

		payload := fmt.Sprintf("{\"ids\":[\"%s\"]}", tempID)
		req := httptest.NewRequest("POST", "/tournaments/bulk-delete", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", sessionCookie)

		resp, _ := app.Test(req)
		if resp.StatusCode != 302 && resp.StatusCode != 200 {
			t.Errorf("expected redirect or 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Create Tournament With Arrays", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Array Cup")
		data.Add("divisionIds[]", "div-champ")
		data.Add("participantIdsSinglesMen[]", p1.ID)
		data.Add("existingTournamentIds[]", eventID)
		
		req := httptest.NewRequest("POST", "/tournaments", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		app.Test(req)
	})

	t.Run("Update Tournament HX Request", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Updated HX")
		data.Add("priority_div-champ", "1,2")
		
		req := httptest.NewRequest("PUT", fmt.Sprintf("/tournaments/%s", eventID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")
		app.Test(req)
	})

	t.Run("Delete Tournament HX Request", func(t *testing.T) {
		// create a temp tournament
		data := url.Values{}
		data.Set("name", "Temp HX Delete")
		req := httptest.NewRequest("POST", "/tournaments", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		app.Test(req)
		
		var t2Models []bunRepo.EventModel
		_ = db.NewSelect().Model(&t2Models).Where("name = ?", "Temp HX Delete").Scan(ctx)
		t2ID := uuid.New().String()
		if len(t2Models) > 0 {
			t2ID = t2Models[0].ID.String()
		}

		// Delete it with HX headers
		reqDel := httptest.NewRequest("DELETE", fmt.Sprintf("/tournaments/%s", t2ID), nil)
		reqDel.Header.Set("Cookie", sessionCookie)
		reqDel.Header.Set("HX-Request", "true")
		reqDel.Header.Set("HX-Current-URL", fmt.Sprintf("/admin/tournaments/%s", t2ID))
		app.Test(reqDel)
		
		// Delete with HX but no URL match
		reqDel2 := httptest.NewRequest("DELETE", fmt.Sprintf("/tournaments/%s", "non-existent"), nil)
		reqDel2.Header.Set("Cookie", sessionCookie)
		reqDel2.Header.Set("HX-Request", "true")
		app.Test(reqDel2)
	})

	t.Run("Delete Bulk HX Request", func(t *testing.T) {
		payload := `{"ids": ["123"]}`
		req := httptest.NewRequest("POST", "/tournaments/bulk-delete", strings.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Cookie", sessionCookie)
		req.Header.Set("HX-Request", "true")
		app.Test(req)
		
		// Invalid JSON
		req2 := httptest.NewRequest("POST", "/tournaments/bulk-delete", strings.NewReader(`{invalid`))
		req2.Header.Set("Content-Type", "application/json")
		req2.Header.Set("Cookie", sessionCookie)
		app.Test(req2)
	})

	t.Run("Admin Board With Matches", func(t *testing.T) {
		// Fetch a child tournament to add match
		var tModel bunRepo.EventModel
		_ = db.NewSelect().Model(&tModel).Where("tournament_id = ?", eventID).Limit(1).Scan(ctx)
		
		matchID := uuid.New().String()
		now := time.Now()
		// insert match into db
		matchUUID, _ := uuid.Parse(matchID)
		_, _ = db.NewInsert().Model(&bunRepo.MatchModel{
			ID: matchUUID,
			TournamentID: tModel.ID,
			TeamAPlayer1ID: uuid.MustParse(p1.ID),
			TeamBPlayer1ID: uuid.MustParse(p2.ID),
			Status: "in_progress",
			Stage: "groups",
			TableNumber: func() *int { i := 1; return &i }(),
			CreatedAt: now,
		}).Exec(ctx)

		// Also add a scheduled and a finished match so the AdminBoard "scheduled" and
		// "finished" card loops (which populate AllDivisions/AllCategories) get exercised too.
		scheduledMatchID := uuid.New()
		_, _ = db.NewInsert().Model(&bunRepo.MatchModel{
			ID:             scheduledMatchID,
			TournamentID:   tModel.ID,
			TeamAPlayer1ID: uuid.MustParse(p1.ID),
			TeamBPlayer1ID: uuid.MustParse(p2.ID),
			Status:         "scheduled",
			Stage:          "groups",
			CreatedAt:      now,
		}).Exec(ctx)

		winner := "A"
		finishedMatchID := uuid.New()
		_, _ = db.NewInsert().Model(&bunRepo.MatchModel{
			ID:             finishedMatchID,
			TournamentID:   tModel.ID,
			TeamAPlayer1ID: uuid.MustParse(p1.ID),
			TeamBPlayer1ID: uuid.MustParse(p2.ID),
			Status:         "finished",
			Stage:          "groups",
			WinnerTeam:     &winner,
			CreatedAt:      now,
		}).Exec(ctx)

		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/board?q=john&div=div-champ&cat=M", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		app.Test(req)
	})

	t.Run("Tournament Health Metrics Data", func(t *testing.T) {
		// Mock metrics data by manually modifying DB or just hitting endpoint again after match finishes
		// NOTE: EventRepository.GetByEventID (internal/infrastructure/persistence/bun/event_repository.go)
		// never copies EventModel.Metrics into the domain event.Event it builds, so this handler's
		// "e.Metrics != nil" branch can never be exercised through the public API today. That's a bug
		// in the persistence layer (outside this package's scope), not a test gap - tracked for a
		// follow-up fix there rather than worked around here.
		var tModel bunRepo.EventModel
		_ = db.NewSelect().Model(&tModel).Where("tournament_id = ?", eventID).Limit(1).Scan(ctx)

		metricsJSON := `{"TotalMatchesPlayed": 5, "TotalSetsPlayed": 15, "TotalPointsScored": 100, "CleanSweeps": 1, "DecidingSets": 2, "Walkovers": 0, "AverageMatchDurationSeconds": 600, "LongestMatchDurationSeconds": 1200, "LongestMatchID": "123"}`
		_, _ = db.NewUpdate().Table("events").Set("metrics = ?", metricsJSON).Where("id = ?", tModel.ID.String()).Exec(ctx)

		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/health/metrics", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Show Edit Form With Priorities", func(t *testing.T) {
		// DB already has priorities from Update HX request.
		// NOTE: TournamentRepository.GetByID never maps table_priorities back into the domain
		// Tournament (and TournamentModel has no such column at all), so the "TablePriorities != nil"
		// branch in ShowEditForm can't actually be exercised until that persistence gap is fixed -
		// that's outside this package's scope.
		req := httptest.NewRequest("GET", fmt.Sprintf("/admin/tournaments/%s/edit", eventID), nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, _ := app.Test(req)
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Error paths - invalid tournament ID", func(t *testing.T) {
		invalidID := "does-not-exist"
		doGet := func(route string) int {
			req := httptest.NewRequest("GET", route, nil)
			req.Header.Set("Cookie", sessionCookie)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("test request failed for %s: %v", route, err)
			}
			return resp.StatusCode
		}

		cases := []string{
			"/admin/tournaments/" + invalidID,
			"/tournaments/" + invalidID + "/public",
			"/tournaments/" + invalidID + "/pdf",
			"/admin/tournaments/" + invalidID + "/board",
			"/tournaments/" + invalidID + "/tv",
			"/admin/tournaments/" + invalidID + "/health",
			"/admin/tournaments/" + invalidID + "/health/metrics",
			"/admin/tournaments/" + invalidID + "/edit",
		}
		for _, route := range cases {
			if status := doGet(route); status == 200 {
				t.Errorf("expected non-200 for invalid ID route %s, got %v", route, status)
			}
		}

		// BoardColumns swallows the error and always renders 200 with an inline error message.
		if status := doGet("/tournaments/" + invalidID + "/board-columns"); status != 200 {
			t.Errorf("expected 200 (error rendered inline) for board-columns, got %v", status)
		}
	})

	t.Run("Board Columns with filters", func(t *testing.T) {
		req := httptest.NewRequest("GET", fmt.Sprintf("/tournaments/%s/board-columns?q=john&div=div-champ&cat=M", eventID), nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Update with malformed priority list", func(t *testing.T) {
		data := url.Values{}
		data.Set("name", "Priority Edge Case")
		data.Set("priority_div-champ", "1,,2, ,3")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/tournaments/%s", eventID), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if resp.StatusCode != 302 && resp.StatusCode != 200 {
			t.Errorf("expected redirect or 200 OK, got %v", resp.StatusCode)
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
		var parentModels []bunRepo.TournamentModel
		_ = db.NewSelect().Model(&parentModels).Where("id = ?", eventID).Scan(ctx)
		if len(parentModels) != 0 {
			t.Errorf("expected 0 parent tournaments, got %d", len(parentModels))
		}

		// Verify child events cascade-deleted from 'events' table (TournamentModel)
		var childModels []bunRepo.EventModel
		_ = db.NewSelect().Model(&childModels).Where("tournament_id = ?", eventID).Scan(ctx)
		if len(childModels) != 0 {
			t.Errorf("expected child events to be cascade-deleted, got %d remaining", len(childModels))
		}
	})
}
