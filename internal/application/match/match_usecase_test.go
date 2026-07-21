package match_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"table-tennis-backend/internal/application/match"
	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	grandDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/identity"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

// ── Mocks ──────────────────────────────────────────────────────────────────

type mockMatchRepo struct {
	matches          map[string]*eventDomain.Match
	subMatches       map[string][]*eventDomain.Match
	occupiedTables   map[int]bool
	scoresUpdated    bool
	subCreated       bool
	squadsUpdated    bool
	saveErr          error
	getAllErr        error
	getSubMatchesErr error
}

func newMockMatchRepo() *mockMatchRepo {
	return &mockMatchRepo{
		matches:        make(map[string]*eventDomain.Match),
		subMatches:     make(map[string][]*eventDomain.Match),
		occupiedTables: make(map[int]bool),
	}
}

func (m *mockMatchRepo) Save(ctx context.Context, match *eventDomain.Match) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.matches[match.ID] = match
	return nil
}
func (m *mockMatchRepo) CountUnfinishedMatches(ctx context.Context, tournamentID string) (int, error) {
	return 0, nil
}
func (m *mockMatchRepo) CountFinishedMatches(ctx context.Context, tournamentID string) (int, error) {
	return 0, nil
}
func (m *mockMatchRepo) GetAll(ctx context.Context) ([]*eventDomain.Match, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	var res []*eventDomain.Match
	for _, match := range m.matches {
		res = append(res, match)
	}
	return res, nil
}
func (m *mockMatchRepo) GetByID(ctx context.Context, id string) (*eventDomain.Match, error) {
	match, ok := m.matches[id]
	if !ok {
		return nil, errors.New("match not found")
	}
	return match, nil
}
func (m *mockMatchRepo) GetSubMatches(ctx context.Context, parentMatchID string) ([]*eventDomain.Match, error) {
	if m.getSubMatchesErr != nil {
		return nil, m.getSubMatchesErr
	}
	return m.subMatches[parentMatchID], nil
}
func (m *mockMatchRepo) GetMatchByParticipants(ctx context.Context, tournamentID, p1ID, p2ID, stage string) (*eventDomain.Match, error) {
	return nil, nil
}
func (m *mockMatchRepo) UpdateScore(ctx context.Context, id string, sets []eventDomain.MatchSet, stageRule eventDomain.StageRule) error {
	m.scoresUpdated = true
	if match, ok := m.matches[id]; ok {
		match.Sets = sets
	}
	return nil
}
func (m *mockMatchRepo) GetOccupiedTablesByEvent(ctx context.Context, eventID string) ([]int, error) {
	var res []int
	for tNum, occupied := range m.occupiedTables {
		if occupied {
			res = append(res, tNum)
		}
	}
	return res, nil
}
func (m *mockMatchRepo) GetOccupiedTablesByTournament(ctx context.Context, tournamentID string) ([]int, error) {
	return m.GetOccupiedTablesByEvent(ctx, tournamentID)
}
func (m *mockMatchRepo) IsTableOccupiedByOtherMatch(ctx context.Context, matchID string, tableNumber int) (bool, error) {
	return m.occupiedTables[tableNumber], nil
}
func (m *mockMatchRepo) UpdateMetadata(ctx context.Context, matchID string, refereeID *string, tableNumber *int) error {
	return nil
}
func (m *mockMatchRepo) HasStartedOrFinishedMatches(ctx context.Context, tournamentID string) (bool, error) {
	return false, nil
}
func (m *mockMatchRepo) DeleteByTournament(ctx context.Context, tournamentID string) error {
	return nil
}
func (m *mockMatchRepo) FinishMatch(ctx context.Context, cmd eventDomain.FinishMatchCommand) error {
	return nil
}
func (m *mockMatchRepo) FindOrCreateMatch(ctx context.Context, tournamentID, p1ID, p2ID, stage, matchType string) (string, error) {
	return "", nil
}
func (m *mockMatchRepo) CreateSubMatches(ctx context.Context, cmd eventDomain.CreateSubMatchesCommand) error {
	m.subCreated = true
	return nil
}
func (m *mockMatchRepo) UpdateSubMatchSquads(ctx context.Context, cmd eventDomain.UpdateSubMatchSquadsCommand) error {
	m.squadsUpdated = true
	return nil
}

type mockEventRepo struct {
	events map[string]*eventDomain.Event
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{
		events: make(map[string]*eventDomain.Event),
	}
}

func (m *mockEventRepo) Save(ctx context.Context, t *eventDomain.Event) error {
	m.events[t.ID] = t
	return nil
}
func (m *mockEventRepo) Update(ctx context.Context, t *eventDomain.Event) error {
	m.events[t.ID] = t
	return nil
}
func (m *mockEventRepo) GetByID(ctx context.Context, id string) (*eventDomain.Event, error) {
	e, ok := m.events[id]
	if !ok {
		return nil, errors.New("event not found")
	}
	return e, nil
}
func (m *mockEventRepo) GetAll(ctx context.Context) ([]*eventDomain.Event, error) { return nil, nil }
func (m *mockEventRepo) UpdateEventIDBulk(ctx context.Context, tournamentIDs []string, eventID string) error {
	return nil
}
func (m *mockEventRepo) UpdateGroups(ctx context.Context, t *eventDomain.Event) error { return nil }
func (m *mockEventRepo) Delete(ctx context.Context, id string) error                  { return nil }
func (m *mockEventRepo) UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockEventRepo) UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*playerDomain.Player) error {
	return nil
}
func (m *mockEventRepo) UpdateParticipantEloBefore(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockEventRepo) AddParticipant(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockEventRepo) RemoveParticipant(ctx context.Context, tournamentID string, playerID string) error {
	return nil
}
func (m *mockEventRepo) SaveTeam(ctx context.Context, team *eventDomain.Team) error { return nil }
func (m *mockEventRepo) DeleteTeam(ctx context.Context, id string) error            { return nil }
func (m *mockEventRepo) AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}
func (m *mockEventRepo) RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}
func (m *mockEventRepo) GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]eventDomain.ParticipantSnapshot, error) {
	return nil, nil
}
func (m *mockEventRepo) GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error) {
	return "", nil
}
func (m *mockEventRepo) AddOfficial(ctx context.Context, tournamentID string, playerID string, pin string) error {
	return nil
}
func (m *mockEventRepo) RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error {
	return nil
}
func (m *mockEventRepo) GetOfficials(ctx context.Context, tournamentID string) ([]eventDomain.ParticipantSnapshot, error) {
	return nil, nil
}
func (m *mockEventRepo) GetEventNumTables(ctx context.Context, eventID string) (int, error) {
	return 4, nil
}

type mockGrandRepo struct {
	grands map[string]*grandDomain.Tournament
}

func newMockGrandRepo() *mockGrandRepo {
	return &mockGrandRepo{
		grands: make(map[string]*grandDomain.Tournament),
	}
}

func (m *mockGrandRepo) Save(ctx context.Context, e *grandDomain.Tournament) error {
	m.grands[e.ID] = e
	return nil
}
func (m *mockGrandRepo) Update(ctx context.Context, e *grandDomain.Tournament) error {
	m.grands[e.ID] = e
	return nil
}
func (m *mockGrandRepo) GetByID(ctx context.Context, id string) (*grandDomain.Tournament, error) {
	g, ok := m.grands[id]
	if !ok {
		return nil, errors.New("grand tournament not found")
	}
	return g, nil
}
func (m *mockGrandRepo) GetByIDDeep(ctx context.Context, id string) (*grandDomain.Tournament, error) {
	return m.GetByID(ctx, id)
}
func (m *mockGrandRepo) GetAll(ctx context.Context) ([]*grandDomain.Tournament, error) {
	return nil, nil
}
func (m *mockGrandRepo) Delete(ctx context.Context, id string) error          { return nil }
func (m *mockGrandRepo) DeleteEvents(ctx context.Context, ids []string) error { return nil }

type mockPlayerRepo struct {
	players map[string]*playerDomain.Player
}

func newMockPlayerRepo() *mockPlayerRepo {
	return &mockPlayerRepo{
		players: make(map[string]*playerDomain.Player),
	}
}

func (m *mockPlayerRepo) GetById(ctx context.Context, id string) (*playerDomain.Player, error) {
	p, ok := m.players[id]
	if !ok {
		return nil, errors.New("player not found")
	}
	return p, nil
}
func (m *mockPlayerRepo) GetByIDs(ctx context.Context, ids []string) ([]*playerDomain.Player, error) {
	var res []*playerDomain.Player
	for _, id := range ids {
		if id == "error_id" {
			return nil, errors.New("forced error")
		}
		if p, ok := m.players[id]; ok {
			res = append(res, p)
		}
	}
	return res, nil
}
func (m *mockPlayerRepo) Save(ctx context.Context, p *playerDomain.Player) error { return nil }
func (m *mockPlayerRepo) SaveMultiple(ctx context.Context, players []*playerDomain.Player) error {
	return nil
}
func (m *mockPlayerRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *mockPlayerRepo) Search(ctx context.Context, query string) ([]*playerDomain.Player, error) {
	return nil, nil
}
func (m *mockPlayerRepo) SearchForSelection(ctx context.Context, query, gender string) ([]*playerDomain.Player, error) {
	return nil, nil
}
func (m *mockPlayerRepo) GetAll(ctx context.Context) ([]*playerDomain.Player, error) { return nil, nil }
func (m *mockPlayerRepo) GetAllSingles(ctx context.Context) ([]*playerDomain.Player, error) {
	return nil, nil
}
func (m *mockPlayerRepo) GetAllDoubles(ctx context.Context) ([]*playerDomain.Player, error) {
	return nil, nil
}
func (m *mockPlayerRepo) GetSinglesByGender(ctx context.Context, gender string) ([]*playerDomain.Player, error) {
	return nil, nil
}
func (m *mockPlayerRepo) GetDoublesByGender(ctx context.Context, gender string) ([]*playerDomain.Player, error) {
	return nil, nil
}

type mockDivisionRepo struct {
	divisions map[string]*divisionDomain.Division
}

func newMockDivisionRepo() *mockDivisionRepo {
	return &mockDivisionRepo{
		divisions: make(map[string]*divisionDomain.Division),
	}
}

func (m *mockDivisionRepo) GetById(ctx context.Context, id string) (*divisionDomain.Division, error) {
	d, ok := m.divisions[id]
	if !ok {
		return nil, errors.New("division not found")
	}
	return d, nil
}
func (m *mockDivisionRepo) GetAll(ctx context.Context) ([]*divisionDomain.Division, error) {
	if _, ok := m.divisions["error_id"]; ok {
		return nil, errors.New("forced error")
	}
	var res []*divisionDomain.Division
	for _, d := range m.divisions {
		res = append(res, d)
	}
	return res, nil
}
func (m *mockDivisionRepo) Save(ctx context.Context, d *divisionDomain.Division) error { return nil }
func (m *mockDivisionRepo) Delete(ctx context.Context, id string) error                { return nil }

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreateMatchUseCase(t *testing.T) {
	matchRepo := newMockMatchRepo()
	eventRepo := newMockEventRepo()
	playerRepo := newMockPlayerRepo()
	divisionRepo := newMockDivisionRepo()

	now := time.Now()
	e, _ := eventDomain.NewTournament("e1", "Singles Event", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
	_ = eventRepo.Save(context.Background(), e)

	p1, _ := playerDomain.NewPlayer("p1", "Player", "One", now, "M", "USA", "", "1")
	p2, _ := playerDomain.NewPlayer("p2", "Player", "Two", now, "M", "USA", "", "2")
	playerRepo.players["p1"] = p1
	playerRepo.players["p2"] = p2

	uc := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	ctx := context.Background()

	t.Run("successful create singles match", func(t *testing.T) {
		m, err := uc.Execute(ctx, "e1", "singles", []string{"p1"}, []string{"p2"}, "quarter_final")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.TournamentID != "e1" || m.Stage != "quarter_final" {
			t.Errorf("unexpected match fields: %+v", m)
		}
		if len(m.TeamA) != 1 || m.TeamA[0].ID != "p1" {
			t.Errorf("unexpected team A: %v", m.TeamA)
		}
	})

	t.Run("event not found", func(t *testing.T) {
		_, err := uc.Execute(ctx, "nonexistent", "singles", []string{"p1"}, []string{"p2"})
		if err == nil {
			t.Fatal("expected error for nonexistent event")
		}
	})

	t.Run("team A player not found", func(t *testing.T) {
		_, err := uc.Execute(ctx, "e1", "singles", []string{"error_id"}, []string{"p2"})
		if err == nil {
			t.Fatal("expected error for team A player not found")
		}
	})

	t.Run("team B player not found", func(t *testing.T) {
		_, err := uc.Execute(ctx, "e1", "singles", []string{"p1"}, []string{"error_id"})
		if err == nil {
			t.Fatal("expected error for team B player not found")
		}
	})

	t.Run("doubles with valid teams", func(t *testing.T) {
		eTeams, _ := eventDomain.NewTournament("e_teams", "Teams Event", "doubles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		eTeams.Teams = []*eventDomain.Team{
			{ID: "t1", Players: []*playerDomain.Player{p1}},
			{ID: "t2", Players: []*playerDomain.Player{p2}},
		}
		eTeams.DivisionRules = []eventDomain.DivisionRule{{}}
		_ = eventRepo.Save(ctx, eTeams)

		maxElo := int16(1500)
		div1, _ := divisionDomain.NewDivision("d1", "Div 1", 1, 0, &maxElo, "both", "#ffffff")
		divisionRepo.divisions["d1"] = div1

		m, err := uc.Execute(ctx, "e_teams", "doubles", []string{"t1"}, []string{"t2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.DivisionID != "d1" {
			t.Errorf("expected division d1, got %s", m.DivisionID)
		}
	})

	t.Run("team A not found in event", func(t *testing.T) {
		_, err := uc.Execute(ctx, "e1", "doubles", []string{"invalid"}, []string{"t2"})
		if err == nil {
			t.Fatal("expected error for team A not found")
		}
	})

	t.Run("team B not found in event", func(t *testing.T) {
		eTeams2, _ := eventDomain.NewTournament("e_teams2", "Teams Event", "doubles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		eTeams2.Teams = []*eventDomain.Team{
			{ID: "t1", Players: []*playerDomain.Player{p1}},
		}
		_ = eventRepo.Save(ctx, eTeams2)
		_, err := uc.Execute(ctx, "e_teams2", "doubles", []string{"t1"}, []string{"invalid"})
		if err == nil {
			t.Fatal("expected error for team B not found")
		}
	})

	t.Run("empty match type defaults to singles", func(t *testing.T) {
		m, err := uc.Execute(ctx, "e1", "", []string{"p1"}, []string{"p2"}, "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.MatchType != "singles" || m.Stage != "group" {
			t.Errorf("unexpected match type %s or stage %s", m.MatchType, m.Stage)
		}
	})

	t.Run("determinePlayerDivision GetAll error yields empty division", func(t *testing.T) {
		eDivErr, _ := eventDomain.NewTournament("e_div_err", "Div Err Event", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		eDivErr.DivisionRules = []eventDomain.DivisionRule{{}}
		_ = eventRepo.Save(ctx, eDivErr)

		errDivRepo := newMockDivisionRepo()
		errDivRepo.divisions["error_id"] = div1ForErrTest()
		ucErr := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, errDivRepo)

		m, err := ucErr.Execute(ctx, "e_div_err", "singles", []string{"p1"}, []string{"p2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.DivisionID != "" {
			t.Errorf("expected empty division on GetAll error, got %s", m.DivisionID)
		}
	})

	t.Run("determinePlayerDivision no matching range yields empty division", func(t *testing.T) {
		eNoMatch, _ := eventDomain.NewTournament("e_no_match", "No Match Event", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		eNoMatch.DivisionRules = []eventDomain.DivisionRule{{}}
		_ = eventRepo.Save(ctx, eNoMatch)

		maxElo := int16(500)
		divLow, _ := divisionDomain.NewDivision("d_low", "Low", 1, 0, &maxElo, "both", "#000000")
		noMatchDivRepo := newMockDivisionRepo()
		noMatchDivRepo.divisions["d_low"] = divLow

		ucNoMatch := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, noMatchDivRepo)
		m, err := ucNoMatch.Execute(ctx, "e_no_match", "singles", []string{"p1"}, []string{"p2"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.DivisionID != "" {
			t.Errorf("expected empty division for out-of-range elo, got %s", m.DivisionID)
		}
	})

	t.Run("matchRepo Save error is propagated", func(t *testing.T) {
		failRepo := newMockMatchRepo()
		failRepo.saveErr = errors.New("save failed")
		ucFail := match.NewCreateMatchUseCase(failRepo, playerRepo, eventRepo, divisionRepo)
		_, err := ucFail.Execute(ctx, "e1", "singles", []string{"p1"}, []string{"p2"})
		if err == nil {
			t.Fatal("expected error when Save fails")
		}
	})
}

// div1ForErrTest returns a division whose presence under the "error_id" key
// forces mockDivisionRepo.GetAll to return an error (see mockDivisionRepo.GetAll).
func div1ForErrTest() *divisionDomain.Division {
	d, _ := divisionDomain.NewDivision("error_id", "Err", 1, 0, nil, "both", "#000000")
	return d
}

func TestFinishMatchUseCase(t *testing.T) {
	uc := match.NewFinishMatchUseCase()
	m := &eventDomain.Match{ID: "m1", Status: "in_progress"}

	t.Run("valid winner A", func(t *testing.T) {
		err := uc.Execute(m, "A")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if m.WinnerTeam != "A" || m.Status != "finished" {
			t.Errorf("unexpected status or winner: %s %s", m.Status, m.WinnerTeam)
		}
	})

	t.Run("invalid winner team", func(t *testing.T) {
		err := uc.Execute(m, "C")
		if err == nil {
			t.Fatal("expected error for invalid winner team")
		}
	})
}

func TestStartMatchUseCase(t *testing.T) {
	matchRepo := newMockMatchRepo()
	eventRepo := newMockEventRepo()
	grandRepo := newMockGrandRepo()
	playerRepo := newMockPlayerRepo()
	divisionRepo := newMockDivisionRepo()

	now := time.Now()
	e, _ := eventDomain.NewTournament("e1", "Event 1", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
	_ = eventRepo.Save(context.Background(), e)

	p1, _ := playerDomain.NewPlayer("p1", "Alice", "Smith", now, "F", "USA", "", "1")
	p2, _ := playerDomain.NewPlayer("p2", "Bob", "Jones", now, "M", "USA", "", "2")

	createUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	uc := match.NewStartMatchUseCase(matchRepo, eventRepo, grandRepo, createUC)
	ctx := context.Background()

	m := &eventDomain.Match{
		ID:           "m1",
		TournamentID: "e1",
		TeamA:        []*playerDomain.Player{p1},
		TeamB:        []*playerDomain.Player{p2},
		Status:       "scheduled",
	}
	_ = matchRepo.Save(ctx, m)

	t.Run("manual table assignment success", func(t *testing.T) {
		tbl := 3
		cmd := eventDomain.StartMatchCommand{
			MatchID:     "m1",
			TableNumber: &tbl,
		}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 3 {
			t.Errorf("expected table 3, got %d", res.TableNumber)
		}
		if res.PlayerAName != "Alice Smith" || res.PlayerBName != "Bob Jones" {
			t.Errorf("unexpected player names: %s vs %s", res.PlayerAName, res.PlayerBName)
		}
	})

	t.Run("manual table assignment occupied error", func(t *testing.T) {
		matchRepo.occupiedTables[1] = true
		tbl := 1
		cmd := eventDomain.StartMatchCommand{
			MatchID:     "m1",
			TableNumber: &tbl,
		}
		_, err := uc.Execute(ctx, cmd)
		if err == nil {
			t.Fatal("expected error for occupied table")
		}
	})

	t.Run("match not found", func(t *testing.T) {
		cmd := eventDomain.StartMatchCommand{MatchID: "invalid_match"}
		_, err := uc.Execute(ctx, cmd)
		if err == nil {
			t.Fatal("expected error for match not found")
		}
	})

	t.Run("event not found", func(t *testing.T) {
		m2 := &eventDomain.Match{ID: "m2", TournamentID: "invalid_event"}
		_ = matchRepo.Save(ctx, m2)
		cmd := eventDomain.StartMatchCommand{MatchID: "m2"}
		_, err := uc.Execute(ctx, cmd)
		if err == nil {
			t.Fatal("expected error for event not found")
		}
	})

	t.Run("no tables available", func(t *testing.T) {
		m3 := &eventDomain.Match{ID: "m3", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, m3)

		// Occupy all tables (assuming totalTables defaults to 4 or 2 based on e1.NumTables)
		matchRepo.occupiedTables[1] = true
		matchRepo.occupiedTables[2] = true
		matchRepo.occupiedTables[3] = true
		matchRepo.occupiedTables[4] = true

		cmd := eventDomain.StartMatchCommand{MatchID: "m3"}
		_, err := uc.Execute(ctx, cmd)
		if err == nil || err != eventDomain.ErrTableOccupied {
			t.Fatalf("expected ErrTableOccupied, got %v", err)
		}
	})

	t.Run("auto-assign priority table", func(t *testing.T) {
		eventID := "g1"
		eWithGrand := &eventDomain.Event{ID: "e_grand", NumTables: 4, EventID: &eventID}
		_ = eventRepo.Save(ctx, eWithGrand)
		mGrand := &eventDomain.Match{ID: "m_grand", TournamentID: "e_grand", DivisionID: "div1"}
		_ = matchRepo.Save(ctx, mGrand)

		g, _ := grandDomain.NewEvent("g1", "Grand", nil, true, now, now)
		g.TablePriorities = map[string][]int{"div1": {3, 4}}
		_ = grandRepo.Save(ctx, g)

		// Free table 3, occupy table 1 and 2
		matchRepo.occupiedTables[1] = true
		matchRepo.occupiedTables[2] = true
		matchRepo.occupiedTables[3] = false
		matchRepo.occupiedTables[4] = false

		cmd := eventDomain.StartMatchCommand{MatchID: "m_grand"}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 3 {
			t.Errorf("expected priority table 3, got %d", res.TableNumber)
		}
	})

	t.Run("high priority assignment", func(t *testing.T) {
		mHigh := &eventDomain.Match{ID: "m_high", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, mHigh)

		matchRepo.occupiedTables[1] = false
		matchRepo.occupiedTables[2] = false

		cmd := eventDomain.StartMatchCommand{MatchID: "m_high", IsHighPriority: true}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 1 {
			t.Errorf("expected high priority table 1, got %d", res.TableNumber)
		}
	})

	t.Run("high priority falls back to table 2 when table 1 occupied", func(t *testing.T) {
		mHigh2 := &eventDomain.Match{ID: "m_high2", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, mHigh2)

		matchRepo.occupiedTables[1] = true
		matchRepo.occupiedTables[2] = false
		matchRepo.occupiedTables[3] = true
		matchRepo.occupiedTables[4] = true

		cmd := eventDomain.StartMatchCommand{MatchID: "m_high2", IsHighPriority: true}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 2 {
			t.Errorf("expected high priority fallback table 2, got %d", res.TableNumber)
		}
	})

	t.Run("high priority falls back to first available when 1 and 2 occupied", func(t *testing.T) {
		mHigh3 := &eventDomain.Match{ID: "m_high3", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, mHigh3)

		matchRepo.occupiedTables[1] = true
		matchRepo.occupiedTables[2] = true
		matchRepo.occupiedTables[3] = false
		matchRepo.occupiedTables[4] = false

		cmd := eventDomain.StartMatchCommand{MatchID: "m_high3", IsHighPriority: true}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 3 {
			t.Errorf("expected fallback to first available table 3, got %d", res.TableNumber)
		}
	})

	t.Run("normal priority picks highest table >= 3", func(t *testing.T) {
		mNorm := &eventDomain.Match{ID: "m_norm", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, mNorm)

		matchRepo.occupiedTables[1] = true
		matchRepo.occupiedTables[2] = true
		matchRepo.occupiedTables[3] = false
		matchRepo.occupiedTables[4] = false

		cmd := eventDomain.StartMatchCommand{MatchID: "m_norm"}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 4 {
			t.Errorf("expected highest table >= 3 (table 4), got %d", res.TableNumber)
		}
	})

	t.Run("normal priority falls back to table 2 when no table >= 3 available", func(t *testing.T) {
		mNorm2 := &eventDomain.Match{ID: "m_norm2", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, mNorm2)

		matchRepo.occupiedTables[1] = true
		matchRepo.occupiedTables[2] = false
		matchRepo.occupiedTables[3] = true
		matchRepo.occupiedTables[4] = true

		cmd := eventDomain.StartMatchCommand{MatchID: "m_norm2"}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 2 {
			t.Errorf("expected fallback table 2, got %d", res.TableNumber)
		}
	})

	t.Run("normal priority falls back to table 1 when table 2 occupied", func(t *testing.T) {
		mNorm3 := &eventDomain.Match{ID: "m_norm3", TournamentID: "e1"}
		_ = matchRepo.Save(ctx, mNorm3)

		matchRepo.occupiedTables[1] = false
		matchRepo.occupiedTables[2] = true
		matchRepo.occupiedTables[3] = true
		matchRepo.occupiedTables[4] = true

		cmd := eventDomain.StartMatchCommand{MatchID: "m_norm3"}
		res, err := uc.Execute(ctx, cmd)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.TableNumber != 1 {
			t.Errorf("expected fallback table 1, got %d", res.TableNumber)
		}
	})

	t.Run("matchRepo Save error is propagated", func(t *testing.T) {
		failRepo := newMockMatchRepo()
		mFail := &eventDomain.Match{ID: "m_fail", TournamentID: "e1"}
		_ = failRepo.Save(ctx, mFail)
		failRepo.saveErr = errors.New("save failed")

		ucFail := match.NewStartMatchUseCase(failRepo, eventRepo, grandRepo, createUC)
		tbl := 1
		_, err := ucFail.Execute(ctx, eventDomain.StartMatchCommand{MatchID: "m_fail", TableNumber: &tbl})
		if err == nil {
			t.Fatal("expected error when Save fails")
		}
	})
}

func TestUpdateMatchScoreUseCase(t *testing.T) {
	matchRepo := newMockMatchRepo()
	eventRepo := newMockEventRepo()
	ctx := context.Background()

	now := time.Now()
	e, _ := eventDomain.NewTournament("e1", "Event 1", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
	_ = eventRepo.Save(ctx, e)

	m := &eventDomain.Match{ID: "m1", TournamentID: "e1"}
	_ = matchRepo.Save(ctx, m)

	uc := match.NewUpdateMatchScoreUseCase(matchRepo, eventRepo)

	t.Run("ParseSetScores valid and invalid", func(t *testing.T) {
		sets, err := match.ParseSetScores([]string{"11-8", "9-11", "11-5"})
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if len(sets) != 3 || sets[0].ScoreA != 11 || sets[0].ScoreB != 8 {
			t.Errorf("unexpected sets: %v", sets)
		}

		_, err = match.ParseSetScores([]string{"invalid"})
		if err == nil {
			t.Fatal("expected parse error for invalid score string")
		}
	})

	t.Run("ParseSetScores skips blank entries", func(t *testing.T) {
		sets, err := match.ParseSetScores([]string{"11-8", "  ", ""})
		if err != nil {
			t.Fatalf("unexpected parse error: %v", err)
		}
		if len(sets) != 1 {
			t.Errorf("expected blank entries to be skipped, got %d sets", len(sets))
		}
	})

	t.Run("ParseSetScores invalid score A", func(t *testing.T) {
		_, err := match.ParseSetScores([]string{"a-8"})
		if err == nil {
			t.Fatal("expected error for non-numeric score A")
		}
	})

	t.Run("ParseSetScores invalid score B", func(t *testing.T) {
		_, err := match.ParseSetScores([]string{"8-b"})
		if err == nil {
			t.Fatal("expected error for non-numeric score B")
		}
	})

	t.Run("execute update score", func(t *testing.T) {
		err := uc.Execute(ctx, "m1", []string{"11-9", "11-7"}, "e1", "group")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !matchRepo.scoresUpdated {
			t.Error("expected scoresUpdated to be true")
		}
	})

	t.Run("execute with divisioned match", func(t *testing.T) {
		e.Matches = append(e.Matches, eventDomain.Match{ID: "m1", DivisionID: "div1"})
		err := uc.Execute(ctx, "m1", []string{"11-9", "11-7"}, "e1", "group")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("execute with invalid raw scores", func(t *testing.T) {
		err := uc.Execute(ctx, "m1", []string{"invalid"}, "e1", "group")
		if err == nil {
			t.Fatal("expected error for invalid raw score")
		}
	})

	t.Run("execute with nonexistent tournament", func(t *testing.T) {
		err := uc.Execute(ctx, "m1", []string{"11-9"}, "nonexistent", "group")
		if err == nil {
			t.Fatal("expected error for nonexistent tournament")
		}
	})

	t.Run("execute on finished tournament error", func(t *testing.T) {
		e.Status = "finished"
		err := uc.Execute(ctx, "m1", []string{"11-9"}, "e1", "group")
		if err == nil {
			t.Fatal("expected error on finished tournament")
		}
	})
}

func TestAutoAssignTablesUseCase(t *testing.T) {
	matchRepo := newMockMatchRepo()
	grandRepo := newMockGrandRepo()
	ctx := context.Background()

	now := time.Now()
	g, _ := grandDomain.NewEvent("g1", "Grand Tourney", []string{"d1"}, true, now, now.Add(24*time.Hour))
	g.NumTables = 4

	p1, _ := playerDomain.NewPlayer("p1", "A", "B", now, "M", "USA", "", "1")
	p2, _ := playerDomain.NewPlayer("p2", "C", "D", now, "M", "USA", "", "2")

	subE, _ := eventDomain.NewTournament("e1", "Sub Event", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
	subE.Matches = []eventDomain.Match{
		{
			ID:           "m1",
			TournamentID: "e1",
			Status:       "scheduled",
			TeamA:        []*playerDomain.Player{p1},
			TeamB:        []*playerDomain.Player{p2},
		},
	}
	g.Events = []*eventDomain.Event{subE}
	_ = grandRepo.Save(ctx, g)

	uc := match.NewAutoAssignTablesUseCase(matchRepo, grandRepo)

	t.Run("assigns table to scheduled match", func(t *testing.T) {
		assigned, err := uc.Execute(ctx, "g1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(assigned) != 1 {
			t.Fatalf("expected 1 assigned match, got %d", len(assigned))
		}
		if assigned[0].TableNumber == nil {
			t.Error("expected TableNumber to be set")
		}
	})

	t.Run("tournament not found", func(t *testing.T) {
		_, err := uc.Execute(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent grand tournament")
		}
	})

	t.Run("no scheduled matches returns nil", func(t *testing.T) {
		gEmpty, _ := grandDomain.NewEvent("g_empty", "Empty", []string{"d1"}, true, now, now.Add(24*time.Hour))
		gEmpty.NumTables = 4
		subEmpty, _ := eventDomain.NewTournament("e_empty", "Empty Sub", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		gEmpty.Events = []*eventDomain.Event{subEmpty}
		_ = grandRepo.Save(ctx, gEmpty)

		assigned, err := uc.Execute(ctx, "g_empty")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if assigned != nil {
			t.Errorf("expected nil assigned matches, got %v", assigned)
		}
	})

	t.Run("no available tables returns nil", func(t *testing.T) {
		gFull, _ := grandDomain.NewEvent("g_full", "Full", []string{"d1"}, true, now, now.Add(24*time.Hour))
		gFull.NumTables = 1
		subFull, _ := eventDomain.NewTournament("e_full", "Full Sub", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		occupiedTbl := 1
		subFull.Matches = []eventDomain.Match{
			{
				ID:           "m_occupied",
				TournamentID: "e_full",
				Status:       "in_progress",
				TableNumber:  &occupiedTbl,
				TeamA:        []*playerDomain.Player{p1},
				TeamB:        []*playerDomain.Player{p2},
			},
			{
				ID:           "m_waiting",
				TournamentID: "e_full",
				Status:       "scheduled",
				TeamA:        []*playerDomain.Player{p1},
				TeamB:        []*playerDomain.Player{p2},
			},
		}
		gFull.Events = []*eventDomain.Event{subFull}
		_ = grandRepo.Save(ctx, gFull)

		assigned, err := uc.Execute(ctx, "g_full")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if assigned != nil {
			t.Errorf("expected nil assigned matches when no tables available, got %v", assigned)
		}
	})

	t.Run("division priority assigns preferred table", func(t *testing.T) {
		gPrio, _ := grandDomain.NewEvent("g_prio", "Prio", []string{"d1"}, true, now, now.Add(24*time.Hour))
		gPrio.NumTables = 4
		gPrio.TablePriorities = map[string][]int{"divP": {2}}
		subPrio, _ := eventDomain.NewTournament("e_prio", "Prio Sub", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		subPrio.Matches = []eventDomain.Match{
			{
				ID:           "m_prio",
				TournamentID: "e_prio",
				DivisionID:   "divP",
				Status:       "scheduled",
				TeamA:        []*playerDomain.Player{p1},
				TeamB:        []*playerDomain.Player{p2},
			},
		}
		gPrio.Events = []*eventDomain.Event{subPrio}
		_ = grandRepo.Save(ctx, gPrio)

		assigned, err := uc.Execute(ctx, "g_prio")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(assigned) != 1 || assigned[0].TableNumber == nil || *assigned[0].TableNumber != 2 {
			t.Errorf("expected priority table 2, got %+v", assigned)
		}
	})

	t.Run("more scheduled matches than tables stops early", func(t *testing.T) {
		gLimited, _ := grandDomain.NewEvent("g_limited", "Limited", []string{"d1"}, true, now, now.Add(24*time.Hour))
		gLimited.NumTables = 1
		subLimited, _ := eventDomain.NewTournament("e_limited", "Limited Sub", "singles", "single_elimination", "men", now, now.Add(24*time.Hour), nil, 2, nil, false)
		subLimited.Matches = []eventDomain.Match{
			{ID: "m_a", TournamentID: "e_limited", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
			{ID: "m_b", TournamentID: "e_limited", Status: "scheduled", TeamA: []*playerDomain.Player{p1}, TeamB: []*playerDomain.Player{p2}},
		}
		gLimited.Events = []*eventDomain.Event{subLimited}
		_ = grandRepo.Save(ctx, gLimited)

		assigned, err := uc.Execute(ctx, "g_limited")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(assigned) != 1 {
			t.Errorf("expected only 1 match assigned (limited tables), got %d", len(assigned))
		}
	})
}

func TestGetMatchesUseCase(t *testing.T) {
	matchRepo := newMockMatchRepo()
	ctx := context.Background()

	m1 := &eventDomain.Match{ID: "m1", TournamentID: "e1", MatchType: "singles", Status: "finished", WinnerTeam: "A"}
	m2 := &eventDomain.Match{ID: "m2", TournamentID: "e1", MatchType: "singles", Status: "scheduled"}
	_ = matchRepo.Save(ctx, m1)
	_ = matchRepo.Save(ctx, m2)

	uc := match.NewGetMatchesUseCase(matchRepo)

	t.Run("returns views for all matches", func(t *testing.T) {
		views, err := uc.GetAllViews(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(views) != 2 {
			t.Errorf("expected 2 views, got %d", len(views))
		}
	})

	t.Run("propagates repo error", func(t *testing.T) {
		errRepo := newMockMatchRepo()
		errRepo.getAllErr = errors.New("db error")
		errUC := match.NewGetMatchesUseCase(errRepo)
		_, err := errUC.GetAllViews(ctx)
		if err == nil {
			t.Fatal("expected error to propagate from GetAll")
		}
	})
}

func TestTeamMatchOrchestratorUseCase(t *testing.T) {
	matchRepo := newMockMatchRepo()
	uc := match.NewTeamMatchOrchestratorUseCase(matchRepo)
	ctx := context.Background()

	p1, _ := playerDomain.NewPlayer("p1", "P1", "A", time.Now(), "M", "USA", "", "1")
	p2, _ := playerDomain.NewPlayer("p2", "P2", "A", time.Now(), "M", "USA", "", "2")
	p3, _ := playerDomain.NewPlayer("p3", "P3", "A", time.Now(), "M", "USA", "", "3")

	p4, _ := playerDomain.NewPlayer("p4", "P4", "B", time.Now(), "M", "USA", "", "4")
	p5, _ := playerDomain.NewPlayer("p5", "P5", "B", time.Now(), "M", "USA", "", "5")
	p6, _ := playerDomain.NewPlayer("p6", "P6", "B", time.Now(), "M", "USA", "", "6")

	teamA := &eventDomain.Team{ID: "tA", Name: "Team A", Players: []*playerDomain.Player{p1, p2, p3}}
	teamB := &eventDomain.Team{ID: "tB", Name: "Team B", Players: []*playerDomain.Player{p4, p5, p6}}

	t.Run("EnsureTeamSubMatches when none exist", func(t *testing.T) {
		err := uc.EnsureTeamSubMatches(ctx, "mParent", teamA, teamB, "olympic", "group")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !matchRepo.subCreated {
			t.Error("expected subCreated to be true")
		}
	})

	t.Run("UpdateTeamSquads with valid squads", func(t *testing.T) {
		matchRepo.subMatches["mParent"] = []*eventDomain.Match{
			{ID: "sub1", RoundNumber: 1},
			{ID: "sub2", RoundNumber: 2},
		}

		err := uc.UpdateTeamSquads(
			ctx,
			"mParent",
			[]string{"p1", "p2", "p3"},
			[]string{"p4", "p5", "p6"},
			"olympic",
			"group",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !matchRepo.squadsUpdated {
			t.Error("expected squadsUpdated to be true")
		}
	})

	t.Run("UpdateTeamSquads with short squads error", func(t *testing.T) {
		err := uc.UpdateTeamSquads(
			ctx,
			"mParent",
			[]string{"p1", "p2"},
			[]string{"p4", "p5", "p6"},
			"olympic",
			"group",
		)
		if err == nil {
			t.Fatal("expected error for short squad")
		}
	})

	t.Run("EnsureTeamSubMatches when subs already exist", func(t *testing.T) {
		alreadyRepo := newMockMatchRepo()
		alreadyRepo.subMatches["mExisting"] = []*eventDomain.Match{{ID: "sub1", RoundNumber: 1}}
		alreadyUC := match.NewTeamMatchOrchestratorUseCase(alreadyRepo)
		err := alreadyUC.EnsureTeamSubMatches(ctx, "mExisting", teamA, teamB, "olympic", "group")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if alreadyRepo.subCreated {
			t.Error("expected subCreated to remain false when subs already exist")
		}
	})

	t.Run("EnsureTeamSubMatches with missing team players errors", func(t *testing.T) {
		emptyTeam := &eventDomain.Team{ID: "tEmpty"}
		err := uc.EnsureTeamSubMatches(ctx, "mNoPlayers", emptyTeam, teamB, "olympic", "group")
		if err == nil {
			t.Fatal("expected error when a team has no players")
		}
		err = uc.EnsureTeamSubMatches(ctx, "mNoPlayers", nil, teamB, "olympic", "group")
		if err == nil {
			t.Fatal("expected error when team A is nil")
		}
	})

	t.Run("EnsureTeamSubMatches propagates GetSubMatches error", func(t *testing.T) {
		errRepo := newMockMatchRepo()
		errRepo.getSubMatchesErr = errors.New("db error")
		errUC := match.NewTeamMatchOrchestratorUseCase(errRepo)
		err := errUC.EnsureTeamSubMatches(ctx, "mParent", teamA, teamB, "olympic", "group")
		if err == nil {
			t.Fatal("expected GetSubMatches error to propagate")
		}
	})

	t.Run("UpdateTeamSquads propagates GetSubMatches error", func(t *testing.T) {
		errRepo := newMockMatchRepo()
		errRepo.getSubMatchesErr = errors.New("db error")
		errUC := match.NewTeamMatchOrchestratorUseCase(errRepo)
		err := errUC.UpdateTeamSquads(ctx, "mParent", []string{"p1", "p2", "p3"}, []string{"p4", "p5", "p6"}, "olympic", "group")
		if err == nil {
			t.Fatal("expected GetSubMatches error to propagate")
		}
	})

	t.Run("UpdateTeamSquads with no sub-matches errors", func(t *testing.T) {
		freshRepo := newMockMatchRepo()
		freshUC := match.NewTeamMatchOrchestratorUseCase(freshRepo)
		err := freshUC.UpdateTeamSquads(ctx, "mNoSubs", []string{"p1", "p2", "p3"}, []string{"p4", "p5", "p6"}, "olympic", "group")
		if err == nil {
			t.Fatal("expected error when sub-matches do not exist")
		}
	})

	t.Run("UpdateTeamSquads covers all olympic rounds", func(t *testing.T) {
		olympicRepo := newMockMatchRepo()
		olympicRepo.subMatches["mOlympic"] = []*eventDomain.Match{
			{ID: "sub1", RoundNumber: 1},
			{ID: "sub2", RoundNumber: 2},
			{ID: "sub3", RoundNumber: 3},
			{ID: "sub4", RoundNumber: 4},
			{ID: "sub5", RoundNumber: 5},
		}
		olympicUC := match.NewTeamMatchOrchestratorUseCase(olympicRepo)
		err := olympicUC.UpdateTeamSquads(ctx, "mOlympic", []string{"p1", "p2", "p3"}, []string{"p4", "p5", "p6"}, "olympic", "group")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !olympicRepo.squadsUpdated {
			t.Error("expected squadsUpdated to be true")
		}
	})

	t.Run("UpdateTeamSquads covers all corbillon rounds with default format", func(t *testing.T) {
		corbillonRepo := newMockMatchRepo()
		corbillonRepo.subMatches["mCorbillon"] = []*eventDomain.Match{
			{ID: "sub1", RoundNumber: 1},
			{ID: "sub2", RoundNumber: 2},
			{ID: "sub3", RoundNumber: 3},
			{ID: "sub4", RoundNumber: 4},
			{ID: "sub5", RoundNumber: 5},
		}
		corbillonUC := match.NewTeamMatchOrchestratorUseCase(corbillonRepo)
		err := corbillonUC.UpdateTeamSquads(ctx, "mCorbillon", []string{"p1", "p2", "p3"}, []string{"p4", "p5", "p6"}, "corbillon", "group")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !corbillonRepo.squadsUpdated {
			t.Error("expected squadsUpdated to be true")
		}
	})
}

func TestIsValidID(t *testing.T) {
	cases := map[string]bool{
		"":          false,
		"nil":       false,
		"null":      false,
		"undefined": false,
		"m1":        true,
		"550e8400":  true,
	}
	for id, want := range cases {
		if got := match.IsValidIDForTest(id); got != want {
			t.Errorf("isValidID(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestGetSubMatchAlignments(t *testing.T) {
	cases := []struct {
		round      int
		teamFormat string
		wantA      string
		wantB      string
	}{
		{1, "olympic", "A & B", "X & Y"},
		{2, "olympic", "C", "Z"},
		{3, "olympic", "A", "X"},
		{4, "olympic", "B", "Y"},
		{5, "olympic", "C", "X"},
		{6, "olympic", "", ""},
		{1, "corbillon", "A", "X"},
		{2, "corbillon", "B", "Y"},
		{3, "corbillon", "C", "Z"},
		{4, "corbillon", "A", "Y"},
		{5, "corbillon", "B", "X"},
		{1, "", "A & B", "X & Y"}, // empty format defaults to olympic
	}
	for _, c := range cases {
		gotA, gotB := match.GetSubMatchAlignmentsForTest(c.round, c.teamFormat)
		if gotA != c.wantA || gotB != c.wantB {
			t.Errorf("getSubMatchAlignments(%d, %q) = (%q, %q), want (%q, %q)", c.round, c.teamFormat, gotA, gotB, c.wantA, c.wantB)
		}
	}
}
