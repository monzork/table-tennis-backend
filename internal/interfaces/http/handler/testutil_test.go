package handler_test

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"html/template"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/gofiber/template/html/v2"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/sqlitedialect"
	_ "modernc.org/sqlite"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	adminDomain "table-tennis-backend/internal/domain/admin"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/infrastructure/identity"
	pdfinfra "table-tennis-backend/internal/infrastructure/pdf"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
	securityInfra "table-tennis-backend/internal/infrastructure/security"
	"table-tennis-backend/internal/interfaces/http/handler"
	"table-tennis-backend/internal/interfaces/http/i18n"
	"table-tennis-backend/internal/interfaces/http/middleware"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

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
		(*bunRepo.EventModel)(nil),
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
	hasher := securityInfra.NewBcryptHasher()
	hashed, _ := hasher.Hash("password")
	a, _ := adminDomain.NewAdmin(idgen.Generate(), "admin", hashed)
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
	searchPlayerSelectionUC := player.NewSearchPlayersForSelectionUseCase(playerRepo)
	importPlayerUC := player.NewImportPlayersUseCase(playerRepo)
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, searchPlayerSelectionUC, importPlayerUC)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)

	divisionRepo := bunRepo.NewDivisionRepository(db)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	tournamentRepo := bunRepo.NewTournamentRepository(db)
	createTournamentUC := tournament.NewCreateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	getTournamentByIDUC := tournament.NewGetTournamentByIDUseCase(tournamentRepo, divisionRepo)
	updateTournamentUC := tournament.NewUpdateTournamentUseCase(tournamentRepo, playerRepo, divisionRepo)
	deleteTournamentUC := tournament.NewDeleteTournamentUseCase(tournamentRepo)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	finishTournamentUC := tournament.NewFinishTournamentUseCase(tournamentRepo, matchRepo, playerRepo)
	exportTournamentUC := tournament.NewExportTournamentReportUseCase(tournamentRepo)
	pdfGen := pdfinfra.NewGoFpdfGenerator()
	exportTournamentPdfUC := tournament.NewExportTournamentPdfUseCase(tournamentRepo, pdfGen)
	movePlayerUC := tournament.NewMovePlayerUseCase(tournamentRepo)
	createTeamUC := tournament.NewCreateTeamUseCase(tournamentRepo)
	deleteTeamUC := tournament.NewDeleteTeamUseCase(tournamentRepo)
	assignPlayerToTeamUC := tournament.NewAssignPlayerToTeamUseCase(tournamentRepo)
	removePlayerFromTeamUC := tournament.NewRemovePlayerFromTeamUseCase(tournamentRepo)
	getTournamentsUC := tournament.NewGetTournamentsUseCase(tournamentRepo)
	tournamentHandler := handler.NewTournamentHandler(
		createTournamentUC, getTournamentByIDUC, updateTournamentUC, deleteTournamentUC,
		leaderboardUC, divisionUC, finishTournamentUC, exportTournamentUC, exportTournamentPdfUC,
		movePlayerUC, createTeamUC, deleteTeamUC, assignPlayerToTeamUC, removePlayerFromTeamUC,
		getTournamentsUC, nil,
	)

	eventRepo := bunRepo.NewEventRepository(db, tournamentRepo)
	exportEventPdfUC := tournament.NewExportEventPdfUseCase(eventRepo, pdfGen)
	createEventUC := event.NewCreateEventUseCase(eventRepo, tournamentRepo, playerRepo, divisionRepo)
	getEventByIDUC := event.NewGetEventByIDUseCase(eventRepo)
	getAllEventsUC := event.NewGetAllEventsUseCase(eventRepo)
	deleteEventUC := event.NewDeleteEventUseCase(eventRepo)
	eventHandler := handler.NewEventHandler(createEventUC, getEventByIDUC, getAllEventsUC, deleteEventUC, divisionUC, leaderboardUC, exportEventPdfUC)
	GetMatchesUC := match.NewGetMatchesUseCase(matchRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, tournamentRepo, divisionRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, tournamentRepo)
	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, tournamentRepo, finishTournamentUC)

	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC)
	divisionHandler := handler.NewDivisionHandler(divisionUC)
	selfRegisterUC := tournament.NewSelfRegisterUseCase(tournamentRepo, playerRepo)
	publicHandler := handler.NewPublicHandler(playerUC, selfRegisterUC)

	adminRepo := bunRepo.NewAdminRepository(db)

	store = session.New()
	authHandler := handler.NewAuthHandler(store, adminRepo, securityInfra.NewBcryptHasher())
	authMiddleware := middleware.Protected(store)

	engine := html.New("../templates", ".html")
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
	engine.AddFunc("t", func(tmap map[string]string, key string) string {
		if tmap != nil {
			if v, ok := tmap[key]; ok {
				return v
			}
		}
		if v, ok := i18n.Translations["en"][key]; ok {
			return v
		}
		return key
	})
	engine.AddFunc("safeHTML", func(s string) template.HTML {
		return template.HTML(s)
	})
	app := fiber.New(fiber.Config{
		Views:             engine,
		PassLocalsToViews: true,
	})

	// Global Translation Middleware for tests
	app.Use(func(c *fiber.Ctx) error {
		lang := c.Cookies("lang")
		if lang != "es" && lang != "en" {
			lang = "en"
		}
		c.Locals("Lang", lang)

		translations := make(map[string]string)
		if lang == "es" {
			translations = map[string]string{
				"nav.dashboard": "Dashboard",
				// minimal mock to avoid panic in layout
			}
		} else {
			translations = map[string]string{
				"nav.dashboard": "Dashboard",
			}
		}
		c.Locals("T", translations)
		return c.Next()
	})

	adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC, getTournamentsUC, divisionUC, getAllEventsUC)

	app.Get("/rankings/singles", leaderboardHandler.GetSingles)
	app.Get("/rankings/doubles", leaderboardHandler.GetDoubles)
	app.Get("/players/department-input", publicHandler.DepartmentInput)
	app.Get("/register", publicHandler.ShowSignup)
	app.Post("/register", publicHandler.Register)
	app.Get("/tournaments/register", publicHandler.ShowTournamentRegistration)
	app.Post("/tournaments/register", publicHandler.RegisterToTournament)

	// Public Score Entry & Match Starting Endpoints
	app.Get("/public/matches/score/form", matchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/form", matchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/update", matchHandler.UpdatePublicScore)
	app.Post("/public/matches/start", matchHandler.Start)
	app.Post("/public/matches/:id/start", matchHandler.Start)

	// Auth endpoints
	app.Get("/admin/login", authHandler.ShowLogin)
	app.Post("/admin/login", authHandler.Login)
	app.Post("/admin/logout", authHandler.Logout)

	admin := app.Group("/admin")
	admin.Use(authMiddleware)
	admin.Get("/", adminHandler.Dashboard)
	admin.Get("/players", adminHandler.Players)
	admin.Get("/tournaments", adminHandler.Tournaments)
	admin.Get("/events", adminHandler.Events)
	admin.Get("/events/division-select", adminHandler.DivisionSelect)
	admin.Get("/events/:id", eventHandler.Detail)
	admin.Get("/divisions", adminHandler.Divisions)

	api := app.Group("/")
	api.Use(authMiddleware)
	api.Post("/players", playerHandler.Register)
	api.Get("/players/search", playerHandler.Search)
	api.Get("/players/search/cards", playerHandler.SearchSelectionCards)
	api.Put("/players/:id", playerHandler.Update)
	api.Delete("/players/:id", playerHandler.Delete)
	api.Post("/players/import", playerHandler.Import)
	api.Post("/tournaments", tournamentHandler.Create)
	api.Post("/events", eventHandler.Create)
	api.Delete("/events/:id", eventHandler.Delete)
	api.Post("/events/bulk-delete", eventHandler.DeleteBulk)
	api.Post("/matches/create", matchHandler.Create)
	api.Post("/matches/finish", matchHandler.Finish)
	api.Post("/matches/:id/start", matchHandler.Start)
	api.Put("/matches/:id/score", matchHandler.UpdateScore)
	api.Post("/divisions", divisionHandler.CreateOrUpdate)
	api.Delete("/divisions/:id", divisionHandler.Delete)

	admin.Get("/tournaments/:id", tournamentHandler.Detail)
	api.Put("/tournaments/:id", tournamentHandler.Update)
	api.Delete("/tournaments/:id", tournamentHandler.Delete)
	admin.Post("/tournaments/:id/finish", tournamentHandler.Finish)
	admin.Get("/tournaments/:id/export", tournamentHandler.Export)
	admin.Get("/tournaments/:id/export/pdf", tournamentHandler.ExportPDF)
	admin.Post("/tournaments/:id/move-player", tournamentHandler.MovePlayer)

	return app, db, store, nil
}
