package tournament

import (
	"context"
	"errors"
	"fmt"
	"table-tennis-backend/internal/domain/player"
	"time"
)

var ErrInvalidDates = errors.New("tournament end date must be after start date")

type Rule struct {
	ID          string
	Name        string
	Description string
}

// StageRule defines how many sets and points are played at a given tournament stage.
type StageRule struct {
	ID           string
	TournamentID string
	Stage        string // "group","r32","r16","quarterfinal","semifinal","final"
	BestOf       int    // e.g. 5 or 7
	PointsToWin  int    // e.g. 11
	PointsMargin int    // must win by this many (e.g. 2)
}

// DefaultStageRules returns WTT-standard rules for all 6 stages.
func DefaultStageRules(tournamentID string) []StageRule {
	short := []string{"group", "r32", "r16"}
	long := []string{"quarterfinal", "semifinal", "final", "3rd_place"}
	rules := make([]StageRule, 0, 6)
	for _, s := range short {
		rules = append(rules, StageRule{ID: fmt.Sprintf("%s-%s", tournamentID, s), TournamentID: tournamentID, Stage: s, BestOf: 5, PointsToWin: 11, PointsMargin: 2})
	}
	for _, s := range long {
		rules = append(rules, StageRule{ID: fmt.Sprintf("%s-%s", tournamentID, s), TournamentID: tournamentID, Stage: s, BestOf: 7, PointsToWin: 11, PointsMargin: 2})
	}
	return rules
}

type Match struct {
	ID           string
	TournamentID string
	MatchType    string // 'singles' or 'doubles'
	TeamA        []*player.Player
	TeamB        []*player.Player
	Status       string // scheduled, in_progress, finished
	WinnerTeam   string // 'A', 'B'
	Sets         []MatchSet
	TeamMatchID  *string
	Stage        string
	DivisionID   string // Division this match belongs to (for division-specific rules)
	UpdatedAt    *time.Time
	RefereeID    *string
	TableNumber  *int
	Pin          string
	RoundNumber  int
	QueuePosition int
}

type MatchSet struct {
	Number int
	ScoreA int
	ScoreB int
}

func (m Match) ScoreA() int {
	if m.MatchType == "teams" && m.TeamMatchID == nil && len(m.Sets) == 1 {
		return m.Sets[0].ScoreA
	}
	score := 0
	for _, s := range m.Sets {
		diff := s.ScoreA - s.ScoreB
		if diff < 0 {
			diff = -diff
		}
		if (s.ScoreA >= 11 || s.ScoreB >= 11) && diff >= 2 {
			if s.ScoreA > s.ScoreB {
				score++
			}
		}
	}
	return score
}

func (m Match) ScoreB() int {
	if m.MatchType == "teams" && m.TeamMatchID == nil && len(m.Sets) == 1 {
		return m.Sets[0].ScoreB
	}
	score := 0
	for _, s := range m.Sets {
		diff := s.ScoreA - s.ScoreB
		if diff < 0 {
			diff = -diff
		}
		if (s.ScoreA >= 11 || s.ScoreB >= 11) && diff >= 2 {
			if s.ScoreB > s.ScoreA {
				score++
			}
		}
	}
	return score
}

type Group struct {
	ID           string
	TournamentID string
	Name         string
	Players      []*player.Player
	Matches      []Match
}

type EloUpset struct {
	MatchID    string `json:"matchId"`
	Difference int    `json:"difference"`
}

type DivisionMetric struct {
	TotalMatchesPlayed          int     `json:"totalMatchesPlayed"`
	AveragePointsPerMatch       float64 `json:"averagePointsPerMatch"`
	AverageMatchDurationSeconds int     `json:"averageMatchDurationSeconds"`
}

type TournamentMetrics struct {
	SchemaVersion int `json:"schemaVersion"`

	TotalMatchesPlayed int `json:"totalMatchesPlayed"`
	TotalSetsPlayed    int `json:"totalSetsPlayed"`
	TotalPointsScored  int `json:"totalPointsScored"`

	AveragePointsPerMatch float64 `json:"averagePointsPerMatch"`
	AverageSetsPerMatch   float64 `json:"averageSetsPerMatch"`

	CleanSweeps  int `json:"cleanSweeps"`
	DecidingSets int `json:"decidingSets"`
	Walkovers    int `json:"walkovers"`

	LongestMatchID              string `json:"longestMatchId,omitempty"`
	LongestMatchDurationSeconds int    `json:"longestMatchDurationSeconds"`

	AverageMatchDurationSeconds int `json:"averageMatchDurationSeconds"`

	AverageEloAtStart float64 `json:"averageEloAtStart"`

	MostEloGainedPlayerID string    `json:"mostEloGainedPlayerId,omitempty"`
	BiggestEloUpset       *EloUpset `json:"biggestEloUpset,omitempty"`

	DivisionMetrics map[string]DivisionMetric `json:"divisionMetrics,omitempty"`
}

type Tournament struct {
	ID                 string
	Name               string
	Type               string // "singles", "doubles", "teams"
	EventCategory      string // "men", "women", "mixed", "open"
	Format             string // "elimination", "groups_elimination", "round_robin"
	DivisionFormats    map[string]string // overrides Format per division
	DivisionGroupPassCounts map[string]int // overrides GroupPassCount per division
	Status             string // "in_progress", "finished"
	WinnerName         string // Name of the winner (player or team)
	Participants       []*player.Player
	StartDate          time.Time
	EndDate            time.Time
	Rules              []Rule
	StageRules         []StageRule
	DivisionRules      []DivisionRule // Division-specific rules override stage rules
	Matches            []Match
	Groups             []Group
	GroupPassCount     int
	RegistrationOpen   bool
	EventID            *string
	SkipElo            bool
	Teams              []*Team
	TeamFormat         string // "olympic", "swaythling", or ""
	NumTables          int
	HasThirdPlaceMatch bool
	Metrics            *TournamentMetrics
	ManualSeedingLocked bool
}

func NewTournament(id string, name string, tournamentType string, format string, category string, start, end time.Time, rules []Rule, groupPassCount int, participants []*player.Player, hasThirdPlaceMatch bool) (*Tournament, error) {
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
		ID:                 id,
		Name:               name,
		Type:               tournamentType,
		EventCategory:      category,
		Format:             format,
		Participants:       participants,
		StartDate:          start,
		EndDate:            end,
		Rules:              rules,
		Matches:            []Match{},
		Groups:             []Group{},
		GroupPassCount:     groupPassCount,
		RegistrationOpen:   false,
		EventID:            nil,
		SkipElo:            false,
		Teams:              []*Team{},
		NumTables:          0,
		HasThirdPlaceMatch: hasThirdPlaceMatch,
	}
	t.StageRules = DefaultStageRules(t.ID)

	if format == "groups_elimination" || format == "round_robin" {
		if err := (&OpenBracketSnakeSeeder{}).AssignGroups(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// GetDivisionFormat returns the specific format for a division if it exists, otherwise falls back to the global tournament format.
func (t *Tournament) GetDivisionFormat(divisionID string) string {
	if t.DivisionFormats != nil {
		if fmt, ok := t.DivisionFormats[divisionID]; ok && fmt != "" {
			return fmt
		}
	}
	return t.Format
}

// GetGroupPassCount returns the specific group pass count for a division if it exists, otherwise falls back to the global tournament pass count.
func (t *Tournament) GetGroupPassCount(divisionID string) int {
	if t.DivisionGroupPassCounts != nil {
		if count, ok := t.DivisionGroupPassCounts[divisionID]; ok && count > 0 {
			return count
		}
	}
	return t.GroupPassCount
}

// GetEffectiveStageRule returns the stage rule to use for a match, considering division overrides.
// Priority: Division Rules > Stage Rules > Default WTT Rules
func (t *Tournament) GetEffectiveStageRule(stage string, divisionID string) StageRule {
	// 1. Check division-specific rules first
	if divisionID != "" && len(t.DivisionRules) > 0 {
		for _, dr := range t.DivisionRules {
			if dr.DivisionID == divisionID && dr.Stage == stage {
				return dr.ToStageRule()
			}
		}
	}

	// 2. Check tournament stage rules
	for _, sr := range t.StageRules {
		if sr.Stage == stage {
			return sr
		}
	}

	// 3. Fallback to default WTT rules
	return StageRule{
		Stage:        stage,
		BestOf:       5,
		PointsToWin:  11,
		PointsMargin: 2,
	}
}

func (t *Tournament) AddMatch(match Match) {
	t.Matches = append(t.Matches, match)
}

func (t *Tournament) FindMatch(matchID string) (*Match, error) {
	for i := range t.Matches {
		if t.Matches[i].ID == matchID {
			return &t.Matches[i], nil
		}
	}
	return nil, errors.New("match not found")
}

// Remove a match
func (t *Tournament) RemoveMatch(matchID string) error {
	for i, m := range t.Matches {
		if m.ID == matchID {
			t.Matches = append(t.Matches[:i], t.Matches[i+1:]...)
			return nil
		}
	}
	return errors.New("match not found")
}

func (t *Tournament) MovePlayer(playerID string, targetGroupID string, targetIndex int) error {
	var movingPlayer *player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		var foundTeam *Team
		for _, team := range t.Teams {
			if team.ID == playerID {
				foundTeam = team
				break
			}
		}
		if foundTeam == nil {
			return errors.New("team is not registered in this tournament")
		}

		avgElo := foundTeam.AverageElo(t.Type)
		movingPlayer = &player.Player{
			ID:         foundTeam.ID,
			FirstName:  foundTeam.Name,
			LastName:   " (Team)",
			SinglesElo: avgElo,
			DoublesElo: avgElo,
		}
	} else {
		for _, p := range t.Participants {
			if p.ID == playerID {
				movingPlayer = p
				break
			}
		}
	}
	if movingPlayer == nil {
		return errors.New("player is not registered in this tournament")
	}


	foundSource := false
	var sourceGroupID string
	for i := range t.Groups {
		g := &t.Groups[i]
		for j, p := range g.Players {
			if p.ID == playerID {
				newPlayers := make([]*player.Player, 0, len(g.Players)-1)
				newPlayers = append(newPlayers, g.Players[:j]...)
				newPlayers = append(newPlayers, g.Players[j+1:]...)
				g.Players = newPlayers
				foundSource = true
				sourceGroupID = g.ID
				break
			}
		}
		if foundSource {
			break
		}
	}

	if targetGroupID == "" {
		targetGroupID = sourceGroupID
	}

	foundTarget := false
	for i := range t.Groups {
		g := &t.Groups[i]
		if g.ID == targetGroupID {
			for _, p := range g.Players {
				if p.ID == playerID {
					return errors.New("player is already in the target group")
				}
			}

			// Determine insertion index
			idx := targetIndex
			if idx < 0 || idx > len(g.Players) {
				idx = len(g.Players)
			}

			// Insert player at idx
			newPlayers := make([]*player.Player, 0, len(g.Players)+1)
			newPlayers = append(newPlayers, g.Players[:idx]...)
			newPlayers = append(newPlayers, movingPlayer)
			newPlayers = append(newPlayers, g.Players[idx:]...)
			g.Players = newPlayers
			foundTarget = true
			break
		}
	}

	if !foundTarget {
		return errors.New("target group not found")
	}

	return nil
}

type ParticipantSnapshot struct {
	PlayerID         string
	Pin              string
	EloBeforeSingles *int16
	EloAfterSingles  *int16
	EloBeforeDoubles *int16
	EloAfterDoubles  *int16
}

type Repository interface {
	Save(ctx context.Context, t *Tournament) error
	GetByID(ctx context.Context, id string) (*Tournament, error)
	GetAll(ctx context.Context) ([]*Tournament, error)
	Update(ctx context.Context, t *Tournament) error
	UpdateEventIDBulk(ctx context.Context, tournamentIDs []string, eventID string) error
	UpdateGroups(ctx context.Context, t *Tournament) error
	Delete(ctx context.Context, id string) error
	UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error
	UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*player.Player) error
	UpdateParticipantEloBefore(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error
	AddParticipant(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error
	RemoveParticipant(ctx context.Context, tournamentID string, playerID string) error
	SaveTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, id string) error
	AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error
	RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error
	GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]ParticipantSnapshot, error)
	GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error)
	AddOfficial(ctx context.Context, tournamentID string, playerID string, pin string) error
	RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error
	GetOfficials(ctx context.Context, tournamentID string) ([]ParticipantSnapshot, error)
}

type MatchRepository interface {
	Save(ctx context.Context, m *Match) error
	CountUnfinishedMatches(ctx context.Context, tournamentID string) (int, error)
	CountFinishedMatches(ctx context.Context, tournamentID string) (int, error)
	GetAll(ctx context.Context) ([]*Match, error)
	UpdateScore(ctx context.Context, id string, sets []MatchSet, stageRule StageRule) error
	GetOccupiedTablesByEvent(ctx context.Context, eventID string) ([]int, error)
	GetOccupiedTablesByTournament(ctx context.Context, tournamentID string) ([]int, error)
	HasStartedOrFinishedMatches(ctx context.Context, tournamentID string) (bool, error)
	DeleteByTournament(ctx context.Context, tournamentID string) error
}

func (t *Tournament) HasMatchesStarted() bool {
	for _, m := range t.Matches {
		if m.Status == "in_progress" || m.Status == "finished" {
			return true
		}
	}
	return false
}
