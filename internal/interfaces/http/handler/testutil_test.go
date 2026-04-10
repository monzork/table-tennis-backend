package handler_test

import (
	"context"
	"database/sql"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	adminDomain "table-tennis-backend/internal/domain/admin"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/application/division"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/handler"
	"table-tennis-backend/internal/interfaces/http/middleware"
)

var (
	testDB *bun.DB
	store  *session.Store
)

func SetupTestDB() (*bun.DB, error) {
	db, err := sql.Open("sqlite", "file::memory:?mode=memory&cache=shared")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	bunDB := bun.NewDB(db, sqlitedialect.New())

	bunDB.RegisterModel(
		(*bunRepo.TournamentParticipantModel)(nil),
		(*bunRepo.GroupParticipantModel)(nil),
	)

	models := []interface{}{
		(*bunRepo.AdminModel)(nil),
		(*bunRepo.DivisionModel)(nil),
		(*bunRepo.MatchModel)(nil),
		(*bunRepo.MatchSetModel)(nil),
		(*bunRepo.PlayerModel)(nil),
		(*bunRepo.StageRuleModel)(nil),
		(*bunRepo.TournamentModel)(nil),
		(*bunRepo.TournamentParticipantModel)(nil),
		(*bunRepo.GroupModel)(nil),
		(*bunRepo.GroupParticipantModel)(nil),
		(*bunRepo.RuleModel)(nil),
	}

	ctx := context.Background()
	for _, model := range models {
		_, err := bunDB.NewCreateTable().Model(model).IfNotExists().Exec(ctx)
		if err != nil {
			return nil, err
		}
	}

	adminRepo := bunRepo.NewAdminRepository(bunDB)
	a, _ := adminDomain.NewAdmin("admin", "password")
	adminRepo.Save(ctx, a)

	return bunDB, nil
}

func SetupTestApp() (*fiber.App, *bun.DB, *session.Store, error) {
	db, err := SetupTestDB()
	if err != nil {
		return nil, nil, nil, err
	}

	playerRepo := bunRepo.NewPlayerRepository(db)
	playerUC := player.NewRegisterPlayerUseCase(playerRepo)
	updatePlayerUC := player.NewUpdatePlayerUseCase(playerRepo)
	deletePlayerUC := player.NewDeletePlayerUseCase(playerRepo)
	getPlayerByIDUC := player.NewGetPlayerByIDUseCase(playerRepo)
	searchPlayerUC := player.NewSearchPlayersUseCase(playerRepo)
	importPlayerUC := player.NewImportPlayersUseCase(playerRepo)
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, importPlayerUC)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(*playerRepo)

	divisionRepo := bunRepo.NewDivisionRepository(db)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	tournamentRepo := bunRepo.NewTournamentRepository(db)
	createTournamentUC := tournament.NewCreateTournamentUseCase(tournamentRepo, playerRepo)
	getTournamentByIDUC := tournament.NewGetTournamentByIDUseCase(tournamentRepo)
	updateTournamentUC := tournament.NewUpdateTournamentUseCase(tournamentRepo, playerRepo)
	deleteTournamentUC := tournament.NewDeleteTournamentUseCase(tournamentRepo)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	finishTournamentUC := tournament.NewFinishTournamentUseCase(tournamentRepo, matchRepo, playerRepo)
	exportTournamentUC := tournament.NewExportTournamentReportUseCase(tournamentRepo)
	tournamentHandler := handler.NewTournamentHandler(createTournamentUC, getTournamentByIDUC, updateTournamentUC, deleteTournamentUC, leaderboardUC, divisionUC, finishTournamentUC, exportTournamentUC)
	GetMatchesUC := match.NewGetMatchesUseCase(*db, *playerRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, *playerRepo, *tournamentRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo)
	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC)

	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)
	divisionHandler := handler.NewDivisionHandler(divisionUC)

	adminRepo := bunRepo.NewAdminRepository(db)

	store = session.New()
	authHandler := handler.NewAuthHandler(store, adminRepo)
	authMiddleware := middleware.Protected(store)

	engine := html.New("../templates", ".html")
	app := fiber.New(fiber.Config{Views: engine})

	getTournamentsUC := tournament.NewGetTournamentsUseCase(tournamentRepo)
	adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC, getTournamentsUC, divisionUC)

	app.Get("/rankings/singles", leaderboardHandler.GetSingles)
	app.Get("/rankings/doubles", leaderboardHandler.GetDoubles)

	// Auth endpoints
	app.Get("/admin/login", authHandler.ShowLogin)
	app.Post("/admin/login", authHandler.Login)
	app.Post("/admin/logout", authHandler.Logout)

	admin := app.Group("/admin")
	admin.Use(authMiddleware)
	admin.Get("/", adminHandler.Dashboard)
	admin.Get("/players", adminHandler.Players)
	admin.Get("/tournaments", adminHandler.Tournaments)
	admin.Get("/divisions", adminHandler.Divisions)

	api := app.Group("/")
	api.Use(authMiddleware)
	api.Post("/players", playerHandler.Register)
	api.Put("/players/:id", playerHandler.Update)
	api.Delete("/players/:id", playerHandler.Delete)
	api.Post("/players/import", playerHandler.Import)
	api.Post("/tournaments", tournamentHandler.Create)
	api.Post("/matches/create", matchHandler.Create)
	api.Post("/matches/finish", matchHandler.Finish)
	api.Put("/matches/:id/score", matchHandler.UpdateScore)
	api.Post("/divisions", divisionHandler.CreateOrUpdate)
	api.Delete("/divisions/:id", divisionHandler.Delete)

	admin.Get("/tournaments/:id", tournamentHandler.Detail)
	api.Put("/tournaments/:id", tournamentHandler.Update)
	api.Delete("/tournaments/:id", tournamentHandler.Delete)
	admin.Post("/tournaments/:id/finish", tournamentHandler.Finish)
	admin.Get("/tournaments/:id/export", tournamentHandler.Export)

	return app, db, store, nil
}
