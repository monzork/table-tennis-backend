package tournament

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"table-tennis-backend/internal/domain/idgen"
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
	UpdatedAt    *time.Time
	RefereeID    *string
	TableNumber  *int
	Pin          string
	RoundNumber  int
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
		if s.ScoreA > s.ScoreB {
			score++
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
		if s.ScoreB > s.ScoreA {
			score++
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

type Tournament struct {
	ID           string
	Name         string
	Type         string // "singles", "doubles", "teams"
	EventCategory string // "men", "women", "mixed", "open"
	Format       string // "elimination", "groups_elimination", "round_robin"
	Status       string // "in_progress", "finished"
	WinnerName   string // Name of the winner (player or team)
	Participants []*player.Player
	StartDate    time.Time
	EndDate      time.Time
	Rules        []Rule
	StageRules   []StageRule
	Matches      []Match
	Groups         []Group
	GroupPassCount int
	RegistrationOpen bool
	EventID      *string
	SkipElo      bool
	Teams        []*Team
	TeamFormat   string // "olympic", "swaythling", or ""
	NumTables    int
	HasThirdPlaceMatch bool
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
		ID:           id,
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
		EventID:      nil,
		SkipElo:      false,
		Teams:        []*Team{},
		NumTables:    0,
		HasThirdPlaceMatch: hasThirdPlaceMatch,
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

func (t *Tournament) AutoAssignGroups() error {
	if t.Format != "groups_elimination" && t.Format != "round_robin" {
		return nil
	}
	// Determine units to group (players or teams)
	var units []*player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		units = make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			avgElo := team.AverageElo(t.Type)
			units[i] = &player.Player{
				ID:         team.ID,
				FirstName:  team.Name,
				LastName:   " (Team)",
				SinglesElo: avgElo,
				DoublesElo: avgElo,
			}
		}
	} else {
		units = make([]*player.Player, len(t.Participants))
		copy(units, t.Participants)
	}

	n := len(units)
	if n == 0 {
		return nil
	}

	// Sort participants/teams by Elo (descending)
	sort.Slice(units, func(i, j int) bool {
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			return units[i].DoublesElo > units[j].DoublesElo
		}
		return units[i].SinglesElo > units[j].SinglesElo
	})

	if t.Format == "round_robin" {
		t.Groups = []Group{
			{
				ID:           idgen.Generate(),
				TournamentID: t.ID,
				Name:         "All Against All",
				Players:      units, // Everyone in one single group
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
			ID:           idgen.Generate(),
			TournamentID: t.ID,
			Name:         fmt.Sprintf("Group %c", 'A'+i),
			Players:      []*player.Player{},
		}
	}

	// Snake seeding
	for i, p := range units {
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

type DivisionSeeding struct {
	Name   string
	MinElo int16
	MaxElo *int16
}

func (t *Tournament) AssignGroupsByDivisions(divs []DivisionSeeding) error {
	if t.Format != "groups_elimination" && t.Format != "round_robin" && t.Format != "elimination" {
		t.Groups = []Group{}
		return nil
	}

	// 1. Group participants by division
	type DivGroup struct {
		Name    string
		Players []*player.Player
	}

	var divGroups []DivGroup

	// Determine units to group (players or teams)
	var units []*player.Player
	if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
		units = make([]*player.Player, len(t.Teams))
		for i, team := range t.Teams {
			avgElo := team.AverageElo(t.Type)
			units[i] = &player.Player{
				ID:         team.ID,
				FirstName:  team.Name,
				LastName:   " (Team)",
				SinglesElo: avgElo,
				DoublesElo: avgElo,
			}
		}
	} else {
		units = make([]*player.Player, len(t.Participants))
		copy(units, t.Participants)
	}

	if t.SkipElo || len(divs) == 0 {
		divGroups = append(divGroups, DivGroup{
			Name:    "Open Bracket",
			Players: units,
		})
	} else {
		assigned := make(map[string]bool)
		for _, d := range divs {
			var dPlayers []*player.Player
			for _, p := range units {
				if assigned[p.ID] {
					continue
				}
				elo := p.SinglesElo
				if t.Type == "doubles" {
					elo = p.DoublesElo
				}
				if elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
					dPlayers = append(dPlayers, p)
					assigned[p.ID] = true
				}
			}
			if len(dPlayers) > 0 {
				divGroups = append(divGroups, DivGroup{
					Name:    d.Name,
					Players: dPlayers,
				})
			}
		}

		var unassigned []*player.Player
		for _, p := range units {
			if !assigned[p.ID] {
				unassigned = append(unassigned, p)
			}
		}
		if len(unassigned) > 0 {
			divGroups = append(divGroups, DivGroup{
				Name:    "Unclassified",
				Players: unassigned,
			})
		}
	}

	t.Groups = []Group{}

	for _, dg := range divGroups {
		n := len(dg.Players)
		if n == 0 {
			continue
		}

		sort.Slice(dg.Players, func(i, j int) bool {
			if t.Type == "doubles" {
				return dg.Players[i].DoublesElo > dg.Players[j].DoublesElo
			}
			return dg.Players[i].SinglesElo > dg.Players[j].SinglesElo
		})

		if t.Format == "round_robin" || t.Format == "elimination" {
			groupName := fmt.Sprintf("%s - Round Robin", dg.Name)
			if t.Format == "elimination" {
				groupName = fmt.Sprintf("%s - Bracket Draw", dg.Name)
			}
			t.Groups = append(t.Groups, Group{
				ID:           idgen.Generate(),
				TournamentID: t.ID,
				Name:         groupName,
				Players:      dg.Players,
			})
			continue
		}

		numGroups := n / 4
		if n%4 != 0 {
			numGroups++
		}

		divGroupsList := make([]Group, numGroups)
		for i := 0; i < numGroups; i++ {
			divGroupsList[i] = Group{
				ID:           idgen.Generate(),
				TournamentID: t.ID,
				Name:         fmt.Sprintf("%s - Group %c", dg.Name, 'A'+i),
				Players:      []*player.Player{},
			}
		}

		for i, p := range dg.Players {
			groupIndex := i % numGroups
			row := i / numGroups
			if row%2 != 0 {
				groupIndex = numGroups - 1 - groupIndex
			}
			divGroupsList[groupIndex].Players = append(divGroupsList[groupIndex].Players, p)
		}

		t.Groups = append(t.Groups, divGroupsList...)
	}

	return nil
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

	for _, m := range t.Matches {
		if m.Status == "in_progress" || m.Status == "finished" {
			return errors.New("cannot move player: matches have already started for this tournament")
		}
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
	UpdateGroups(ctx context.Context, t *Tournament) error
	Delete(ctx context.Context, id string) error
	UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error
	UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*player.Player) error
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
}
