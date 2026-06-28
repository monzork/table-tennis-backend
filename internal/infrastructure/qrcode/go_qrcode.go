package qrcode

import (
	"github.com/skip2/go-qrcode"
)

// GoQRCodeGenerator implements the domain.Generator interface
// using the github.com/skip2/go-qrcode library.
type GoQRCodeGenerator struct{}

func NewGoQRCodeGenerator() *GoQRCodeGenerator {
	return &GoQRCodeGenerator{}
}

func (g *GoQRCodeGenerator) Generate(data string, size int) ([]byte, error) {
	// Medium recovery level gives a good balance between data capacity and error correction
	return qrcode.Encode(data, qrcode.Medium, size)
}
