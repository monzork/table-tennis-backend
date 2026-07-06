package notification

import "time"

type PushSubscription struct {
	ID        string
	Endpoint  string
	P256dh    string
	Auth      string
	UserAgent string
	CreatedAt time.Time
}

type PushSubscriptionRepository interface {
	Save(sub *PushSubscription) error
	GetAll() ([]*PushSubscription, error)
	DeleteByEndpoint(endpoint string) error
}
