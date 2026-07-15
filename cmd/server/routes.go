package main

import (
	"table-tennis-backend/internal/interfaces/http/handler"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberws "github.com/gofiber/websocket/v2"
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

	// Public Events List
	app.Get("/events", c.EventHandler.PublicList)

	// Public Event Self-Registration (must be before /events/:id)
	/* signupLimiter := limiter.New(limiter.Config{
		Max:        5,
		Expiration: 1 * time.Minute,
		KeyGenerator: func(ctx *fiber.Ctx) string {
			return ctx.IP()
		},
	}) */
	// Disabled: self-registration
	// app.Get("/events/register", c.PublicHandler.ShowTournamentRegistration)
	// app.Post("/events/register", signupLimiter, c.PublicHandler.RegisterToTournament)

	// Public Detail Views
	app.Get("/events/:id", c.EventHandler.PublicDetail)
	app.Get("/events/:id/tv", c.EventHandler.PublicTVDashboard)
	app.Get("/tournaments/:id", c.TournamentHandler.PublicDetail)
	app.Get("/tournaments/:id/tv", c.TournamentHandler.PublicTVDashboard)

	// User Registration
	// Disabled: self-registration
	// app.Get("/register", c.PublicHandler.ShowSignup)
	// app.Post("/register", signupLimiter, c.PublicHandler.Register)
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

	// QR Code generation endpoint
	app.Get("/qr", c.QRHandler.Generate)

	// Admin Score Form (read-only partial — no auth needed, no sensitive data)
	app.Get("/admin/matches/score/form", c.MatchHandler.ShowScoreForm)
	app.Post("/admin/matches/score/form", c.MatchHandler.ShowScoreForm)

	app.Get("/players/import/template", c.PlayerHandler.ImportTemplate) // public template download

	// Push Notifications
	app.Get("/api/vapid-public-key", func(ctx *fiber.Ctx) error {
		return ctx.SendString(c.NotificationHandler.GetVAPIDPublicKey())
	})
	app.Post("/api/notifications/subscribe", c.NotificationHandler.Subscribe)

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
	admin.Get("/events", c.AdminHandler.Events)
	admin.Get("/tournaments", c.AdminHandler.Tournaments)
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

	// Events API
	admin.Get("/events/:id", c.EventHandler.Detail)
	admin.Get("/events/:id/board", c.EventHandler.Board)
	admin.Get("/events/:id/board/columns", c.EventHandler.BoardColumns)
	api.Post("/events", c.EventHandler.Create)
	api.Get("/events/:id/edit", c.EventHandler.ShowEditForm)
	api.Put("/events/:id", c.EventHandler.Update)
	api.Delete("/events/:id", c.EventHandler.Delete)
	admin.Post("/events/:id/finish", c.EventHandler.Finish)
	admin.Get("/events/:id/export", c.EventHandler.Export)
	admin.Get("/events/:id/export/pdf", c.EventHandler.ExportPDF)
	admin.Post("/events/:id/teams", c.EventHandler.CreateTeam)
	admin.Delete("/events/:id/teams/:teamId", c.EventHandler.DeleteTeam)
	admin.Post("/events/:id/teams/:teamId/players", c.EventHandler.AssignPlayerToTeam)
	admin.Delete("/events/:id/teams/:teamId/players/:playerId", c.EventHandler.RemovePlayerFromTeam)
	admin.Post("/events/:id/groups", c.EventHandler.AddGroup)
	admin.Post("/events/:id/officials", c.EventHandler.AddOfficial)
	admin.Delete("/events/:id/officials/:playerId", c.EventHandler.RemoveOfficial)
	admin.Post("/events/:id/move-player", c.EventHandler.MovePlayer)
	admin.Post("/events/:id/save-knockout-seeds", c.EventHandler.SaveKnockoutSeeds)
	admin.Post("/events/:id/toggle-seeding-lock", c.EventHandler.ToggleSeedingLock)
	admin.Post("/events/:id/divisions/:divId/start-knockout", c.EventHandler.StartKnockout)
	admin.Post("/events/:id/regenerate-seeds", c.EventHandler.RegenerateGroupSeeds)
	admin.Post("/events/:id/participants/elo-before", c.EventHandler.UpdateParticipantEloBefore)
	admin.Post("/events/:id/recalculate-elo", c.EventHandler.RecalculateElo)
	api.Delete("/events/:id/participants/:playerId", c.EventHandler.RemoveParticipant)

	// Tournaments API
	admin.Get("/tournaments/division-select", c.AdminHandler.DivisionSelect)
	admin.Get("/tournaments/:id", c.TournamentHandler.Detail)
	admin.Get("/tournaments/:id/board", c.TournamentHandler.AdminBoard)
	admin.Get("/tournaments/:id/board/columns", c.TournamentHandler.BoardColumns)
	admin.Get("/tournaments/:id/edit", c.TournamentHandler.ShowEditForm)
	api.Post("/tournaments", c.TournamentHandler.Create)
	api.Put("/tournaments/:id", c.TournamentHandler.Update)
	api.Delete("/tournaments/:id", c.TournamentHandler.Delete)
	api.Post("/tournaments/bulk-delete", c.TournamentHandler.DeleteBulk)

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
	admin.Get("/tournaments/:id/export/pdf", c.TournamentHandler.ExportEventPDF)
	admin.Post("/notifications/broadcast", c.NotificationHandler.Broadcast)
}
