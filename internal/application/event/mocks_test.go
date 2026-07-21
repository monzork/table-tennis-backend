package event

import (
	"context"
	"errors"

	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	singleTournamentDomain "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/domain/tournaments"
	"table-tennis-backend/internal/infrastructure/identity"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

var errNotFound = errors.New("not found")

// ─── event.Repository mock ─────────────────────────────────────────────────

type mockRepo struct {
	events map[string]*tournamentDomain.Event

	getErr                   error
	getAllErr                error
	saveErr                  error
	updateErr                error
	updateGroupsErr          error
	deleteErr                error
	snapshotsErr             error
	snapshots                []tournamentDomain.ParticipantSnapshot
	officialsErr             error
	officials                []tournamentDomain.ParticipantSnapshot
	addOfficialErr           error
	removeOfficErr           error
	addParticipErr           error
	removeParticiErr         error
	saveTeamErr              error
	deleteTeamErr            error
	addToTeamErr             error
	removeFromTeamErr        error
	numTables                int
	numTablesErr             error
	pinLookup                string
	pinLookupErr             error
	updateEloBeforeErr       error
	updateParticipantsEloErr error
	updateEventIDBulkErr     error

	saveCalls         int
	updateCalls       int
	updateGroupsCalls int
	deleteCalls       int
}

func newMockRepo() *mockRepo {
	return &mockRepo{events: make(map[string]*tournamentDomain.Event)}
}

func (m *mockRepo) Save(ctx context.Context, t *tournamentDomain.Event) error {
	m.saveCalls++
	if m.saveErr != nil {
		return m.saveErr
	}
	m.events[t.ID] = t
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*tournamentDomain.Event, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	t, ok := m.events[id]
	if !ok {
		return nil, errNotFound
	}
	return t, nil
}

func (m *mockRepo) GetAll(ctx context.Context) ([]*tournamentDomain.Event, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	var res []*tournamentDomain.Event
	for _, t := range m.events {
		res = append(res, t)
	}
	return res, nil
}

func (m *mockRepo) Update(ctx context.Context, t *tournamentDomain.Event) error {
	m.updateCalls++
	if m.updateErr != nil {
		return m.updateErr
	}
	m.events[t.ID] = t
	return nil
}

func (m *mockRepo) UpdateEventIDBulk(ctx context.Context, tournamentIDs []string, eventID string) error {
	return m.updateEventIDBulkErr
}

func (m *mockRepo) UpdateGroups(ctx context.Context, t *tournamentDomain.Event) error {
	m.updateGroupsCalls++
	if m.updateGroupsErr != nil {
		return m.updateGroupsErr
	}
	m.events[t.ID] = t
	return nil
}

func (m *mockRepo) Delete(ctx context.Context, id string) error {
	m.deleteCalls++
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.events, id)
	return nil
}

func (m *mockRepo) UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}

func (m *mockRepo) UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*playerDomain.Player) error {
	return m.updateParticipantsEloErr
}

func (m *mockRepo) UpdateParticipantEloBefore(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return m.updateEloBeforeErr
}

func (m *mockRepo) AddParticipant(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return m.addParticipErr
}

func (m *mockRepo) RemoveParticipant(ctx context.Context, tournamentID string, playerID string) error {
	return m.removeParticiErr
}

func (m *mockRepo) SaveTeam(ctx context.Context, team *tournamentDomain.Team) error {
	return m.saveTeamErr
}

func (m *mockRepo) DeleteTeam(ctx context.Context, id string) error {
	return m.deleteTeamErr
}

func (m *mockRepo) AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error {
	return m.addToTeamErr
}

func (m *mockRepo) RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error {
	return m.removeFromTeamErr
}

func (m *mockRepo) GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]tournamentDomain.ParticipantSnapshot, error) {
	if m.snapshotsErr != nil {
		return nil, m.snapshotsErr
	}
	return m.snapshots, nil
}

func (m *mockRepo) GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error) {
	return m.pinLookup, m.pinLookupErr
}

func (m *mockRepo) AddOfficial(ctx context.Context, tournamentID string, playerID string, pin string) error {
	return m.addOfficialErr
}

func (m *mockRepo) RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error {
	return m.removeOfficErr
}

func (m *mockRepo) GetOfficials(ctx context.Context, tournamentID string) ([]tournamentDomain.ParticipantSnapshot, error) {
	if m.officialsErr != nil {
		return nil, m.officialsErr
	}
	return m.officials, nil
}

func (m *mockRepo) GetEventNumTables(ctx context.Context, eventID string) (int, error) {
	return m.numTables, m.numTablesErr
}

// ─── event.MatchRepository mock ────────────────────────────────────────────

type mockMatchRepo struct {
	unfinishedCount    int
	unfinishedErr      error
	finishedCount      int
	finishedErr        error
	occupiedByEvent    []int
	occupiedByEventErr error
	occupiedByTourn    []int
	occupiedByTournErr error
	hasActivity        bool
	hasActivityErr     error
	deleteByTournErr   error
	saveErr            error
	savedMatches       []*tournamentDomain.Match
}

func (m *mockMatchRepo) Save(ctx context.Context, match *tournamentDomain.Match) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.savedMatches = append(m.savedMatches, match)
	return nil
}
func (m *mockMatchRepo) CountUnfinishedMatches(ctx context.Context, tournamentID string) (int, error) {
	return m.unfinishedCount, m.unfinishedErr
}
func (m *mockMatchRepo) CountFinishedMatches(ctx context.Context, tournamentID string) (int, error) {
	return m.finishedCount, m.finishedErr
}
func (m *mockMatchRepo) GetAll(ctx context.Context) ([]*tournamentDomain.Match, error) {
	return nil, nil
}
func (m *mockMatchRepo) GetByID(ctx context.Context, id string) (*tournamentDomain.Match, error) {
	return nil, nil
}
func (m *mockMatchRepo) GetSubMatches(ctx context.Context, parentMatchID string) ([]*tournamentDomain.Match, error) {
	return nil, nil
}
func (m *mockMatchRepo) GetMatchByParticipants(ctx context.Context, tournamentID, p1ID, p2ID, stage string) (*tournamentDomain.Match, error) {
	return nil, nil
}
func (m *mockMatchRepo) UpdateScore(ctx context.Context, id string, sets []tournamentDomain.MatchSet, stageRule tournamentDomain.StageRule) error {
	return nil
}
func (m *mockMatchRepo) GetOccupiedTablesByEvent(ctx context.Context, eventID string) ([]int, error) {
	return m.occupiedByEvent, m.occupiedByEventErr
}
func (m *mockMatchRepo) GetOccupiedTablesByTournament(ctx context.Context, tournamentID string) ([]int, error) {
	return m.occupiedByTourn, m.occupiedByTournErr
}
func (m *mockMatchRepo) IsTableOccupiedByOtherMatch(ctx context.Context, matchID string, tableNumber int) (bool, error) {
	return false, nil
}
func (m *mockMatchRepo) UpdateMetadata(ctx context.Context, matchID string, refereeID *string, tableNumber *int) error {
	return nil
}
func (m *mockMatchRepo) HasStartedOrFinishedMatches(ctx context.Context, tournamentID string) (bool, error) {
	return m.hasActivity, m.hasActivityErr
}
func (m *mockMatchRepo) DeleteByTournament(ctx context.Context, tournamentID string) error {
	return m.deleteByTournErr
}
func (m *mockMatchRepo) FinishMatch(ctx context.Context, cmd tournamentDomain.FinishMatchCommand) error {
	return nil
}
func (m *mockMatchRepo) FindOrCreateMatch(ctx context.Context, tournamentID, p1ID, p2ID, stage, matchType string) (string, error) {
	return "", nil
}
func (m *mockMatchRepo) CreateSubMatches(ctx context.Context, cmd tournamentDomain.CreateSubMatchesCommand) error {
	return nil
}
func (m *mockMatchRepo) UpdateSubMatchSquads(ctx context.Context, cmd tournamentDomain.UpdateSubMatchSquadsCommand) error {
	return nil
}

// ─── player.Repository mock ────────────────────────────────────────────────

type mockPlayerRepo struct {
	players         map[string]*playerDomain.Player
	getByIdErr      error
	getByIDsErr     error
	saveErr         error
	saveMultipleErr error
	deleteErr       error
	searchErr       error
	getAllErr       error
	savedPlayers    []*playerDomain.Player
}

func newMockPlayerRepo() *mockPlayerRepo {
	return &mockPlayerRepo{players: make(map[string]*playerDomain.Player)}
}

func (m *mockPlayerRepo) GetById(ctx context.Context, id string) (*playerDomain.Player, error) {
	if m.getByIdErr != nil {
		return nil, m.getByIdErr
	}
	p, ok := m.players[id]
	if !ok {
		return nil, errNotFound
	}
	return p, nil
}

func (m *mockPlayerRepo) GetByIDs(ctx context.Context, ids []string) ([]*playerDomain.Player, error) {
	if m.getByIDsErr != nil {
		return nil, m.getByIDsErr
	}
	var res []*playerDomain.Player
	for _, id := range ids {
		if p, ok := m.players[id]; ok {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockPlayerRepo) Save(ctx context.Context, p *playerDomain.Player) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.players[p.ID] = p
	m.savedPlayers = append(m.savedPlayers, p)
	return nil
}

func (m *mockPlayerRepo) SaveMultiple(ctx context.Context, players []*playerDomain.Player) error {
	if m.saveMultipleErr != nil {
		return m.saveMultipleErr
	}
	for _, p := range players {
		m.players[p.ID] = p
	}
	return nil
}

func (m *mockPlayerRepo) Delete(ctx context.Context, id string) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.players, id)
	return nil
}

func (m *mockPlayerRepo) Search(ctx context.Context, query string) ([]*playerDomain.Player, error) {
	return nil, m.searchErr
}

func (m *mockPlayerRepo) SearchForSelection(ctx context.Context, query, gender string) ([]*playerDomain.Player, error) {
	return nil, nil
}

func (m *mockPlayerRepo) GetAll(ctx context.Context) ([]*playerDomain.Player, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	var res []*playerDomain.Player
	for _, p := range m.players {
		res = append(res, p)
	}
	return res, nil
}

func (m *mockPlayerRepo) GetAllSingles(ctx context.Context) ([]*playerDomain.Player, error) {
	return m.GetAll(ctx)
}

func (m *mockPlayerRepo) GetAllDoubles(ctx context.Context) ([]*playerDomain.Player, error) {
	return m.GetAll(ctx)
}

func (m *mockPlayerRepo) GetSinglesByGender(ctx context.Context, gender string) ([]*playerDomain.Player, error) {
	return nil, nil
}

func (m *mockPlayerRepo) GetDoublesByGender(ctx context.Context, gender string) ([]*playerDomain.Player, error) {
	return nil, nil
}

// ─── division.Repository mock ──────────────────────────────────────────────

type mockDivisionRepo struct {
	divisions  []*divisionDomain.Division
	getAllErr  error
	getByIdErr error
	saveErr    error
	deleteErr  error
}

func (m *mockDivisionRepo) Save(ctx context.Context, d *divisionDomain.Division) error {
	return m.saveErr
}
func (m *mockDivisionRepo) GetAll(ctx context.Context) ([]*divisionDomain.Division, error) {
	if m.getAllErr != nil {
		return nil, m.getAllErr
	}
	return m.divisions, nil
}
func (m *mockDivisionRepo) Delete(ctx context.Context, id string) error {
	return m.deleteErr
}
func (m *mockDivisionRepo) GetById(ctx context.Context, id string) (*divisionDomain.Division, error) {
	if m.getByIdErr != nil {
		return nil, m.getByIdErr
	}
	for _, d := range m.divisions {
		if d.ID == id {
			return d, nil
		}
	}
	return nil, errNotFound
}

// ─── tournaments.Dispatcher mock ───────────────────────────────────────────

type mockDispatcher struct {
	dispatchedAsync []tournaments.Tournament
	dispatchErr     error
}

func (m *mockDispatcher) Subscribe(eventName string, handler tournaments.EventHandler) {}
func (m *mockDispatcher) Dispatch(ctx context.Context, t tournaments.Tournament) error {
	return m.dispatchErr
}
func (m *mockDispatcher) DispatchAsync(ctx context.Context, t tournaments.Tournament) {
	m.dispatchedAsync = append(m.dispatchedAsync, t)
}

// ─── pdf.Generator mock ────────────────────────────────────────────────────

type mockPdfGenerator struct {
	tournamentReportBytes []byte
	tournamentReportErr   error
	eventReportBytes      []byte
	eventReportErr        error
}

func (m *mockPdfGenerator) GenerateTournamentReport(t *tournamentDomain.Event, divs []*divisionDomain.Division) ([]byte, error) {
	return m.tournamentReportBytes, m.tournamentReportErr
}

func (m *mockPdfGenerator) GenerateEventReport(e *singleTournamentDomain.Tournament, divs []*divisionDomain.Division) ([]byte, error) {
	return m.eventReportBytes, m.eventReportErr
}

// ─── domain/tournament (singular) Repository mock ──────────────────────────

type mockSingleTournamentRepo struct {
	tournaments    map[string]*singleTournamentDomain.Tournament
	getByIDErr     error
	getByIDDeepErr error
}

func newMockSingleTournamentRepo() *mockSingleTournamentRepo {
	return &mockSingleTournamentRepo{tournaments: make(map[string]*singleTournamentDomain.Tournament)}
}

func (m *mockSingleTournamentRepo) Save(ctx context.Context, e *singleTournamentDomain.Tournament) error {
	m.tournaments[e.ID] = e
	return nil
}
func (m *mockSingleTournamentRepo) Update(ctx context.Context, e *singleTournamentDomain.Tournament) error {
	m.tournaments[e.ID] = e
	return nil
}
func (m *mockSingleTournamentRepo) GetByID(ctx context.Context, id string) (*singleTournamentDomain.Tournament, error) {
	if m.getByIDErr != nil {
		return nil, m.getByIDErr
	}
	t, ok := m.tournaments[id]
	if !ok {
		return nil, errNotFound
	}
	return t, nil
}
func (m *mockSingleTournamentRepo) GetByIDDeep(ctx context.Context, id string) (*singleTournamentDomain.Tournament, error) {
	if m.getByIDDeepErr != nil {
		return nil, m.getByIDDeepErr
	}
	t, ok := m.tournaments[id]
	if !ok {
		return nil, errNotFound
	}
	return t, nil
}
func (m *mockSingleTournamentRepo) GetAll(ctx context.Context) ([]*singleTournamentDomain.Tournament, error) {
	var res []*singleTournamentDomain.Tournament
	for _, t := range m.tournaments {
		res = append(res, t)
	}
	return res, nil
}
func (m *mockSingleTournamentRepo) Delete(ctx context.Context, id string) error {
	delete(m.tournaments, id)
	return nil
}
func (m *mockSingleTournamentRepo) DeleteEvents(ctx context.Context, ids []string) error {
	return nil
}
