package handler

import (
	"strconv"
	"table-tennis-backend/internal/domain/qrcode"

	"github.com/gofiber/fiber/v2"
)

type QRHandler struct {
	generator qrcode.Generator
}

func NewQRHandler(generator qrcode.Generator) *QRHandler {
	return &QRHandler{generator: generator}
}

func (h *QRHandler) Generate(c *fiber.Ctx) error {
	data := c.Query("data")
	if data == "" {
		return c.Status(fiber.StatusBadRequest).SendString("data parameter is required")
	}

	sizeStr := c.Query("size", "250")
	size, err := strconv.Atoi(sizeStr)
	if err != nil || size <= 0 || size > 1024 {
		size = 250 // default to 250 if invalid
	}

	pngBytes, err := h.generator.Generate(data, size)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("failed to generate QR code")
	}

	c.Set("Content-Type", "image/png")
	// Cache it for a while since it's static for the same URL
	c.Set("Cache-Control", "public, max-age=31536000") 
	return c.Send(pngBytes)
}
