package notification_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/notification"
)

type mockPushSubscriptionRepo struct {
	subs map[string]*notification.PushSubscription
}

func newMockRepo() *mockPushSubscriptionRepo {
	return &mockPushSubscriptionRepo{
		subs: make(map[string]*notification.PushSubscription),
	}
}

func (m *mockPushSubscriptionRepo) Save(sub *notification.PushSubscription) error {
	m.subs[sub.Endpoint] = sub
	return nil
}

func (m *mockPushSubscriptionRepo) GetAll() ([]*notification.PushSubscription, error) {
	var result []*notification.PushSubscription
	for _, sub := range m.subs {
		result = append(result, sub)
	}
	return result, nil
}

func (m *mockPushSubscriptionRepo) DeleteByEndpoint(endpoint string) error {
	delete(m.subs, endpoint)
	return nil
}

func TestPushSubscription_StructInitialization(t *testing.T) {
	now := time.Now()
	sub := &notification.PushSubscription{
		ID:        "sub-1",
		Endpoint:  "https://push.example.com/sub/1",
		P256dh:    "p256dh_key_data",
		Auth:      "auth_secret_data",
		UserAgent: "Mozilla/5.0",
		CreatedAt: now,
	}

	if sub.ID != "sub-1" {
		t.Errorf("expected ID 'sub-1', got '%s'", sub.ID)
	}
	if sub.Endpoint != "https://push.example.com/sub/1" {
		t.Errorf("expected Endpoint 'https://push.example.com/sub/1', got '%s'", sub.Endpoint)
	}
	if sub.P256dh != "p256dh_key_data" {
		t.Errorf("expected P256dh 'p256dh_key_data', got '%s'", sub.P256dh)
	}
	if sub.Auth != "auth_secret_data" {
		t.Errorf("expected Auth 'auth_secret_data', got '%s'", sub.Auth)
	}
	if sub.UserAgent != "Mozilla/5.0" {
		t.Errorf("expected UserAgent 'Mozilla/5.0', got '%s'", sub.UserAgent)
	}
	if !sub.CreatedAt.Equal(now) {
		t.Errorf("expected CreatedAt %v, got %v", now, sub.CreatedAt)
	}
}

func TestPushSubscriptionRepository_Interface(t *testing.T) {
	var repo notification.PushSubscriptionRepository = newMockRepo()

	sub := &notification.PushSubscription{
		ID:       "sub-1",
		Endpoint: "https://example.com/ep1",
	}

	if err := repo.Save(sub); err != nil {
		t.Fatalf("unexpected error saving sub: %v", err)
	}

	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("unexpected error getting all subs: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 subscription, got %d", len(all))
	}

	if err := repo.DeleteByEndpoint("https://example.com/ep1"); err != nil {
		t.Fatalf("unexpected error deleting sub: %v", err)
	}

	all, _ = repo.GetAll()
	if len(all) != 0 {
		t.Errorf("expected 0 subscriptions after delete, got %d", len(all))
	}
}
