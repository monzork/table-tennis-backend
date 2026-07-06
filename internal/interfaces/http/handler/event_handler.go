package handler

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	eventDomain "table-tennis-backend/internal/domain/event"

	"github.com/gofiber/fiber/v2"
)

type EventHandler struct {
	createUC      *event.CreateEventUseCase
	updateUC      *event.UpdateEventUseCase
	getByID       *event.GetEventByIDUseCase
	getAll        *event.GetAllEventsUseCase
	deleteUC      *event.DeleteEventUseCase
	divisionUC    *division.DivisionUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
	exportPdfUC   *tournament.ExportEventPdfUseCase
}

func NewEventHandler(
	createUC *event.CreateEventUseCase,
	updateUC *event.UpdateEventUseCase,
	getByID *event.GetEventByIDUseCase,
	getAll *event.GetAllEventsUseCase,
	deleteUC *event.DeleteEventUseCase,
	divisionUC *division.DivisionUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	exportPdfUC *tournament.ExportEventPdfUseCase,
) *EventHandler {
	return &EventHandler{
		createUC:      createUC,
		updateUC:      updateUC,
		getByID:       getByID,
		getAll:        getAll,
		deleteUC:      deleteUC,
		divisionUC:    divisionUC,
		leaderboardUC: leaderboardUC,
		exportPdfUC:   exportPdfUC,
	}
}

func (h *EventHandler) Create(c *fiber.Ctx) error {
	name := c.FormValue("name")
	skipElo := c.FormValue("skipElo") == "on"
	var divisionIDs []string
	for _, rawId := range c.Request().PostArgs().PeekMulti("divisionIds[]") {
		divisionIDs = append(divisionIDs, string(rawId))
	}
	if len(divisionIDs) == 0 {
		if divID := c.FormValue("divisionId"); divID != "" {
			divisionIDs = append(divisionIDs, divID)
		}
	}
	if skipElo {
		divisionIDs = []string{"none"}
	}
	startDate := c.FormValue("startDate")
	endDate := c.FormValue("endDate")

	parseCategoryConfig := func(catKey string, defaultFormat string) event.CategoryConfig {
		auto := c.FormValue("auto"+catKey) == "on"
		format := c.FormValue("format" + catKey)
		if format == "" {
			format = c.FormValue("format")
			if format == "" {
				format = defaultFormat
			}
		}
		passCount := 2
		fmt.Sscanf(c.FormValue("groupPassCount"+catKey), "%d", &passCount)

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
		c.Context(), name, divisionIDs, skipElo, startDate, endDate,
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
<td style="display:none;">
	<div id="toast-container" hx-swap-oob="beforeend">
		<div class="flex items-center gap-3 px-5 py-4 rounded-2xl bg-club-panel border border-white/10 shadow-2xl transition-all duration-500 max-w-sm toast-slide-in pointer-events-auto">
			<svg class="w-5 h-5 text-emerald-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
			<span class="text-xs font-bold uppercase tracking-wider text-white/90">Grand Event "%s" initialized successfully!</span>
		</div>
	</div>
</td>`, e.Name)

	rowStr := rowBuf.String()
	if idx := strings.LastIndex(rowStr, "</tr>"); idx != -1 {
		rowStr = rowStr[:idx] + toastHTML + rowStr[idx:]
	} else {
		rowStr = rowStr + toastHTML
	}

	c.Set("Content-Type", "text/html")
	return c.SendString(rowStr)
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
		"Event":        res.event,
		"Divisions":    res.divisions,
		"Type":         "Tournaments", // highlight tournaments tab in layout
		"OGImage":      c.BaseURL() + "/open_tdm.jpeg",
		"CanonicalURL": c.BaseURL() + c.Path(),
	}), "layouts/public")
}

func (h *EventHandler) ExportEventPDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pdfBytes, err := h.exportPdfUC.Execute(c.Context(), idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_report_%s.pdf\"", idStr))
	return c.Send(pdfBytes)
}

func (h *EventHandler) getBoardData(c *fiber.Ctx, eventID string) (*eventDomain.Event, []*divisionDomain.Division, []BoardCard, []BoardCard, []BoardCard, error) {
	ctx := c.Context()
	
	// Ensure deep fetch of the event so all tournaments and matches are loaded.
	// Actually, getByID uses GetByIDDeep behind the scenes which includes tournaments and matches.
	e, err := h.getByID.Execute(ctx, eventID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	
	divs, _ := h.divisionUC.GetAll(ctx)
	
	var globalScheduled []BoardCard
	var inProgress []BoardCard
	var finished []BoardCard

	// Global tracker for player activity
	lastActivity := make(map[string]time.Time)

	for _, t := range e.Tournaments {
		s, i, f := BuildBoardCards(t, divs)
		// Inject TournamentName into the cards for display
		for idx := range s {
			s[idx].TournamentName = t.Name
			s[idx].TournamentID = t.ID
		}
		for idx := range i {
			i[idx].TournamentName = t.Name
			i[idx].TournamentID = t.ID
		}
		for idx := range f {
			f[idx].TournamentName = t.Name
			f[idx].TournamentID = t.ID
		}
		
		globalScheduled = append(globalScheduled, s...)
		inProgress = append(inProgress, i...)
		finished = append(finished, f...)

		// Track last activity across all tournaments in this event
		for _, m := range t.Matches {
			if m.Status == "in_progress" || m.Status == "finished" {
				tAct := time.Time{}
				if m.UpdatedAt != nil {
					tAct = *m.UpdatedAt
				}
				for _, p := range m.TeamA {
					if lastActivity[p.ID].Before(tAct) {
						lastActivity[p.ID] = tAct
					}
				}
				for _, p := range m.TeamB {
					if lastActivity[p.ID].Before(tAct) {
						lastActivity[p.ID] = tAct
					}
				}
			}
		}
	}

	var reordered []BoardCard
	var unstarted []BoardCard
	var virtualScheduled []BoardCard

	for _, c := range globalScheduled {
		if c.MatchID == "" {
			virtualScheduled = append(virtualScheduled, c)
		} else {
			unstarted = append(unstarted, c)
		}
	}

	simClock := time.Now().Add(24 * time.Hour)
	
	scheduleMatchGreedy := func(pool *[]BoardCard) {
		for len(*pool) > 0 {
			bestIdx := -1
			var bestPenalty time.Time
			var bestSum int64

			for i, c := range *pool {
				t1 := lastActivity[c.P1Id]
				t2 := lastActivity[c.P2Id]
				penalty := t1
				if t2.After(t1) {
					penalty = t2
				}
				sum := t1.UnixNano() + t2.UnixNano()

				if bestIdx == -1 || penalty.Before(bestPenalty) || (penalty.Equal(bestPenalty) && sum < bestSum) {
					bestIdx = i
					bestPenalty = penalty
					bestSum = sum
				}
			}
			
			bestCard := (*pool)[bestIdx]
			reordered = append(reordered, bestCard)
			lastActivity[bestCard.P1Id] = simClock
			lastActivity[bestCard.P2Id] = simClock
			simClock = simClock.Add(time.Second)
			*pool = append((*pool)[:bestIdx], (*pool)[bestIdx+1:]...)
		}
	}

	scheduleMatchGreedy(&unstarted)
	scheduleMatchGreedy(&virtualScheduled)

	return e, divs, reordered, inProgress, finished, nil
}

func (h *EventHandler) AdminBoard(c *fiber.Ctx) error {
	lang := getLang(c)
	e, _, scheduled, inProgress, finished, err := h.getBoardData(c, c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Build AllDivisions from the board cards (same approach as tournament Board handler)
	uniqueDivsMap := make(map[string]bool)
	for _, card := range scheduled {
		if card.DivisionName != "" {
			uniqueDivsMap[card.DivisionName] = true
		}
	}
	for _, card := range inProgress {
		if card.DivisionName != "" {
			uniqueDivsMap[card.DivisionName] = true
		}
	}
	for _, card := range finished {
		if card.DivisionName != "" {
			uniqueDivsMap[card.DivisionName] = true
		}
	}
	var allDivs []string
	for d := range uniqueDivsMap {
		allDivs = append(allDivs, d)
	}

	// Parse filter params
	q := strings.ToLower(c.Query("q"))
	var selectedDivs []string
	for _, d := range c.Request().URI().QueryArgs().PeekMulti("div") {
		selectedDivs = append(selectedDivs, string(d))
	}
	if c.Query("q") != "" || len(selectedDivs) > 0 {
		scheduled = FilterBoardCards(scheduled, q, selectedDivs)
		inProgress = FilterBoardCards(inProgress, q, selectedDivs)
		finished = FilterBoardCards(finished, q, selectedDivs)
	}
	selectedDivsMap := make(map[string]bool)
	for _, d := range selectedDivs {
		selectedDivsMap[d] = true
	}

	// Build tables from event NumTables
	tables := buildEventTables(e, inProgress)

	return c.Render("admin/event-board", merge(tMap(lang), fiber.Map{
		"Event":        e,
		"Scheduled":    scheduled,
		"InProgress":   inProgress,
		"Finished":     finished,
		"AllDivisions": allDivs,
		"QueryQ":       c.Query("q"),
		"SelectedDivs": selectedDivsMap,
		"Tables":       tables,
	}), "layouts/admin")
}

func (h *EventHandler) PublicTVDashboard(c *fiber.Ctx) error {
	lang := getLang(c)
	e, _, scheduled, inProgress, finished, err := h.getBoardData(c, c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("public/event-board", merge(tMap(lang), fiber.Map{
		"Event":      e,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
	}))
}

func (h *EventHandler) BoardColumns(c *fiber.Ctx) error {
	lang := getLang(c)
	e, _, scheduled, inProgress, finished, err := h.getBoardData(c, c.Params("id"))
	if err != nil {
		return c.SendString("<div class='text-red-400'>Error loading board</div>")
	}

	q := strings.ToLower(c.Query("q"))
	var selectedDivs []string
	for _, d := range c.Request().URI().QueryArgs().PeekMulti("div") {
		selectedDivs = append(selectedDivs, string(d))
	}

	if c.Query("q") != "" || len(selectedDivs) > 0 {
		scheduled = FilterBoardCards(scheduled, q, selectedDivs)
		inProgress = FilterBoardCards(inProgress, q, selectedDivs)
		finished = FilterBoardCards(finished, q, selectedDivs)
	}

	// Build tables from event NumTables
	tables := buildEventTables(e, inProgress)

	return c.Render("admin/partials/event-board-columns", fiber.Map{
		"Event":      e,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
		"T":          tMap(lang)["T"],
	})
}

// buildEventTables creates a TableVM slice from an event's NumTables + occupied tables.
func buildEventTables(e *eventDomain.Event, inProgress []BoardCard) []TableVM {
	var tables []TableVM
	if e == nil {
		return tables
	}

	numTables := e.NumTables
	for _, t := range e.Tournaments {
		if t.NumTables > numTables {
			numTables = t.NumTables
		}
	}

	if numTables <= 0 {
		return tables
	}
	
	usedTables := make(map[int]bool)
	for _, match := range inProgress {
		if match.TableNumber != nil {
			usedTables[*match.TableNumber] = true
		}
	}
	
	for i := 1; i <= numTables; i++ {
		tables = append(tables, TableVM{
			Number: i,
			IsUsed: usedTables[i],
		})
	}
	return tables
}

func (h *EventHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	e, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	return c.Render("admin/partials/event-edit-form", fiber.Map{
		"Event": e,
	})
}

func (h *EventHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	name := c.FormValue("name")
	startDate := c.FormValue("startDate")
	endDate := c.FormValue("endDate")
	var numTables int
	fmt.Sscanf(c.FormValue("numTables"), "%d", &numTables)

	e, err := h.updateUC.Execute(c.Context(), id, name, startDate, endDate, numTables)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "event-updated")
		return c.Render("admin/partials/event-edit-form", fiber.Map{
			"Event":   e,
			"Success": true,
		})
	}
	return c.Redirect("/admin/events/" + id)
}
