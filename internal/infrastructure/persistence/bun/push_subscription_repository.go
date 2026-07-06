package bun

import (
	"context"
	"table-tennis-backend/internal/domain/notification"

	"github.com/uptrace/bun"
)

type PushSubscriptionRepository struct {
	db bun.IDB
}

func NewPushSubscriptionRepository(db bun.IDB) notification.PushSubscriptionRepository {
	return &PushSubscriptionRepository{db: db}
}

func (r *PushSubscriptionRepository) Save(sub *notification.PushSubscription) error {
	model := PushSubscriptionModelFromDomain(sub)
	_, err := r.db.NewInsert().
		Model(model).
		On("CONFLICT (endpoint) DO UPDATE").
		Set("p256dh = EXCLUDED.p256dh").
		Set("auth = EXCLUDED.auth").
		Set("user_agent = EXCLUDED.user_agent").
		Exec(context.Background())
	return err
}

func (r *PushSubscriptionRepository) GetAll() ([]*notification.PushSubscription, error) {
	var models []PushSubscriptionModel
	err := r.db.NewSelect().Model(&models).Scan(context.Background())
	if err != nil {
		return nil, err
	}

	subs := make([]*notification.PushSubscription, len(models))
	for i, m := range models {
		subs[i] = m.ToDomain()
	}
	return subs, nil
}

func (r *PushSubscriptionRepository) DeleteByEndpoint(endpoint string) error {
	_, err := r.db.NewDelete().
		Model((*PushSubscriptionModel)(nil)).
		Where("endpoint = ?", endpoint).
		Exec(context.Background())
	return err
}
