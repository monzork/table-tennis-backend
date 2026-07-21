package pin_test

import (
	"testing"

	"table-tennis-backend/internal/domain/pin"
)

type mockPinGenerator struct {
	pins []string
	idx  int
}

func (m *mockPinGenerator) GenerateUnique() string {
	if m.idx < len(m.pins) {
		p := m.pins[m.idx]
		m.idx++
		return p
	}
	return "0000"
}

func TestPinGeneratorInterface(t *testing.T) {
	var gen pin.Generator = &mockPinGenerator{
		pins: []string{"1234", "5678"},
	}

	p1 := gen.GenerateUnique()
	if p1 != "1234" {
		t.Errorf("expected '1234', got '%s'", p1)
	}

	p2 := gen.GenerateUnique()
	if p2 != "5678" {
		t.Errorf("expected '5678', got '%s'", p2)
	}
}
