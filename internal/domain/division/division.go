package division

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrInvalidName     = errors.New("division name is required")
	ErrInvalidEloRange = errors.New("min_elo must be less than max_elo when max_elo is set")
)

type Division struct {
	ID           string
	Name         string
	DisplayOrder int
	MinElo       int16
	MaxElo       *int16 // nil means no upper limit (top division)
	Category     string // "singles", "doubles", or "both"
	Color        string
}

func NewDivision(name string, displayOrder int, minElo int16, maxElo *int16, category, color string) (*Division, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if maxElo != nil && minElo >= *maxElo {
		return nil, ErrInvalidEloRange
	}
	if category == "" {
		category = "both"
	}
	if color == "" {
		color = "#ffffff"
	}
	return &Division{
		ID:           uuid.New().String(),
		Name:         name,
		DisplayOrder: displayOrder,
		MinElo:       minElo,
		MaxElo:       maxElo,
		Category:     category,
		Color:        color,
	}, nil
}

// ContainsElo checks if a given ELO rating falls within this division's range.
func (d *Division) ContainsElo(elo int16) bool {
	if elo < d.MinElo {
		return false
	}
	if d.MaxElo != nil && elo > *d.MaxElo {
		return false
	}
	return true
}
