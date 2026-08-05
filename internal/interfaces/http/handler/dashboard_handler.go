package handler

import (
	"html/template"

	"table-tennis-backend/internal/application/dashboard"

	"github.com/gofiber/fiber/v2"
)

type DashboardHandler struct {
	getViewUC *dashboard.GetPublicDashboardViewUseCase
}

func NewDashboardHandler(getViewUC *dashboard.GetPublicDashboardViewUseCase) *DashboardHandler {
	return &DashboardHandler{getViewUC: getViewUC}
}

// Public renders the public dashboard: headline counts plus a handful of
// charts built from narrow SQL aggregate queries.
// Route: GET /dashboard
func (h *DashboardHandler) Public(c *fiber.Ctx) error {
	lang := getLang(c)

	view, err := h.getViewUC.Execute(c.Context())
	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render("public/dashboard", merge(tMap(lang), fiber.Map{
		"Stats":            view.Stats,
		"PlayersByCountry": template.HTML(view.PlayersByCountrySVG),
		"EventsByFormat":   template.HTML(view.EventsByFormatSVG),
		"ActivityByMonth":  template.HTML(view.ActivityByMonthSVG),
		"TopGainers":       template.HTML(view.TopGainersSVG),
		"Type":             "Dashboard",
		"CanonicalURL":     c.BaseURL() + c.Path(),
	}), "layouts/public")
}
