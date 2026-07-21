package pin_test

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"table-tennis-backend/internal/infrastructure/pin"
)

type mockDBChecker struct {
	takenPINs map[string]bool
	failErr   error
	calls     int
}

func (m *mockDBChecker) IsActivePINTaken(ctx context.Context, p string) (bool, error) {
	m.calls++
	if m.failErr != nil && m.calls == 1 {
		return false, m.failErr
	}
	return m.takenPINs[p], nil
}

func TestCryptoGenerator_GenerateUnique_NilChecker(t *testing.T) {
	gen := pin.NewCryptoGenerator(nil)
	p := gen.GenerateUnique()
	if len(p) != 4 {
		t.Errorf("expected 4-digit PIN, got %q", p)
	}
	val, err := strconv.Atoi(p)
	if err != nil || val < 1000 || val > 9999 {
		t.Errorf("expected PIN in range 1000-9999, got %q", p)
	}
}

func TestCryptoGenerator_GenerateUnique_WithChecker(t *testing.T) {
	checker := &mockDBChecker{
		takenPINs: make(map[string]bool),
	}
	gen := pin.NewCryptoGenerator(checker)

	p1 := gen.GenerateUnique()
	if len(p1) != 4 {
		t.Errorf("expected 4-digit PIN, got %q", p1)
	}

	// Mark p1 as taken so next call must retry until it gets a new one
	checker.takenPINs[p1] = true
	p2 := gen.GenerateUnique()
	if p1 == p2 {
		t.Errorf("expected different PIN after marking %s as taken", p1)
	}
}

func TestCryptoGenerator_GenerateUnique_CheckerErrorRetry(t *testing.T) {
	checker := &mockDBChecker{
		takenPINs: make(map[string]bool),
		failErr:   errors.New("db connection failed"),
	}
	gen := pin.NewCryptoGenerator(checker)

	p := gen.GenerateUnique()
	if len(p) != 4 {
		t.Errorf("expected 4-digit PIN, got %q", p)
	}
	if checker.calls < 2 {
		t.Errorf("expected at least 2 calls due to retry on DB error, got %d", checker.calls)
	}
}

func TestGenerateUniqueInBatch(t *testing.T) {
	used := make(map[string]bool)

	p1 := pin.GenerateUniqueInBatch(used)
	if len(p1) != 4 {
		t.Errorf("expected 4-digit PIN, got %q", p1)
	}
	if !used[p1] {
		t.Errorf("expected PIN %s to be marked in used map", p1)
	}

	p2 := pin.GenerateUniqueInBatch(used)
	if p1 == p2 {
		t.Errorf("expected distinct PINs in batch, got duplicate %s", p1)
	}
	if !used[p2] {
		t.Errorf("expected PIN %s to be marked in used map", p2)
	}
}

// generateRaw's crypto/rand error branch is not covered here: as of Go 1.24,
// crypto/rand.Read treats any read failure (including from a swapped-in
// rand.Reader mock) as fatal and crashes the process instead of returning an
// error, so that branch can't be exercised without killing the test binary.
