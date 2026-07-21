package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// TestMatchRepository_TeamMatch_UpdateScore_AggregatesAndAdvances exercises the
// TeamMatchID aggregation branch inside UpdateScore (and indirectly the
// unexported containsMatch helper): finishing enough sub-matches must flip
// the parent team match to "finished" and advance its winner into the next
// bracket match.
func TestMatchRepository_TeamMatch_UpdateScore_AggregatesAndAdvances(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	// The bracket match the team-match winner should advance into.
	next := f.newMatch(t, "champion")
	if err := f.matchRepo.Save(ctx, next); err != nil {
		t.Fatalf("Save next: %v", err)
	}

	parent := &event.Match{
		ID:            uuid.NewString(),
		TournamentID:  f.tournament.ID,
		MatchType:     "teams",
		TeamA:         []*player.Player{f.players[0]},
		TeamB:         []*player.Player{f.players[1]},
		Status:        "in_progress",
		Stage:         "final",
		NextMatchID:   next.ID,
		NextMatchSlot: "A",
	}
	if err := f.matchRepo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}
	// Save() doesn't currently persist NextMatchID/NextMatchSlot, so patch them
	// directly via UpdateMetadata-equivalent raw update to mirror bracket wiring.
	if err := setNextMatch(ctx, f, parent.ID, next.ID, "A"); err != nil {
		t.Fatalf("setNextMatch: %v", err)
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
		t.Fatalf("expected 5 subs, got %d", len(subs))
	}

	stageRule := event.StageRule{BestOf: 5, PointsToWin: 11, PointsMargin: 2}
	winSets := []event.MatchSet{{Number: 1, ScoreA: 11, ScoreB: 5}, {Number: 2, ScoreA: 11, ScoreB: 5}, {Number: 3, ScoreA: 11, ScoreB: 5}}

	// Win 3 of 5 sub-matches for team A via UpdateScore; the 3rd should flip the parent.
	for i := 0; i < 3; i++ {
		if err := f.matchRepo.UpdateScore(ctx, subs[i].ID, winSets, stageRule); err != nil {
			t.Fatalf("UpdateScore sub[%d]: %v", i, err)
		}
	}

	parentGot, err := f.matchRepo.GetByID(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetByID parent: %v", err)
	}
	if parentGot.Status != "finished" || parentGot.WinnerTeam != "A" {
		t.Fatalf("expected parent team match to be finished with A winning, got %+v", parentGot)
	}

	// Remaining unplayed subs should have been reset to scheduled.
	remaining, err := f.matchRepo.GetSubMatches(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetSubMatches (after decision): %v", err)
	}
	for _, s := range remaining[3:] {
		if s.Status == "in_progress" {
			t.Fatalf("expected unplayed sub-match to be reset to scheduled, got %+v", s)
		}
	}

	nextGot, err := f.matchRepo.GetByID(ctx, next.ID)
	if err != nil {
		t.Fatalf("GetByID next: %v", err)
	}
	if nextGot.TeamA[0].ID != f.players[0].ID {
		t.Fatalf("expected winner to advance into next match slot A, got %+v", nextGot.TeamA)
	}
}

// setNextMatch is a small test-only helper to wire up NextMatchID/NextMatchSlot,
// since MatchRepository.Save doesn't persist bracket-advancement columns (they're
// set by other flows, e.g. bracket generation, that live outside this package).
func setNextMatch(ctx context.Context, f *matchTestFixture, matchID, nextMatchID, slot string) error {
	_, err := f.matchRepo.DB().NewUpdate().
		TableExpr("matches").
		Set("next_match_id = ?, next_match_slot = ?", nextMatchID, slot).
		Where("id = ?", matchID).
		Exec(ctx)
	return err
}

// TestMatchRepository_TeamMatch_FinishMatch_AggregatesAndAdvances mirrors the
// UpdateScore-based aggregation test above but drives sub-match decisions
// through FinishMatch directly, exercising FinishMatch's own team-match
// aggregation branch (siblings loop, parent decision, next-match advance)
// which UpdateScore doesn't reach.
func TestMatchRepository_TeamMatch_FinishMatch_AggregatesAndAdvances(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	next := f.newMatch(t, "champion")
	if err := f.matchRepo.Save(ctx, next); err != nil {
		t.Fatalf("Save next: %v", err)
	}

	parent := &event.Match{
		ID:           uuid.NewString(),
		TournamentID: f.tournament.ID,
		MatchType:    "teams",
		TeamA:        []*player.Player{f.players[0]},
		TeamB:        []*player.Player{f.players[1]},
		Status:       "in_progress",
		Stage:        "final",
	}
	if err := f.matchRepo.Save(ctx, parent); err != nil {
		t.Fatalf("Save parent: %v", err)
	}
	if err := setNextMatch(ctx, f, parent.ID, next.ID, "B"); err != nil {
		t.Fatalf("setNextMatch: %v", err)
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

	// Finish 3 of 5 subs for team B directly via FinishMatch.
	for i := 0; i < 3; i++ {
		if err := f.matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: subs[i].ID, WinnerTeam: "B"}); err != nil {
			t.Fatalf("FinishMatch sub[%d]: %v", i, err)
		}
	}

	parentGot, err := f.matchRepo.GetByID(ctx, parent.ID)
	if err != nil {
		t.Fatalf("GetByID parent: %v", err)
	}
	if parentGot.Status != "finished" || parentGot.WinnerTeam != "B" {
		t.Fatalf("expected parent team match finished with B winning, got %+v", parentGot)
	}

	nextGot, err := f.matchRepo.GetByID(ctx, next.ID)
	if err != nil {
		t.Fatalf("GetByID next: %v", err)
	}
	if nextGot.TeamB[0].ID != f.players[1].ID {
		t.Fatalf("expected winner to advance into next match slot B, got %+v", nextGot.TeamB)
	}
}

// TestMatchRepository_FinishMatch_AdvancesNextMatch_SlotA covers the
// NextMatchSlot == "A" branch of FinishMatch's own bracket-advance logic
// (the sibling test above exercises slot "B").
func TestMatchRepository_FinishMatch_AdvancesNextMatch_SlotA(t *testing.T) {
	f := newMatchTestFixture(t)
	ctx := context.Background()

	next := f.newMatch(t, "champion")
	if err := f.matchRepo.Save(ctx, next); err != nil {
		t.Fatalf("Save next: %v", err)
	}
	m := f.newMatch(t, "final")
	if err := f.matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := setNextMatch(ctx, f, m.ID, next.ID, "A"); err != nil {
		t.Fatalf("setNextMatch: %v", err)
	}

	if err := f.matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: m.ID, WinnerTeam: "A"}); err != nil {
		t.Fatalf("FinishMatch: %v", err)
	}

	nextGot, err := f.matchRepo.GetByID(ctx, next.ID)
	if err != nil {
		t.Fatalf("GetByID next: %v", err)
	}
	if nextGot.TeamA[0].ID != f.players[0].ID {
		t.Fatalf("expected winner to advance into slot A, got %+v", nextGot.TeamA)
	}
}

func TestMatchRepository_GetOccupiedTablesByEvent(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	tournamentRepo := bunRepo.NewTournamentRepository(db, eventRepo)
	ctx := context.Background()

	tr := newTestTournament(t, "Occupied Tables Tournament")
	sub := newTestEvent(t, "Occupied Tables Event")
	attachEvent(tr, sub)
	if err := tournamentRepo.Save(ctx, tr); err != nil {
		t.Fatalf("Save tournament: %v", err)
	}

	p1 := savePlayer(t, playerRepo, "Occ", "One", "M")
	p2 := savePlayer(t, playerRepo, "Occ", "Two", "M")

	table := 9
	m := &event.Match{
		ID:           uuid.NewString(),
		TournamentID: sub.ID,
		MatchType:    "singles",
		TeamA:        []*player.Player{p1},
		TeamB:        []*player.Player{p2},
		Status:       "in_progress",
		Stage:        "group",
		TableNumber:  &table,
	}
	if err := matchRepo.Save(ctx, m); err != nil {
		t.Fatalf("Save match: %v", err)
	}

	occupied, err := matchRepo.GetOccupiedTablesByEvent(ctx, tr.ID)
	if err != nil {
		t.Fatalf("GetOccupiedTablesByEvent: %v", err)
	}
	if len(occupied) != 1 || occupied[0] != 9 {
		t.Fatalf("expected table 9 occupied via parent tournament id, got %+v", occupied)
	}

	none, err := matchRepo.GetOccupiedTablesByEvent(ctx, uuid.NewString())
	if err != nil {
		t.Fatalf("GetOccupiedTablesByEvent (unrelated tournament): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("expected 0 occupied tables for unrelated tournament, got %+v", none)
	}
}
