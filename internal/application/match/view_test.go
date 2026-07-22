package match_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"table-tennis-backend/internal/application/match"
	eventDomain "table-tennis-backend/internal/domain/event"
	playerDomain "table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

var viewTestDBCounter int64

// setupViewTestDB spins up an in-memory sqlite-backed bun.DB, mirroring the pattern
// used in internal/infrastructure/persistence/bun/testutil_test.go. GetScoreFormViewUseCase
// and GetTeamMatchFormViewUseCase are hard-wired to the concrete bun repositories, so a real
// DB-backed test is the only way to exercise them.
func setupViewTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:matchviewtestdb%d?mode=memory&cache=shared", atomic.AddInt64(&viewTestDBCounter, 1))
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	sqldb.SetMaxOpenConns(1)
	t.Cleanup(func() { sqldb.Close() })

	bunDB := bun.NewDB(sqldb, sqlitedialect.New())
	bunDB.RegisterModel(
		(*bunRepo.EventParticipantModel)(nil),
		(*bunRepo.GroupParticipantModel)(nil),
		(*bunRepo.TeamPlayerModel)(nil),
	)
	models := []interface{}{
		(*bunRepo.AdminModel)(nil),
		(*bunRepo.DivisionModel)(nil),
		(*bunRepo.EventModel)(nil),
		(*bunRepo.MatchModel)(nil),
		(*bunRepo.MatchSetModel)(nil),
		(*bunRepo.PlayerModel)(nil),
		(*bunRepo.StageRuleModel)(nil),
		(*bunRepo.TournamentModel)(nil),
		(*bunRepo.EventParticipantModel)(nil),
		(*bunRepo.GroupModel)(nil),
		(*bunRepo.GroupParticipantModel)(nil),
		(*bunRepo.RuleModel)(nil),
		(*bunRepo.TeamModel)(nil),
		(*bunRepo.TeamPlayerModel)(nil),
		(*bunRepo.EventOfficialModel)(nil),
		(*bunRepo.PushSubscriptionModel)(nil),
		(*bunRepo.DivisionRuleModel)(nil),
	}
	ctx := context.Background()
	for _, model := range models {
		if _, err := bunDB.NewCreateTable().Model(model).IfNotExists().Exec(ctx); err != nil {
			t.Fatalf("create table for %T: %v", model, err)
		}
	}
	return bunDB
}

type viewFixture struct {
	playerRepo *bunRepo.PlayerRepository
	eventRepo  *bunRepo.EventRepository
	matchRepo  *bunRepo.MatchRepository

	scoreFormUC *match.GetScoreFormViewUseCase
	teamFormUC  *match.GetTeamMatchFormViewUseCase
}

func newViewFixture(t *testing.T) *viewFixture {
	t.Helper()
	db := setupViewTestDB(t)
	playerRepo := bunRepo.NewPlayerRepository(db)
	eventRepo := bunRepo.NewEventRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	divisionRepo := bunRepo.NewDivisionRepository(db)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	teamMatchUC := match.NewTeamMatchOrchestratorUseCase(matchRepo)

	return &viewFixture{
		playerRepo:  playerRepo,
		eventRepo:   eventRepo,
		matchRepo:   matchRepo,
		scoreFormUC: match.NewGetScoreFormViewUseCase(matchRepo, eventRepo, playerRepo, createMatchUC, teamMatchUC),
		teamFormUC:  match.NewGetTeamMatchFormViewUseCase(matchRepo, eventRepo),
	}
}

func (f *viewFixture) savePlayer(t *testing.T, first, last, gender string) *playerDomain.Player {
	t.Helper()
	p, err := playerDomain.NewPlayer(uuid.NewString(), first, last, time.Now(), gender, "HN", "", "")
	if err != nil {
		t.Fatalf("NewPlayer: %v", err)
	}
	if err := f.playerRepo.Save(context.Background(), p); err != nil {
		t.Fatalf("Save player: %v", err)
	}
	return p
}

func (f *viewFixture) saveEvent(t *testing.T, name, typ, format string, participants []*playerDomain.Player) *eventDomain.Event {
	t.Helper()
	start := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	e, err := eventDomain.NewTournament(uuid.NewString(), name, typ, format, "open", start, end, nil, 4, participants, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	if err := f.eventRepo.Save(context.Background(), e); err != nil {
		t.Fatalf("Save event: %v", err)
	}
	return e
}

func TestGetScoreFormViewUseCase_NewSinglesMatch(t *testing.T) {
	f := newViewFixture(t)
	p1 := f.savePlayer(t, "Alice", "Anderson", "F")
	p2 := f.savePlayer(t, "Beth", "Baker", "F")
	e := f.saveEvent(t, "Singles Open", "singles", "elimination", []*playerDomain.Player{p1, p2})

	view, err := f.scoreFormUC.Execute(context.Background(), "", e.ID, "group", 5, p1.ID, p2.ID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if view.IsTeams || view.IsDoubles {
		t.Fatalf("expected singles non-team view, got %+v", view)
	}
	if view.PlayerA != "Alice Anderson" || view.PlayerB != "Beth Baker" {
		t.Fatalf("unexpected player names: %q vs %q", view.PlayerA, view.PlayerB)
	}
	if len(view.Sets) != 5 {
		t.Fatalf("expected 5 sets for bestOf=5, got %d", len(view.Sets))
	}
	if len(view.Participants) != 2 {
		t.Fatalf("expected 2 participants, got %d", len(view.Participants))
	}
}

func TestGetScoreFormViewUseCase_ExistingMatchWithSets(t *testing.T) {
	f := newViewFixture(t)
	p1 := f.savePlayer(t, "Carol", "Castro", "F")
	p2 := f.savePlayer(t, "Diana", "Delgado", "F")
	e := f.saveEvent(t, "Singles Open 2", "singles", "elimination", []*playerDomain.Player{p1, p2})

	m := &eventDomain.Match{
		ID:           uuid.NewString(),
		TournamentID: e.ID,
		MatchType:    "singles",
		TeamA:        []*playerDomain.Player{p1},
		TeamB:        []*playerDomain.Player{p2},
		Status:       "in_progress",
		Stage:        "group",
	}
	if err := f.matchRepo.Save(context.Background(), m); err != nil {
		t.Fatalf("Save match: %v", err)
	}
	sets := []eventDomain.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 7}}
	stageRule := eventDomain.StageRule{Stage: "group", BestOf: 5, PointsToWin: 11, PointsMargin: 2}
	if err := f.matchRepo.UpdateScore(context.Background(), m.ID, sets, stageRule); err != nil {
		t.Fatalf("UpdateScore: %v", err)
	}

	view, err := f.scoreFormUC.Execute(context.Background(), m.ID, e.ID, "group", 5, "", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if view.Status != "in_progress" {
		t.Fatalf("expected status in_progress, got %q", view.Status)
	}
	if view.PlayerA != "Carol Castro" || view.PlayerB != "Diana Delgado" {
		t.Fatalf("unexpected player names resolved from existing match: %q vs %q", view.PlayerA, view.PlayerB)
	}
	found := false
	for _, s := range view.Sets {
		if s.Number == 1 && s.ScoreA == 11 && s.ScoreB == 7 {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected set 1 score 11-7 in view, got %+v", view.Sets)
	}
}

func TestGetScoreFormViewUseCase_Doubles(t *testing.T) {
	f := newViewFixture(t)
	p1 := f.savePlayer(t, "E1", "L1", "M")
	p2 := f.savePlayer(t, "E2", "L2", "M")
	p3 := f.savePlayer(t, "E3", "L3", "M")
	p4 := f.savePlayer(t, "E4", "L4", "M")
	e := f.saveEvent(t, "Doubles Open", "doubles", "elimination", []*playerDomain.Player{p1, p2, p3, p4})

	m := &eventDomain.Match{
		ID:           uuid.NewString(),
		TournamentID: e.ID,
		MatchType:    "doubles",
		TeamA:        []*playerDomain.Player{p1, p2},
		TeamB:        []*playerDomain.Player{p3, p4},
		Status:       "scheduled",
		Stage:        "group",
	}
	if err := f.matchRepo.Save(context.Background(), m); err != nil {
		t.Fatalf("Save match: %v", err)
	}

	view, err := f.scoreFormUC.Execute(context.Background(), m.ID, e.ID, "group", 5, "", "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !view.IsDoubles {
		t.Fatal("expected IsDoubles=true")
	}
	if view.PlayerANames == "" || view.PlayerBNames == "" {
		t.Fatalf("expected combined doubles names to be populated, got %q / %q", view.PlayerANames, view.PlayerBNames)
	}
}

func TestGetScoreFormViewUseCase_TeamsFormat(t *testing.T) {
	f := newViewFixture(t)
	p1 := f.savePlayer(t, "T1", "P1", "M")
	p2 := f.savePlayer(t, "T2", "P2", "M")
	p3 := f.savePlayer(t, "T3", "P3", "M")
	p4 := f.savePlayer(t, "T4", "P4", "M")
	e := f.saveEvent(t, "Teams Open", "teams", "elimination", []*playerDomain.Player{p1, p2, p3, p4})

	teamA, err := eventDomain.NewTeam(uuid.NewString(), e.ID, "Team A")
	if err != nil {
		t.Fatalf("NewTeam A: %v", err)
	}
	teamB, err := eventDomain.NewTeam(uuid.NewString(), e.ID, "Team B")
	if err != nil {
		t.Fatalf("NewTeam B: %v", err)
	}
	ctx := context.Background()
	if err := f.eventRepo.SaveTeam(ctx, teamA); err != nil {
		t.Fatalf("SaveTeam A: %v", err)
	}
	if err := f.eventRepo.SaveTeam(ctx, teamB); err != nil {
		t.Fatalf("SaveTeam B: %v", err)
	}
	for _, p := range []*playerDomain.Player{p1, p2} {
		if err := f.eventRepo.AddPlayerToTeam(ctx, teamA.ID, p.ID); err != nil {
			t.Fatalf("AddPlayerToTeam A: %v", err)
		}
	}
	for _, p := range []*playerDomain.Player{p3, p4} {
		if err := f.eventRepo.AddPlayerToTeam(ctx, teamB.ID, p.ID); err != nil {
			t.Fatalf("AddPlayerToTeam B: %v", err)
		}
	}

	// No existing match: should create the parent team match and its sub-matches,
	// then signal the handler to render the team-match form.
	view, err := f.scoreFormUC.Execute(ctx, "", e.ID, "final", 5, teamA.ID, teamB.ID)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !view.IsTeams {
		t.Fatal("expected IsTeams=true for teams-format tournament")
	}
	if view.MatchID == "" {
		t.Fatal("expected a parent team match to have been created")
	}

	subs, err := f.matchRepo.GetSubMatches(ctx, view.MatchID)
	if err != nil {
		t.Fatalf("GetSubMatches: %v", err)
	}
	if len(subs) != 5 {
		t.Fatalf("expected 5 sub-matches to have been ensured, got %d", len(subs))
	}
}

func TestGetTeamMatchFormViewUseCase(t *testing.T) {
	f := newViewFixture(t)
	p1 := f.savePlayer(t, "S1", "Q1", "M")
	p2 := f.savePlayer(t, "S2", "Q2", "M")
	p3 := f.savePlayer(t, "S3", "Q3", "M")
	p4 := f.savePlayer(t, "S4", "Q4", "M")
	e := f.saveEvent(t, "Teams Cup", "teams", "elimination", []*playerDomain.Player{p1, p2, p3, p4})

	teamA, err := eventDomain.NewTeam(uuid.NewString(), e.ID, "Squad A")
	if err != nil {
		t.Fatalf("NewTeam A: %v", err)
	}
	teamB, err := eventDomain.NewTeam(uuid.NewString(), e.ID, "Squad B")
	if err != nil {
		t.Fatalf("NewTeam B: %v", err)
	}
	ctx := context.Background()
	if err := f.eventRepo.SaveTeam(ctx, teamA); err != nil {
		t.Fatalf("SaveTeam A: %v", err)
	}
	if err := f.eventRepo.SaveTeam(ctx, teamB); err != nil {
		t.Fatalf("SaveTeam B: %v", err)
	}
	for _, p := range []*playerDomain.Player{p1, p2} {
		if err := f.eventRepo.AddPlayerToTeam(ctx, teamA.ID, p.ID); err != nil {
			t.Fatalf("AddPlayerToTeam A: %v", err)
		}
	}
	for _, p := range []*playerDomain.Player{p3, p4} {
		if err := f.eventRepo.AddPlayerToTeam(ctx, teamB.ID, p.ID); err != nil {
			t.Fatalf("AddPlayerToTeam B: %v", err)
		}
	}

	parent := &eventDomain.Match{
		ID:           uuid.NewString(),
		TournamentID: e.ID,
		MatchType:    "teams",
		TeamA:        []*playerDomain.Player{p1},
		TeamB:        []*playerDomain.Player{p3},
		Status:       "scheduled",
		Stage:        "final",
	}
	if err := f.matchRepo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}
	cmd := eventDomain.CreateSubMatchesCommand{
		ParentMatchID: parent.ID,
		TournamentID:  e.ID,
		Stage:         "final",
		TeamFormat:    "olympic",
		TeamAPlayers:  []string{p1.ID, p2.ID},
		TeamBPlayers:  []string{p3.ID, p4.ID},
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err != nil {
		t.Fatalf("CreateSubMatches: %v", err)
	}

	view, err := f.teamFormUC.Execute(ctx, parent.ID, e.ID, "final")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if view.TeamA == nil || view.TeamB == nil {
		t.Fatalf("expected both teams resolved, got TeamA=%v TeamB=%v", view.TeamA, view.TeamB)
	}
	if len(view.SubMatches) != 5 {
		t.Fatalf("expected 5 sub-match view models, got %d", len(view.SubMatches))
	}
	if view.SquadAP1 == "" || view.SquadBP1 == "" {
		t.Fatalf("expected squad assignments to be populated, got A=%q B=%q", view.SquadAP1, view.SquadBP1)
	}
	if len(view.Participants) != 4 {
		t.Fatalf("expected 4 tournament participants, got %d", len(view.Participants))
	}
}

func TestGetTeamMatchFormViewUseCase_CorbillonFormat(t *testing.T) {
	f := newViewFixture(t)
	p1 := f.savePlayer(t, "C1", "R1", "F")
	p2 := f.savePlayer(t, "C2", "R2", "F")
	p3 := f.savePlayer(t, "C3", "R3", "F")
	p4 := f.savePlayer(t, "C4", "R4", "F")
	e, err := eventDomain.NewTournament(uuid.NewString(), "Corbillon Cup", "teams", "elimination", "open",
		time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC),
		nil, 4, []*playerDomain.Player{p1, p2, p3, p4}, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	e.TeamFormat = "corbillon"
	ctx := context.Background()
	if err := f.eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	teamA, _ := eventDomain.NewTeam(uuid.NewString(), e.ID, "Squad C")
	teamB, _ := eventDomain.NewTeam(uuid.NewString(), e.ID, "Squad D")
	if err := f.eventRepo.SaveTeam(ctx, teamA); err != nil {
		t.Fatalf("SaveTeam A: %v", err)
	}
	if err := f.eventRepo.SaveTeam(ctx, teamB); err != nil {
		t.Fatalf("SaveTeam B: %v", err)
	}
	for _, p := range []*playerDomain.Player{p1, p2} {
		_ = f.eventRepo.AddPlayerToTeam(ctx, teamA.ID, p.ID)
	}
	for _, p := range []*playerDomain.Player{p3, p4} {
		_ = f.eventRepo.AddPlayerToTeam(ctx, teamB.ID, p.ID)
	}

	parent := &eventDomain.Match{
		ID:           uuid.NewString(),
		TournamentID: e.ID,
		MatchType:    "teams",
		TeamA:        []*playerDomain.Player{p1},
		TeamB:        []*playerDomain.Player{p3},
		Status:       "scheduled",
		Stage:        "final",
	}
	if err := f.matchRepo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}
	cmd := eventDomain.CreateSubMatchesCommand{
		ParentMatchID: parent.ID,
		TournamentID:  e.ID,
		Stage:         "final",
		TeamFormat:    "corbillon",
		TeamAPlayers:  []string{p1.ID, p2.ID},
		TeamBPlayers:  []string{p3.ID, p4.ID},
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err != nil {
		t.Fatalf("CreateSubMatches: %v", err)
	}

	view, err := f.teamFormUC.Execute(ctx, parent.ID, e.ID, "final")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if view.TeamFormat != "corbillon" {
		t.Fatalf("expected corbillon format preserved, got %q", view.TeamFormat)
	}
	if len(view.SubMatches) != 5 {
		t.Fatalf("expected 5 sub-match view models, got %d", len(view.SubMatches))
	}
}

func TestTeamPlayerHelpers(t *testing.T) {
	p1 := &playerDomain.Player{ID: "p1", FirstName: "Ann", LastName: "Lee"}
	p2 := &playerDomain.Player{ID: "p2", FirstName: "Bea", LastName: "Kim"}
	team := []*playerDomain.Player{p1, p2}

	if got := match.TeamPlayerIDForTest(team, 0); got != "p1" {
		t.Errorf("expected p1, got %q", got)
	}
	if got := match.TeamPlayerIDForTest(team, 1); got != "p2" {
		t.Errorf("expected p2, got %q", got)
	}
	if got := match.TeamPlayerIDForTest(team, 2); got != "" {
		t.Errorf("expected empty string for out-of-range index, got %q", got)
	}
	if got := match.TeamPlayerIDForTest(nil, 0); got != "" {
		t.Errorf("expected empty string for empty team, got %q", got)
	}

	if got := match.TeamPlayerNameForTest(team, 0); got != p1.FullName() {
		t.Errorf("expected %q, got %q", p1.FullName(), got)
	}
	if got := match.TeamPlayerNameForTest(team, 1); got != p2.FullName() {
		t.Errorf("expected %q, got %q", p2.FullName(), got)
	}
	if got := match.TeamPlayerNameForTest(nil, 0); got != "" {
		t.Errorf("expected empty string for empty team, got %q", got)
	}
}
