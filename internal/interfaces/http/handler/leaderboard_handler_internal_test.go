package handler

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"

	"github.com/gofiber/fiber/v2"
	"github.com/valyala/fasthttp"
)

type mockPlayerRepo struct {
	player.Repository
	players []*player.Player
}

func (m *mockPlayerRepo) GetAllSingles(ctx context.Context) ([]*player.Player, error) {
	return m.players, nil
}

func (m *mockPlayerRepo) GetAllDoubles(ctx context.Context) ([]*player.Player, error) {
	return m.players, nil
}

func (m *mockPlayerRepo) GetSinglesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var res []*player.Player
	for _, p := range m.players {
		if p.Gender == gender {
			res = append(res, p)
		}
	}
	return res, nil
}

func (m *mockPlayerRepo) GetDoublesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	var res []*player.Player
	for _, p := range m.players {
		if p.Gender == gender {
			res = append(res, p)
		}
	}
	return res, nil
}

type mockDivisionRepo struct {
	divisionDomain.Repository
	divisions []*divisionDomain.Division
}

func (m *mockDivisionRepo) GetAll(ctx context.Context) ([]*divisionDomain.Division, error) {
	return m.divisions, nil
}

type erroringPlayerRepo struct {
	player.Repository
}

func (m *erroringPlayerRepo) GetAllSingles(ctx context.Context) ([]*player.Player, error) {
	return nil, errPlayerRepo
}
func (m *erroringPlayerRepo) GetAllDoubles(ctx context.Context) ([]*player.Player, error) {
	return nil, errPlayerRepo
}
func (m *erroringPlayerRepo) GetSinglesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	return nil, errPlayerRepo
}
func (m *erroringPlayerRepo) GetDoublesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	return nil, errPlayerRepo
}

type erroringDivisionRepo struct {
	divisionDomain.Repository
}

func (m *erroringDivisionRepo) GetAll(ctx context.Context) ([]*divisionDomain.Division, error) {
	return nil, errDivisionRepo
}

var (
	errPlayerRepo   = fiber.NewError(500, "player repo boom")
	errDivisionRepo = fiber.NewError(500, "division repo boom")
)

func TestLeaderboardHandler_GetGroupedPlayers_Errors(t *testing.T) {
	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	okDivRepo := &mockDivisionRepo{divisions: nil}
	okPlayerRepo := &mockPlayerRepo{players: nil}

	t.Run("getGroupedPlayers player error", func(t *testing.T) {
		h := NewLeaderboardHandler(leaderboard.NewGetLeaderboardUseCase(&erroringPlayerRepo{}), division.NewDivisionUseCase(okDivRepo))
		if _, err := h.getGroupedPlayers(ctx, "singles"); err == nil {
			t.Fatal("expected error from player repo")
		}
	})

	t.Run("getGroupedPlayers division error", func(t *testing.T) {
		h := NewLeaderboardHandler(leaderboard.NewGetLeaderboardUseCase(okPlayerRepo), division.NewDivisionUseCase(&erroringDivisionRepo{}))
		if _, err := h.getGroupedPlayers(ctx, "singles"); err == nil {
			t.Fatal("expected error from division repo")
		}
	})

	t.Run("getGroupedPlayersByGender player error", func(t *testing.T) {
		h := NewLeaderboardHandler(leaderboard.NewGetLeaderboardUseCase(&erroringPlayerRepo{}), division.NewDivisionUseCase(okDivRepo))
		if _, err := h.getGroupedPlayersByGender(ctx, "singles", "M"); err == nil {
			t.Fatal("expected error from player repo")
		}
	})

	t.Run("getGroupedPlayersByGender division error", func(t *testing.T) {
		h := NewLeaderboardHandler(leaderboard.NewGetLeaderboardUseCase(okPlayerRepo), division.NewDivisionUseCase(&erroringDivisionRepo{}))
		if _, err := h.getGroupedPlayersByGender(ctx, "singles", "M"); err == nil {
			t.Fatal("expected error from division repo")
		}
	})
}

func TestLeaderboardHandler_RenderRanking_Errors(t *testing.T) {
	t.Run("renderRanking player error", func(t *testing.T) {
		h := NewLeaderboardHandler(leaderboard.NewGetLeaderboardUseCase(&erroringPlayerRepo{}), division.NewDivisionUseCase(&mockDivisionRepo{}))
		app := fiber.New()
		app.Get("/rankings/singles", h.GetSingles)
		req := httptest.NewRequest("GET", "/rankings/singles", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode == fiber.StatusOK {
			t.Errorf("expected non-200 for player repo error, got %d", resp.StatusCode)
		}
	})

	t.Run("renderRanking division error", func(t *testing.T) {
		h := NewLeaderboardHandler(leaderboard.NewGetLeaderboardUseCase(&mockPlayerRepo{}), division.NewDivisionUseCase(&erroringDivisionRepo{}))
		app := fiber.New()
		app.Get("/rankings/singles", h.GetSingles)
		req := httptest.NewRequest("GET", "/rankings/singles", nil)
		resp, _ := app.Test(req)
		if resp.StatusCode == fiber.StatusOK {
			t.Errorf("expected non-200 for division repo error, got %d", resp.StatusCode)
		}
	})
}

func TestLeaderboardHandler_GetGroupedPlayers(t *testing.T) {
	p1, _ := player.NewPlayer("1", "Alice", "Smith", time.Now(), "F", "USA", "Dept 1", "ID001")
	p1.SinglesElo = 1500
	p1.DoublesElo = 1400

	p2, _ := player.NewPlayer("2", "Bob", "Jones", time.Now(), "M", "MEX", "Dept 2", "ID002")
	p2.SinglesElo = 800
	p2.DoublesElo = 900

	p3, _ := player.NewPlayer("3", "Carol", "White", time.Now(), "F", "USA", "Dept 1", "ID003")
	p3.SinglesElo = 1200
	p3.DoublesElo = 1600

	maxElo1 := int16(2000)
	div1, _ := divisionDomain.NewDivision("div1", "Division A", 1, 1000, &maxElo1, "singles", "#ffffff")
	maxElo2 := int16(999)
	div2, _ := divisionDomain.NewDivision("div2", "Division B", 2, 0, &maxElo2, "singles", "#000000")

	noDiv, _ := divisionDomain.NewDivision("none", "No Division", 3, 0, nil, "singles", "#000000")

	playerRepo := &mockPlayerRepo{players: []*player.Player{p1, p2, p3}}
	divisionRepo := &mockDivisionRepo{divisions: []*divisionDomain.Division{div1, div2, noDiv}}

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	h := NewLeaderboardHandler(leaderboardUC, divisionUC)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	t.Run("Singles", func(t *testing.T) {
		groups, err := h.getGroupedPlayers(ctx, "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(groups))
		}
		if groups[0].Division.ID != "div1" {
			t.Errorf("expected div1, got %s", groups[0].Division.ID)
		}
		if len(groups[0].Players) != 2 {
			t.Errorf("expected 2 players in div1, got %d", len(groups[0].Players))
		}
		if groups[1].Division.ID != "div2" {
			t.Errorf("expected div2, got %s", groups[1].Division.ID)
		}
		if len(groups[1].Players) != 1 || groups[1].Players[0].ID != "2" {
			t.Errorf("expected Bob in div2")
		}
	})

	t.Run("Doubles", func(t *testing.T) {
		groups, err := h.getGroupedPlayers(ctx, "doubles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 2 {
			t.Fatalf("expected 2 groups, got %d", len(groups))
		}
		if groups[0].Division.ID != "div1" || len(groups[0].Players) != 2 {
			t.Errorf("expected 2 players in div1 doubles")
		}
		if groups[1].Division.ID != "div2" || len(groups[1].Players) != 1 {
			t.Errorf("expected 1 player in div2 doubles")
		}
	})
}

func TestLeaderboardHandler_GetGroupedPlayersByGender(t *testing.T) {
	p1, _ := player.NewPlayer("1", "Alice", "Smith", time.Now(), "F", "USA", "Dept 1", "ID001")
	p1.SinglesElo = 1500
	p1.DoublesElo = 1400

	p2, _ := player.NewPlayer("2", "Bob", "Jones", time.Now(), "M", "MEX", "Dept 2", "ID002")
	p2.SinglesElo = 800
	p2.DoublesElo = 900

	p3, _ := player.NewPlayer("3", "Carol", "White", time.Now(), "F", "USA", "Dept 1", "ID003")
	p3.SinglesElo = 1200
	p3.DoublesElo = 1600

	maxElo1 := int16(2000)
	div1, _ := divisionDomain.NewDivision("div1", "Division A", 1, 1000, &maxElo1, "singles", "#ffffff")
	maxElo2 := int16(999)
	div2, _ := divisionDomain.NewDivision("div2", "Division B", 2, 0, &maxElo2, "singles", "#000000")

	playerRepo := &mockPlayerRepo{players: []*player.Player{p1, p2, p3}}
	divisionRepo := &mockDivisionRepo{divisions: []*divisionDomain.Division{div1, div2}}

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	h := NewLeaderboardHandler(leaderboardUC, divisionUC)

	app := fiber.New()
	ctx := app.AcquireCtx(&fasthttp.RequestCtx{})
	defer app.ReleaseCtx(ctx)

	t.Run("Women Singles", func(t *testing.T) {
		groups, err := h.getGroupedPlayersByGender(ctx, "singles", "F")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(groups))
		}
		if groups[0].Division.ID != "div1" || len(groups[0].Players) != 2 {
			t.Errorf("expected 2 women in div1 singles")
		}
	})

	t.Run("Men Doubles", func(t *testing.T) {
		groups, err := h.getGroupedPlayersByGender(ctx, "doubles", "M")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(groups) != 1 {
			t.Fatalf("expected 1 group, got %d", len(groups))
		}
		if groups[0].Division.ID != "div2" || len(groups[0].Players) != 1 {
			t.Errorf("expected 1 man in div2 doubles")
		}
	})
}
