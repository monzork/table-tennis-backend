package identity

import "github.com/google/uuid"

type UUIDGenerator struct{}

func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

func (g *UUIDGenerator) Generate() string {
	return uuid.NewString()
}
