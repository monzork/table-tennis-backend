package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"table-tennis-backend/internal/interfaces/http/i18n"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	adminDomain "table-tennis-backend/internal/domain/admin"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/handler"
	"table-tennis-backend/internal/interfaces/http/middleware"

	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/infrastructure/identity"
	"table-tennis-backend/internal/infrastructure/security"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	fiberws "github.com/gofiber/websocket/v2"
	"github.com/joho/godotenv"
)

func main() {
	idgen.Register(identity.NewUUIDGenerator())
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or failed to load")
	}

	bun.Connect()

	playerRepo := bun.NewPlayerRepository(bun.DB)
	playerUC := player.NewRegisterPlayerUseCase(playerRepo)
	updatePlayerUC := player.NewUpdatePlayerUseCase(playerRepo)
	deletePlayerUC := player.NewDeletePlayerUseCase(playerRepo)
	importPlayerUC := player.NewImportPlayersUseCase(playerRepo)
	getPlayerByIDUC := player.NewGetPlayerByIDUseCase(playerRepo)
	searchPlayerUC := player.NewSearchPlayersUseCase(playerRepo)
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, importPlayerUC)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)

	divisionRepo := bun.NewDivisionRepository(bun.DB)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	tournamentRepo := bun.NewTournamentRepository(bun.DB)
	createTournamentUC := tournament.NewCreateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	getTournamentByIDUC := tournament.NewGetTournamentByIDUseCase(tournamentRepo, divisionRepo)
	getTournamentsUC := tournament.NewGetTournamentsUseCase(tournamentRepo)
	updateTournamentUC := tournament.NewUpdateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	deleteTournamentUC := tournament.NewDeleteTournamentUseCase(tournamentRepo)
	matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)
	finishTournamentUC := tournament.NewFinishTournamentUseCase(tournamentRepo, matchRepo, playerRepo)
	exportTournamentUC := tournament.NewExportTournamentReportUseCase(tournamentRepo)
	exportTournamentPdfUC := tournament.NewExportTournamentPdfUseCase(tournamentRepo)
	exportAllTournamentsPdfUC := tournament.NewExportAllTournamentsPdfUseCase(tournamentRepo)
	movePlayerUC := tournament.NewMovePlayerUseCase(tournamentRepo)
	createTeamUC := tournament.NewCreateTeamUseCase(tournamentRepo)
	deleteTeamUC := tournament.NewDeleteTeamUseCase(tournamentRepo)
	assignPlayerToTeamUC := tournament.NewAssignPlayerToTeamUseCase(tournamentRepo)
	removePlayerFromTeamUC := tournament.NewRemovePlayerFromTeamUseCase(tournamentRepo)

	tournamentHandler := handler.NewTournamentHandler(
		createTournamentUC,
		getTournamentByIDUC,
		updateTournamentUC,
		deleteTournamentUC,
		leaderboardUC,
		divisionUC,
		finishTournamentUC,
		exportTournamentUC,
		exportTournamentPdfUC,
		exportAllTournamentsPdfUC,
		movePlayerUC,
		createTeamUC,
		deleteTeamUC,
		assignPlayerToTeamUC,
		removePlayerFromTeamUC,
		getTournamentsUC,
	)
	eventRepo := bun.NewEventRepository(bun.DB, tournamentRepo)
	createEventUC := event.NewCreateEventUseCase(eventRepo, tournamentRepo, playerRepo, divisionRepo)
	getEventByIDUC := event.NewGetEventByIDUseCase(eventRepo)
	getAllEventsUC := event.NewGetAllEventsUseCase(eventRepo)
	deleteEventUC := event.NewDeleteEventUseCase(eventRepo)
	eventHandler := handler.NewEventHandler(createEventUC, getEventByIDUC, getAllEventsUC, deleteEventUC, divisionUC, leaderboardUC)

	GetMatchesUC := match.NewGetMatchesUseCase(matchRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, tournamentRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, tournamentRepo)
	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, tournamentRepo, finishTournamentUC)

	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)
	divisionHandler := handler.NewDivisionHandler(divisionUC)
	selfRegisterUC := tournament.NewSelfRegisterUseCase(tournamentRepo, playerRepo)
	publicHandler := handler.NewPublicHandler(playerUC, selfRegisterUC)

	adminRepo := bun.NewAdminRepository(bun.DB)

	hasher := security.NewBcryptHasher()

	// Seed default admin if DB empty
	count, _ := adminRepo.Count(context.Background())
	if count == 0 {
		user := os.Getenv("ADMIN_USERNAME")
		pass := os.Getenv("ADMIN_PASSWORD")
		if user == "" {
			user = "admin"
		}
		if pass == "" {
			pass = "password"
		}
		hashed, err := hasher.Hash(pass)
		if err == nil {
			if a, err := adminDomain.NewAdmin(idgen.Generate(), user, hashed); err == nil {
				adminRepo.Save(context.Background(), a)
			}
		}
	}

	store := session.New()
	authHandler := handler.NewAuthHandler(store, adminRepo, hasher)
	authMiddleware := middleware.Protected(store)

	engine := html.New("./internal/interfaces/http/templates", ".html")
	type CountryInfo struct {
		Code string
		Name string
	}
	countriesList := []CountryInfo{
		{"NIC", "Nicaragua"},
		{"ARG", "Argentina"},
		{"BRA", "Brazil"},
		{"CAN", "Canada"},
		{"CHL", "Chile"},
		{"CHN", "China"},
		{"COL", "Colombia"},
		{"CRC", "Costa Rica"},
		{"CUB", "Cuba"},
		{"DOM", "Dominican Republic"},
		{"ECU", "Ecuador"},
		{"SLV", "El Salvador"},
		{"ESP", "Spain"},
		{"FRA", "France"},
		{"GER", "Germany"},
		{"GTM", "Guatemala"},
		{"HON", "Honduras"},
		{"JPN", "Japan"},
		{"KOR", "South Korea"},
		{"MEX", "Mexico"},
		{"PAN", "Panama"},
		{"PER", "Peru"},
		{"PRI", "Puerto Rico"},
		{"SWE", "Sweden"},
		{"TPE", "Chinese Taipei"},
		{"USA", "United States"},
		{"VEN", "Venezuela"},
	}
	engine.AddFunc("countries", func() []CountryInfo {
		return countriesList
	})
	engine.AddFunc("add", func(a, b int) int {
		return a + b
	})
	engine.AddFunc("dict", func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("invalid dict call, must have even number of arguments")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	})
	engine.AddFunc("isNicaragua", func(country string) bool {
		c := strings.TrimSpace(strings.ToUpper(country))
		return c == "NIC" || c == "NICARAGUA" || c == "NI"
	})
	engine.AddFunc("nicaraguaDepartments", func() []string {
		return handler.NicaraguaDepartments
	})
	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
	})

	app.Static("/static", "./static")
	
	// Global Translation Middleware
	app.Use(func(c *fiber.Ctx) error {
		lang := c.Cookies("lang")
		if lang != "es" && lang != "en" {
			lang = "en"
		}
		m := make(map[string]string)
		// i18n package should be imported if not already. We'll check imports next.
		for k := range i18n.Translations["en"] {
			m[k] = i18n.T(lang, k)
		}
		c.Locals("T", m)
		c.Locals("Lang", lang)
		return c.Next()
	})

	adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC, getTournamentsUC, divisionUC, getAllEventsUC)

	// ==========================================
	// PUBLIC ROUTES
	// ==========================================

	// Rankings
	app.Get("/rankings/singles", leaderboardHandler.GetSingles)
	app.Get("/rankings/doubles", leaderboardHandler.GetDoubles)
	app.Get("/rankings/mens/singles", leaderboardHandler.GetMensSingles)
	app.Get("/rankings/womens/singles", leaderboardHandler.GetWomensSingles)
	app.Get("/rankings/mens/doubles", leaderboardHandler.GetMensDoubles)
	app.Get("/rankings/womens/doubles", leaderboardHandler.GetWomensDoubles)
	app.Get("/rankings/mixed/doubles", leaderboardHandler.GetMixedDoubles)

	// Redirect Root to Public Rankings
	app.Get("/", func(c *fiber.Ctx) error {
		return c.Redirect("/rankings/singles")
	})

	// Public Tournaments List
	app.Get("/tournaments", tournamentHandler.PublicList)

	// Public Tournament Self-Registration (must be before /tournaments/:id)
	signupLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	})
	app.Get("/tournaments/register", publicHandler.ShowTournamentRegistration)
	app.Post("/tournaments/register", signupLimiter, publicHandler.RegisterToTournament)

	// Public Detail Views
	app.Get("/tournaments/:id", tournamentHandler.PublicDetail)
	app.Get("/events/:id", eventHandler.PublicDetail)

	// User Registration
	app.Get("/register", publicHandler.ShowSignup)
	app.Post("/register", signupLimiter, publicHandler.Register)
	app.Get("/players/department-input", publicHandler.DepartmentInput)

	// Language Switcher
	app.Get("/lang/:locale", publicHandler.SetLang)

	// Public Score Entry & Match Starting Endpoints
	app.Get("/public/matches/score/form", matchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/form", matchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/update", matchHandler.UpdatePublicScore)
	app.Post("/public/matches/start", matchHandler.Start)
	app.Post("/public/matches/:id/start", matchHandler.Start)

	// QR-code score entry — shareable per-match URL
	app.Get("/score/:matchId", matchHandler.ShowMatchScorePage)
	app.Post("/score/:matchId/verify", matchHandler.ValidateMatchPIN)

	// Admin Score Form (read-only partial — no auth needed, no sensitive data)
	app.Get("/admin/matches/score/form", matchHandler.ShowScoreForm)
	app.Post("/admin/matches/score/form", matchHandler.ShowScoreForm)

	app.Get("/players/import/template", playerHandler.ImportTemplate) // public template download

	// ==========================================
	// WEBSOCKET ROUTES
	// ==========================================

	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/brackets/:tournamentId", fiberws.New(handler.WsBracketHandler))

	// ==========================================
	// AUTHENTICATION ROUTES
	// ==========================================

	app.Get("/admin/login", authHandler.ShowLogin)
	app.Post("/admin/login", authHandler.Login)
	app.Post("/admin/logout", authHandler.Logout)

	// ==========================================
	// PROTECTED ADMIN DASHBOARD / VIEW GROUPS
	// ==========================================

	admin := app.Group("/admin")
	admin.Use(authMiddleware)

	admin.Get("/", adminHandler.Dashboard)
	admin.Get("/players", adminHandler.Players)
	admin.Get("/tournaments", adminHandler.Tournaments)
	admin.Get("/events", adminHandler.Events)
	admin.Get("/divisions", adminHandler.Divisions)
	admin.Get("/player-field", adminHandler.NewPlayerField)

	// ==========================================
	// PROTECTED API / FORM ENDPOINTS
	// ==========================================

	api := app.Group("/")
	api.Use(authMiddleware)

	// Players API
	api.Post("/players", playerHandler.Register)
	api.Get("/players/search", playerHandler.Search)
	api.Get("/players/search/cards", playerHandler.SearchSelectionCards)
	api.Get("/players/:id/edit", playerHandler.ShowEditForm)
	api.Put("/players/:id", playerHandler.Update)
	api.Delete("/players/:id", playerHandler.Delete)
	api.Post("/players/import", playerHandler.Import)


	// Tournaments API
	admin.Get("/tournaments/:id", tournamentHandler.Detail)
	admin.Get("/tournaments/:id/board", tournamentHandler.Board)
	admin.Get("/tournaments/:id/board/columns", tournamentHandler.BoardColumns)
	api.Post("/tournaments", tournamentHandler.Create)
	api.Get("/tournaments/:id/edit", tournamentHandler.ShowEditForm)
	api.Put("/tournaments/:id", tournamentHandler.Update)
	api.Delete("/tournaments/:id", tournamentHandler.Delete)
	admin.Post("/tournaments/:id/finish", tournamentHandler.Finish)
	admin.Get("/tournaments/:id/export", tournamentHandler.Export)
	admin.Get("/tournaments/:id/export/pdf", tournamentHandler.ExportPDF)
	admin.Post("/tournaments/:id/teams", tournamentHandler.CreateTeam)
	admin.Delete("/tournaments/:id/teams/:teamId", tournamentHandler.DeleteTeam)
	admin.Post("/tournaments/:id/teams/:teamId/players", tournamentHandler.AssignPlayerToTeam)
	admin.Delete("/tournaments/:id/teams/:teamId/players/:playerId", tournamentHandler.RemovePlayerFromTeam)
	admin.Post("/tournaments/:id/move-player", tournamentHandler.MovePlayer)

	// Events API
	admin.Get("/events/division-select", adminHandler.DivisionSelect)
	admin.Get("/events/:id", eventHandler.Detail)
	api.Post("/events", eventHandler.Create)
	api.Delete("/events/:id", eventHandler.Delete)
	api.Post("/events/bulk-delete", eventHandler.DeleteBulk)

	// Matches API
	admin.Post("/matches/score/update", matchHandler.UpdateScore)
	api.Post("/matches/create", matchHandler.Create)
	api.Post("/matches/finish", matchHandler.Finish)
	api.Post("/matches/:id/start", matchHandler.Start)
	api.Post("/matches/:id/reset", matchHandler.Reset)
	api.Get("/matches/:id/score-form", matchHandler.ShowScoreForm)
	api.Put("/matches/:id/score", matchHandler.UpdateScore)

	// Divisions API
	api.Get("/divisions", divisionHandler.ShowEditForm)
	api.Get("/divisions/:id/edit", divisionHandler.ShowEditForm)
	api.Post("/divisions", divisionHandler.CreateOrUpdate)
	api.Delete("/divisions/:id", divisionHandler.Delete)

	// Reports API
	admin.Get("/reports/all-tournaments/pdf", tournamentHandler.ExportAllPDF)

	log.Fatal(app.Listen(":8080"))
}
