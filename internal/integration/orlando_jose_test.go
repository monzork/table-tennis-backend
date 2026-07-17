//go:build integration

// Package integration contains integration tests that run against a live PostgreSQL database.
// Run with: go test -tags=integration -run TestOrlandoJose ./internal/integration/...
package integration

import (
	"context"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"table-tennis-backend/internal/domain/idgen"
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

// TestOrlandoJoseSecondDivisionReplay creates a clone tournament using the same
// 2nd-division participants, submits all group-stage match scores from the original,
// verifies the group standings match, and then deletes the clone.
// The original tournament data is never modified.
