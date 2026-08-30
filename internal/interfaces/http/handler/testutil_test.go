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

	accountApp "table-tennis-backend/internal/application/account"
	dashboardApp "table-tennis-backend/internal/application/dashboard"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/notification"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"
	adminDomain "table-tennis-backend/internal/domain/admin"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/tournaments"
	"table-tennis-backend/internal/infrastructure/identity"
	oauthinfra "table-tennis-backend/internal/infrastructure/oauth"
	pdfinfra "table-tennis-backend/internal/infrastructure/pdf"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"
	securityInfra "table-tennis-backend/internal/infrastructure/security"
	svgchartinfra "table-tennis-backend/internal/infrastructure/svgchart"
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
		(*bunRepo.EventParticipantModel)(nil),
		(*bunRepo.GroupParticipantModel)(nil),
		(*bunRepo.TeamPlayerModel)(nil),
	)

	models := []interface{}{
		(*bunRepo.AdminModel)(nil),
		(*bunRepo.AccountModel)(nil),
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
	tournamentRepoForEnroll := bunRepo.NewEventRepository(db)
	dispatcher := tournaments.NewInMemoryDispatcher()
	enrollPlayerUC := event.NewEnrollPlayerUseCase(tournamentRepoForEnroll, dispatcher)
	getTournamentsUC := event.NewGetTournamentsUseCase(tournamentRepoForEnroll)
	parentTournamentRepoForStats := bunRepo.NewTournamentRepository(db, tournamentRepoForEnroll)
	getPlayerStatsUC := player.NewGetPlayerTournamentStatsUseCase(playerRepo, tournamentRepoForEnroll, parentTournamentRepoForStats)
	getEloTrendUC := player.NewGetPlayerEloTrendUseCase(svgchartinfra.NewSVGGenerator())
	playerHandler := handler.NewPlayerHandler(playerUC, updatePlayerUC, deletePlayerUC, getPlayerByIDUC, searchPlayerUC, searchPlayerSelectionUC, importPlayerUC, enrollPlayerUC, getTournamentsUC, getPlayerStatsUC, getEloTrendUC)

	leaderboardUC := leaderboard.NewGetLeaderboardUseCase(playerRepo)

	divisionRepo := bunRepo.NewDivisionRepository(db)
	divisionUC := division.NewDivisionUseCase(divisionRepo)

	tournamentRepo := bunRepo.NewEventRepository(db)
	createTournamentUC := event.NewCreateTournamentUseCase(tournamentRepo, playerRepo)
	getTournamentByIDUC := event.NewGetTournamentByIDUseCase(tournamentRepo)
	updateTournamentUC := event.NewUpdateTournamentUseCase(tournamentRepo, playerRepo)
	deleteTournamentUC := event.NewDeleteTournamentUseCase(tournamentRepo)
	matchRepo := bunRepo.NewMatchRepository(db, playerRepo)
	finishTournamentUC := event.NewFinishTournamentUseCase(tournamentRepo, matchRepo, playerRepo)
	exportTournamentUC := event.NewExportTournamentReportUseCase(tournamentRepo)
	pdfGen := pdfinfra.NewGoFpdfGenerator()
	exportTournamentPdfUC := event.NewExportTournamentPdfUseCase(tournamentRepo, divisionRepo, pdfGen)
	movePlayerUC := event.NewMovePlayerUseCase(tournamentRepo)
	createTeamUC := event.NewCreateTeamUseCase(tournamentRepo)
	deleteTeamUC := event.NewDeleteTeamUseCase(tournamentRepo)
	assignPlayerToTeamUC := event.NewAssignPlayerToTeamUseCase(tournamentRepo)
	removePlayerFromTeamUC := event.NewRemovePlayerFromTeamUseCase(tournamentRepo)
	getTournamentsUC = event.NewGetTournamentsUseCase(tournamentRepo)
	regenerateSeedsUC := event.NewRegenerateGroupSeedsUseCase(tournamentRepo, matchRepo)
	updateParticipantEloUC := event.NewUpdateParticipantEloBeforeUseCase(tournamentRepo, regenerateSeedsUC)
	getOccupiedTablesUC := event.NewGetOccupiedTablesUseCase(matchRepo)
	removeParticipantUC := event.NewRemoveParticipantUseCase(tournamentRepo)
	saveKnockoutSeedsUC := event.NewSaveKnockoutSeedsUseCase(tournamentRepo, divisionRepo)
	toggleSeedingLockUC := event.NewToggleSeedingLockUseCase(tournamentRepo)
	addGroupUC := event.NewAddGroupUseCase(tournamentRepo)
	recalculateEloUC := event.NewRecalculateTournamentEloUseCase(tournamentRepo, matchRepo, playerRepo)

	startKnockoutUC := event.NewStartKnockoutStageUseCase(tournamentRepo, matchRepo, divisionRepo)
	getEventDetailViewUC := event.NewGetEventDetailViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC)
	getPublicDetailViewUC := event.NewGetPublicEventDetailViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC)
	tvDashboardUC := event.NewGetPublicTVDashboardViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC)
	boardViewUC := event.NewGetBoardViewUseCase(getTournamentByIDUC, divisionUC)
	editFormViewUC := event.NewGetEditFormViewUseCase(getTournamentByIDUC, leaderboardUC, divisionUC)

	tournamentHandler := handler.NewEventHandler(
		createTournamentUC, getTournamentByIDUC, updateTournamentUC, deleteTournamentUC,
		leaderboardUC, divisionUC, finishTournamentUC, exportTournamentUC, exportTournamentPdfUC,
		movePlayerUC, createTeamUC, deleteTeamUC, assignPlayerToTeamUC, removePlayerFromTeamUC,
		getTournamentsUC, getOccupiedTablesUC, regenerateSeedsUC, updateParticipantEloUC,
		removeParticipantUC, saveKnockoutSeedsUC, toggleSeedingLockUC, addGroupUC, recalculateEloUC,
		startKnockoutUC, getEventDetailViewUC, getPublicDetailViewUC, tvDashboardUC, boardViewUC, editFormViewUC,
	)

	eventRepo := bunRepo.NewTournamentRepository(db, tournamentRepo)
	exportEventPdfUC := event.NewExportEventPdfUseCase(eventRepo, tournamentRepo, divisionRepo, pdfGen)
	createEventUC := tournament.NewCreateEventUseCase(eventRepo, tournamentRepo, playerRepo, divisionRepo)
	updateEventUC := tournament.NewUpdateEventUseCase(eventRepo)
	getEventByIDUC := tournament.NewGetEventByIDUseCase(eventRepo)
	getAllEventsUC := tournament.NewGetAllEventsUseCase(eventRepo)
	deleteEventUC := tournament.NewDeleteEventUseCase(eventRepo)
	getBoardUC := tournament.NewGetBoardDataUseCase(eventRepo, divisionRepo)
	eventHandler := handler.NewTournamentHandler(createEventUC, updateEventUC, getEventByIDUC, getAllEventsUC, deleteEventUC, divisionUC, leaderboardUC, exportEventPdfUC, getBoardUC)
	GetMatchesUC := match.NewGetMatchesUseCase(matchRepo)

	createMatchUC := match.NewCreateMatchUseCase(matchRepo, playerRepo, tournamentRepo, divisionRepo)
	finishMatchUC := match.NewFinishMatchUseCase()
	updateScoreUC := match.NewUpdateMatchScoreUseCase(matchRepo, tournamentRepo)
	teamMatchUC := match.NewTeamMatchOrchestratorUseCase(matchRepo)
	startMatchUC := match.NewStartMatchUseCase(matchRepo, tournamentRepo, eventRepo, createMatchUC)
	pushSubRepo := bunRepo.NewPushSubscriptionRepository(db)
	broadcastPushUC := notification.NewBroadcastPushNotificationUseCase(pushSubRepo, "test-pubkey", "test-privkey")
	matchHandler := handler.NewMatchHandler(createMatchUC, finishMatchUC, updateScoreUC, playerRepo, matchRepo, tournamentRepo, eventRepo, finishTournamentUC, broadcastPushUC, teamMatchUC, startMatchUC, divisionRepo)

	distUC := leaderboard.NewGetDivisionDistributionUseCase(svgchartinfra.NewSVGGenerator())
	leaderboardHandler := handler.NewLeaderboardHandler(leaderboardUC, divisionUC, distUC)
	divisionHandler := handler.NewDivisionHandler(divisionUC)
	dashboardRepo := bunRepo.NewDashboardRepository(db)
	getDashboardViewUC := dashboardApp.NewGetPublicDashboardViewUseCase(dashboardRepo, svgchartinfra.NewSVGGenerator())
	dashboardHandler := handler.NewDashboardHandler(getDashboardViewUC)
	selfRegisterUC := event.NewSelfRegisterUseCase(tournamentRepo, playerRepo)
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
	engine.AddFunc("mul", func(a, b int) int {
		return a * b
	})
	engine.AddFunc("div", func(a, b int) int {
		if b == 0 {
			return 0
		}
		return a / b
	})
	engine.AddFunc("seq", func(n int) []int {
		if n <= 0 {
			return nil
		}
		s := make([]int, n)
		for i := range s {
			s[i] = i + 1
		}
		return s
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
	engine.AddFunc("cleanPhone", func(phone string) string {
		var b strings.Builder
		for _, ch := range phone {
			if ch >= '0' && ch <= '9' {
				b.WriteRune(ch)
			}
		}
		return b.String()
	})
	engine.AddFunc("safeHTML", func(s string) template.HTML {
		return template.HTML(s)
	})
	engine.AddFunc("eventDivision", handler.EventDivisionName)
	engine.AddFunc("eloDelta", func(delta *float64) string {
		if delta == nil {
			return ""
		}
		return fmt.Sprintf("%+.0f", *delta)
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

	// Non-blocking admin-session check, mirroring cmd/server/main.go.
	sessStore := store // capture by value: `store` is a shared package var some
	// nested SetupTestApp() calls in other subtests reassign; middleware.Protected
	// below captures its store argument at construction time the same way.
	app.Use(func(c *fiber.Ctx) error {
		isAdmin := false
		if sess, err := sessStore.Get(c); err == nil {
			isAdmin, _ = sess.Get("authenticated").(bool)
		}
		c.Locals("IsAdminSession", isAdmin)
		return c.Next()
	})

	adminHandler := handler.NewAdminHandler(playerUC, createTournamentUC, createMatchUC, GetMatchesUC, leaderboardUC, getTournamentsUC, divisionUC, getAllEventsUC)

	app.Get("/rankings/singles", leaderboardHandler.GetSingles)
	app.Get("/dashboard", dashboardHandler.Public)
	app.Get("/rankings/doubles", leaderboardHandler.GetDoubles)
	app.Get("/players/department-input", publicHandler.DepartmentInput)
	app.Get("/players/:id/stats", playerHandler.PublicStats)
	app.Get("/register", publicHandler.ShowSignup)
	app.Post("/register", publicHandler.Register)
	app.Get("/events/register", publicHandler.ShowTournamentRegistration)
	app.Get("/events/register/:id", publicHandler.ShowTournamentRegisterForm)
	app.Post("/events/register", publicHandler.RegisterToTournament)
	app.Get("/lang/:locale", publicHandler.SetLang)
	app.Get("/sitemap.xml", publicHandler.Sitemap)

	// Public Score Entry & Match Starting Endpoints
	app.Get("/public/matches/score/form", matchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/form", matchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/update", matchHandler.UpdatePublicScore)
	app.Post("/public/matches/start", matchHandler.Start)
	app.Post("/public/matches/:id/start", matchHandler.Start)
	app.Get("/public/score/:matchId", matchHandler.ShowMatchScorePage)
	app.Get("/public/score/table/:tableNumber/event/:eventId", matchHandler.ShowTableScorePage)
	app.Get("/public/score/table/:tableNumber/tournament/:tournamentId", matchHandler.ShowTableScorePage)
	app.Post("/public/score/:matchId/verify", matchHandler.ValidateMatchPIN)

	// Public tournament endpoints
	app.Get("/public/tournaments", eventHandler.PublicList)
	app.Get("/tournaments/:id/public", eventHandler.PublicDetail)
	app.Get("/tournaments/:id/tv", eventHandler.PublicTVDashboard)
	app.Get("/tournaments/:id/board-columns", eventHandler.BoardColumns)

	// Auth endpoints
	app.Get("/admin/login", authHandler.ShowLogin)
	app.Post("/admin/login", authHandler.Login)
	app.Post("/admin/logout", authHandler.Logout)

	admin := app.Group("/admin")
	admin.Use(authMiddleware)
	admin.Get("/", adminHandler.Dashboard)
	admin.Get("/players", adminHandler.Players)

	admin.Get("/tournaments", adminHandler.Tournaments)
	admin.Get("/tournaments/division-select", adminHandler.DivisionSelect)
	admin.Get("/tournaments/:id", eventHandler.Detail)
	admin.Get("/divisions", adminHandler.Divisions)
	admin.Get("/new-player-field", adminHandler.NewPlayerField)
	admin.Get("/close-modal", adminHandler.CloseModal)

	// ── Guardian account area (entirely separate from /admin) ──────────────
	// Registered before `api := app.Group("/"); api.Use(authMiddleware)` below:
	// Fiber applies Use() middleware to every route registered afterward that
	// matches its prefix, and api's prefix is "/" (matches everything), so
	// these public/account routes must exist before that Use() call or they'd
	// incorrectly require an admin session.
	accountRepo := bunRepo.NewAccountRepository(db)
	googleClient := oauthinfra.NewGoogleClient("test-client-id", "test-client-secret", "http://localhost/account/google/callback")
	loginWithGoogleUC := accountApp.NewLoginWithGoogleUseCase(accountRepo)
	getAccountByIDUC := accountApp.NewGetAccountByIDUseCase(accountRepo)
	updateAccountUC := accountApp.NewUpdateAccountUseCase(accountRepo)
	createChildPlayerUC := accountApp.NewCreateChildPlayerUseCase(playerRepo)
	getLinkedPlayersUC := accountApp.NewGetLinkedPlayersUseCase(playerRepo)
	getPlayerPendingMatchesUC := player.NewGetPlayerPendingMatchesUseCase(tournamentRepo, divisionRepo)
	getGuardianPendingMatchesUC := accountApp.NewGetGuardianPendingMatchesUseCase(getLinkedPlayersUC, getPlayerPendingMatchesUC)
	proposeMatchScoreUC := match.NewProposeMatchScoreUseCase(matchRepo, tournamentRepo, playerRepo)
	confirmMatchScoreUC := match.NewConfirmMatchScoreUseCase(matchRepo, tournamentRepo, updateScoreUC)
	rejectMatchScoreProposalUC := match.NewRejectMatchScoreProposalUseCase(matchRepo)

	accountHandler := handler.NewAccountHandler(
		store, googleClient,
		loginWithGoogleUC, getAccountByIDUC, updateAccountUC, createChildPlayerUC,
		getLinkedPlayersUC, getGuardianPendingMatchesUC,
		getPlayerByIDUC, updatePlayerUC, getPlayerPendingMatchesUC,
		proposeMatchScoreUC, confirmMatchScoreUC, rejectMatchScoreProposalUC,
	)

	app.Get("/account/login", accountHandler.ShowLogin)
	app.Get("/account/google/login", accountHandler.GoogleLogin)
	app.Get("/account/google/callback", accountHandler.GoogleCallback)
	app.Post("/account/logout", accountHandler.Logout)

	// Test-only bypass to establish an account session without a live Google
	// OAuth client (exercised end-to-end separately in internal/infrastructure/oauth).
	app.Post("/test/account-login", func(c *fiber.Ctx) error {
		var body struct {
			AccountID string `form:"accountId"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		sess, err := store.Get(c)
		if err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		sess.Set("account_authenticated", true)
		sess.Set("account_id", body.AccountID)
		if err := sess.Save(); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		return c.SendString("account logged in")
	})

	accountGroup := app.Group("/account")
	accountGroup.Use(middleware.AccountProtected(store))
	accountGroup.Get("/", accountHandler.Dashboard)
	accountGroup.Get("/me", accountHandler.ShowMyInfo)
	accountGroup.Put("/me", accountHandler.UpdateMyInfo)
	accountGroup.Get("/players/new", accountHandler.ShowAddChildForm)
	accountGroup.Post("/players", accountHandler.CreateChild)
	claimPlayerUC := accountApp.NewClaimPlayerUseCase(playerRepo)
	searchClaimableUC := accountApp.NewSearchClaimablePlayersUseCase(playerRepo)
	accountHandler.WithClaimUseCases(claimPlayerUC, searchClaimableUC)
	accountGroup.Get("/players/claim", accountHandler.ShowClaimForm)
	accountGroup.Get("/players/claim/search", accountHandler.SearchClaimable)
	accountGroup.Post("/players/:id/claim", accountHandler.ClaimPlayer)
	accountGroup.Get("/players/:id", accountHandler.PlayerDetail)
	accountGroup.Get("/players/:id/edit", accountHandler.EditPlayer)
	accountGroup.Put("/players/:id", accountHandler.UpdatePlayer)
	accountGroup.Get("/pending-matches", accountHandler.PendingMatches)
	accountGroup.Post("/matches/:matchId/propose-score", accountHandler.ProposeScore)
	accountGroup.Post("/matches/:matchId/confirm-score", accountHandler.ConfirmScore)
	accountGroup.Post("/matches/:matchId/reject-score", accountHandler.RejectScore)

	api := app.Group("/")
	api.Use(authMiddleware)
	api.Post("/players", playerHandler.Register)
	api.Get("/players/search", playerHandler.Search)
	api.Get("/players/search/cards", playerHandler.SearchSelectionCards)
	api.Put("/players/:id", playerHandler.Update)
	api.Delete("/players/:id", playerHandler.Delete)
	api.Post("/players/import", playerHandler.Import)
	api.Get("/players/:id/edit", playerHandler.ShowEditForm)
	app.Get("/players/import/template", playerHandler.ImportTemplate)
	api.Post("/events", tournamentHandler.Create)
	api.Post("/tournaments", eventHandler.Create)
	api.Delete("/tournaments/:id", eventHandler.Delete)
	api.Post("/tournaments/bulk-delete", eventHandler.DeleteBulk)
	api.Put("/tournaments/:id", eventHandler.Update)
	api.Get("/tournaments/:id/pdf", eventHandler.ExportEventPDF)
	admin.Get("/tournaments/:id/board", eventHandler.AdminBoard)
	admin.Get("/tournaments/:id/health", eventHandler.TournamentHealth)
	admin.Get("/tournaments/:id/health/metrics", eventHandler.TournamentHealthMetrics)
	admin.Get("/tournaments/:id/edit", eventHandler.ShowEditForm)
	api.Post("/matches/create", matchHandler.Create)
	api.Post("/matches/finish", matchHandler.Finish)
	api.Post("/matches/:id/start", matchHandler.Start)
	api.Put("/matches/:id/score", matchHandler.UpdateScore)
	api.Get("/matches/:id/score", matchHandler.ShowScoreForm)
	api.Post("/matches/:id/reset", matchHandler.Reset)
	api.Post("/divisions", divisionHandler.CreateOrUpdate)
	api.Delete("/divisions/:id", divisionHandler.Delete)
	api.Get("/divisions/edit", divisionHandler.ShowEditForm)
	api.Get("/divisions/:id/edit", divisionHandler.ShowEditForm)

	admin.Get("/events/:id", tournamentHandler.Detail)
	api.Put("/events/:id", tournamentHandler.Update)
	api.Delete("/events/:id", tournamentHandler.Delete)
	admin.Post("/events/:id/finish", tournamentHandler.Finish)
	admin.Get("/events/:id/export", tournamentHandler.Export)
	admin.Get("/events/:id/export/pdf", tournamentHandler.ExportPDF)
	admin.Post("/events/:id/move-player", tournamentHandler.MovePlayer)
	admin.Post("/events/:id/regenerate-seeds", tournamentHandler.RegenerateGroupSeeds)
	admin.Post("/events/:id/groups", tournamentHandler.AddGroup)
	admin.Get("/events/:id/edit", tournamentHandler.ShowEditForm)
	admin.Post("/events/:id/officials", tournamentHandler.AddOfficial)
	admin.Delete("/events/:id/officials/:playerId", tournamentHandler.RemoveOfficial)
	admin.Delete("/events/:id/participants/:playerId", tournamentHandler.RemoveParticipant)
	admin.Post("/events/:id/divisions/:divId/knockout/seeds", tournamentHandler.SaveKnockoutSeeds)
	admin.Post("/events/:id/divisions/:divId/start-knockout", tournamentHandler.StartKnockout)
	admin.Post("/events/:id/participants/elo-before", tournamentHandler.UpdateParticipantEloBefore)
	api.Post("/events/:id/teams", tournamentHandler.CreateTeam)

	api.Delete("/events/:id/teams/:teamId", tournamentHandler.DeleteTeam)
	api.Post("/events/:id/teams/:teamId/players", tournamentHandler.AssignPlayerToTeam)
	api.Delete("/events/:id/teams/:teamId/players/:playerId", tournamentHandler.RemovePlayerFromTeam)
	app.Get("/public/events", tournamentHandler.PublicList)
	app.Get("/public/events/:id", tournamentHandler.PublicDetail)
	app.Get("/public/events/:id/tv", tournamentHandler.PublicTVDashboard)
	app.Get("/public/redirect/events/:id", tournamentHandler.PublicRedirectToTournament)
	app.Get("/events/:id/board", tournamentHandler.Board)
	app.Get("/events/:id/board/columns", tournamentHandler.BoardColumns)
	admin.Post("/events/:id/toggle-seeding-lock", tournamentHandler.ToggleSeedingLock)
	admin.Post("/events/:id/recalculate-elo", tournamentHandler.RecalculateElo)

	// Admin-protected additions wired via api (defined above, authMiddleware-guarded).
	assignPlayerToAccountUC := accountApp.NewAssignPlayerToAccountUseCase(playerRepo, accountRepo)
	playerHandler.WithAssignPlayerToAccountUseCase(assignPlayerToAccountUC)
	api.Post("/players/:id/link-account", playerHandler.LinkAccount)
	api.Post("/players/:id/unlink-account", playerHandler.UnlinkAccount)

	getPendingClaimsUC := accountApp.NewGetPendingPlayerClaimsUseCase(playerRepo, accountRepo)
	approveClaimUC := accountApp.NewApprovePlayerClaimUseCase(playerRepo)
	rejectClaimUC := accountApp.NewRejectPlayerClaimUseCase(playerRepo)
	adminHandler.WithClaimReviewUseCases(getPendingClaimsUC, approveClaimUC, rejectClaimUC)
	api.Post("/players/:id/approve-claim", adminHandler.ApproveClaim)
	api.Post("/players/:id/reject-claim", adminHandler.RejectClaim)

	matchHandler.WithConfirmMatchScoreUseCase(confirmMatchScoreUC)
	api.Post("/matches/:id/confirm-proposal", matchHandler.AdminConfirmProposal)

	return app, db, store, nil
}
