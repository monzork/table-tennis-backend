//go:build integration

// Package integration contains integration tests that run against a live PostgreSQL database.
// Run with: go test -tags=integration -run TestOrlandoJose ./internal/integration/...
package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appEvent "table-tennis-backend/internal/application/event"
	appMatch "table-tennis-backend/internal/application/match"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	tournamentParent "table-tennis-backend/internal/domain/tournament"
	"table-tennis-backend/internal/infrastructure/identity"
	infraBun "table-tennis-backend/internal/infrastructure/persistence/bun"
)

func init() {
	godotenv.Load("../../.env")
	infraBun.Connect()
	idgen.Register(identity.NewUUIDGenerator())
}

const (
	// Orlando José Perez In Memoriam — Men's Singles Second Division event ID
	orlandoJose2ndDivEventID = "113c97cf-d74a-43f7-945f-4141027d54f3"
	secondDivisionID         = "div-second"
	// Confirmed champion of the 2nd division based on original tournament results
	expectedChampion2ndDiv = "Mario Espinoza"
)

// TestOrlandoJoseKnockoutResults is a non-destructive read-only test that verifies the
// final knockout standings of the original Orlando José tournament match expected results.
func TestOrlandoJoseKnockoutResults(t *testing.T) {
	db := infraBun.DB
	require.NotNil(t, db, "database connection must be established")

	ctx := context.Background()
	repo := infraBun.NewEventRepository(db)

	origEvent, err := repo.GetByID(ctx, orlandoJose2ndDivEventID)
	require.NoError(t, err, "should fetch original event from DB")

	// Build knockout bracket results — ignore group stage
	type KOResult struct {
		Stage   string
		Round   int
		PlayerA string
		PlayerB string
		Winner  string
	}

	var results []KOResult
	maxRound := 0
	for _, m := range origEvent.Matches {
		if m.DivisionID != secondDivisionID || m.Stage == "group" {
			continue
		}
		r := KOResult{Stage: m.Stage, Round: m.RoundNumber}
		if len(m.TeamA) > 0 {
			r.PlayerA = m.TeamA[0].FirstName + " " + m.TeamA[0].LastName
		}
		if len(m.TeamB) > 0 {
			r.PlayerB = m.TeamB[0].FirstName + " " + m.TeamB[0].LastName
		}
		r.Winner = m.WinnerTeam
		results = append(results, r)
		if m.RoundNumber > maxRound {
			maxRound = m.RoundNumber
		}
	}

	t.Logf("Orlando José 2nd Division — Knockout Results (%d matches, max round %d):", len(results), maxRound)
	for _, r := range results {
		t.Logf("  [%s R%d] %s vs %s → Winner: Team%s", r.Stage, r.Round, r.PlayerA, r.PlayerB, r.Winner)
	}

	require.Greater(t, len(results), 0, "should have knockout results")

	// Find the champion (winner of the highest-round match)
	var champions []string
	for _, r := range results {
		if r.Round != maxRound {
			continue
		}
		if r.Winner == "A" {
			champions = append(champions, r.PlayerA)
		} else if r.Winner == "B" {
			champions = append(champions, r.PlayerB)
		}
	}
	t.Logf("🏆 Champions (final round %d): %v", maxRound, champions)
	assert.Contains(t, champions, expectedChampion2ndDiv,
		"Mario Espinoza should be the 2nd division champion")
}

// TestOrlandoJoseSecondDivisionReplay creates a clone tournament (a real parent
// Tournament row plus its one child Event, same as the admin UI would produce)
// using the same 2nd-division participants, submits all group-stage match
// scores from the original, verifies the group standings match, and then
// deletes the clone. The original tournament data is never modified.
//
// Two env vars support a manual inspect-before-delete workflow:
//   - ORLANDO_REPLAY_KEEP=1 skips the final delete so the clone stays in the DB
//     for review; the clone's parent tournament ID is logged.
//   - ORLANDO_REPLAY_DELETE_ID=<id> switches the test into cleanup-only mode:
//     it deletes exactly that parent tournament ID (cascading to its child
//     event, matches, and groups) and returns without doing any replay.
func TestOrlandoJoseSecondDivisionReplay(t *testing.T) {
	ctx := context.Background()
	db := infraBun.DB
	require.NotNil(t, db, "database connection must be established")

	eventRepo := infraBun.NewEventRepository(db)
	tournamentRepo := infraBun.NewTournamentRepository(db, eventRepo)

	if deleteID := os.Getenv("ORLANDO_REPLAY_DELETE_ID"); deleteID != "" {
		require.NoError(t, tournamentRepo.Delete(ctx, deleteID))
		t.Logf("Deleted clone tournament %s (child event/matches/groups cascade)", deleteID)
		return
	}

	playerRepo := infraBun.NewPlayerRepository(db)
	matchRepo := infraBun.NewMatchRepository(db, playerRepo)
	divisionRepo := infraBun.NewDivisionRepository(db)

	orig, err := eventRepo.GetByID(ctx, orlandoJose2ndDivEventID)
	require.NoError(t, err, "should fetch original event from DB")

	// The original event mixes all three divisions' groups/matches together
	// (SkipDivisionSplit=false); isolate the 2nd division's groups by finding
	// which groups contain a player who appears in a div-second group match.
	origGroupMatchesByID := make(map[string][]tournamentDomain.Match)
	var origGroupMatches []tournamentDomain.Match
	for _, m := range orig.Matches {
		if m.DivisionID == secondDivisionID && m.Stage == "group" && m.Status == "finished" {
			origGroupMatches = append(origGroupMatches, m)
			if len(m.TeamA) > 0 {
				origGroupMatchesByID[m.TeamA[0].ID] = append(origGroupMatchesByID[m.TeamA[0].ID], m)
			}
			if len(m.TeamB) > 0 {
				origGroupMatchesByID[m.TeamB[0].ID] = append(origGroupMatchesByID[m.TeamB[0].ID], m)
			}
		}
	}
	require.NotEmpty(t, origGroupMatches, "should find finished 2nd division group matches")

	var divGroups []tournamentDomain.Group
	for _, g := range orig.Groups {
		for _, p := range g.Players {
			if len(origGroupMatchesByID[p.ID]) > 0 {
				divGroups = append(divGroups, g)
				break
			}
		}
	}
	require.NotEmpty(t, divGroups, "should find 2nd division groups in the original event")

	var participantIDs []string
	for _, g := range divGroups {
		for _, p := range g.Players {
			participantIDs = append(participantIDs, p.ID)
		}
	}
	require.NotEmpty(t, participantIDs, "should find 2nd division participants")

	// 1. Create a real parent Tournament for the clone — an Event with no
	// TournamentID doesn't show up the way a real admin-created tournament
	// does (it won't appear in the /admin/tournaments listing at all).
	today := time.Now().Format("2006-01-02")
	todayDate, err := time.Parse("2006-01-02", today)
	require.NoError(t, err)
	parentID := idgen.Generate()
	parentTournament, err := tournamentParent.NewTournament(
		parentID, "TEST CLONE — Orlando José 2nd Division Replay", nil, true, todayDate, todayDate,
	)
	require.NoError(t, err)
	require.NoError(t, tournamentRepo.Save(ctx, parentTournament))

	// 2. Create the clone child event under that parent. Category "open"
	// (rather than "men") avoids CreateTournamentUseCase's side effect of
	// auto-creating an empty paired event for the opposite gender, keeping
	// cleanup to a single child event.
	createUC := appEvent.NewCreateTournamentUseCase(eventRepo, playerRepo)
	clone, err := createUC.Execute(ctx, appEvent.CreateEventCommand{
		Name:                 "Men's Singles — 2nd Division",
		Type:                 orig.Type,
		Format:               orig.Format,
		Category:             "open",
		StartDate:            today,
		EndDate:              today,
		ParticipantIDs:       participantIDs,
		GroupPassCount:       orig.GroupPassCount,
		LosersGroupPassCount: orig.LosersGroupPassCount,
		SkipElo:              true,
		HasThirdPlaceMatch:   orig.HasThirdPlaceMatch,
		TournamentID:         &parentID,
		// The original group matches were actually played best-of-3 (every
		// recorded match has 2 or 3 sets, and winners never reach 3 set-wins)
		// even though this event's own StageRules table declares group=BestOf
		// 5 — an inconsistency in the historical/imported data, not something
		// to reproduce. Overriding to what was truly played is what lets
		// UpdateScore mark these matches "finished" during replay.
		StageRuleOverrides: []appEvent.StageRuleOverride{
			{Stage: "group", BestOf: 3, PointsToWin: 11, PointsMargin: 2},
		},
	})
	require.NoError(t, err)
	t.Logf("Created clone tournament %s / event %s (SkipElo=%v) — inspect via the admin UI (/admin/tournaments/%s) before it is deleted",
		parentID, clone.ID, clone.SkipElo, parentID)

	// 2. Overwrite the auto-seeded groups (based on today's live Elo, which
	// would not reproduce the original grouping) with the original historical
	// grouping, so the replay is apples-to-apples.
	clonePlayerByID := make(map[string]*player.Player)
	for _, p := range clone.Participants {
		clonePlayerByID[p.ID] = p
	}
	clone.Groups = nil
	for _, g := range divGroups {
		newGroup := tournamentDomain.Group{ID: idgen.Generate(), EventID: clone.ID, Name: g.Name}
		for _, op := range g.Players {
			if np, ok := clonePlayerByID[op.ID]; ok {
				newGroup.Players = append(newGroup.Players, np)
			}
		}
		clone.Groups = append(clone.Groups, newGroup)
	}
	require.NoError(t, eventRepo.UpdateGroups(ctx, clone))

	// 3. Replay every original group-stage score through the same
	// application-layer use cases the QR/public score-entry endpoint
	// (/score/:matchId -> POST /public/matches/score/update) calls under the
	// hood — CreateMatchUseCase to create the pairing and
	// UpdateMatchScoreUseCase to submit the score — rather than reaching
	// into the repository directly, so the replay exercises the exact same
	// code path a player scanning a QR code at the table would hit.
	createMatchUC := appMatch.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	updateScoreUC := appMatch.NewUpdateMatchScoreUseCase(matchRepo, eventRepo)
	for _, m := range origGroupMatches {
		require.Len(t, m.TeamA, 1)
		require.Len(t, m.TeamB, 1)
		newMatch, err := createMatchUC.Execute(ctx, clone.ID, "singles", []string{m.TeamA[0].ID}, []string{m.TeamB[0].ID}, "group")
		require.NoError(t, err)
		rawScores := make([]string, len(m.Sets))
		for i, s := range m.Sets {
			rawScores[i] = fmt.Sprintf("%d-%d", s.ScoreA, s.ScoreB)
		}
		require.NoError(t, updateScoreUC.Execute(ctx, newMatch.ID, rawScores, clone.ID, "group"))
	}

	// 4. Re-fetch the clone and verify per-group standings match the
	// original, player-for-player and rank-for-rank.
	reloaded, err := eventRepo.GetByID(ctx, clone.ID)
	require.NoError(t, err)

	cloneGroupByName := make(map[string]tournamentDomain.Group)
	for _, g := range reloaded.Groups {
		cloneGroupByName[g.Name] = g
	}

	for _, origGroup := range divGroups {
		cloneGroup, ok := cloneGroupByName[origGroup.Name]
		require.True(t, ok, "clone should have a %q group", origGroup.Name)

		origStandings := tournamentDomain.BuildStandings(origGroup.Players, orig.Matches)
		cloneStandings := tournamentDomain.BuildStandings(cloneGroup.Players, reloaded.Matches)

		require.Equal(t, len(origStandings), len(cloneStandings), "group %q: standings length mismatch", origGroup.Name)
		for i := range origStandings {
			assert.Equal(t, origStandings[i].Player.ID, cloneStandings[i].Player.ID,
				"group %q rank %d: expected %s %s, got %s %s",
				origGroup.Name, i+1,
				origStandings[i].Player.FirstName, origStandings[i].Player.LastName,
				cloneStandings[i].Player.FirstName, cloneStandings[i].Player.LastName)
			assert.Equal(t, origStandings[i].Wins, cloneStandings[i].Wins)
			assert.Equal(t, origStandings[i].SetsWon, cloneStandings[i].SetsWon)
			assert.Equal(t, origStandings[i].SetsLost, cloneStandings[i].SetsLost)
		}
		t.Logf("Group %q standings verified (%d players)", origGroup.Name, len(origStandings))
	}

	// 5. Clean up, unless the caller asked to keep the clone around for
	// manual inspection first (ORLANDO_REPLAY_KEEP=1). To delete it later,
	// re-run with ORLANDO_REPLAY_DELETE_ID=<parentID> set to the ID logged above.
	if os.Getenv("ORLANDO_REPLAY_KEEP") != "" {
		t.Logf("ORLANDO_REPLAY_KEEP set — leaving clone tournament %s in place for inspection", parentID)
		return
	}
	require.NoError(t, tournamentRepo.Delete(ctx, parentID))
	t.Logf("Deleted clone tournament %s", parentID)
}
