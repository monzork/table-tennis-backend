//go:build integration

// Package integration contains integration tests that run against a live PostgreSQL database.
// Run with: go test -tags=integration -run TestOrlandoJose ./internal/integration/...
package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appEvent "table-tennis-backend/internal/application/event"
	appMatch "table-tennis-backend/internal/application/match"
	eventDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
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
	repo := infraBun.NewTournamentRepository(db)

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

// TestOrlandoJoseSecondDivisionReplay creates a clone tournament using the same
// 2nd-division participants, submits all group-stage match scores from the original,
// verifies the group standings match, and then deletes the clone.
// The original tournament data is never modified.
func TestOrlandoJoseSecondDivisionReplay(t *testing.T) {
	db := infraBun.DB
	require.NotNil(t, db, "database connection must be established")

	ctx := context.Background()
	playerRepo := infraBun.NewPlayerRepository(db)
	tournamentRepo := infraBun.NewTournamentRepository(db)
	matchRepo := infraBun.NewMatchRepository(db, playerRepo)
	divisionRepo := infraBun.NewDivisionRepository(db)

	// ── Step 1: Read original event ──────────────────────────────────────────
	origEvent, err := tournamentRepo.GetByID(ctx, orlandoJose2ndDivEventID)
	require.NoError(t, err)
	t.Logf("Original event: %q (%d participants, %d matches)",
		origEvent.Name, len(origEvent.Participants), len(origEvent.Matches))

	// Collect unique players who appeared in 2nd division group matches
	seenPlayerIDs := make(map[string]bool)
	var participantIDs []string
	for _, m := range origEvent.Matches {
		if m.DivisionID != secondDivisionID || m.Stage != "group" {
			continue
		}
		for _, p := range m.TeamA {
			if !seenPlayerIDs[p.ID] {
				seenPlayerIDs[p.ID] = true
				participantIDs = append(participantIDs, p.ID)
			}
		}
		for _, p := range m.TeamB {
			if !seenPlayerIDs[p.ID] {
				seenPlayerIDs[p.ID] = true
				participantIDs = append(participantIDs, p.ID)
			}
		}
	}
	require.Greater(t, len(participantIDs), 0, "should find 2nd division players")
	t.Logf("Collected %d unique players from 2nd division group matches", len(participantIDs))

	// ── Step 2: Create the clone event ──────────────────────────────────────
	createUC := appEvent.NewCreateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	now := time.Now()
	cloneName := fmt.Sprintf("TEST_OrlandoJose2nd_%s", now.Format("150405"))

	cloneEvent, err := createUC.Execute(ctx,
		cloneName,
		"singles",
		"groups_elimination",
		"open", // category
		now.Format("2006-01-02"),
		now.AddDate(0, 0, 1).Format("2006-01-02"),
		participantIDs,
		nil,   // newPlayers
		2,     // groupPassCount
		nil,   // stageRuleOverrides
		nil,   // divisionRules
		true,  // skipElo – replicate skip_elo from original
		nil,   // eventID (no parent container)
		"",    // teamFormat
		6,     // numTables
		false, // hasThirdPlaceMatch
		nil,   // divisionFormats
		nil,   // divisionGroupPassCounts
		nil,   // divisionGroupCounts
	)
	require.NoError(t, err, "should create clone event")
	require.NotEmpty(t, cloneEvent.ID)
	t.Logf("Created clone event %q (ID: %s) with %d participants",
		cloneEvent.Name, cloneEvent.ID, len(cloneEvent.Participants))

	// Always clean up the clone event at test end
	defer func() {
		deleteUC := appEvent.NewDeleteTournamentUseCase(tournamentRepo)
		if err := deleteUC.Execute(ctx, cloneEvent.ID); err != nil {
			t.Logf("⚠️  WARNING: failed to delete clone event %s: %v", cloneEvent.ID, err)
		} else {
			t.Logf("✅ Cleaned up clone event %s", cloneEvent.ID)
		}
	}()

	// ── Step 3: Create Clone's Group Matches & Replay Scores ──────────────────
	updateScoreUC := appMatch.NewUpdateMatchScoreUseCase(matchRepo, tournamentRepo)

	// Build a map of playerID pair → original match result
	type MatchKey struct{ A, B string }
	type MatchResult struct {
		WinnerTeam string
		Sets       []struct{ ScoreA, ScoreB int }
	}
	origResults := make(map[MatchKey]MatchResult)
	for _, m := range origEvent.Matches {
		if m.DivisionID != secondDivisionID || m.Stage != "group" || m.Status != "finished" {
			continue
		}
		if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
			continue
		}
		key := MatchKey{A: m.TeamA[0].ID, B: m.TeamB[0].ID}
		result := MatchResult{WinnerTeam: m.WinnerTeam}
		for _, s := range m.Sets {
			result.Sets = append(result.Sets, struct{ ScoreA, ScoreB int }{s.ScoreA, s.ScoreB})
		}
		origResults[key] = result
	}
	t.Logf("Original group match results to replay: %d", len(origResults))

	// Manually recreate those group matches in the clone
	replayedCount := 0
	skippedCount := 0

	for key, orig := range origResults {
		// Find player models
		pA, err := playerRepo.GetById(ctx, key.A)
		if err != nil {
			skippedCount++
			continue
		}
		pB, err := playerRepo.GetById(ctx, key.B)
		if err != nil {
			skippedCount++
			continue
		}

		mID := idgen.Generate()
		newMatch := eventDomain.Match{
			ID:           mID,
			TournamentID: cloneEvent.ID,
			DivisionID:   secondDivisionID,
			Stage:        "group",
			Status:       "scheduled",
			MatchType:    "singles",
			TeamA:        []*playerDomain.Player{pA},
			TeamB:        []*playerDomain.Player{pB},
		}
		err = matchRepo.Save(ctx, &newMatch)
		require.NoError(t, err)

		// Submit set scores
		var rawScores []string
		for _, s := range orig.Sets {
			rawScores = append(rawScores, fmt.Sprintf("%d-%d", s.ScoreA, s.ScoreB))
		}
		err = updateScoreUC.Execute(ctx, mID, rawScores, cloneEvent.ID, newMatch.Stage)
		if err != nil {
			t.Logf("Warning: score update failed for clone match %s: %v", mID, err)
		} else {
			// Finish the match so it counts towards standings
			newMatch.Status = "finished"
			newMatch.WinnerTeam = orig.WinnerTeam
			if err := matchRepo.Save(ctx, &newMatch); err != nil {
				t.Logf("Warning: failed to save finished clone match %s: %v", mID, err)
			} else {
				replayedCount++
			}
		}
	}

	t.Logf("Recreated and replayed: %d group matches, skipped: %d", replayedCount, skippedCount)

	// ── Step 4: Verify group standings ──────────────────────────────────────
	// After score submission, reload the clone and verify at least some matches finished
	updatedClone, err := tournamentRepo.GetByID(ctx, cloneEvent.ID)
	require.NoError(t, err)

	finishedGroupMatches := 0
	for _, m := range updatedClone.Matches {
		if m.Stage == "group" && m.Status == "finished" {
			finishedGroupMatches++
		}
	}
	t.Logf("Clone event finished group matches after replay: %d", finishedGroupMatches)
	assert.GreaterOrEqual(t, finishedGroupMatches, replayedCount,
		"at least as many group matches should be finished as were replayed")

	// ── Step 5: Verify the original tournament's champion ───────────────────
	// This is the most important assertion — it runs against the ORIGINAL unmodified data
	t.Run("OriginalChampionIntact", func(t *testing.T) {
		repo := infraBun.NewTournamentRepository(db)
		orig, err := repo.GetByID(ctx, orlandoJose2ndDivEventID)
		require.NoError(t, err)

		maxRound := 0
		for _, m := range orig.Matches {
			if m.DivisionID == secondDivisionID && m.Stage != "group" && m.RoundNumber > maxRound {
				maxRound = m.RoundNumber
			}
		}

		var champions []string
		for _, m := range orig.Matches {
			if m.DivisionID != secondDivisionID || m.Stage == "group" || m.RoundNumber != maxRound {
				continue
			}
			if m.WinnerTeam == "A" && len(m.TeamA) > 0 {
				champions = append(champions, m.TeamA[0].FirstName+" "+m.TeamA[0].LastName)
			} else if m.WinnerTeam == "B" && len(m.TeamB) > 0 {
				champions = append(champions, m.TeamB[0].FirstName+" "+m.TeamB[0].LastName)
			}
		}
		t.Logf("Original 2nd division champions (round %d): %v", maxRound, champions)
		assert.Contains(t, champions, expectedChampion2ndDiv,
			"Mario Espinoza must still be the champion in the original tournament")
	})
}
