package handler

import (
	"fmt"
	"strings"
	appTournament "table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
)

type PublicHandler struct {
	registerPlayerUC *player.RegisterPlayerUseCase
	selfRegisterUC   *appTournament.SelfRegisterUseCase
}

func NewPublicHandler(
	uc *player.RegisterPlayerUseCase,
	selfRegUC *appTournament.SelfRegisterUseCase,
) *PublicHandler {
	return &PublicHandler{
		registerPlayerUC: uc,
		selfRegisterUC:   selfRegUC,
	}
}

// ── Language helpers ──────────────────────────────────────────────────────────

// getLang reads the preferred language exclusively from the cookie.
// Defaults to "en" if no valid cookie is present.
func getLang(c *fiber.Ctx) string {
	if l := c.Cookies("lang"); l == "es" || l == "en" {
		return l
	}
	return "es"
}

// SetLang writes the chosen locale to a 1-year cookie and redirects the user
// back to wherever they came from (Referer header) or to "/" as a fallback.
// Route: GET /lang/:locale   (e.g. /lang/es  or  /lang/en)
func (h *PublicHandler) SetLang(c *fiber.Ctx) error {
	locale := c.Params("locale")
	if locale != "es" && locale != "en" {
		locale = "en"
	}
	c.Cookie(&fiber.Cookie{
		Name:     "lang",
		Value:    locale,
		MaxAge:   60 * 60 * 24 * 365, // 1 year
		HTTPOnly: false,              // JS can read for future use if needed
		SameSite: "Lax",
	})
	// Redirect back to the page the user was on
	referer := c.Query("returnTo")
	if referer == "" {
		referer = c.Get("HX-Current-URL")
	}
	if referer == "" {
		referer = c.Get("Referer")
	}
	if referer == "" {
		referer = "/"
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Redirect", referer)
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(referer, fiber.StatusFound)
}

// tMap builds a fiber.Map with all translated strings under key "T".
func tMap(lang string) fiber.Map {
	return fiber.Map{
		"T":    i18n.PrecomputedMaps[lang],
		"Lang": lang,
	}
}

// merge adds extra keys into a fiber.Map (shallow).
func merge(base fiber.Map, extra fiber.Map) fiber.Map {
	for k, v := range extra {
		base[k] = v
	}
	return base
}

// ── Player self-signup ────────────────────────────────────────────────────────

var NicaraguaDepartments = []string{
	"Boaco", "Carazo", "Chinandega", "Chontales", "Estelí",
	"Granada", "Jinotega", "León", "Madriz", "Managua",
	"Masaya", "Matagalpa", "Nueva Segovia", "Rivas", "Río San Juan",
	"RACCN", "RACCS",
}

func (h *PublicHandler) DepartmentInput(c *fiber.Ctx) error {
	country := strings.TrimSpace(strings.ToUpper(c.Query("country")))
	currentDept := c.Query("currentDepartment")
	theme := c.Query("theme") // "register", "admin-add", "admin-edit"

	isNicaragua := country == "NIC" || country == "NICARAGUA" || country == "NI"

	return c.Render("partials/department-input", fiber.Map{
		"IsNicaragua":          isNicaragua,
		"NicaraguaDepartments": NicaraguaDepartments,
		"Department":           currentDept,
		"Theme":                theme,
	})
}
func (h *PublicHandler) ShowSignup(c *fiber.Ctx) error {
	lang := getLang(c)
	return c.Render("register", merge(tMap(lang), fiber.Map{
		"Title": i18n.T(lang, "register.title"),
	}), "layouts/public")
}

func (h *PublicHandler) Register(c *fiber.Ctx) error {
	lang := getLang(c)
	var body struct {
		FirstName      string `form:"firstName"`
		SecondName     string `form:"secondName"`
		LastName       string `form:"lastName"`
		SecondLastName string `form:"secondLastName"`
		Birthdate      string `form:"birthdate"`
		Country        string `form:"country"`
		Department     string `form:"department"`
		Gender         string `form:"gender"`
		WhatsAppNumber string `form:"whatsAppNumber"`
		NationalID     string `form:"nationalId"`
		Honeypot       string `form:"website"` // Honeypot field
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	// Honeypot check: if 'website' is filled, it's likely a bot
	if body.Honeypot != "" {
		return c.Render("register-success", merge(tMap(lang), fiber.Map{
			"Message": i18n.T(lang, "register.btn"),
		}), "layouts/public")
	}

	_, err := h.registerPlayerUC.Execute(
		c.Context(),
		body.FirstName,
		body.SecondName,
		body.LastName,
		body.SecondLastName,
		body.Birthdate,
		body.Gender,
		body.Country,
		body.Department,
		body.WhatsAppNumber,
		body.NationalID,
		500, // Default starting elo
		500,
	)

	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render("register-success", merge(tMap(lang), fiber.Map{
		"Title":   "Success",
		"Message": body.FirstName + ", you are registered! Go to the rankings to see your profile.",
	}), "layouts/public")
}

// ── Event self-registration ──────────────────────────────────────────────

// ShowTournamentRegistration lists all open events.
func (h *PublicHandler) ShowTournamentRegistration(c *fiber.Ctx) error {
	lang := getLang(c)
	events, err := h.selfRegisterUC.GetOpenTournaments(c.Context())
	if err != nil {
		return ErrorHandler(err)
	}
	return c.Render("event-register", merge(tMap(lang), fiber.Map{
		"Title":  i18n.T(lang, "tourney_reg.title"),
		"Events": events,
	}), "layouts/public")
}

// ShowTournamentRegisterForm renders the form for a specific event.
func (h *PublicHandler) ShowTournamentRegisterForm(c *fiber.Ctx) error {
	lang := getLang(c)
	tid := c.Params("id")
	events, err := h.selfRegisterUC.GetOpenTournaments(c.Context())
	if err != nil {
		return ErrorHandler(err)
	}
	// Find the specific event
	var target interface{}
	for _, t := range events {
		if t.ID == tid {
			target = t
			break
		}
	}
	if target == nil {
		return c.Render("event-register", merge(tMap(lang), fiber.Map{
			"Title":  i18n.T(lang, "tourney_reg.title"),
			"Events": events,
			"Error":  i18n.T(lang, "tourney_reg.not_found"),
		}), "layouts/public")
	}
	return c.Render("event-register", merge(tMap(lang), fiber.Map{
		"Title":              i18n.T(lang, "tourney_reg.title"),
		"Events":             events,
		"SelectedTournament": target,
		"TournamentID":       tid,
	}), "layouts/public")
}

// RegisterToTournament handles the form submission for event self-registration.
func (h *PublicHandler) RegisterToTournament(c *fiber.Ctx) error {
	lang := getLang(c)
	var body struct {
		TournamentID   string `form:"tournamentId"`
		FirstName      string `form:"firstName"`
		SecondName     string `form:"secondName"`
		LastName       string `form:"lastName"`
		SecondLastName string `form:"secondLastName"`
		Country        string `form:"country"`
		Department     string `form:"department"`
		WhatsAppNumber string `form:"whatsAppNumber"`
		Birthdate      string `form:"birthdate"`
		Gender         string `form:"gender"`
		NationalID     string `form:"nationalId"`
		Honeypot       string `form:"website"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	// Bot protection
	if body.Honeypot != "" {
		return c.Render("event-register-success", merge(tMap(lang), fiber.Map{
			"Title":   i18n.T(lang, "tourney_reg.success_title"),
			"Message": "Thank you!",
		}), "layouts/public")
	}

	t, playerName, err := h.selfRegisterUC.Execute(
		c.Context(),
		body.TournamentID,
		body.FirstName,
		body.SecondName,
		body.LastName,
		body.SecondLastName,
		body.Country,
		body.Department,
		body.WhatsAppNumber,
		body.Birthdate,
		body.Gender,
		body.NationalID,
	)

	if err != nil {
		// Re-render with error
		events, _ := h.selfRegisterUC.GetOpenTournaments(c.Context())
		return c.Render("event-register", merge(tMap(lang), fiber.Map{
			"Title":        i18n.T(lang, "tourney_reg.title"),
			"Events":       events,
			"TournamentID": body.TournamentID,
			"Error":        err.Error(),
		}), "layouts/public")
	}

	return c.Render("event-register-success", merge(tMap(lang), fiber.Map{
		"Title":          i18n.T(lang, "tourney_reg.success_title"),
		"Message":        i18n.T(lang, "tourney_reg.success_msg"),
		"TournamentName": t.Name,
		"PlayerName":     playerName,
	}), "layouts/public")
}

// Sitemap generates a dynamic XML sitemap.
func (h *PublicHandler) Sitemap(c *fiber.Ctx) error {
	baseURL := c.BaseURL()
	events, _ := h.selfRegisterUC.GetOpenTournaments(c.Context()) // we can use GetOpenTournaments or ideally all events if available. Actually, just public ones.

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>`)
	sb.WriteString("\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	sb.WriteString("\n")

	// Static routes
	staticRoutes := []string{
		"/rankings/singles",
		"/events",
		"/register",
	}
	for _, route := range staticRoutes {
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s%s</loc>\n", baseURL, route))
		sb.WriteString("    <changefreq>daily</changefreq>\n")
		sb.WriteString("    <priority>0.8</priority>\n")
		sb.WriteString("  </url>\n")
	}

	// Dynamic routes (Events)
	for _, t := range events {
		sb.WriteString("  <url>\n")
		sb.WriteString(fmt.Sprintf("    <loc>%s/events/%s</loc>\n", baseURL, t.ID))
		sb.WriteString("    <changefreq>hourly</changefreq>\n")
		sb.WriteString("    <priority>1.0</priority>\n")
		sb.WriteString("  </url>\n")
	}

	sb.WriteString(`</urlset>`)

	c.Set("Content-Type", "application/xml")
	return c.SendString(sb.String())
}
