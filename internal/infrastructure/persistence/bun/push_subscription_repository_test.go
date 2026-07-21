package bun_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/notification"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestPushSubscriptionRepository_SaveGetAllDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPushSubscriptionRepository(db)

	sub := &notification.PushSubscription{
		ID:        uuid.NewString(),
		Endpoint:  "https://push.example.com/abc",
		P256dh:    "key",
		Auth:      "auth",
		UserAgent: "test-agent",
		CreatedAt: time.Now(),
	}

	if err := repo.Save(sub); err != nil {
		t.Fatalf("Save: %v", err)
	}

	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 1 || all[0].Endpoint != sub.Endpoint {
		t.Fatalf("unexpected subscriptions: %+v", all)
	}

	if err := repo.DeleteByEndpoint(sub.Endpoint); err != nil {
		t.Fatalf("DeleteByEndpoint: %v", err)
	}

	all, err = repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll after delete: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected 0 subscriptions after delete, got %d", len(all))
	}
}

func TestPushSubscriptionRepository_Save_UpsertsOnConflict(t *testing.T) {
	db := setupTestDB(t)
	repo := bunRepo.NewPushSubscriptionRepository(db)

	sub := &notification.PushSubscription{
		ID:       uuid.NewString(),
		Endpoint: "https://push.example.com/xyz",
		P256dh:   "key1",
		Auth:     "auth1",
	}
	if err := repo.Save(sub); err != nil {
		t.Fatalf("Save: %v", err)
	}

	sub2 := &notification.PushSubscription{
		ID:       uuid.NewString(),
		Endpoint: "https://push.example.com/xyz",
		P256dh:   "key2",
		Auth:     "auth2",
	}
	if err := repo.Save(sub2); err != nil {
		t.Fatalf("Save (upsert): %v", err)
	}

	all, err := repo.GetAll()
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 1 || all[0].P256dh != "key2" {
		t.Fatalf("expected upserted subscription, got %+v", all)
	}
}
