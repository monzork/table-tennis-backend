// Package pin provides a cryptographically secure implementation of the
// domain/pin.Generator interface using crypto/rand.
package pin

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
)

// DBChecker is the minimal DB interface needed to verify PIN uniqueness.
// Keeping it narrow prevents a hard dependency on the full bun.DB type.
type DBChecker interface {
	// IsActivePINTaken returns true if the given PIN is already assigned to a
	// non-finished row in the relevant table.
	IsActivePINTaken(ctx context.Context, pin string) (bool, error)
}

// CryptoGenerator generates 4-digit PINs (1000–9999) using crypto/rand.
// It optionally accepts a DBChecker so it can verify uniqueness against the DB.
type CryptoGenerator struct {
	checker DBChecker // may be nil for batch/in-memory uniqueness only
}

// NewCryptoGenerator creates a CryptoGenerator that uses the provided checker
// to verify PIN uniqueness against the database.
// Pass nil if you only need in-memory deduplication (e.g. event batch).
func NewCryptoGenerator(checker DBChecker) *CryptoGenerator {
	return &CryptoGenerator{checker: checker}
}

// generateRaw produces a single random 4-digit PIN using crypto/rand.
func generateRaw() (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	pinVal := int(binary.BigEndian.Uint32(b[:]))%9000 + 1000
	return fmt.Sprintf("%d", pinVal), nil
}

// GenerateUnique returns a 4-digit PIN that is not currently in use.
// It loops until it finds a free PIN, performing a DB check on each attempt
// if a checker is registered.
func (g *CryptoGenerator) GenerateUnique() string {
	ctx := context.Background()
	for {
		pin, err := generateRaw()
		if err != nil {
			// crypto/rand failure is extremely unlikely; retry
			continue
		}
		if g.checker == nil {
			return pin
		}
		taken, err := g.checker.IsActivePINTaken(ctx, pin)
		if err == nil && !taken {
			return pin
		}
	}
}

// GenerateUniqueInBatch returns a 4-digit PIN not present in the provided
// usedPINs map, then marks it as used to prevent duplicates within the same batch.
// Use this when inserting multiple rows in one transaction (no DB round-trip needed).
func GenerateUniqueInBatch(usedPINs map[string]bool) string {
	for {
		pin, err := generateRaw()
		if err != nil {
			continue
		}
		if !usedPINs[pin] {
			usedPINs[pin] = true
			return pin
		}
	}
}
