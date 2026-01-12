package main

import (
	"log"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/handler"

	"github.com/gofiber/fiber/v2"
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

	matchRepo := bun.NewMatchRepository(bun.DB)
	createMatchUC := match.NewCreateMatchUseCase(matchRepo, *playerRepo, *tournamentRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(*playerRepo)
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC)

	engine := html.New("./internal/interfaces/http/templates", ".html")

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	app.Get("/leaderboard", leaderboardHandler.Get)

	// Dashboard
	dashboardHandler := handler.NewDashboardHandler()
	app.Get("/", dashboardHandler.Show)

	// Players
	app.Post("/players", playerHandler.Register)

	// Tournaments
	app.Post("/tournaments", tournamentHandler.Create)

	// Matches
	app.Post("/matches/create", matchHandler.Create)
	app.Post("/matches/finish", matchHandler.Finish)
	log.Fatal(app.Listen(":8080"))
}
