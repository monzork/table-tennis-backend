package bun_test

import (
	"testing"
	"time"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

func TestPlayerModel_FullName(t *testing.T) {
	m := &bunRepo.PlayerModel{
		ID:        uuid.New(),
		FirstName: "Ada",
		LastName:  "Lovelace",
		Birthdate: time.Now(),
	}
	if got := m.FullName(); got != "Ada Lovelace" {
		t.Fatalf("expected %q, got %q", "Ada Lovelace", got)
	}
}
