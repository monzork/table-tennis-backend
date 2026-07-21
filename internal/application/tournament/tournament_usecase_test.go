package tournament_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	subTourneyDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/identity"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

// ── Mock Tournament Repository (eventDomain in tournament_crud.go) ────────

type mockEventRepo struct {
	events    map[string]*tournamentDomain.Tournament
	updateErr error
}

func newMockEventRepo() *mockEventRepo {
	return &mockEventRepo{
		events: make(map[string]*tournamentDomain.Tournament),
	}
}

func (m *mockEventRepo) Save(ctx context.Context, e *tournamentDomain.Tournament) error {
	m.events[e.ID] = e
	return nil
}

func (m *mockEventRepo) Update(ctx context.Context, e *tournamentDomain.Tournament) error {
	if m.updateErr != nil {
		return m.updateErr
	}
	m.events[e.ID] = e
	return nil
}

func (m *mockEventRepo) GetByID(ctx context.Context, id string) (*tournamentDomain.Tournament, error) {
	e, ok := m.events[id]
	if !ok {
		return nil, errors.New("event not found")
	}
	return e, nil
}

func (m *mockEventRepo) GetByIDDeep(ctx context.Context, id string) (*tournamentDomain.Tournament, error) {
	return m.GetByID(ctx, id)
}

func (m *mockEventRepo) GetAll(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	var res []*tournamentDomain.Tournament
	for _, e := range m.events {
		res = append(res, e)
	}
	return res, nil
}

func (m *mockEventRepo) Delete(ctx context.Context, id string) error {
	delete(m.events, id)
	return nil
}

func (m *mockEventRepo) DeleteEvents(ctx context.Context, ids []string) error {
	for _, id := range ids {
		delete(m.events, id)
	}
	return nil
}

// ── Mock Sub-Tournament Repository ────────────────────────────────────────

type mockSubTourneyRepo struct {
	subTourneys map[string]*subTourneyDomain.Event
}

func newMockSubTourneyRepo() *mockSubTourneyRepo {
	return &mockSubTourneyRepo{
		subTourneys: make(map[string]*subTourneyDomain.Event),
	}
}

func (m *mockSubTourneyRepo) Save(ctx context.Context, t *subTourneyDomain.Event) error {
	m.subTourneys[t.ID] = t
	return nil
}

func (m *mockSubTourneyRepo) GetByID(ctx context.Context, id string) (*subTourneyDomain.Event, error) {
	t, ok := m.subTourneys[id]
	if !ok {
		return nil, errors.New("sub-tournament not found")
	}
	return t, nil
}

func (m *mockSubTourneyRepo) GetAll(ctx context.Context) ([]*subTourneyDomain.Event, error) {
	var res []*subTourneyDomain.Event
	for _, t := range m.subTourneys {
		res = append(res, t)
	}
	return res, nil
}

func (m *mockSubTourneyRepo) Update(ctx context.Context, t *subTourneyDomain.Event) error {
	m.subTourneys[t.ID] = t
	return nil
}

func (m *mockSubTourneyRepo) UpdateEventIDBulk(ctx context.Context, tournamentIDs []string, eventID string) error {
	for _, id := range tournamentIDs {
		if t, ok := m.subTourneys[id]; ok {
			t.EventID = &eventID
		}
	}
	return nil
}

func (m *mockSubTourneyRepo) UpdateGroups(ctx context.Context, t *subTourneyDomain.Event) error {
	return nil
}
func (m *mockSubTourneyRepo) Delete(ctx context.Context, id string) error { return nil }
func (m *mockSubTourneyRepo) UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockSubTourneyRepo) UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*playerDomain.Player) error {
	return nil
}
func (m *mockSubTourneyRepo) UpdateParticipantEloBefore(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockSubTourneyRepo) AddParticipant(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockSubTourneyRepo) RemoveParticipant(ctx context.Context, tournamentID string, playerID string) error {
	return nil
}
func (m *mockSubTourneyRepo) SaveTeam(ctx context.Context, team *subTourneyDomain.Team) error {
	return nil
}
func (m *mockSubTourneyRepo) DeleteTeam(ctx context.Context, id string) error { return nil }
func (m *mockSubTourneyRepo) AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}
func (m *mockSubTourneyRepo) RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}
func (m *mockSubTourneyRepo) GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]subTourneyDomain.ParticipantSnapshot, error) {
	return nil, nil
}
func (m *mockSubTourneyRepo) GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error) {
	return "", nil
}
func (m *mockSubTourneyRepo) AddOfficial(ctx context.Context, tournamentID string, playerID string, pin string) error {
	return nil
}
func (m *mockSubTourneyRepo) RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error {
	return nil
}
func (m *mockSubTourneyRepo) GetOfficials(ctx context.Context, tournamentID string) ([]subTourneyDomain.ParticipantSnapshot, error) {
	return nil, nil
}
func (m *mockSubTourneyRepo) GetEventNumTables(ctx context.Context, eventID string) (int, error) {
	return 4, nil
}

// ── Mock Player Repository ────────────────────────────────────────────────

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

// ── Mock Division Repository ──────────────────────────────────────────────

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
	var res []*divisionDomain.Division
	for _, d := range m.divisions {
		res = append(res, d)
	}
	return res, nil
}

func (m *mockDivisionRepo) Save(ctx context.Context, d *divisionDomain.Division) error { return nil }
func (m *mockDivisionRepo) Delete(ctx context.Context, id string) error                { return nil }

// ── Tests ──────────────────────────────────────────────────────────────────

func TestCreateEventUseCase(t *testing.T) {
	eventRepo := newMockEventRepo()
	subTourneyRepo := newMockSubTourneyRepo()
	playerRepo := newMockPlayerRepo()
	divisionRepo := newMockDivisionRepo()

	p1, _ := playerDomain.NewPlayer("p1", "John", "Doe", time.Now(), "M", "USA", "", "1")
	playerRepo.players["p1"] = p1

	div1, _ := divisionDomain.NewDivision("d1", "Division 1", 1, 1000, nil, "singles", "#00ff00")
	divisionRepo.divisions["d1"] = div1

	uc := tournament.NewCreateEventUseCase(eventRepo, subTourneyRepo, playerRepo, divisionRepo)
	ctx := context.Background()

	t.Run("create event with skip elo", func(t *testing.T) {
		res, err := uc.Execute(
			ctx,
			"Summer Open",
			[]string{"d1"},
			true,
			"2026-08-01",
			"2026-08-05",
			tournament.CategoryConfig{
				Auto:           true,
				Format:         "single_elimination",
				GroupPassCount: 2,
				PlayerIDs:      []string{"p1"},
			},
			tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{},
			tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{},
			nil,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Name != "Summer Open" {
			t.Errorf("expected Summer Open, got %s", res.Name)
		}
		if len(res.Events) != 1 {
			t.Errorf("expected 1 sub-event created, got %d", len(res.Events))
		}
	})

	t.Run("invalid start/end date", func(t *testing.T) {
		_, err := uc.Execute(
			ctx,
			"Invalid Dates",
			[]string{"d1"},
			true,
			"invalid-date",
			"2026-08-05",
			tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{},
			tournament.CategoryConfig{}, tournament.CategoryConfig{}, tournament.CategoryConfig{},
			tournament.CategoryConfig{}, nil,
		)
		if err == nil {
			t.Fatal("expected error for invalid start date")
		}
	})
}

func TestTournamentCRUDUseCases(t *testing.T) {
	eventRepo := newMockEventRepo()
	ctx := context.Background()

	now := time.Now()
	t1, _ := tournamentDomain.NewEvent("t1", "Tourney 1", []string{"d1"}, false, now, now.Add(24*time.Hour))
	t2, _ := tournamentDomain.NewEvent("t2", "Tourney 2", []string{"d1"}, false, now, now.Add(24*time.Hour))
	_ = eventRepo.Save(ctx, t1)
	_ = eventRepo.Save(ctx, t2)

	t.Run("GetEventByIDUseCase", func(t *testing.T) {
		uc := tournament.NewGetEventByIDUseCase(eventRepo)
		res, err := uc.Execute(ctx, "t1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Name != "Tourney 1" {
			t.Errorf("expected Tourney 1, got %s", res.Name)
		}
	})

	t.Run("GetAllEventsUseCase", func(t *testing.T) {
		uc := tournament.NewGetAllEventsUseCase(eventRepo)
		res, err := uc.Execute(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("expected 2 events, got %d", len(res))
		}
	})

	t.Run("UpdateEventUseCase", func(t *testing.T) {
		uc := tournament.NewUpdateEventUseCase(eventRepo)
		updated, err := uc.Execute(ctx, "t1", "Updated Tourney 1", "2026-09-01", "2026-09-05", 8, map[string][]int{"d1": {1, 2}})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if updated.Name != "Updated Tourney 1" {
			t.Errorf("expected Updated Tourney 1, got %s", updated.Name)
		}
		if updated.NumTables != 8 {
			t.Errorf("expected 8 tables, got %d", updated.NumTables)
		}
	})

	t.Run("DeleteEventUseCase", func(t *testing.T) {
		uc := tournament.NewDeleteEventUseCase(eventRepo)
		if err := uc.Execute(ctx, "t1"); err != nil {
			t.Fatalf("unexpected error on single delete: %v", err)
		}
		if err := uc.ExecuteBulk(ctx, []string{"t2"}); err != nil {
			t.Fatalf("unexpected error on bulk delete: %v", err)
		}
		all, _ := eventRepo.GetAll(ctx)
		if len(all) != 0 {
			t.Errorf("expected 0 events left, got %d", len(all))
		}
	})
}

func TestGetBoardDataUseCase(t *testing.T) {
	eventRepo := newMockEventRepo()
	divisionRepo := newMockDivisionRepo()
	ctx := context.Background()

	now := time.Now()
	t1, _ := tournamentDomain.NewEvent("t1", "Grand Event", []string{"d1"}, false, now, now.Add(24*time.Hour))
	_ = eventRepo.Save(ctx, t1)

	uc := tournament.NewGetBoardDataUseCase(eventRepo, divisionRepo)

	resT, resDivs, sched, inProg, fin, err := uc.Execute(ctx, "t1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resT == nil {
		t.Errorf("expected non-nil tournament")
	}
	_ = resDivs
	_ = sched
	_ = inProg
	_ = fin
}
