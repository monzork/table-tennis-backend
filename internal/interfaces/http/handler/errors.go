package handler

import (
	"errors"
	"strings"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/gofiber/fiber/v2"
)

// ErrorHandler maps domain and infrastructure errors to standard HTTP responses.
func ErrorHandler(err error) error {
	if err == nil {
		return nil
	}

	// If it's already a fiber error, pass it through
	if e, ok := err.(*fiber.Error); ok {
		return e
	}

	// Handle specific domain errors
	if errors.Is(err, tournament.ErrInvalidDates) {
		return ErrorHandler(err)
	}

	errMsg := err.Error()

	// Handle string matching for common validation/domain errors
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "no rows in result set") {
		return fiber.NewError(fiber.StatusNotFound, "Resource not found")
	}

	if strings.Contains(errMsg, "restricted:") || strings.Contains(errMsg, "already") || strings.Contains(errMsg, "cannot") {
		return fiber.NewError(fiber.StatusBadRequest, errMsg)
	}

	// Bun duplicate key constraint violation
	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "UNIQUE constraint") {
		return fiber.NewError(fiber.StatusConflict, "A resource with this identifier already exists")
	}

	// For everything else, return a generic 500 error.
	// You might want to log the original error here using a logger instance.
	return fiber.NewError(fiber.StatusInternalServerError, "Internal Server Error")
}
