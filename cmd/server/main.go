package main

	import (
		"context"
		"log"
		"os"
		"time"

		adminDomain "table-tennis-backend/internal/domain/admin"
		"table-tennis-backend/internal/application/leaderboard"
		"table-tennis-backend/internal/application/match"
		"table-tennis-backend/internal/application/player"
		"table-tennis-backend/internal/application/tournament"
		"table-tennis-backend/internal/application/division"
		"table-tennis-backend/internal/infrastructure/persistence/bun"
		"table-tennis-backend/internal/interfaces/http/handler"
		"table-tennis-backend/internal/interfaces/http/middleware"

		"github.com/gofiber/fiber/v2"
		"github.com/gofiber/fiber/v2/middleware/limiter"
		"github.com/gofiber/fiber/v2/middleware/session"
		"github.com/gofiber/template/html/v2"
	)

	func main() {
		bun.Connect()

		playerRepo := bun.NewPlayerRepository(bun.DB)
		playerUC := player.NewRegisterPlayerUseCase(playerRepo)
		updatePlayerUC := player.NewUpdatePlayerUseCase(playerRepo)
		deletePlayerUC := player.NewDeletePlayerUseCase(playerRepo)
		importPlayerUC := player.NewImportPlayersUseCase(playerRepo)
		getPlayerByIDUC := player.NewGetPlayerByIDUseCase(playerRepo)
		searchPlayerUC := player.NewSearchPlayersUseCase(playerRepo)
		playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, importPlayerUC)

		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(*playerRepo)

		divisionRepo := bun.NewDivisionRepository(bun.DB)
		divisionUC := division.NewDivisionUseCase(divisionRepo)

		tournamentRepo := bun.NewTournamentRepository(bun.DB)
		createTournamentUC := tournament.NewCreateTournamentUseCase(tournamentRepo, playerRepo)
		getTournamentByIDUC := tournament.NewGetTournamentByIDUseCase(tournamentRepo)
		updateTournamentUC := tournament.NewUpdateTournamentUseCase(tournamentRepo, playerRepo)
		deleteTournamentUC := tournament.NewDeleteTournamentUseCase(tournamentRepo)
		matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)
		finishTournamentUC := tournament.NewFinishTournamentUseCase(tournamentRepo, matchRepo, playerRepo)
		exportTournamentUC := tournament.NewExportTournamentReportUseCase(tournamentRepo)
		exportTournamentPdfUC := tournament.NewExportTournamentPdfUseCase(tournamentRepo)
		tournamentHandler := handler.NewTournamentHandler(createTournamentUC, getTournamentByIDUC, updateTournamentUC, deleteTournamentUC, leaderboardUC, divisionUC, finishTournamentUC, exportTournamentUC, exportTournamentPdfUC)
		GetMatchesUC := match.NewGetMatchesUseCase(*bun.DB, *playerRepo)

		createMatchUC := match.NewCreateMatchUseCase(matchRepo, *playerRepo, *tournamentRepo)
		finishMatchUC := match.NewFinishMatchUseCase()
		updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo)
		matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, finishTournamentUC)



		leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)
		divisionHandler := handler.NewDivisionHandler(divisionUC)
		publicHandler := handler.NewPublicHandler(playerUC)

		adminRepo := bun.NewAdminRepository(bun.DB)

		// Seed default admin if DB empty
		count, _ := adminRepo.Count(context.Background())
		if count == 0 {
			user := os.Getenv("ADMIN_USERNAME")
			pass := os.Getenv("ADMIN_PASSWORD")
			if user == "" { user = "admin" }
			if pass == "" { pass = "password" }
			if a, err := adminDomain.NewAdmin(user, pass); err == nil {
				adminRepo.Save(context.Background(), a)
			}
		}

		store := session.New()
		authHandler := handler.NewAuthHandler(store, adminRepo)
		authMiddleware := middleware.Protected(store)

		engine := html.New("./internal/interfaces/http/templates", ".html")
		app := fiber.New(fiber.Config{
			Views: engine,
		})

		getTournamentsUC := tournament.NewGetTournamentsUseCase(tournamentRepo)
		adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC, getTournamentsUC, divisionUC)

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

		// Public Signup with Rate Limiting (5 requests per min)
		signupLimiter := limiter.New(limiter.Config{
			Max:        5,
			Expiration: 1 * time.Minute,
			KeyGenerator: func(c *fiber.Ctx) string {
				return c.IP()
			},
		})
		app.Get("/register", publicHandler.ShowSignup)
		app.Post("/register", signupLimiter, publicHandler.Register)

		// Auth endpoints
		app.Get("/admin/login", authHandler.ShowLogin)
		app.Post("/admin/login", authHandler.Login)
		app.Post("/admin/logout", authHandler.Logout)

		// Admin functionality protected by session auth
		admin := app.Group("/admin")
		admin.Use(authMiddleware)
		admin.Get("/", adminHandler.Dashboard)
		admin.Get("/players", adminHandler.Players)
		admin.Get("/tournaments", adminHandler.Tournaments)
		admin.Get("/divisions", adminHandler.Divisions)
		admin.Get("/player-field", adminHandler.NewPlayerField)
		admin.Get("/matches/score/form", matchHandler.ShowScoreForm)
		admin.Post("/matches/score/update", matchHandler.UpdateScore)
		admin.Post("/tournaments/:id/finish", tournamentHandler.Finish)

		// Existing Form Post Endpoints mapped internally, protected
		api := app.Group("/")
		api.Use(authMiddleware)
		api.Post("/players", playerHandler.Register)
		api.Get("/players/search", playerHandler.Search)
		api.Get("/players/search/cards", playerHandler.SearchSelectionCards)
		api.Get("/players/:id/edit", playerHandler.ShowEditForm)
		api.Put("/players/:id", playerHandler.Update)
		api.Delete("/players/:id", playerHandler.Delete)
		api.Post("/players/import", playerHandler.Import)
		app.Get("/players/import/template", playerHandler.ImportTemplate)
		api.Post("/tournaments", tournamentHandler.Create)
		api.Post("/matches/create", matchHandler.Create)
		api.Post("/matches/finish", matchHandler.Finish)
		api.Get("/matches/:id/score-form", matchHandler.ShowScoreForm)
		api.Put("/matches/:id/score", matchHandler.UpdateScore)
		api.Get("/divisions", divisionHandler.ShowEditForm) // for new
		api.Get("/divisions/:id/edit", divisionHandler.ShowEditForm)
		api.Post("/divisions", divisionHandler.CreateOrUpdate)
		api.Delete("/divisions/:id", divisionHandler.Delete)

		// Tournament CRUD routes (admin protected)
		admin.Get("/tournaments/:id", tournamentHandler.Detail)
		api.Get("/tournaments/:id/edit", tournamentHandler.ShowEditForm)
		api.Put("/tournaments/:id", tournamentHandler.Update)
		api.Delete("/tournaments/:id", tournamentHandler.Delete)
		admin.Post("/tournaments/:id/finish", tournamentHandler.Finish)
		admin.Get("/tournaments/:id/export", tournamentHandler.Export)
		admin.Get("/tournaments/:id/export/pdf", tournamentHandler.ExportPDF)

		log.Fatal(app.Listen(":8080"))
	}
