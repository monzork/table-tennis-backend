package idgen_test

import (
	"testing"

	"table-tennis-backend/internal/domain/idgen"
)

type mockGenerator struct {
	nextID string
}

func (m *mockGenerator) Generate() string {
	return m.nextID
}

func TestGenerate_UnregisteredPanics(t *testing.T) {
	// Reset global generator to nil before testing panic behavior
	idgen.Register(nil)

	defer func() {
		r := recover()
		if r == nil {
			t.Errorf("expected Generate() to panic when generator is nil")
		} else if msg, ok := r.(string); !ok || msg != "ID generator not registered" {
			t.Errorf("expected panic message 'ID generator not registered', got %v", r)
		}
	}()

	idgen.Generate()
}

func TestGenerate_Success(t *testing.T) {
	mock := &mockGenerator{nextID: "custom-id-123"}
	idgen.Register(mock)

	got := idgen.Generate()
	if got != "custom-id-123" {
		t.Errorf("expected 'custom-id-123', got '%s'", got)
	}

	// Clean up global state after test
	idgen.Register(nil)
}
