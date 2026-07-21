package identity_test

import (
	"testing"

	"table-tennis-backend/internal/infrastructure/identity"

	"github.com/google/uuid"
)

func TestUUIDGenerator_Generate(t *testing.T) {
	gen := identity.NewUUIDGenerator()
	if gen == nil {
		t.Fatal("expected NewUUIDGenerator to return non-nil instance")
	}

	id1 := gen.Generate()
	if id1 == "" {
		t.Error("expected non-empty UUID string")
	}

	_, err := uuid.Parse(id1)
	if err != nil {
		t.Errorf("expected valid UUID, got error: %v", err)
	}

	id2 := gen.Generate()
	if id1 == id2 {
		t.Errorf("expected distinct UUIDs, got identical: %s", id1)
	}
}
