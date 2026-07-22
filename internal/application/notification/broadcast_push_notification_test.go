package notification_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"table-tennis-backend/internal/application/notification"
	domain "table-tennis-backend/internal/domain/notification"
)

type mockPushSubRepo struct {
	subs            []*domain.PushSubscription
	getAllErr       error
	deletedEndpoint string
}

func (m *mockPushSubRepo) Save(sub *domain.PushSubscription) error {
	m.subs = append(m.subs, sub)
	return nil
}

func (m *mockPushSubRepo) GetAll() ([]*domain.PushSubscription, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.subs, nil
}

func (m *mockPushSubRepo) DeleteByEndpoint(endpoint string) error {
	m.deletedEndpoint = endpoint
	var updated []*domain.PushSubscription
	for _, s := range m.subs {
		if s.Endpoint != endpoint {
			updated = append(updated, s)
		}
	}
	m.subs = updated
	return nil
}

func TestBroadcastPushNotification_GetAllError(t *testing.T) {
	repo := &mockPushSubRepo{
		getAllErr: errors.New("db error"),
	}
	uc := notification.NewBroadcastPushNotificationUseCase(repo, "pubkey", "privkey")

	err := uc.Execute(notification.PushMessage{
		Title: "Test",
		Body:  "Test Body",
		URL:   "https://example.com",
	})
	if err == nil {
		t.Fatal("expected error when repo GetAll fails, got nil")
	}
}

func TestBroadcastPushNotification_EmptySubs(t *testing.T) {
	repo := &mockPushSubRepo{
		subs: []*domain.PushSubscription{},
	}
	uc := notification.NewBroadcastPushNotificationUseCase(repo, "pubkey", "privkey")

	err := uc.Execute(notification.PushMessage{
		Title: "Test",
		Body:  "Test Body",
		URL:   "https://example.com",
	})
	if err != nil {
		t.Fatalf("expected no error with empty subscriptions, got: %v", err)
	}
}

func TestBroadcastPushNotification_WithSubs(t *testing.T) {
	repo := &mockPushSubRepo{
		subs: []*domain.PushSubscription{
			{
				ID:       "1",
				Endpoint: "https://example.com/push/1",
				P256dh:   "mock-p256dh",
				Auth:     "mock-auth",
			},
		},
	}
	uc := notification.NewBroadcastPushNotificationUseCase(repo, "pubkey", "privkey")

	// webpush.SendNotification will fail with mock keys/endpoint, but Execute handles errors and continues
	err := uc.Execute(notification.PushMessage{
		Title: "Match Starting",
		Body:  "Table 1: Player A vs Player B",
		URL:   "/matches/1",
	})
	if err != nil {
		t.Fatalf("expected Execute to complete without returning error, got: %v", err)
	}
}

func TestBroadcastPushNotification_SuccessAndExpired(t *testing.T) {
	// Create a test server to simulate webpush provider
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/push/success" {
			w.WriteHeader(http.StatusCreated)
		} else if r.URL.Path == "/push/expired" {
			w.WriteHeader(http.StatusGone) // 410
		} else {
			w.WriteHeader(http.StatusNotFound) // 404
		}
	}))
	defer ts.Close()

	// Valid base64url-encoded EC P-256 uncompressed point (65 bytes) and a
	// valid 16-byte auth secret; webpush-go decodes these before making the
	// HTTP request, so malformed values error out before the mock server
	// (and its 410 response) is ever reached.
	const testP256dh = "BKpskB45ypbSqppGWIugeBob9KliDblvFr1nScHAuo45glESdm5LkwXDcv17N3JeTJbusff0NSIfF7cm-cgrG_A"
	const testAuth = "0zKrnFGtUNa-rsqZZY36Ug"

	repo := &mockPushSubRepo{
		subs: []*domain.PushSubscription{
			{
				ID:       "1",
				Endpoint: ts.URL + "/push/success",
				P256dh:   testP256dh,
				Auth:     testAuth,
			},
			{
				ID:       "2",
				Endpoint: ts.URL + "/push/expired",
				P256dh:   testP256dh,
				Auth:     testAuth,
			},
		},
	}
	uc := notification.NewBroadcastPushNotificationUseCase(repo, "BOt8myi-n2seelwPVbG7qiCV-v79nvUXeHaBloLSWWEPYXTlKoiknq_crCMZfJ2H5683DeNeUFFsF6nBfGJ1zIo", "vTIcZmYZmWNcCHGqkppnEWI4i43_unbMj5xBIXBQzhY")

	err := uc.Execute(notification.PushMessage{
		Title: "Test",
		Body:  "Test Body",
		URL:   "/",
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}

	if repo.deletedEndpoint != ts.URL+"/push/expired" {
		t.Errorf("expected deleted endpoint %s, got %s", ts.URL+"/push/expired", repo.deletedEndpoint)
	}
}
