package handler_test

import (
	"context"
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestPublicHandler_TournamentSelfRegistration(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	tournamentRepo := bunRepo.NewTournamentRepository(db)

	// Create a player that is already registered
	existingPlayer, err := playerDomain.NewPlayer(uuid.New().String(), "Jane", "Doe", time.Now().AddDate(-25, 0, 0), "F", "NIC", "", "")
	if err != nil {
		t.Fatalf("failed to create existing player: %v", err)
	}
	existingPlayer.UpdateSinglesElo(600)
	existingPlayer.UpdateDoublesElo(600)
	if err := playerRepo.Save(ctx, existingPlayer); err != nil {
		t.Fatalf("failed to save existing player: %v", err)
	}

	// Create a event that is open for registration
	tourney, err := tournamentDomain.NewTournament(uuid.New().String(), "Open Championship", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, nil, false)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}
	tourney.RegistrationOpen = true
	if err := tournamentRepo.Save(ctx, tourney); err != nil {
		t.Fatalf("failed to save event: %v", err)
	}

	t.Run("Register existing player", func(t *testing.T) {
		data := url.Values{}
		data.Set("tournamentId", tourney.ID)
		data.Set("firstName", "Jane")
		data.Set("lastName", "Doe")
		data.Set("country", "NIC")

		req := httptest.NewRequest("POST", "/events/register", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify participant was added to event in DB
		updatedTourney, err := tournamentRepo.GetByID(ctx, tourney.ID)
		if err != nil {
			t.Fatalf("failed to get event: %v", err)
		}

		found := false
		for _, p := range updatedTourney.Participants {
			if p.ID == existingPlayer.ID {
				found = true
				if p.SinglesElo != 600 || p.DoublesElo != 600 {
					t.Errorf("expected participant Elo to match existing player's Elo (600), got Singles=%v, Doubles=%v", p.SinglesElo, p.DoublesElo)
				}
			}
		}
		if !found {
			t.Errorf("expected existing player to be added as participant")
		}
	})

	t.Run("Register non-existent player (creates new player with starting ELO 500)", func(t *testing.T) {
		data := url.Values{}
		data.Set("tournamentId", tourney.ID)
		data.Set("firstName", "Bob")
		data.Set("lastName", "Newguy")
		data.Set("country", "CRC")

		req := httptest.NewRequest("POST", "/events/register", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify new player was created in DB
		players, err := playerRepo.GetAll(ctx)
		if err != nil {
			t.Fatalf("failed to list players: %v", err)
		}

		var createdPlayer *playerDomain.Player
		for _, p := range players {
			if p.FirstName == "Bob" && p.LastName == "Newguy" {
				createdPlayer = p
				break
			}
		}

		if createdPlayer == nil {
			t.Fatalf("expected new player to be created in DB")
		}

		if createdPlayer.Country != "CRC" {
			t.Errorf("expected country 'CRC', got '%s'", createdPlayer.Country)
		}

		if createdPlayer.SinglesElo != 500 || createdPlayer.DoublesElo != 500 {
			t.Errorf("expected new player to have starting Elo 500, got Singles=%v, Doubles=%v", createdPlayer.SinglesElo, createdPlayer.DoublesElo)
		}

		// Verify participant was added to event
		updatedTourney, err := tournamentRepo.GetByID(ctx, tourney.ID)
		if err != nil {
			t.Fatalf("failed to get event: %v", err)
		}

		found := false
		for _, p := range updatedTourney.Participants {
			if p.ID == createdPlayer.ID {
				found = true
				if p.SinglesElo != 500 || p.DoublesElo != 500 {
					t.Errorf("expected participant Elo to match starting Elo (500), got Singles=%v, Doubles=%v", p.SinglesElo, p.DoublesElo)
				}
			}
		}
		if !found {
			t.Errorf("expected created player to be added as participant")
		}
	})

	t.Run("Fail with single name (no last name)", func(t *testing.T) {
		data := url.Values{}
		data.Set("tournamentId", tourney.ID)
		data.Set("firstName", "Cher")
		data.Set("lastName", "")
		data.Set("country", "USA")

		req := httptest.NewRequest("POST", "/events/register", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		// Re-renders the registration page with error message, which returns a 200 page showing the error
		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Read response body to verify the error is presented
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("failed to read response body: %v", err)
		}
		bodyStr := string(bodyBytes)
		if !strings.Contains(bodyStr, "first and last name are required for registration") {
			t.Errorf("expected page to show error message, body was:\n%s", bodyStr)
		}
	})
}
