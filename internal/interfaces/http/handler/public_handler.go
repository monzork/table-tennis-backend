package handler

import (
	"context"
	"strings"
	"table-tennis-backend/internal/application/player"
	appTournament "table-tennis-backend/internal/application/tournament"
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
	return "en"
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
		HTTPOnly: false,               // JS can read for future use if needed
		SameSite: "Lax",
	})
	// Redirect back to the page the user was on
	referer := c.Get("Referer")
	if referer == "" {
		referer = "/"
	}
	return c.Redirect(referer, fiber.StatusFound)
}

// tMap builds a fiber.Map with all translated strings under key "T".
func tMap(lang string) fiber.Map {
	m := make(map[string]string)
	for k := range i18n.Translations["en"] {
		m[k] = i18n.T(lang, k)
	}
	return fiber.Map{
		"T":    m,
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
		Honeypot       string `form:"website"` // Honeypot field
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Honeypot check: if 'website' is filled, it's likely a bot
	if body.Honeypot != "" {
		return c.Render("register-success", merge(tMap(lang), fiber.Map{
			"Message": i18n.T(lang, "register.btn"),
		}), "layouts/public")
	}

	_, err := h.registerPlayerUC.Execute(
		context.Background(),
		body.FirstName,
		body.SecondName,
		body.LastName,
		body.SecondLastName,
		body.Birthdate,
		body.Gender,
		body.Country,
		body.Department,
		body.WhatsAppNumber,
		"", // National ID
		500, // Default starting elo
		500,
	)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("register-success", merge(tMap(lang), fiber.Map{
		"Title":   "Success",
		"Message": body.FirstName + ", you are registered! Go to the rankings to see your profile.",
	}), "layouts/public")
}

// ── Tournament self-registration ──────────────────────────────────────────────

// ShowTournamentRegistration lists all open tournaments.
func (h *PublicHandler) ShowTournamentRegistration(c *fiber.Ctx) error {
	lang := getLang(c)
	tournaments, err := h.selfRegisterUC.GetOpenTournaments(context.Background())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Render("tournament-register", merge(tMap(lang), fiber.Map{
		"Title":       i18n.T(lang, "tourney_reg.title"),
		"Tournaments": tournaments,
	}), "layouts/public")
}

// ShowTournamentRegisterForm renders the form for a specific tournament.
func (h *PublicHandler) ShowTournamentRegisterForm(c *fiber.Ctx) error {
	lang := getLang(c)
	tid := c.Params("id")
	tournaments, err := h.selfRegisterUC.GetOpenTournaments(context.Background())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	// Find the specific tournament
	var target interface{}
	for _, t := range tournaments {
		if t.ID == tid {
			target = t
			break
		}
	}
	if target == nil {
		return c.Render("tournament-register", merge(tMap(lang), fiber.Map{
			"Title":       i18n.T(lang, "tourney_reg.title"),
			"Tournaments": tournaments,
			"Error":       i18n.T(lang, "tourney_reg.not_found"),
		}), "layouts/public")
	}
	return c.Render("tournament-register", merge(tMap(lang), fiber.Map{
		"Title":              i18n.T(lang, "tourney_reg.title"),
		"Tournaments":        tournaments,
		"SelectedTournament": target,
		"TournamentID":       tid,
	}), "layouts/public")
}

// RegisterToTournament handles the form submission for tournament self-registration.
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
		Honeypot       string `form:"website"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Bot protection
	if body.Honeypot != "" {
		return c.Render("tournament-register-success", merge(tMap(lang), fiber.Map{
			"Title":   i18n.T(lang, "tourney_reg.success_title"),
			"Message": "Thank you!",
		}), "layouts/public")
	}

	t, playerName, err := h.selfRegisterUC.Execute(
		context.Background(),
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
	)

	if err != nil {
		// Re-render with error
		tournaments, _ := h.selfRegisterUC.GetOpenTournaments(context.Background())
		return c.Render("tournament-register", merge(tMap(lang), fiber.Map{
			"Title":        i18n.T(lang, "tourney_reg.title"),
			"Tournaments":  tournaments,
			"TournamentID": body.TournamentID,
			"Error":        err.Error(),
		}), "layouts/public")
	}

	return c.Render("tournament-register-success", merge(tMap(lang), fiber.Map{
		"Title":          i18n.T(lang, "tourney_reg.success_title"),
		"Message":        i18n.T(lang, "tourney_reg.success_msg"),
		"TournamentName": t.Name,
		"PlayerName":     playerName,
	}), "layouts/public")
}
