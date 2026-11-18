package handler_test

import (
	"errors"
	"testing"

	"github.com/gofiber/fiber/v2"

	"table-tennis-backend/internal/interfaces/http/handler"
)

// Team-assignment validation failures are user errors, not server faults:
// they must surface as 400 with the real message (they used to be a bare 500).
func TestErrorHandler_TeamAssignmentValidationIsBadRequest(t *testing.T) {
	for _, msg := range []string{
		"Only female athletes are allowed in women's events",
		"Only male athletes are allowed in men's events",
		"doubles teams can only have a maximum of two players",
		"mixed doubles teams must consist of one male and one female player",
	} {
		var fe *fiber.Error
		if !errors.As(handler.ErrorHandler(errors.New(msg)), &fe) || fe.Code != fiber.StatusBadRequest || fe.Message != msg {
			t.Errorf("%q: expected 400 with message, got %v", msg, fe)
		}
	}
}
