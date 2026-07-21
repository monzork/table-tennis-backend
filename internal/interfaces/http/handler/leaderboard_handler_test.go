package handler_test

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func createTestLeaderboardApp(t *testing.T) *fiber.App {
	db, err := SetupTestDB()
	if err != nil {
		t.Fatalf("failed to setup test db: %v", err)
	}

	ctx := context.Background()
	playerRepo := bunRepo.NewPlayerRepository(db)
	divisionRepo := bunRepo.NewDivisionRepository(db)

	// Seed some divisions
	maxElo1 := int16(2000)
	div1, _ := divisionDomain.NewDivision("div1", "Division A", 1, 1000, &maxElo1, "singles", "#ffffff")
	maxElo2 := int16(999)
	div2, _ := divisionDomain.NewDivision("div2", "Division B", 2, 0, &maxElo2, "singles", "#000000")
	_ = divisionRepo.Save(ctx, div1)
	_ = divisionRepo.Save(ctx, div2)

	// Seed some players
	now := time.Now()
	p1, _ := player.NewPlayer(idgen.Generate(), "Alice", "Smith", now, "M", "USA", "Dept 1", "ID001")
	p1.SinglesElo = 1500
	p1.DoublesElo = 1400
	_ = playerRepo.Save(ctx, p1)

	p2, _ := player.NewPlayer(idgen.Generate(), "Bob", "Jones", now, "M", "MEX", "Dept 2", "ID002")
	p2.SinglesElo = 800
	p2.DoublesElo = 900
	_ = playerRepo.Save(ctx, p2)

	p3, _ := player.NewPlayer(idgen.Generate(), "Carol", "White", now, "F", "USA", "Dept 1", "ID003")
	p3.SinglesElo = 1200
	p3.DoublesElo = 1600
	_ = playerRepo.Save(ctx, p3)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)
	divisionUC := division.NewDivisionUseCase(divisionRepo)
	lh := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)

	engine := html.New("../templates", ".html")
	type CountryInfo struct {
		Code string
		Name string
	}
	engine.AddFunc("countries", func() []CountryInfo { return nil })
	engine.AddFunc("add", func(a, b int) int { return a + b })
	engine.AddFunc("dict", func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid dict call")
		}
		dict := make(map[string]interface{})
		for i := 0; i < len(values); i += 2 {
			dict[values[i].(string)] = values[i+1]
		}
		return dict, nil
	})
	engine.AddFunc("isNicaragua", func(country string) bool { return false })
	engine.AddFunc("nicaraguaDepartments", func() []string { return nil })
	engine.AddFunc("t", func(tmap map[string]string, key string) string { return key })
	engine.AddFunc("cleanPhone", func(phone string) string { return phone })
	engine.AddFunc("safeHTML", func(s string) template.HTML { return template.HTML(s) })

	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
	})

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("Lang", "en")
		c.Locals("T", make(map[string]string))
		return c.Next()
	})

	app.Get("/rankings/singles", lh.GetSingles)
	app.Get("/rankings/doubles", lh.GetDoubles)
	app.Get("/rankings/mens-singles", lh.GetMensSingles)
	app.Get("/rankings/womens-singles", lh.GetWomensSingles)
	app.Get("/rankings/mens-doubles", lh.GetMensDoubles)
	app.Get("/rankings/womens-doubles", lh.GetWomensDoubles)
	app.Get("/rankings/mixed-doubles", lh.GetMixedDoubles)

	return app
}

func TestLeaderboardHandler_GetSingles(t *testing.T) {
	app := createTestLeaderboardApp(t)

	t.Run("GetSingles standard request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rankings/singles", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GetSingles HTMX partial request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rankings/singles?q=Alice&sort=points_asc", nil)
		req.Header.Set("HX-Request", "true")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("GetSingles with division filter", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/rankings/singles?division=Division%20A", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})
}

func TestLeaderboardHandler_Categories(t *testing.T) {
	app := createTestLeaderboardApp(t)

	routes := []string{
		"/rankings/doubles",
		"/rankings/mens-singles",
		"/rankings/womens-singles",
		"/rankings/mens-doubles",
		"/rankings/womens-doubles",
		"/rankings/mixed-doubles",
	}

	for _, route := range routes {
		t.Run("GET "+route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", route, err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", route, resp.StatusCode)
			}
		})
	}
}

func TestLeaderboardHandler_SortingAndFiltering(t *testing.T) {
	app := createTestLeaderboardApp(t)

	testCases := []struct {
		name string
		url  string
	}{
		{"Sort by name asc", "/rankings/singles?sort=name_asc"},
		{"Sort by points asc", "/rankings/singles?sort=points_asc"},
		{"Filter by search query name", "/rankings/singles?q=Alice"},
		{"Filter by search query country", "/rankings/singles?q=USA"},
		{"Filter by search query department", "/rankings/singles?q=Dept%201"},
		{"Doubles with search query", "/rankings/doubles?q=Bob&sort=points_desc"},
		{"Doubles with division filter", "/rankings/doubles?division=Division%20A"},
		{"Singles with unknown division filter", "/rankings/singles?division=NoSuchDivision"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.url, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.name, err)
			}
			if resp.StatusCode != http.StatusOK {
				t.Errorf("%s: expected status 200, got %d", tc.name, resp.StatusCode)
			}
		})
	}
}
