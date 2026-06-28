package qrcode

// Generator defines the interface for generating QR codes.
// It follows Domain-Driven Design (DDD) principles to decouple
// the domain layer from specific third-party implementations.
type Generator interface {
	// Generate creates a QR code from the given data and returns it as a PNG byte slice.
	Generate(data string, size int) ([]byte, error)
}
