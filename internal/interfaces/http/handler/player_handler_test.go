package handler_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"bytes"
	"mime/multipart"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func TestPlayerHandler(t *testing.T) {
	app, db, _, err := SetupTestApp()
	if err != nil {
		t.Fatalf("failed to setup test app: %v", err)
	}

	// Helper to login and get session cookie
	loginReq := httptest.NewRequest("POST", "/admin/login", strings.NewReader("username=admin&password=password"))
	loginReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	loginResp, _ := app.Test(loginReq)
	
	var sessionCookie string
	for _, v := range loginResp.Header.Values("Set-Cookie") {
		if strings.HasPrefix(v, "session_id=") {
			sessionCookie = strings.Split(v, ";")[0]
		}
	}

	t.Run("Create Player", func(t *testing.T) {
		data := url.Values{}
		data.Set("firstName", "John")
		data.Set("lastName", "Doe")
		data.Set("birthdate", "1990-01-01")
		data.Set("gender", "M")
		data.Set("country", "USA")

		req := httptest.NewRequest("POST", "/players", strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Update Player", func(t *testing.T) {
		// Fetch the player created in the previous step
		var pm bunRepo.PlayerModel
		err := db.NewSelect().Model(&pm).Where("first_name = ?", "John").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find seeded player: %v", err)
		}

		data := url.Values{}
		data.Set("firstName", "John Updated")
		data.Set("lastName", "Doe")
		data.Set("birthdate", "1990-01-01")
		data.Set("gender", "M")
		data.Set("country", "USA")

		req := httptest.NewRequest("PUT", fmt.Sprintf("/players/%s", pm.ID.String()), strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Delete Player", func(t *testing.T) {
		var pm bunRepo.PlayerModel
		err := db.NewSelect().Model(&pm).Where("first_name = ?", "John Updated").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find seeded player: %v", err)
		}

		req := httptest.NewRequest("DELETE", fmt.Sprintf("/players/%s", pm.ID.String()), bytes.NewReader([]byte{}))
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}
	})

	t.Run("Import Players CSV", func(t *testing.T) {
		csvContent := "first_name,last_name,birthdate,gender,country,department,singles_elo,doubles_elo,whatsapp_number,pin\n" +
			"Alice,Imported,1995-06-15,F,MEX,IT,1200,1150,+5212345678,9876\n"

		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file", "players.csv")
		if err != nil {
			t.Fatalf("failed to create form file: %v", err)
		}
		part.Write([]byte(csvContent))
		writer.Close()

		req := httptest.NewRequest("POST", "/players/import", &body)
		req.Header.Set("Content-Type", writer.FormDataContentType())
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		// Verify the imported player in DB
		var pm bunRepo.PlayerModel
		err = db.NewSelect().Model(&pm).Where("first_name = ? AND last_name = ?", "Alice", "Imported").Scan(context.Background())
		if err != nil {
			t.Fatalf("failed to find imported player: %v", err)
		}
		if pm.WhatsAppNumber != "+5212345678" {
			t.Errorf("expected WhatsAppNumber '+5212345678', got '%s'", pm.WhatsAppNumber)
		}
		if pm.Pin != "9876" {
			t.Errorf("expected Pin '9876', got '%s'", pm.Pin)
		}
	})

	t.Run("Search Player Case Insensitive", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/players/search?q=alice", nil)
		req.Header.Set("Cookie", sessionCookie)

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if resp.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", resp.StatusCode)
		}

		reqUpper := httptest.NewRequest("GET", "/players/search?q=ALICE", nil)
		reqUpper.Header.Set("Cookie", sessionCookie)

		respUpper, err := app.Test(reqUpper)
		if err != nil {
			t.Fatalf("test request failed: %v", err)
		}

		if respUpper.StatusCode != 200 {
			t.Errorf("expected 200 OK, got %v", respUpper.StatusCode)
		}
	})
}
