package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberws "github.com/gofiber/websocket/v2"
	"table-tennis-backend/internal/interfaces/http/handler"
	"time"
)

func SetupRoutes(app *fiber.App, c *Container, authMiddleware fiber.Handler) {
	// ==========================================
	// HEALTH CHECK
	// ==========================================

	app.Get("/health", func(ctx *fiber.Ctx) error {
		return ctx.JSON(fiber.Map{"status": "ok"})
	})

	// ==========================================
	// PUBLIC ROUTES
	// ==========================================

	// Rankings
	app.Get("/rankings/singles", c.LeaderboardHandler.GetSingles)
	app.Get("/rankings/doubles", c.LeaderboardHandler.GetDoubles)
	app.Get("/rankings/mens/singles", c.LeaderboardHandler.GetMensSingles)
	app.Get("/rankings/womens/singles", c.LeaderboardHandler.GetWomensSingles)
	app.Get("/rankings/mens/doubles", c.LeaderboardHandler.GetMensDoubles)
	app.Get("/rankings/womens/doubles", c.LeaderboardHandler.GetWomensDoubles)
	app.Get("/rankings/mixed/doubles", c.LeaderboardHandler.GetMixedDoubles)

	// Redirect Root to Public Rankings
	app.Get("/", func(ctx *fiber.Ctx) error {
		return ctx.Redirect("/rankings/singles")
	})

	// Public Tournaments List
	app.Get("/tournaments", c.TournamentHandler.PublicList)

	// Public Tournament Self-Registration (must be before /tournaments/:id)
	signupLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return ctx.IP()
		},
	})
	app.Get("/tournaments/register", c.PublicHandler.ShowTournamentRegistration)
	app.Post("/tournaments/register", signupLimiter, c.PublicHandler.RegisterToTournament)

	// Public Detail Views
	app.Get("/tournaments/:id", c.TournamentHandler.PublicDetail)
	app.Get("/tournaments/:id/tv", c.TournamentHandler.PublicTVDashboard)
	app.Get("/events/:id", c.EventHandler.PublicDetail)
	app.Get("/events/:id/tv", c.EventHandler.PublicTVDashboard)

	// User Registration
	app.Get("/register", c.PublicHandler.ShowSignup)
	app.Post("/register", signupLimiter, c.PublicHandler.Register)
	app.Get("/players/department-input", c.PublicHandler.DepartmentInput)

	// Language Switcher
	app.Get("/lang/:locale", c.PublicHandler.SetLang)

	// Sitemap
	app.Get("/sitemap.xml", c.PublicHandler.Sitemap)

	// Public Score Entry & Match Starting Endpoints
	app.Get("/public/matches/score/form", c.MatchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/form", c.MatchHandler.ShowPublicScoreForm)
	app.Post("/public/matches/score/update", c.MatchHandler.UpdatePublicScore)
	app.Post("/public/matches/start", c.MatchHandler.Start)
	app.Post("/public/matches/:id/start", c.MatchHandler.Start)

	// QR-code score entry — shareable per-match URL
	app.Get("/score/:matchId", c.MatchHandler.ShowMatchScorePage)
	app.Get("/score/t/:tournamentId/table/:tableNumber", c.MatchHandler.ShowTableScorePage)
	app.Get("/score/e/:eventId/table/:tableNumber", c.MatchHandler.ShowTableScorePage)
	app.Post("/score/:matchId/verify", c.MatchHandler.ValidateMatchPIN)

	// QR Code generation endpoint
	app.Get("/qr", c.QRHandler.Generate)

	// Admin Score Form (read-only partial — no auth needed, no sensitive data)
	app.Get("/admin/matches/score/form", c.MatchHandler.ShowScoreForm)
	app.Post("/admin/matches/score/form", c.MatchHandler.ShowScoreForm)

	app.Get("/players/import/template", c.PlayerHandler.ImportTemplate) // public template download

	// ==========================================
	// WEBSOCKET ROUTES
	// ==========================================

	app.Use("/ws", func(ctx *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(ctx) {
			// Only allow same-origin or explicitly permitted origins
			origin := ctx.Get("Origin")
			host := ctx.Get("Host")
			if origin != "" && origin != "http://"+host && origin != "https://"+host {
				return ctx.Status(fiber.StatusForbidden).SendString("Cross-origin WebSocket connections not allowed")
			}
			ctx.Locals("allowed", true)
			return ctx.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/brackets/:tournamentId", fiberws.New(handler.WsBracketHandler))

	// ==========================================
	// AUTHENTICATION ROUTES
	// ==========================================

	loginLimiter := limiter.New(limiter.Config{
		Max:        10,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return ctx.IP()
		},
		LimitReached: func(ctx *fiber.Ctx) error {
			return ctx.Status(fiber.StatusTooManyRequests).SendString("Too many login attempts. Please wait a minute.")
		},
	})

	app.Get("/admin/login", c.AuthHandler.ShowLogin)
	app.Post("/admin/login", loginLimiter, c.AuthHandler.Login)
	app.Post("/admin/logout", c.AuthHandler.Logout)

	// ==========================================
	// PROTECTED ADMIN DASHBOARD / VIEW GROUPS
	// ==========================================

	admin := app.Group("/admin")
	admin.Use(authMiddleware)

	admin.Get("/", c.AdminHandler.Dashboard)
	admin.Get("/players", c.AdminHandler.Players)
	admin.Get("/tournaments", c.AdminHandler.Tournaments)
	admin.Get("/events", c.AdminHandler.Events)
	admin.Get("/divisions", c.AdminHandler.Divisions)
	admin.Get("/player-field", c.AdminHandler.NewPlayerField)

	// ==========================================
	// PROTECTED API / FORM ENDPOINTS
	// ==========================================

	api := app.Group("/")
	api.Use(authMiddleware)

	// Players API
	api.Post("/players", c.PlayerHandler.Register)
	api.Get("/players/search", c.PlayerHandler.Search)
	api.Get("/players/search/cards", c.PlayerHandler.SearchSelectionCards)
	api.Get("/players/:id/edit", c.PlayerHandler.ShowEditForm)
	api.Put("/players/:id", c.PlayerHandler.Update)
	api.Delete("/players/:id", c.PlayerHandler.Delete)
	api.Post("/players/import", c.PlayerHandler.Import)

	// Tournaments API
	admin.Get("/tournaments/:id", c.TournamentHandler.Detail)
	admin.Get("/tournaments/:id/board", c.TournamentHandler.Board)
	admin.Get("/tournaments/:id/board/columns", c.TournamentHandler.BoardColumns)
	api.Post("/tournaments", c.TournamentHandler.Create)
	api.Get("/tournaments/:id/edit", c.TournamentHandler.ShowEditForm)
	api.Put("/tournaments/:id", c.TournamentHandler.Update)
	api.Delete("/tournaments/:id", c.TournamentHandler.Delete)
	admin.Post("/tournaments/:id/finish", c.TournamentHandler.Finish)
	admin.Get("/tournaments/:id/export", c.TournamentHandler.Export)
	admin.Get("/tournaments/:id/export/pdf", c.TournamentHandler.ExportPDF)
	admin.Post("/tournaments/:id/teams", c.TournamentHandler.CreateTeam)
	admin.Delete("/tournaments/:id/teams/:teamId", c.TournamentHandler.DeleteTeam)
	admin.Post("/tournaments/:id/teams/:teamId/players", c.TournamentHandler.AssignPlayerToTeam)
	admin.Delete("/tournaments/:id/teams/:teamId/players/:playerId", c.TournamentHandler.RemovePlayerFromTeam)
	admin.Post("/tournaments/:id/officials", c.TournamentHandler.AddOfficial)
	admin.Delete("/tournaments/:id/officials/:playerId", c.TournamentHandler.RemoveOfficial)
	admin.Post("/tournaments/:id/move-player", c.TournamentHandler.MovePlayer)
	admin.Post("/tournaments/:id/regenerate-seeds", c.TournamentHandler.RegenerateGroupSeeds)
	admin.Post("/tournaments/:id/participants/elo-before", c.TournamentHandler.UpdateParticipantEloBefore)

	// Events API
	admin.Get("/events/division-select", c.AdminHandler.DivisionSelect)
	admin.Get("/events/:id", c.EventHandler.Detail)
	admin.Get("/events/:id/board", c.EventHandler.AdminBoard)
	admin.Get("/events/:id/board/columns", c.EventHandler.BoardColumns)
	api.Post("/events", c.EventHandler.Create)
	api.Delete("/events/:id", c.EventHandler.Delete)
	api.Post("/events/bulk-delete", c.EventHandler.DeleteBulk)

	// Matches API
	admin.Post("/matches/score/update", c.MatchHandler.UpdateScore)
	api.Post("/matches/create", c.MatchHandler.Create)
	api.Post("/matches/finish", c.MatchHandler.Finish)
	api.Post("/matches/:id/start", c.MatchHandler.Start)
	api.Post("/matches/:id/reset", c.MatchHandler.Reset)
	api.Get("/matches/:id/score-form", c.MatchHandler.ShowScoreForm)
	api.Put("/matches/:id/score", c.MatchHandler.UpdateScore)

	// Divisions API
	api.Get("/divisions", c.DivisionHandler.ShowEditForm)
	api.Get("/divisions/:id/edit", c.DivisionHandler.ShowEditForm)
	api.Post("/divisions", c.DivisionHandler.CreateOrUpdate)
	api.Delete("/divisions/:id", c.DivisionHandler.Delete)

	// Reports API
	admin.Get("/events/:id/export/pdf", c.EventHandler.ExportEventPDF)
}
