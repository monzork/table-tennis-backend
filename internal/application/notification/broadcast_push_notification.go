package notification

import (
	"encoding/json"
	"log/slog"
	domain "table-tennis-backend/internal/domain/notification"

	"github.com/SherClockHolmes/webpush-go"
)

type BroadcastPushNotificationUseCase struct {
	repo       domain.PushSubscriptionRepository
	vapidPub   string
	vapidPriv  string
}

func NewBroadcastPushNotificationUseCase(repo domain.PushSubscriptionRepository, pub, priv string) *BroadcastPushNotificationUseCase {
	return &BroadcastPushNotificationUseCase{
		repo:      repo,
		vapidPub:  pub,
		vapidPriv: priv,
	}
}

type PushMessage struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

func (uc *BroadcastPushNotificationUseCase) Execute(message PushMessage) error {
	subs, err := uc.repo.GetAll()
	if err != nil {
		return err
	}

	payload, err := json.Marshal(message)
	if err != nil {
		return err
	}

	for _, sub := range subs {
		s := &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys: webpush.Keys{
				P256dh: sub.P256dh,
				Auth:   sub.Auth,
			},
		}

		resp, err := webpush.SendNotification(payload, s, &webpush.Options{
			Subscriber:      "mailto:admin@table-tennis.local",
			VAPIDPublicKey:  uc.vapidPub,
			VAPIDPrivateKey: uc.vapidPriv,
			TTL:             30,
		})

		if err != nil {
			slog.Error("Failed to send push notification", "endpoint", sub.Endpoint, "err", err)
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 410 || resp.StatusCode == 404 {
			// Subscription has expired or is no longer valid
			_ = uc.repo.DeleteByEndpoint(sub.Endpoint)
		}
	}

	return nil
}
