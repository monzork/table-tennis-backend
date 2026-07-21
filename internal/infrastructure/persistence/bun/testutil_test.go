package bun_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
)

var testDBCounter int64

// setupTestDB spins up an in-memory sqlite-backed bun.DB with all repository
// tables created, mirroring the pattern used in
// internal/interfaces/http/handler/testutil_test.go. Each call gets its own
// isolated in-memory database (unique DSN) so tests don't clobber each
// other's data via sqlite's shared cache mode.
func setupTestDB(t *testing.T) *bun.DB {
	t.Helper()

	dsn := fmt.Sprintf("file:testdb%d?mode=memory&cache=shared", atomic.AddInt64(&testDBCounter, 1))
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
