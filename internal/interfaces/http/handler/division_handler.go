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

	return c.Redirect("/admin/divisions")
}

func (h *DivisionHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	err := h.uc.Delete(c.Context(), id)
	if err != nil {
		return c.Status(500).SendString(err.Error())
	}

	// Because we might be hitting this with a form doing a POST with _method=DELETE
	return c.Redirect("/admin/divisions")
}
