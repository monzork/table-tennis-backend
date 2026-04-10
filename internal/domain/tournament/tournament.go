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

// StageRule defines how many sets and points are played at a given tournament stage.
type StageRule struct {
	ID           uuid.UUID
	TournamentID uuid.UUID
	Stage        string // "group","r32","r16","quarterfinal","semifinal","final"
	BestOf       int    // e.g. 5 or 7
	PointsToWin  int    // e.g. 11
	PointsMargin int    // must win by this many (e.g. 2)
}

// DefaultStageRules returns WTT-standard rules for all 6 stages.
func DefaultStageRules(tournamentID uuid.UUID) []StageRule {
	short := []string{"group", "r32", "r16"}
	long := []string{"quarterfinal", "semifinal", "final"}
	rules := make([]StageRule, 0, 6)
	for _, s := range short {
		rules = append(rules, StageRule{ID: uuid.New(), TournamentID: tournamentID, Stage: s, BestOf: 5, PointsToWin: 11, PointsMargin: 2})
	}
	for _, s := range long {
		rules = append(rules, StageRule{ID: uuid.New(), TournamentID: tournamentID, Stage: s, BestOf: 7, PointsToWin: 11, PointsMargin: 2})
	}
	return rules
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

func (m Match) ScoreA() int {
	score := 0
	for _, s := range m.Sets {
		if s.ScoreA > s.ScoreB {
			score++
		}
	}
	return score
}

func (m Match) ScoreB() int {
	score := 0
	for _, s := range m.Sets {
		if s.ScoreB > s.ScoreA {
			score++
		}
	}
	return score
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
	EventCategory string // "men", "women", "mixed", "open"
	Format       string // "elimination", "groups_elimination", "round_robin"
	Status       string // "in_progress", "finished"
	Participants []*player.Player
	StartDate    time.Time
	EndDate      time.Time
	Rules        []Rule
	StageRules   []StageRule
	Matches      []Match
	Groups         []Group
	GroupPassCount int
	RegistrationOpen bool
}

func NewTournament(name string, tournamentType string, format string, category string, start, end time.Time, rules []Rule, groupPassCount int, participants []*player.Player) (*Tournament, error) {
	if end.Before(start) {
		return nil, ErrInvalidDates
	}
	if tournamentType == "" {
		tournamentType = "singles"
	}
	if format == "" {
		format = "elimination"
	}
	if category == "" {
		category = "open"
	}

	// Validation mapping mapping depending on tournament category
	for _, p := range participants {
		if category == "men" && p.Gender != "M" {
			return nil, errors.New("restricted: mens category cannot contain female players")
		}
		if category == "women" && p.Gender != "F" {
			return nil, errors.New("restricted: womens category cannot contain male players")
		}
	}

	t := &Tournament{
		ID:           uuid.New(),
		Name:         name,
		Type:         tournamentType,
		EventCategory: category,
		Format:       format,
		Participants: participants,
		StartDate:    start,
		EndDate:      end,
		Rules:        rules,
		Matches:      []Match{},
		Groups:         []Group{},
		GroupPassCount: groupPassCount,
		RegistrationOpen: false,
	}
	t.StageRules = DefaultStageRules(t.ID)

	if format == "groups_elimination" || format == "round_robin" {
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
	if t.Format != "groups_elimination" && t.Format != "round_robin" {
		return nil
	}
	// Number of participants
	n := len(t.Participants)
	if n == 0 {
		return nil
	}

	// Sort participants by Elo (descending)
	sort.Slice(t.Participants, func(i, j int) bool {
		if t.Type == "doubles" {
			return t.Participants[i].DoublesElo > t.Participants[j].DoublesElo
		}
		return t.Participants[i].SinglesElo > t.Participants[j].SinglesElo
	})

	if t.Format == "round_robin" {
		t.Groups = []Group{
			{
				ID:           uuid.New(),
				TournamentID: t.ID,
				Name:         "All Against All",
				Players:      t.Participants, // Everyone in one single group
			},
		}
		return nil
	}

	// WTT standard: groups of 3 or 4.
	// Let's aim for groups of 4 if possible, otherwise 3.
	numGroups := n / 4
	if n%4 != 0 {
		numGroups++
	}

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
		row := i / numGroups
		if row%2 != 0 {
			groupIndex = numGroups - 1 - groupIndex
		}
		t.Groups[groupIndex].Players = append(t.Groups[groupIndex].Players, p)
	}

	return nil
}
