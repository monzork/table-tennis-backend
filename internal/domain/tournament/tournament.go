package tournament

import (
	"errors"
	"fmt"
	"sort"
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

type Group struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	Name         string
	Players      []*player.Player
	Matches      []Match
}

type Tournament struct {
	ID           uuid.UUID
	Name         string
	Type         string // "singles", "doubles", "teams"
	Format       string // "elimination", "groups_elimination"
	Participants []*player.Player
	StartDate    time.Time
	EndDate      time.Time
	Rules        []Rule
	Matches      []Match
	Groups       []Group
}

func NewTournament(name string, tournamentType string, format string, start, end time.Time, rules []Rule, participants []*player.Player) (*Tournament, error) {
	if end.Before(start) {
		return nil, ErrInvalidDates
	}
	if tournamentType == "" {
		tournamentType = "singles"
	}
	if format == "" {
		format = "elimination"
	}
	t := &Tournament{
		ID:           uuid.New(),
		Name:         name,
		Type:         tournamentType,
		Format:       format,
		Participants: participants,
		StartDate:    start,
		EndDate:      end,
		Rules:        rules,
		Matches:      []Match{},
		Groups:       []Group{},
	}

	if format == "groups_elimination" {
		if err := t.AutoAssignGroups(); err != nil {
			return nil, err
		}
	}

	return t, nil
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

func (t *Tournament) AutoAssignGroups() error {
	if t.Format != "groups_elimination" {
		return nil
	}
	// Number of participants
	n := len(t.Participants)
	if n == 0 {
		return nil
	}

	// WTT standard: groups of 3 or 4.
	// Let's aim for groups of 4 if possible, otherwise 3.
	numGroups := n / 4
	if n%4 != 0 {
		numGroups++
	}

	// Sort participants by Elo (descending)
	sort.Slice(t.Participants, func(i, j int) bool {
		if t.Type == "doubles" {
			return t.Participants[i].DoublesElo > t.Participants[j].DoublesElo
		}
		return t.Participants[i].SinglesElo > t.Participants[j].SinglesElo
	})

	t.Groups = make([]Group, numGroups)
	for i := 0; i < numGroups; i++ {
		t.Groups[i] = Group{
			ID:           uuid.New(),
			TournamentID: t.ID,
			Name:         fmt.Sprintf("Group %c", 'A'+i),
			Players:      []*player.Player{},
		}
	}

	// Snake seeding
	for i, p := range t.Participants {
		groupIndex := i % numGroups
		// In snake seeding:
		// Row 0: 0, 1, 2, 3
		// Row 1: 7, 6, 5, 4
		// Row 2: 8, 9, 10, 11
		row := i / numGroups
		if row%2 != 0 {
			groupIndex = numGroups - 1 - groupIndex
		}
		t.Groups[groupIndex].Players = append(t.Groups[groupIndex].Players, p)
	}

	return nil
}
