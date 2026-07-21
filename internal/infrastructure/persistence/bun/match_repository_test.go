package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// matchTestFixture sets up a tournament (event), 4 players, and repos wired together.
type matchTestFixture struct {
	eventRepo  *bunRepo.EventRepository
	playerRepo *bunRepo.PlayerRepository
	matchRepo  *bunRepo.MatchRepository
	tournament *event.Event
	players    []*player.Player
}

func newMatchTestFixture(t *testing.T) *matchTestFixture {
	t.Helper()
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	ctx := context.Background()

	var players []*player.Player
	for i := 0; i < 4; i++ {
		p := savePlayer(t, playerRepo, "Match", "Player", "M")
		players = append(players, p)
	}

	tr := newBareEvent(t, "Match Fixture Event", players)
	if err := eventRepo.Save(ctx, tr); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	return &matchTestFixture{
		eventRepo:  eventRepo,
		playerRepo: playerRepo,
		matchRepo:  matchRepo,
		tournament: tr,
		players:    players,
	}
}

func (f *matchTestFixture) newMatch(t *testing.T, stage string) *event.Match {
	t.Helper()
	return &event.Match{
		ID:           uuid.NewString(),
		TournamentID: f.tournament.ID,
		MatchType:    "singles",
		TeamA:        []*player.Player{f.players[0]},
		TeamB:        []*player.Player{f.players[1]},
		Status:       "scheduled",
		Stage:        stage,
	}
}

func TestMatchRepository_SaveAndGetByID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if m.Pin == "" {
		t.Fatal("expected Save to assign a PIN")
	}

	got, err := f.matchRepo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "scheduled" || len(got.TeamA) != 1 || got.TeamA[0].ID != f.players[0].ID {
		t.Fatalf("unexpected match: %+v", got)
	}
}

func TestMatchRepository_GetByID_NotFound(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.GetByID(ctx, uuid.NewString()); err == nil {
		t.Fatal("expected error for missing match, got nil")
	}
}

func TestMatchRepository_GetByID_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.GetByID(ctx, "bad-id"); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestMatchRepository_Save_DoublesTeams(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := &event.Match{
		ID:           uuid.NewString(),
		TournamentID: f.tournament.ID,
		MatchType:    "doubles",
		TeamA:        []*player.Player{f.players[0], f.players[1]},
		TeamB:        []*player.Player{f.players[2], f.players[3]},
		Status:       "scheduled",
		Stage:        "group",
	}
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := f.matchRepo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.TeamA) != 2 || len(got.TeamB) != 2 {
		t.Fatalf("expected 2 players per team, got teamA=%d teamB=%d", len(got.TeamA), len(got.TeamB))
	}
}

func TestMatchRepository_UpdateScore_DecidesMatch(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "final")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	stageRule := event.StageRule{BestOf: 5, PointsToWin: 11, PointsMargin: 2}
	sets := []event.MatchSet{
		{Number: 1, ScoreA: 11, ScoreB: 5},
		{Number: 2, ScoreA: 11, ScoreB: 7},
		{Number: 3, ScoreA: 11, ScoreB: 9},
	}
	if err := f.matchRepo.UpdateScore(ctx, m.ID, sets, stageRule); err != nil {
		t.Fatalf("UpdateScore: %v", err)
	}

	got, err := f.matchRepo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "finished" || got.WinnerTeam != "A" {
		t.Fatalf("expected match A to have won and be finished, got %+v", got)
	}
	if len(got.Sets) != 3 {
		t.Fatalf("expected 3 sets, got %d", len(got.Sets))
	}
}

func TestMatchRepository_UpdateScore_InProgress(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "final")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	stageRule := event.StageRule{BestOf: 5, PointsToWin: 11, PointsMargin: 2}
	sets := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	if err := f.matchRepo.UpdateScore(ctx, m.ID, sets, stageRule); err != nil {
		t.Fatalf("UpdateScore: %v", err)
	}

	got, err := f.matchRepo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Status != "in_progress" {
		t.Fatalf("expected in_progress status, got %q", got.Status)
	}
}

func TestMatchRepository_UpdateScore_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	err := f.matchRepo.UpdateScore(ctx, "bad-id", nil, event.StageRule{})
	if err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestMatchRepository_CountUnfinishedAndFinishedMatches(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m1 := f.newMatch(t, "group")
	m2 := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m1); err != nil {
		t.Fatalf("Save m1: %v", err)
	}
	if err := f.matchRepo.Save(ctx, m2); err != nil {
		t.Fatalf("Save m2: %v", err)
	}

	unfinished, err := f.matchRepo.CountUnfinishedMatches(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("CountUnfinishedMatches: %v", err)
	}
	if unfinished != 2 {
		t.Fatalf("expected 2 unfinished matches, got %d", unfinished)
	}

	if err := f.matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: m1.ID, WinnerTeam: "A"}); err != nil {
		t.Fatalf("FinishMatch: %v", err)
	}

	unfinished, err = f.matchRepo.CountUnfinishedMatches(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("CountUnfinishedMatches (after finish): %v", err)
	}
	if unfinished != 1 {
		t.Fatalf("expected 1 unfinished match, got %d", unfinished)
	}

	finished, err := f.matchRepo.CountFinishedMatches(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("CountFinishedMatches: %v", err)
	}
	if finished != 1 {
		t.Fatalf("expected 1 finished match, got %d", finished)
	}
}

func TestMatchRepository_HasStartedOrFinishedMatches(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	started, err := f.matchRepo.HasStartedOrFinishedMatches(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("HasStartedOrFinishedMatches: %v", err)
	}
	if started {
		t.Fatal("expected no started/finished matches yet")
	}

	m := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := f.matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: m.ID, WinnerTeam: "A"}); err != nil {
		t.Fatalf("FinishMatch: %v", err)
	}

	started, err = f.matchRepo.HasStartedOrFinishedMatches(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("HasStartedOrFinishedMatches (after finish): %v", err)
	}
	if !started {
		t.Fatal("expected a finished match to be reported")
	}
}

func TestMatchRepository_DeleteByTournament(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := f.matchRepo.DeleteByTournament(ctx, f.tournament.ID); err != nil {
		t.Fatalf("DeleteByTournament: %v", err)
	}

	if _, err := f.matchRepo.GetByID(ctx, m.ID); err == nil {
		t.Fatal("expected match to be deleted")
	}
}

func TestMatchRepository_GetAll(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	empty, err := f.matchRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll (empty): %v", err)
	}
	if len(empty) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(empty))
	}

	m1 := f.newMatch(t, "group")
	m2 := f.newMatch(t, "r16")
	if err := f.matchRepo.Save(ctx, m1); err != nil {
		t.Fatalf("Save m1: %v", err)
	}
	if err := f.matchRepo.Save(ctx, m2); err != nil {
		t.Fatalf("Save m2: %v", err)
	}

	all, err := f.matchRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 matches, got %d", len(all))
	}
}

func TestMatchRepository_GetOccupiedTables(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "group")
	table := 3
	m.Status = "in_progress"
	m.TableNumber = &table
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	occupiedByTournament, err := f.matchRepo.GetOccupiedTablesByTournament(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("GetOccupiedTablesByTournament: %v", err)
	}
	if len(occupiedByTournament) != 1 || occupiedByTournament[0] != 3 {
		t.Fatalf("expected table 3 occupied, got %+v", occupiedByTournament)
	}

	occupiedByEvent, err := f.matchRepo.GetOccupiedTablesByEvent(ctx, f.tournament.ID)
	if err != nil {
		t.Fatalf("GetOccupiedTablesByEvent: %v", err)
	}
	if len(occupiedByEvent) != 0 {
		// GetOccupiedTablesByEvent looks up events whose tournament_id == the given
		// eventID, i.e. it expects a *parent* tournament id, not an event id
		// directly. With no EventModel rows pointing at f.tournament.ID it
		// should come back empty.
		t.Fatalf("expected 0 occupied tables via GetOccupiedTablesByEvent (no parent link), got %+v", occupiedByEvent)
	}
}

func TestMatchRepository_IsTableOccupiedByOtherMatch(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m1 := f.newMatch(t, "group")
	table := 7
	m1.Status = "in_progress"
	m1.TableNumber = &table
	if err := f.matchRepo.Save(ctx, m1); err != nil {
		t.Fatalf("Save m1: %v", err)
	}

	m2 := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m2); err != nil {
		t.Fatalf("Save m2: %v", err)
	}

	occupied, err := f.matchRepo.IsTableOccupiedByOtherMatch(ctx, m2.ID, 7)
	if err != nil {
		t.Fatalf("IsTableOccupiedByOtherMatch: %v", err)
	}
	if !occupied {
		t.Fatal("expected table 7 to be occupied by m1")
	}

	notOccupied, err := f.matchRepo.IsTableOccupiedByOtherMatch(ctx, m1.ID, 7)
	if err != nil {
		t.Fatalf("IsTableOccupiedByOtherMatch (self): %v", err)
	}
	if notOccupied {
		t.Fatal("expected the occupying match itself to be excluded")
	}
}

func TestMatchRepository_UpdateMetadata(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	refID := f.players[2].ID
	table := 4
	if err := f.matchRepo.UpdateMetadata(ctx, m.ID, &refID, &table); err != nil {
		t.Fatalf("UpdateMetadata: %v", err)
	}

	got, err := f.matchRepo.GetByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.RefereeID == nil || *got.RefereeID != refID {
		t.Fatalf("expected referee to be set, got %+v", got.RefereeID)
	}
	if got.TableNumber == nil || *got.TableNumber != 4 {
		t.Fatalf("expected table 4, got %+v", got.TableNumber)
	}
}

func TestMatchRepository_UpdateMetadata_InvalidID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if err := f.matchRepo.UpdateMetadata(ctx, "bad-id", nil, nil); err == nil {
		t.Fatal("expected error for invalid UUID, got nil")
	}
}

func TestMatchRepository_GetMatchByParticipants(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := f.matchRepo.GetMatchByParticipants(ctx, f.tournament.ID, f.players[0].ID, f.players[1].ID, "group")
	if err != nil {
		t.Fatalf("GetMatchByParticipants: %v", err)
	}
	if got.ID != m.ID {
		t.Fatalf("expected to find match %s, got %s", m.ID, got.ID)
	}

	// Order reversed should also match.
	got2, err := f.matchRepo.GetMatchByParticipants(ctx, f.tournament.ID, f.players[1].ID, f.players[0].ID, "group")
	if err != nil {
		t.Fatalf("GetMatchByParticipants (reversed): %v", err)
	}
	if got2.ID != m.ID {
		t.Fatalf("expected to find match %s (reversed), got %s", m.ID, got2.ID)
	}
}

func TestMatchRepository_GetMatchByParticipants_NotFound(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	if _, err := f.matchRepo.GetMatchByParticipants(ctx, f.tournament.ID, f.players[0].ID, f.players[1].ID, "group"); err == nil {
		t.Fatal("expected error when no match exists, got nil")
	}
}

func TestMatchRepository_FindOrCreateMatch(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	id1, err := f.matchRepo.FindOrCreateMatch(ctx, f.tournament.ID, f.players[0].ID, f.players[1].ID, "group", "singles")
	if err != nil {
		t.Fatalf("FindOrCreateMatch: %v", err)
	}
	if id1 == "" {
		t.Fatal("expected a match ID to be returned")
	}

	// Calling again with the same pair/stage should return the same match.
	id2, err := f.matchRepo.FindOrCreateMatch(ctx, f.tournament.ID, f.players[0].ID, f.players[1].ID, "group", "singles")
	if err != nil {
		t.Fatalf("FindOrCreateMatch (again): %v", err)
	}
	if id1 != id2 {
		t.Fatalf("expected idempotent match creation, got %s and %s", id1, id2)
	}
}

func TestMatchRepository_GetSubMatches_And_CreateSubMatches(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	parent := &event.Match{
		ID:           uuid.NewString(),
		TournamentID: f.tournament.ID,
		MatchType:    "teams",
		TeamA:        []*player.Player{f.players[0]},
		TeamB:        []*player.Player{f.players[1]},
		Status:       "scheduled",
		Stage:        "final",
	}
	if err := f.matchRepo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}

	cmd := event.CreateSubMatchesCommand{
		ParentMatchID: parent.ID,
		TournamentID:  f.tournament.ID,
		Stage:         "final",
		TeamFormat:    "olympic",
		TeamAPlayers:  []string{f.players[0].ID, f.players[1].ID},
		TeamBPlayers:  []string{f.players[2].ID, f.players[3].ID},
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err != nil {
		t.Fatalf("CreateSubMatches: %v", err)
	}

	subs, err := f.matchRepo.GetSubMatches(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetSubMatches: %v", err)
	}
	if len(subs) != 5 {
		t.Fatalf("expected 5 sub-matches, got %d", len(subs))
	}

	// Idempotency: calling again should not create duplicates.
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err != nil {
		t.Fatalf("CreateSubMatches (again): %v", err)
	}
	subs, err = f.matchRepo.GetSubMatches(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetSubMatches (after repeat): %v", err)
	}
	if len(subs) != 5 {
		t.Fatalf("expected still 5 sub-matches after repeat call, got %d", len(subs))
	}
}

func TestMatchRepository_CreateSubMatches_MissingPlayers(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	cmd := event.CreateSubMatchesCommand{
		ParentMatchID: uuid.NewString(),
		TournamentID:  f.tournament.ID,
		Stage:         "final",
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err == nil {
		t.Fatal("expected error when teams have no players")
	}
}

func TestMatchRepository_UpdateSubMatchSquads(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	parent := &event.Match{
		ID:           uuid.NewString(),
		TournamentID: f.tournament.ID,
		MatchType:    "teams",
		TeamA:        []*player.Player{f.players[0]},
		TeamB:        []*player.Player{f.players[1]},
		Status:       "scheduled",
		Stage:        "final",
	}
	if err := f.matchRepo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}

	cmd := event.CreateSubMatchesCommand{
		ParentMatchID: parent.ID,
		TournamentID:  f.tournament.ID,
		Stage:         "final",
		TeamFormat:    "olympic",
		TeamAPlayers:  []string{f.players[0].ID, f.players[1].ID},
		TeamBPlayers:  []string{f.players[2].ID, f.players[3].ID},
	}
	if err := f.matchRepo.CreateSubMatches(ctx, cmd); err != nil {
		t.Fatalf("CreateSubMatches: %v", err)
	}
	subs, err := f.matchRepo.GetSubMatches(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetSubMatches: %v", err)
	}

	updateCmd := event.UpdateSubMatchSquadsCommand{
		ParentMatchID: parent.ID,
		Assignments: []event.SubMatchSquadAssignment{
			{
				SubMatchID:     subs[0].ID,
				TeamAPlayer1ID: f.players[1].ID,
				TeamBPlayer1ID: f.players[3].ID,
			},
		},
	}
	if err := f.matchRepo.UpdateSubMatchSquads(ctx, updateCmd); err != nil {
		t.Fatalf("UpdateSubMatchSquads: %v", err)
	}

	got, err := f.matchRepo.GetByID(ctx, subs[0].ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.TeamA[0].ID != f.players[1].ID || got.TeamB[0].ID != f.players[3].ID {
		t.Fatalf("expected squad reassignment to take effect, got %+v", got)
	}
}

func TestMatchRepository_GenerateUniquePin(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	pin := f.matchRepo.GenerateUniquePin(ctx)
	if len(pin) != 4 {
		t.Fatalf("expected a 4-digit pin, got %q", pin)
	}
}

func TestMatchRepository_GetSets_And_GetModelByID(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	m := f.newMatch(t, "group")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	stageRule := event.StageRule{BestOf: 5, PointsToWin: 11, PointsMargin: 2}
	sets := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}}
	if err := f.matchRepo.UpdateScore(ctx, m.ID, sets, stageRule); err != nil {
		t.Fatalf("UpdateScore: %v", err)
	}

	gotSets, err := f.matchRepo.GetSets(ctx, m.ID)
	if err != nil {
		t.Fatalf("GetSets: %v", err)
	}
	if len(gotSets) != 1 || gotSets[0].ScoreA != 11 {
		t.Fatalf("unexpected sets: %+v", gotSets)
	}

	mUUID, err := uuid.Parse(m.ID)
	if err != nil {
		t.Fatalf("parse uuid: %v", err)
	}
	model, err := f.matchRepo.GetModelByID(ctx, mUUID)
	if err != nil {
		t.Fatalf("GetModelByID: %v", err)
	}
	if model.ID.String() != m.ID {
		t.Fatalf("unexpected model: %+v", model)
	}
}

func TestMatchRepository_DB(t *testing.T) {
	f := newMatchTestFixture(t)
	if f.matchRepo.DB() == nil {
		t.Fatal("expected DB() to return a non-nil bun.DB")
	}
}
