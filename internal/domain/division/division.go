package division

import (
	"context"
	"errors"
	"strings"
)

var (
	ErrInvalidName     = errors.New("division name is required")
	ErrInvalidEloRange = errors.New("min_elo must be less than max_elo when max_elo is set")
)

type Repository interface {
	Save(ctx context.Context, d *Division) error
	GetAll(ctx context.Context) ([]*Division, error)
	Delete(ctx context.Context, id string) error
	GetById(ctx context.Context, id string) (*Division, error)
}

type Division struct {
	ID           string
	Name         string
	DisplayOrder int
	MinElo       int16
	MaxElo       *int16 // nil means no upper limit (top division)
	Category     string // "singles", "doubles", or "both"
	Gender       string // "M", "F", or "both" (default) -- lets men's/women's bands use different Elo ranges over one shared Elo pool
	Color        string
}

func NewDivision(id, name string, displayOrder int, minElo int16, maxElo *int16, category, color string) (*Division, error) {
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
		ID:           id,
		Name:         name,
		DisplayOrder: displayOrder,
		MinElo:       minElo,
		MaxElo:       maxElo,
		Category:     category,
		Gender:       "both",
		Color:        color,
	}, nil
}

// MatchesGender reports whether a player of the given gender ("M"/"F") falls
// under this division. Divisions with no gender set (empty or "both") apply
// to everyone; empty is treated the same as "both" for backward compatibility
// with divisions/fixtures created before this field existed.
func (d *Division) MatchesGender(playerGender string) bool {
	if d.Gender == "" || strings.EqualFold(d.Gender, "both") {
		return true
	}
	return strings.EqualFold(d.Gender, playerGender)
}

// OnlyGendered returns just the divisions explicitly scoped to a single
// gender (Gender == "M" or "F"), dropping the legacy gender-agnostic
// ("both") bands. Used wherever only the newer per-gender division scheme
// should be shown (e.g. the new-tournament division picker, the public
// rankings distribution chart) -- the legacy rows are never deleted, since
// bracket rendering for tournaments already built against them still looks
// them up by name, so this filters display only, not the underlying data.
func OnlyGendered(divisions []*Division) []*Division {
	var out []*Division
	for _, d := range divisions {
		if strings.EqualFold(d.Gender, "M") || strings.EqualFold(d.Gender, "F") {
			out = append(out, d)
		}
	}
	return out
}

// ContainsElo checks if a given ELO rating falls within this division's range.
// MinElo is inclusive, MaxElo is exclusive — so a player at exactly MaxElo
// belongs to the next (higher) division whose MinElo equals this MaxElo.
func (d *Division) ContainsElo(elo int16) bool {
	if elo < d.MinElo {
		return false
	}
	if d.MaxElo != nil && elo >= *d.MaxElo {
		return false
	}
	return true
}
