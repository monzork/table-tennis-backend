package handler_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"

	tournamentDomain "table-tennis-backend/internal/domain/event"
	parentDomain "table-tennis-backend/internal/domain/tournament"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestTournamentHandler_PublicList(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	sessionCookie := getSessionCookie(app)
	ctx := context.Background()
	eventRepo := bunRepo.NewEventRepository(db)
	parentRepo := bunRepo.NewTournamentRepository(db, eventRepo)
	now := time.Now()

	parentID := uuid.New().String()
	if err := parentRepo.Save(ctx, &parentDomain.Tournament{
		ID: parentID, Name: "Listed Parent", StartDate: now, EndDate: now.Add(24 * time.Hour),
	}); err != nil {
		t.Fatalf("failed to save parent tournament: %v", err)
	}

	req := httptest.NewRequest("GET", "/public/tournaments", nil)
	req.Header.Set("Cookie", sessionCookie)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestEventHandler_PublicRedirectToTournament(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	sessionCookie := getSessionCookie(app)
	ctx := context.Background()
	eventRepo := bunRepo.NewEventRepository(db)
	parentRepo := bunRepo.NewTournamentRepository(db, eventRepo)
	now := time.Now()

	t.Run("redirects to the parent tournament when the event has one", func(t *testing.T) {
		parentID := uuid.New().String()
		if err := parentRepo.Save(ctx, &parentDomain.Tournament{
			ID: parentID, Name: "Parent", StartDate: now, EndDate: now.Add(24 * time.Hour),
		}); err != nil {
			t.Fatalf("failed to save parent tournament: %v", err)
		}

		ev, _ := tournamentDomain.NewEvent(uuid.New().String(), "Men's Singles", "singles", "elimination", "open", now, now.Add(24*time.Hour), nil, 1, nil, false)
		ev.TournamentID = &parentID
		if err := eventRepo.Save(ctx, ev); err != nil {
			t.Fatalf("failed to save event: %v", err)
		}

		req := httptest.NewRequest("GET", "/public/redirect/events/"+ev.ID, nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 302 {
			t.Fatalf("expected 302, got %d", resp.StatusCode)
		}
		if got := resp.Header.Get("Location"); got != "/tournaments/"+parentID {
			t.Fatalf("expected redirect to /tournaments/%s, got %s", parentID, got)
		}
	})

	t.Run("errors on an unknown event id", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/public/redirect/events/does-not-exist", nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode == 200 || resp.StatusCode == 302 {
			t.Fatalf("expected an error status, got %d", resp.StatusCode)
		}
	})

	t.Run("falls back to rendering the event when it has no parent tournament", func(t *testing.T) {
		ev, _ := tournamentDomain.NewEvent(uuid.New().String(), "Standalone Event", "singles", "elimination", "open", now, now.Add(24*time.Hour), nil, 1, nil, false)
		if err := eventRepo.Save(ctx, ev); err != nil {
			t.Fatalf("failed to save event: %v", err)
		}

		req := httptest.NewRequest("GET", "/public/redirect/events/"+ev.ID, nil)
		req.Header.Set("Cookie", sessionCookie)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("request failed: %v", err)
		}
		if resp.StatusCode != 200 {
			t.Fatalf("expected 200, got %d", resp.StatusCode)
		}
	})
}
