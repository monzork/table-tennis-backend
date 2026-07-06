package bun

import (
	"table-tennis-backend/internal/domain/notification"
	"time"

	"github.com/uptrace/bun"
)

type PushSubscriptionModel struct {
	bun.BaseModel `bun:"table:push_subscriptions,alias:ps"`
	ID            string    `bun:"id,pk,type:uuid,default:gen_random_uuid()"`
	Endpoint      string    `bun:"endpoint,notnull"`
	P256dh        string    `bun:"p256dh,notnull"`
	Auth          string    `bun:"auth,notnull"`
	UserAgent     string    `bun:"user_agent"`
	CreatedAt     time.Time `bun:"created_at,nullzero,notnull,default:current_timestamp"`
}

func (m *PushSubscriptionModel) ToDomain() *notification.PushSubscription {
	return &notification.PushSubscription{
		ID:        m.ID,
		Endpoint:  m.Endpoint,
		P256dh:    m.P256dh,
		Auth:      m.Auth,
		UserAgent: m.UserAgent,
		CreatedAt: m.CreatedAt,
	}
}

func PushSubscriptionModelFromDomain(d *notification.PushSubscription) *PushSubscriptionModel {
	return &PushSubscriptionModel{
		ID:        d.ID,
		Endpoint:  d.Endpoint,
		P256dh:    d.P256dh,
		Auth:      d.Auth,
		UserAgent: d.UserAgent,
		CreatedAt: d.CreatedAt,
	}
}
