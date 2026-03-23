package tournament

import (
	"errors"
	"table-tennis-backend/internal/domain/player"
	"time"

	"github.com/google/uuid"
)

var ErrInvalidDates = errors.New("tournament end date must be after start date")

type Rule struct {
	ID          uuid.UUID
	Name        string
	Description string
}

type Match struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	MatchType    string // 'singles' or 'doubles'
	TeamA        []*player.Player
	TeamB        []*player.Player
	Status       string // scheduled, in_progress, finished
	WinnerTeam   string // 'A', 'B'
	Sets         []MatchSet
}

type MatchSet struct {
	Number int
	ScoreA int
	ScoreB int
}

type Tournament struct {
	ID        uuid.UUID
	Name      string
	Type      string // "singles", "doubles", "teams"
	StartDate time.Time
	EndDate   time.Time
	Rules     []Rule
	Matches   []Match
}

func NewTournament(name string, tournamentType string, start, end time.Time, rules []Rule) (*Tournament, error) {
	if end.Before(start) {
		return nil, ErrInvalidDates
	}
	if tournamentType == "" {
		tournamentType = "singles"
	}
	return &Tournament{
		ID:        uuid.New(),
		Name:      name,
		Type:      tournamentType,
		StartDate: start,
		EndDate:   end,
		Rules:     rules,
		Matches:   []Match{},
	}, nil
}

func (t *Tournament) AddMatch(match Match) {
	t.Matches = append(t.Matches, match)
}

func (t *Tournament) FindMatch(matchID uuid.UUID) (*Match, error) {
	for i := range t.Matches {
		if t.Matches[i].ID == matchID {
			return &t.Matches[i], nil
		}
	}
	return nil, errors.New("match not found")
}

// Remove a match
func (t *Tournament) RemoveMatch(matchID uuid.UUID) error {
	for i, m := range t.Matches {
		if m.ID == matchID {
			t.Matches = append(t.Matches[:i], t.Matches[i+1:]...)
			return nil
		}
	}
	return errors.New("match not found")
}
