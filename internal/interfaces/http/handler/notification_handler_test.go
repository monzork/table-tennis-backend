package handler_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	appNotification "table-tennis-backend/internal/application/notification"
	"table-tennis-backend/internal/domain/notification"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/gofiber/fiber/v2"
)

type mockPushRepo struct {
	saveErr   error
	getAllErr error
	subs      []*notification.PushSubscription
}

func (m *mockPushRepo) Save(sub *notification.PushSubscription) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.subs = append(m.subs, sub)
	return nil
}

func (m *mockPushRepo) GetAll() ([]*notification.PushSubscription, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.subs, nil
}

func (m *mockPushRepo) DeleteByEndpoint(endpoint string) error {
	return nil
}

func TestNotificationHandler_GetVAPIDPublicKey(t *testing.T) {
	vapidKey := "test-vapid-public-key-12345"
	mockRepo := &mockPushRepo{}
	broadcastUC := appNotification.NewBroadcastPushNotificationUseCase(mockRepo, vapidKey, "test-private-key")
	h := handler.NewNotificationHandler(mockRepo, vapidKey, broadcastUC)

	if got := h.GetVAPIDPublicKey(); got != vapidKey {
		t.Errorf("expected VAPID public key %q, got %q", vapidKey, got)
	}
}

func TestNotificationHandler_Subscribe(t *testing.T) {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}
	repo := bunRepo.NewPushSubscriptionRepository(db)
	broadcastUC := appNotification.NewBroadcastPushNotificationUseCase(repo, "pubkey", "privkey")
	h := handler.NewNotificationHandler(repo, "pubkey", broadcastUC)

	app := fiber.New()
	app.Post("/api/subscribe", h.Subscribe)

	t.Run("Valid subscription request", func(t *testing.T) {
		payload := map[string]interface{}{
			"endpoint": "https://push.example.com/sub/123",
			"keys": map[string]string{
				"p256dh": "key_p256dh_sample",
				"auth":   "key_auth_sample",
			},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/subscribe", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "TestBrowser/1.0")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}

		// Verify saved in DB
		subs, err := repo.GetAll()
		if err != nil {
			t.Fatalf("failed to get subs: %v", err)
		}
		if len(subs) != 1 {
			t.Fatalf("expected 1 subscription in repo, got %d", len(subs))
		}
		if subs[0].Endpoint != "https://push.example.com/sub/123" {
			t.Errorf("unexpected endpoint %q", subs[0].Endpoint)
		}
		if subs[0].UserAgent != "TestBrowser/1.0" {
			t.Errorf("unexpected User-Agent %q", subs[0].UserAgent)
		}
	})

	t.Run("Invalid JSON request body", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/subscribe", bytes.NewReader([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Repository error returns 500", func(t *testing.T) {
		errRepo := &mockPushRepo{saveErr: errors.New("db save error")}
		errHandler := handler.NewNotificationHandler(errRepo, "pubkey", broadcastUC)

		errApp := fiber.New()
		errApp.Post("/api/subscribe", errHandler.Subscribe)

		payload := map[string]interface{}{
			"endpoint": "https://push.example.com/sub/err",
			"keys": map[string]string{
				"p256dh": "p",
				"auth":   "a",
			},
		}
		body, _ := json.Marshal(payload)

		req := httptest.NewRequest(http.MethodPost, "/api/subscribe", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := errApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.StatusCode)
		}
	})
}

func TestNotificationHandler_Broadcast(t *testing.T) {
	mockRepo := &mockPushRepo{}
	broadcastUC := appNotification.NewBroadcastPushNotificationUseCase(mockRepo, "pubkey", "privkey")
	h := handler.NewNotificationHandler(mockRepo, "pubkey", broadcastUC)

	app := fiber.New()
	app.Post("/api/broadcast", h.Broadcast)

	t.Run("Valid broadcast message", func(t *testing.T) {
		msg := appNotification.PushMessage{
			Title: "Match Starting",
			Body:  "Table 1 is ready",
			URL:   "/matches/1",
		}
		body, _ := json.Marshal(msg)

		req := httptest.NewRequest(http.MethodPost, "/api/broadcast", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("Invalid broadcast JSON", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/broadcast", bytes.NewReader([]byte("{invalid")))
		req.Header.Set("Content-Type", "application/json")

		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", resp.StatusCode)
		}
	})

	t.Run("Broadcast use case error returns 500", func(t *testing.T) {
		errRepo := &mockPushRepo{getAllErr: errors.New("db read error")}
		errUC := appNotification.NewBroadcastPushNotificationUseCase(errRepo, "pubkey", "privkey")
		errHandler := handler.NewNotificationHandler(errRepo, "pubkey", errUC)

		errApp := fiber.New()
		errApp.Post("/api/broadcast", errHandler.Broadcast)

		msg := appNotification.PushMessage{Title: "T", Body: "B"}
		body, _ := json.Marshal(msg)

		req := httptest.NewRequest(http.MethodPost, "/api/broadcast", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")

		resp, err := errApp.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusInternalServerError {
			t.Errorf("expected status 500, got %d", resp.StatusCode)
		}
	})
}
