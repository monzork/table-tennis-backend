package handler

import (
	"strconv"
	"strings"

	"table-tennis-backend/internal/application/division"

	"github.com/gofiber/fiber/v2"
)

type DivisionHandler struct {
	uc *division.DivisionUseCase
}

func NewDivisionHandler(uc *division.DivisionUseCase) *DivisionHandler {
	return &DivisionHandler{uc: uc}
}

func (h *DivisionHandler) CreateOrUpdate(c *fiber.Ctx) error {
	id := c.FormValue("id")
	name := c.FormValue("name")
	
	order, _ := strconv.Atoi(c.FormValue("display_order"))
	minElo, _ := strconv.Atoi(c.FormValue("min_elo"))
	
	maxEloStr := c.FormValue("max_elo")
	var maxElo *int16
	if strings.TrimSpace(maxEloStr) != "" {
		val, err := strconv.Atoi(maxEloStr)
		if err == nil {
			v16 := int16(val)
			maxElo = &v16
		}
	}

	color := c.FormValue("color")
	if color == "" {
		color = "#ffffff"
	}

	err := h.uc.Save(c.Context(), id, name, order, int16(minElo), maxElo, "both", color)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}

	// For HTMX, return the row or the whole list. Let's try to get the one we just saved.
	// We don't have a direct "Return object from Save" yet, so we fetch if we have an ID or get all if new.
	// Actually, if it's new, we might not know the ID easily without refactoring UseCase.
	// Let's just return a redirect or a signal to reload if new, or the partial if update.
	// Better: just fetch all and return rows for simplicity.
	
	if id != "" {
		d, _ := h.uc.GetById(c.Context(), id)
		return c.Render("admin/partials/division-row", d)
	}

	// For new ones, it's easier to just redirect for now or return all rows.
	// Let's return all rows.
	divisions, _ := h.uc.GetAll(c.Context())
	return c.Render("admin/divisions", fiber.Map{"Divisions": divisions}, "layouts/admin")
}

func (h *DivisionHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.uc.Delete(c.Context(), id)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *DivisionHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		// New division
		return c.Render("admin/partials/division-edit-form", fiber.Map{})
	}
	d, err := h.uc.GetById(c.Context(), id)
	if err != nil {
		return c.Status(404).SendString("Division not found")
	}
	return c.Render("admin/partials/division-edit-form", d)
}
