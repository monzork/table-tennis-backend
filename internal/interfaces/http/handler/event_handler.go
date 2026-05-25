package handler

import (
	"fmt"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"

	"github.com/gofiber/fiber/v2"
)

type EventHandler struct {
	createUC      *event.CreateEventUseCase
	getByID       *event.GetEventByIDUseCase
	getAll        *event.GetAllEventsUseCase
	deleteUC      *event.DeleteEventUseCase
	divisionUC    *division.DivisionUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
}

func NewEventHandler(
	createUC *event.CreateEventUseCase,
	getByID *event.GetEventByIDUseCase,
	getAll *event.GetAllEventsUseCase,
	deleteUC *event.DeleteEventUseCase,
	divisionUC *division.DivisionUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
) *EventHandler {
	return &EventHandler{
		createUC:      createUC,
		getByID:       getByID,
		getAll:        getAll,
		deleteUC:      deleteUC,
		divisionUC:    divisionUC,
		leaderboardUC: leaderboardUC,
	}
}

func (h *EventHandler) Create(c *fiber.Ctx) error {
	name := c.FormValue("name")
	skipElo := c.FormValue("skipElo") == "on"
	divisionID := c.FormValue("divisionId")
	if skipElo {
		divisionID = "none"
	}
	startDate := c.FormValue("startDate")
	endDate := c.FormValue("endDate")

	parseCategoryConfig := func(catKey string, defaultFormat string) event.CategoryConfig {
		auto := c.FormValue("auto" + catKey) == "on"
		format := c.FormValue("format" + catKey)
		if format == "" {
			format = c.FormValue("format")
			if format == "" {
				format = defaultFormat
			}
		}
		passCount := 2
		fmt.Sscanf(c.FormValue("groupPassCount" + catKey), "%d", &passCount)
		
		var ids []string
		for _, rawId := range c.Request().PostArgs().PeekMulti("participantIds" + catKey + "[]") {
			ids = append(ids, string(rawId))
		}
		
		// Fallback to global player pool if specific category pool is empty
		if len(ids) == 0 {
			for _, rawId := range c.Request().PostArgs().PeekMulti("participant_ids[]") {
				ids = append(ids, string(rawId))
			}
		}
		
		return event.CategoryConfig{
			Auto:           auto,
			Format:         format,
			GroupPassCount: passCount,
			PlayerIDs:      ids,
		}
	}

	singlesMen := parseCategoryConfig("SinglesMen", "groups_elimination")
	singlesWomen := parseCategoryConfig("SinglesWomen", "groups_elimination")
	doublesMen := parseCategoryConfig("DoublesMen", "elimination")
	doublesWomen := parseCategoryConfig("DoublesWomen", "elimination")
	doublesMixed := parseCategoryConfig("DoublesMixed", "elimination")
	teamsMen := parseCategoryConfig("TeamsMen", "round_robin")
	teamsWomen := parseCategoryConfig("TeamsWomen", "round_robin")

	e, err := h.createUC.Execute(
		c.Context(), name, divisionID, skipElo, startDate, endDate,
		singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("HX-Trigger", "eventCreated")
	return c.Render("admin/partials/event-row", e)
}

func (h *EventHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")
	e, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	divisions, _ := h.divisionUC.GetAll(c.Context())

	return c.Render("admin/event-detail", fiber.Map{
		"Event":     e,
		"Divisions": divisions,
	}, "layouts/admin")
}

func (h *EventHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if c.Get("HX-Request") != "" {
		if c.Get("HX-Current-URL") != "" && fmt.Sprintf("/admin/events/%s", id) == c.Get("HX-Current-URL") {
			c.Set("HX-Redirect", "/admin/events")
		}
		return c.SendString("")
	}
	return c.SendString("")
}
