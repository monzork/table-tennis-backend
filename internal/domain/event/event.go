package event

import (
	"context"
	"errors"
	"fmt"
	"table-tennis-backend/internal/domain/player"
	"time"
)

var ErrInvalidDates = errors.New("event end date must be after start date")

type Rule struct {
	ID          string
	Name        string
	Description string
}

// StageRule defines how many sets and points are played at a given event stage.
type StageRule struct {
	ID           string
	EventID      string
	Stage        string // "group","r32","r16","quarterfinal","semifinal","final"
	BestOf       int    // e.g. 5 or 7
	PointsToWin  int    // e.g. 11
	PointsMargin int    // must win by this many (e.g. 2)
}

// DefaultStageRules returns WTT-standard rules for all 6 stages.
func DefaultStageRules(eventID string) []StageRule {
	short := []string{"group", "r32", "r16"}
	long := []string{"quarterfinal", "semifinal", "final", "3rd_place"}
	rules := make([]StageRule, 0, 6)
	for _, s := range short {
		rules = append(rules, StageRule{ID: fmt.Sprintf("%s-%s", eventID, s), EventID: eventID, Stage: s, BestOf: 5, PointsToWin: 11, PointsMargin: 2})
	}
	for _, s := range long {
		rules = append(rules, StageRule{ID: fmt.Sprintf("%s-%s", eventID, s), EventID: eventID, Stage: s, BestOf: 7, PointsToWin: 11, PointsMargin: 2})
	}
	return rules
}

type Match struct {
	ID            string
	EventID       string
	MatchType     string // 'singles' or 'doubles'
	TeamA         []*player.Player
	TeamB         []*player.Player
	Status        string // scheduled, in_progress, finished
	WinnerTeam    string // 'A', 'B'
	Sets          []MatchSet
	TeamMatchID   *string
	Stage         string
	DivisionID    string // Division this match belongs to (for division-specific rules)
	UpdatedAt     *time.Time
	RefereeID     *string
	TableNumber   *int
	Pin           string
	RoundNumber   int
	QueuePosition int
	GroupID       string
	NextMatchID   string
	NextMatchSlot string
	// ProposedSets/ProposedByPlayerID/ProposedAt stage a player-submitted
	// score that has not yet been finalized. It sits here until either the
	// opposing player confirms it (or submits a correction) or an admin
	// verifies it directly — see MatchRepository.ProposeScore/ClearScoreProposal.
	ProposedSets       []MatchSet
	ProposedByPlayerID *string
	ProposedAt         *time.Time
	// EloDeltaA/EloDeltaB are the Elo points gained (positive) or lost
	// (negative) by team A/B from this specific match, set once Elo is
	// applied for the event (see FinishTournamentUseCase/
	// RecalculateTournamentEloUseCase). Nil when Elo hasn't been applied yet
	// (match unfinished, event has SkipElo, or match type "teams").
	EloDeltaA *float64
	EloDeltaB *float64
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

// ProposedWinner determines the winner ("A" or "B") implied by ProposedSets,
// using the same set-counting rule as ScoreA/ScoreB. Returns "" if there's no
// proposal or the proposed sets don't yet decide a winner.
func (m Match) ProposedWinner() string {
	scoreA, scoreB := 0, 0
	for _, s := range m.ProposedSets {
		diff := s.ScoreA - s.ScoreB
		if diff < 0 {
			diff = -diff
		}
		if (s.ScoreA >= 11 || s.ScoreB >= 11) && diff >= 2 {
			if s.ScoreA > s.ScoreB {
				scoreA++
			} else {
				scoreB++
			}
		}
	}
	if scoreA > scoreB {
		return "A"
	}
	if scoreB > scoreA {
		return "B"
	}
	return ""
}

type Group struct {
	ID      string
	EventID string
	Name    string
	Players []*player.Player
	Matches []Match
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

type Event struct {
	ID                    string
	Name                  string
	Type                  string // "singles", "doubles", "teams"
	EventCategory         string // "men", "women", "mixed", "open"
	Format                string // "elimination", "groups_elimination", "round_robin"
	Status                string // "in_progress", "finished"
	WinnerName            string // Name of the winner (player or team)
	Participants          []*player.Player
	StartDate             time.Time
	EndDate               time.Time
	Rules                 []Rule
	StageRules            []StageRule
	Matches               []Match
	Groups                []Group
	GroupPassCount        int
	LosersGroupPassCount  int
	RegistrationOpen      bool
	TournamentID          *string
	SkipElo               bool
	Teams                 []*Team
	TeamFormat            string // "olympic", "swaythling", or ""
	NumTables             int
	HasThirdPlaceMatch    bool
	Metrics               *TournamentMetrics
	ManualSeedingLocked   bool
	KnockoutBracketsCount int
	// SkipDivisionSplit marks events whose roster was hand-picked across Elo
	// bands (e.g. an "Open" tournament category) so the bracket view must not
	// re-bucket participants into per-division sub-brackets by their live Elo.
	SkipDivisionSplit bool
	// UseGenderDivisions marks events whose bracket should use gender-specific
	// division Elo bands (e.g. div-first-male/div-first-female) instead of the
	// shared gender-agnostic bands (div-first/div-second/...). Defaults false
	// so every event created before this field existed keeps its exact
	// existing bracket grouping.
	UseGenderDivisions bool
	// ParticipantSnapshots carries each participant's Elo before/after this
	// event. It is only populated by callers that need it (e.g. PDF export);
	// nil elsewhere.
	ParticipantSnapshots []ParticipantSnapshot
}

func NewEvent(id string, name string, tournamentType string, format string, category string, start, end time.Time, rules []Rule, groupPassCount int, participants []*player.Player, hasThirdPlaceMatch bool) (*Event, error) {
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

	// Validation mapping mapping depending on event category
	for _, p := range participants {
		if category == "men" && p.Gender != "M" {
			return nil, errors.New("restricted: mens category cannot contain female players")
		}
		if category == "women" && p.Gender != "F" {
			return nil, errors.New("restricted: womens category cannot contain male players")
		}
	}

	t := &Event{
		ID:                    id,
		Name:                  name,
		Type:                  tournamentType,
		EventCategory:         category,
		Format:                format,
		Participants:          participants,
		StartDate:             start,
		EndDate:               end,
		Rules:                 rules,
		Matches:               []Match{},
		Groups:                []Group{},
		GroupPassCount:        groupPassCount,
		RegistrationOpen:      false,
		TournamentID:          nil,
		SkipElo:               false,
		Teams:                 []*Team{},
		NumTables:             0,
		HasThirdPlaceMatch:    hasThirdPlaceMatch,
		KnockoutBracketsCount: 1,
	}
	t.StageRules = DefaultStageRules(t.ID)

	if format == "groups_elimination" || format == "round_robin" {
		if err := (&OpenBracketSnakeSeeder{}).AssignGroups(t); err != nil {
			return nil, err
		}
	}

	return t, nil
}

// GetEffectiveStageRule returns the stage rule to use for a match.
// Priority: Event Stage Rules > Default WTT Rules
func (t *Event) GetEffectiveStageRule(stage string) StageRule {
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

func (t *Event) AddMatch(match Match) {
	t.Matches = append(t.Matches, match)
}

func (t *Event) FindMatch(matchID string) (*Match, error) {
	for i := range t.Matches {
		if t.Matches[i].ID == matchID {
			return &t.Matches[i], nil
		}
	}
	return nil, errors.New("match not found")
}

// Remove a match
func (t *Event) RemoveMatch(matchID string) error {
	for i, m := range t.Matches {
		if m.ID == matchID {
			t.Matches = append(t.Matches[:i], t.Matches[i+1:]...)
			return nil
		}
	}
	return errors.New("match not found")
}

func (t *Event) MovePlayer(playerID string, targetGroupID string, targetIndex int) error {
	if t.ManualSeedingLocked {
		return errors.New("cannot move player: seeding is locked")
	}
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
			return errors.New("team is not registered in this event")
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
		return errors.New("player is not registered in this event")
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

var ErrTableOccupied = errors.New("table occupied by another in-progress match")

// EventRepository is the core CRUD interface for the Event aggregate root.
type EventRepository interface {
	Save(ctx context.Context, t *Event) error
	GetByID(ctx context.Context, id string) (*Event, error)
	GetAll(ctx context.Context) ([]*Event, error)
	Update(ctx context.Context, t *Event) error
	UpdateEventIDBulk(ctx context.Context, eventIDs []string, tournamentID string) error
	UpdateGroups(ctx context.Context, t *Event) error
	Delete(ctx context.Context, id string) error
	GetTournamentNumTables(ctx context.Context, tournamentID string) (int, error)
	GetByParticipantID(ctx context.Context, playerID string) ([]*Event, error)
}

// ParticipantRepository manages player participation and Elo snapshots within an event.
type ParticipantRepository interface {
	UpdateParticipantElo(ctx context.Context, eventID string, playerID string, singlesElo, doublesElo int16) error
	UpdateParticipantsElo(ctx context.Context, eventID string, players []*player.Player) error
	UpdateParticipantEloBefore(ctx context.Context, eventID string, playerID string, singlesElo, doublesElo int16) error
	AddParticipant(ctx context.Context, eventID string, playerID string, singlesElo, doublesElo int16) error
	RemoveParticipant(ctx context.Context, eventID string, playerID string) error
	GetParticipantSnapshots(ctx context.Context, eventID string) ([]ParticipantSnapshot, error)
	GetParticipantOrOfficialByPIN(ctx context.Context, eventID string, pin string) (string, error)
}

// TeamRepository manages teams and their player rosters (doubles/team-format events).
type TeamRepository interface {
	SaveTeam(ctx context.Context, team *Team) error
	DeleteTeam(ctx context.Context, id string) error
	AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error
	RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error
}

// OfficialRepository manages referees/officials assigned to a tournament. An
// official is scoped to the whole parent tournament (shared across all its
// child events/categories), not just the single event ID passed in — the
// implementation resolves that event's parent tournament internally.
type OfficialRepository interface {
	AddOfficial(ctx context.Context, eventID string, playerID string, pin string) error
	RemoveOfficial(ctx context.Context, eventID string, playerID string) error
	GetOfficials(ctx context.Context, eventID string) ([]ParticipantSnapshot, error)
	GetParticipantOrOfficialByPIN(ctx context.Context, eventID string, pin string) (string, error)
}

// Repository is the full Event aggregate repository. Prefer depending on the narrower
// EventRepository/ParticipantRepository/TeamRepository/OfficialRepository interfaces where a
// use case only needs one slice of this capability.
type Repository interface {
	EventRepository
	ParticipantRepository
	TeamRepository
	OfficialRepository
}

type MatchRepository interface {
	Save(ctx context.Context, m *Match) error
	CountUnfinishedMatches(ctx context.Context, eventID string) (int, error)
	CountFinishedMatches(ctx context.Context, eventID string) (int, error)
	GetAll(ctx context.Context) ([]*Match, error)
	GetByID(ctx context.Context, id string) (*Match, error)
	GetSubMatches(ctx context.Context, parentMatchID string) ([]*Match, error)
	GetMatchByParticipants(ctx context.Context, eventID, p1ID, p2ID, stage string) (*Match, error)
	GetInProgressMatchOnTable(ctx context.Context, tableNumber int, eventID, tournamentID string) (*Match, error)
	UpdateScore(ctx context.Context, id string, sets []MatchSet, stageRule StageRule) error
	// DoubleForfeit marks a match as a no-contest: both sides defaulted, so
	// neither wins, no Elo is applied, and no one advances in the bracket.
	DoubleForfeit(ctx context.Context, id string) error
	ProposeScore(ctx context.Context, matchID string, sets []MatchSet, proposedByPlayerID string, stageRule StageRule) error
	ClearScoreProposal(ctx context.Context, matchID string) error
	GetOccupiedTablesByEvent(ctx context.Context, eventID string) ([]int, error)
	GetOccupiedTablesByTournament(ctx context.Context, tournamentID string) ([]int, error)
	IsTableOccupiedByOtherMatch(ctx context.Context, matchID string, tableNumber int) (bool, error)
	UpdateMetadata(ctx context.Context, matchID string, refereeID *string, tableNumber *int) error
	ResetMatch(ctx context.Context, matchID string) error
	HasStartedOrFinishedMatches(ctx context.Context, eventID string) (bool, error)
	DeleteByEvent(ctx context.Context, eventID string) error
	FinishMatch(ctx context.Context, cmd FinishMatchCommand) error
	FindOrCreateMatch(ctx context.Context, eventID, p1ID, p2ID, stage, matchType string) (string, error)
	// Team match orchestration
	CreateSubMatches(ctx context.Context, cmd CreateSubMatchesCommand) error
	UpdateSubMatchSquads(ctx context.Context, cmd UpdateSubMatchSquadsCommand) error
	// UpdateEloDelta persists the per-match Elo points gained/lost by each
	// team, computed once Elo is applied for the event.
	UpdateEloDelta(ctx context.Context, matchID string, deltaA, deltaB *float64) error
}

// FinishMatchCommand carries all data needed to finish a match including bracket advancement.
type FinishMatchCommand struct {
	MatchID    string
	WinnerTeam string
}

// StartMatchCommand carries all inputs needed to atomically start a match.
type StartMatchCommand struct {
	MatchID        string
	EventID        string
	TableNumber    *int // nil = auto-assign
	TotalTables    int
	IsHighPriority bool // true for 1st division, semi/final stages
}

// StartMatchResult holds outcome of starting a match.
type StartMatchResult struct {
	TableNumber int
	Pin         string
	PlayerAName string
	PlayerBName string
}

// CreateSubMatchesCommand carries all data needed to create sub-matches for a team match.
type CreateSubMatchesCommand struct {
	ParentMatchID string
	EventID       string
	Stage         string
	TeamFormat    string   // "olympic" or "corbillon"
	TeamAPlayers  []string // player IDs
	TeamBPlayers  []string
}

// SubMatchSquadAssignment describes player assignments for a single sub-match.
type SubMatchSquadAssignment struct {
	SubMatchID     string
	TeamAPlayer1ID string
	TeamAPlayer2ID string // empty = no second player
	TeamBPlayer1ID string
	TeamBPlayer2ID string
}

// UpdateSubMatchSquadsCommand carries all player assignments for the sub-matches of a team match.
type UpdateSubMatchSquadsCommand struct {
	ParentMatchID string
	Assignments   []SubMatchSquadAssignment
}

func (t *Event) HasMatchesStarted() bool {
	for _, m := range t.Matches {
		if m.Status == "in_progress" || m.Status == "finished" {
			return true
		}
	}
	return false
}
