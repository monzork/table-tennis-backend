package handler

import (
	"fmt"
	"strings"
	"sync"
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

	var existingTournamentIDs []string
	for _, rawId := range c.Request().PostArgs().PeekMulti("existingTournamentIds[]") {
		existingTournamentIDs = append(existingTournamentIDs, string(rawId))
	}

	e, err := h.createUC.Execute(
		c.Context(), name, divisionID, skipElo, startDate, endDate,
		singlesMen, singlesWomen, doublesMen, doublesWomen, doublesMixed, teamsMen, teamsWomen,
		existingTournamentIDs,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var rowBuf strings.Builder
	if err := c.App().Config().Views.Render(&rowBuf, "admin/partials/event-row", e); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	toastHTML := fmt.Sprintf(`
<div id="toast-container" hx-swap-oob="beforeend">
	<div class="flex items-center gap-3 px-5 py-4 rounded-2xl bg-club-panel border border-white/10 shadow-2xl transition-all duration-500 max-w-sm toast-slide-in pointer-events-auto">
		<svg class="w-5 h-5 text-emerald-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
		<span class="text-xs font-bold uppercase tracking-wider text-white/90">Grand Event "%s" initialized successfully!</span>
	</div>
</div>`, e.Name)

	c.Set("Content-Type", "text/html")
	return c.SendString(rowBuf.String() + toastHTML)
}

func (h *EventHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")
	
	type result struct {
		event     any
		err       error
		divisions any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.event, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}

	return c.Render("admin/event-detail", fiber.Map{
		"Event":     res.event,
		"Divisions": res.divisions,
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

func (h *EventHandler) DeleteBulk(c *fiber.Ctx) error {
	var body struct {
		IDs []string `json:"ids" form:"ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if len(body.IDs) > 0 {
		if err := h.deleteUC.ExecuteBulk(c.Context(), body.IDs); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Redirect", "/admin/events")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect("/admin/events")
}

func (h *EventHandler) PublicDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")
	
	type result struct {
		event     any
		err       error
		divisions any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.event, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}

	return c.Render("public/event-detail", merge(tMap(lang), fiber.Map{
		"Event":     res.event,
		"Divisions": res.divisions,
		"Type":      "Tournaments", // highlight tournaments tab in layout
	}), "layouts/public")
}
