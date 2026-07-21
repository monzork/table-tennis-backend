package handler_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestPublicHandler_SetLang(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	t.Run("Set valid lang es", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lang/es", nil)
		req.Header.Set("Referer", "/some-page")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}

		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected status 302, got %v", resp.StatusCode)
		}
		if loc := resp.Header.Get("Location"); loc != "/some-page" {
			t.Errorf("expected location /some-page, got %v", loc)
		}

		cookies := resp.Cookies()
		found := false
		for _, cookie := range cookies {
			if cookie.Name == "lang" {
				found = true
				if cookie.Value != "es" {
					t.Errorf("expected lang cookie value es, got %v", cookie.Value)
				}
			}
		}
		if !found {
			t.Errorf("expected lang cookie to be set")
		}
	})

	t.Run("Set invalid lang defaults to en", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lang/invalid", nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}

		if resp.StatusCode != http.StatusFound {
			t.Errorf("expected status 302, got %v", resp.StatusCode)
		}

		cookies := resp.Cookies()
		for _, cookie := range cookies {
			if cookie.Name == "lang" && cookie.Value != "en" {
				t.Errorf("expected lang cookie value en, got %v", cookie.Value)
			}
		}
	})

	t.Run("HX-Request sets HX-Redirect", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/lang/en", nil)
		req.Header.Set("HX-Request", "true")
		req.Header.Set("HX-Current-URL", "/hx-page")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", resp.StatusCode)
		}
		if hxRedir := resp.Header.Get("HX-Redirect"); hxRedir != "/hx-page" {
			t.Errorf("expected HX-Redirect /hx-page, got %v", hxRedir)
		}
	})
}

func TestPublicHandler_DepartmentInput(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	t.Run("Country Nicaragua", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/players/department-input?country=NIC", nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", resp.StatusCode)
		}
	})

	t.Run("Country other", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/players/department-input?country=USA", nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", resp.StatusCode)
		}
	})
}

func TestPublicHandler_ShowSignup(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/register", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %v", resp.StatusCode)
	}

	t.Run("With explicit lang cookie", func(t *testing.T) {
		reqEn := httptest.NewRequest(http.MethodGet, "/register", nil)
		reqEn.Header.Set("Cookie", "lang=en")
		respEn, err := app.Test(reqEn)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if respEn.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", respEn.StatusCode)
		}
	})
}

func TestPublicHandler_Register(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)

	t.Run("Valid registration", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "John")
		data.Set("lastName", "Doe")
		data.Set("country", "NIC")
		data.Set("birthdate", "1990-01-01")
		data.Set("gender", "M")

		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			t.Errorf("expected status 200, got %v, body: %s", resp.StatusCode, string(body))
		}

		players, _ := playerRepo.GetAll(ctx)
		found := false
		for _, p := range players {
			if p.FirstName == "John" && p.LastName == "Doe" {
				found = true
			}
		}
		if !found {
			t.Errorf("expected player to be created")
		}
	})

	t.Run("Honeypot filled", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "Bot")
		data.Set("lastName", "User")
		data.Set("website", "http://spam.com") // Honeypot

		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", resp.StatusCode)
		}

		players, _ := playerRepo.GetAll(ctx)
		for _, p := range players {
			if p.FirstName == "Bot" && p.LastName == "User" {
				t.Errorf("expected player NOT to be created due to honeypot")
			}
		}
	})

	t.Run("Invalid body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/register", strings.NewReader("{not-valid-json"))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode == http.StatusOK {
			t.Errorf("expected non-200 for invalid body, got %v", resp.StatusCode)
		}
	})
}

func TestPublicHandler_ShowTournamentRegistration(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/events/register", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %v", resp.StatusCode)
	}
}

func TestPublicHandler_ShowTournamentRegisterForm(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	ctx := context.Background()
	tournamentRepo := bunRepo.NewEventRepository(db)

	tourney, err := tournamentDomain.NewTournament(uuid.New().String(), "Open Championship 2", "singles", "elimination", "open", time.Now(), time.Now().Add(24*time.Hour), []tournamentDomain.Rule{}, 2, nil, false)
	if err != nil {
		t.Fatalf("failed to create event: %v", err)
	}
	tourney.RegistrationOpen = true
	if err := tournamentRepo.Save(ctx, tourney); err != nil {
		t.Fatalf("failed to save event: %v", err)
	}

	t.Run("Valid event ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/events/register/"+tourney.ID, nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", resp.StatusCode)
		}
	})

	t.Run("Invalid event ID", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/events/register/invalid-id", nil)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %v", resp.StatusCode)
		}
	})
}

func TestPublicHandler_Sitemap(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %v", resp.StatusCode)
	}
	if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "application/xml") {
		t.Errorf("expected Content-Type application/xml, got %v", contentType)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read body: %v", err)
	}
	if !strings.Contains(string(body), "<urlset") {
		t.Errorf("expected body to contain <urlset, got %v", string(body))
	}
}

func TestPublicHandler_RegisterToTournament_Honeypot(t *testing.T) {
	app, _, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	data := url.Values{}
	data.Set("tournamentId", "some-id")
	data.Set("firstName", "Bot")
	data.Set("lastName", "User")
	data.Set("website", "http://spam.com") // Honeypot

	req := httptest.NewRequest(http.MethodPost, "/events/register", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("test request failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %v", resp.StatusCode)
	}

	t.Run("Invalid body", func(t *testing.T) {
		reqBad := httptest.NewRequest(http.MethodPost, "/events/register", strings.NewReader("{not-valid-json"))
		reqBad.Header.Set("Content-Type", "application/json")

		respBad, err := app.Test(reqBad)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if respBad.StatusCode == http.StatusOK {
			t.Errorf("expected non-200 for invalid body, got %v", respBad.StatusCode)
		}
	})

	t.Run("Unknown tournament ID re-renders with error", func(t *testing.T) {
		dataErr := url.Values{}
		dataErr.Set("tournamentId", "does-not-exist")
		dataErr.Set("firstName", "Jane")
		dataErr.Set("lastName", "Doe")

		reqErr := httptest.NewRequest(http.MethodPost, "/events/register", strings.NewReader(dataErr.Encode()))
		reqErr.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		respErr, err := app.Test(reqErr)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}
		if respErr.StatusCode != http.StatusOK {
			t.Errorf("expected status 200 (re-rendered with error), got %v", respErr.StatusCode)
		}
	})
}
