package bun_test

import (
	"context"
	"testing"
	"time"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// TestEventRepository_GetByID_And_GetByEventID_Deep_TeamsWithGroupsAndMatches
// exercises the team/doubles-specific branches (team-based group participants,
// parent team matches with sub-match win aggregation) in both GetByID and the
// deep variant of GetByEventID, which are otherwise hard to reach through the
// simpler singles-event tests.
func TestEventRepository_GetByID_And_GetByEventID_Deep_TeamsWithGroupsAndMatches(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	playerRepo := bunRepo.NewPlayerRepository(db)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	ctx := context.Background()

	tournamentID := uuid.New()
	tIDStr := tournamentID.String()

	// A doubles event so isTeamType branches are exercised.
	start := time.Now()
	e, err := event.NewTournament(uuid.NewString(), "Deep Doubles", "doubles", "elimination", "open", start, start.Add(time.Hour), nil, 2, nil, false)
	if err != nil {
		t.Fatalf("NewTournament: %v", err)
	}
	e.EventID = &tIDStr
	if err := eventRepo.Save(ctx, e); err != nil {
		t.Fatalf("Save event: %v", err)
	}

	pA1 := savePlayer(t, playerRepo, "TeamA", "One", "M")
	pA2 := savePlayer(t, playerRepo, "TeamA", "Two", "M")
	pB1 := savePlayer(t, playerRepo, "TeamB", "One", "M")
	pB2 := savePlayer(t, playerRepo, "TeamB", "Two", "M")

	teamA, err := event.NewTeam(uuid.NewString(), e.ID, "Team A")
	if err != nil {
		t.Fatalf("NewTeam A: %v", err)
	}
	teamB, err := event.NewTeam(uuid.NewString(), e.ID, "Team B")
	if err != nil {
		t.Fatalf("NewTeam B: %v", err)
	}
	if err := eventRepo.SaveTeam(ctx, teamA); err != nil {
		t.Fatalf("SaveTeam A: %v", err)
	}
	if err := eventRepo.SaveTeam(ctx, teamB); err != nil {
		t.Fatalf("SaveTeam B: %v", err)
	}
	for _, p := range []*player.Player{pA1, pA2} {
		if err := eventRepo.AddPlayerToTeam(ctx, teamA.ID, p.ID); err != nil {
			t.Fatalf("AddPlayerToTeam A/%s: %v", p.ID, err)
		}
	}
	for _, p := range []*player.Player{pB1, pB2} {
		if err := eventRepo.AddPlayerToTeam(ctx, teamB.ID, p.ID); err != nil {
			t.Fatalf("AddPlayerToTeam B/%s: %v", p.ID, err)
		}
	}

	// A group whose "players" are team placeholders (ID == team ID), mirroring
	// how the application represents team/doubles groups (see Event.MovePlayer).
	teamAPlaceholder := &player.Player{ID: teamA.ID, FirstName: teamA.Name}
	teamBPlaceholder := &player.Player{ID: teamB.ID, FirstName: teamB.Name}
	e.Groups = []event.Group{
		{ID: uuid.NewString(), TournamentID: e.ID, Name: "Group A", Players: []*player.Player{teamAPlaceholder, teamBPlaceholder}},
	}
	if err := eventRepo.UpdateGroups(ctx, e); err != nil {
		t.Fatalf("UpdateGroups: %v", err)
	}

	// Parent team match, represented with one member per team as the nominal player.
	parentMatch := &event.Match{
		ID:           uuid.NewString(),
		TournamentID: e.ID,
		MatchType:    "teams",
		TeamA:        []*player.Player{pA1},
		TeamB:        []*player.Player{pB1},
		Status:       "in_progress",
		Stage:        "final",
	}
	if err := matchRepo.Save(ctx, parentMatch); err != nil {
		t.Fatalf("Save parent match: %v", err)
	}

	// Sub-matches under the parent team match.
	cmd := event.CreateSubMatchesCommand{
		ParentMatchID: parentMatch.ID,
		TournamentID:  e.ID,
		Stage:         "final",
		TeamFormat:    "olympic",
		TeamAPlayers:  []string{pA1.ID, pA2.ID},
		TeamBPlayers:  []string{pB1.ID, pB2.ID},
	}
	if err := matchRepo.CreateSubMatches(ctx, cmd); err != nil {
		t.Fatalf("CreateSubMatches: %v", err)
	}
	subs, err := matchRepo.GetSubMatches(ctx, parentMatch.ID)
	if err != nil {
		t.Fatalf("GetSubMatches: %v", err)
	}
	if len(subs) != 5 {
		t.Fatalf("expected 5 sub-matches, got %d", len(subs))
	}
	// Finish the first sub-match so the parent's virtual-set aggregation branch runs.
	if err := matchRepo.FinishMatch(ctx, event.FinishMatchCommand{MatchID: subs[0].ID, WinnerTeam: "A"}); err != nil {
		t.Fatalf("FinishMatch sub[0]: %v", err)
	}

	// --- GetByID (non-deep helper method that loads the full match/team/group graph) ---
	got, err := eventRepo.GetByID(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if len(got.Teams) != 2 {
		t.Fatalf("expected 2 teams, got %d", len(got.Teams))
	}
	if len(got.Groups) != 1 || len(got.Groups[0].Players) != 2 {
		t.Fatalf("expected 1 group with 2 team-based players, got %+v", got.Groups)
	}
	foundParent := false
	for _, m := range got.Matches {
		if m.ID == parentMatch.ID {
			foundParent = true
			if len(m.Sets) != 1 || m.Sets[0].ScoreA != 1 {
				t.Fatalf("expected virtual set reflecting 1 sub-win for A, got %+v", m.Sets)
			}
		}
	}
	if !foundParent {
		t.Fatal("expected to find the parent team match in GetByID results")
	}

	// --- GetByEventID with deep=true (tournament-level aggregate view) ---
	deepEvents, err := eventRepo.GetByEventID(ctx, tournamentID, true)
	if err != nil {
		t.Fatalf("GetByEventID (deep): %v", err)
	}
	if len(deepEvents) != 1 {
		t.Fatalf("expected 1 event, got %d", len(deepEvents))
	}
	de := deepEvents[0]
	if len(de.Teams) != 2 {
		t.Fatalf("expected 2 teams in deep view, got %d", len(de.Teams))
	}
	if len(de.Groups) != 1 || len(de.Groups[0].Players) != 2 {
		t.Fatalf("expected 1 group with 2 team-based players in deep view, got %+v", de.Groups)
	}
	if len(de.Matches) == 0 {
		t.Fatalf("expected matches to be populated in deep view")
	}

	// --- GetByEventID with deep=false should skip match loading ---
	liteEvents, err := eventRepo.GetByEventID(ctx, tournamentID, false)
	if err != nil {
		t.Fatalf("GetByEventID (lite): %v", err)
	}
	if len(liteEvents) != 1 || len(liteEvents[0].Matches) != 0 {
		t.Fatalf("expected 0 matches when deep=false, got %+v", liteEvents[0].Matches)
	}
}
