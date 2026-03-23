package main

	import (
		"context"
		"log"
		"os"

		adminDomain "table-tennis-backend/internal/domain/admin"
		"table-tennis-backend/internal/application/leaderboard"
		"table-tennis-backend/internal/application/match"
		"table-tennis-backend/internal/application/player"
		"table-tennis-backend/internal/application/tournament"
		"table-tennis-backend/internal/infrastructure/persistence/bun"
		"table-tennis-backend/internal/interfaces/http/handler"
		"table-tennis-backend/internal/interfaces/http/middleware"

		"github.com/gofiber/fiber/v2"
		"github.com/gofiber/fiber/v2/middleware/session"
		"github.com/gofiber/template/html/v2"
	)

	func main() {
		bun.Connect()

		playerRepo := bun.NewPlayerRepository(bun.DB)
		playerUC := player.NewRegisterPlayerUseCase(playerRepo)
		playerHandler := handler.NewPlayerHandler(playerUC)

		tournamentRepo := bun.NewTournamentRepository(bun.DB)
		createTournamentUC := tournament.NewCreateTournamentUseCase(tournamentRepo)
		tournamentHandler := handler.NewTournamentHandler(createTournamentUC)

		matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)
		GetMatchesUC := match.NewGetMatchesUseCase(*bun.DB, *playerRepo)

		createMatchUC := match.NewCreateMatchUseCase(matchRepo, *playerRepo, *tournamentRepo)
		finishMatchUC := match.NewFinishMatchUseCase()
		matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC)

		leaderboardUC := leaderboard.NewGetLeaderboardUseCase(*playerRepo)
		leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC)
		
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

		adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC)

		app.Get("/rankings/singles", leaderboardHandler.GetSingles)
		app.Get("/rankings/doubles", leaderboardHandler.GetDoubles)


		// Redirect Root to Public Rankings
		app.Get("/", func(c *fiber.Ctx) error {
			return c.Redirect("/rankings/singles")
		})

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
		admin.Get("/matches", adminHandler.Matches)

		// Existing Form Post Endpoints mapped internally, protected
		api := app.Group("/")
		api.Use(authMiddleware)
		api.Post("/players", playerHandler.Register)
		api.Post("/tournaments", tournamentHandler.Create)
		api.Post("/matches/create", matchHandler.Create)
		api.Post("/matches/finish", matchHandler.Finish)

		log.Fatal(app.Listen(":8080"))
	}
