// Package pin defines the domain interface for generating match/event PINs.
package pin

// Generator is the domain contract for producing unique numeric PINs.
// Implementations must use a cryptographically secure source of randomness.
type Generator interface {
	// GenerateUnique returns a 4-digit PIN string (1000–9999) that is unique
	// among currently active (non-finished) entities tracked by the implementation.
	GenerateUnique() string
}
