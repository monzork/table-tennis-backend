package handler

import (
	"log/slog"
	appNotification "table-tennis-backend/internal/application/notification"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/notification"

	"github.com/gofiber/fiber/v2"
)

type NotificationHandler struct {
	repo        notification.PushSubscriptionRepository
	vapidPubKey string
	broadcastUC *appNotification.BroadcastPushNotificationUseCase
}

func NewNotificationHandler(repo notification.PushSubscriptionRepository, vapidPubKey string, broadcastUC *appNotification.BroadcastPushNotificationUseCase) *NotificationHandler {
	return &NotificationHandler{
		repo:        repo,
		vapidPubKey: vapidPubKey,
		broadcastUC: broadcastUC,
	}
}

func (h *NotificationHandler) GetVAPIDPublicKey() string {
	return h.vapidPubKey
}

func (h *NotificationHandler) Subscribe(c *fiber.Ctx) error {
	type WebPushSub struct {
		Endpoint string `json:"endpoint"`
		Keys     struct {
			P256dh string `json:"p256dh"`
			Auth   string `json:"auth"`
		} `json:"keys"`
	}

	var req WebPushSub
	if err := c.BodyParser(&req); err != nil {
		slog.Error("Failed to parse push subscription", "err", err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request"})
	}

	sub := notification.PushSubscription{
		ID:        idgen.Generate(),
		Endpoint:  req.Endpoint,
		P256dh:    req.Keys.P256dh,
		Auth:      req.Keys.Auth,
		UserAgent: c.Get("User-Agent"),
	}

	if err := h.repo.Save(&sub); err != nil {
		slog.Error("Failed to save push subscription", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusOK)
}

func (h *NotificationHandler) Broadcast(c *fiber.Ctx) error {
	var msg appNotification.PushMessage
	if err := c.BodyParser(&msg); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid message payload"})
	}

	if err := h.broadcastUC.Execute(msg); err != nil {
		slog.Error("Failed to broadcast push notification", "err", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.SendStatus(fiber.StatusOK)
}
