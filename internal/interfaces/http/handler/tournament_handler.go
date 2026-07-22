package handler

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	domainEvent "table-tennis-backend/internal/domain/event"
	eventDomain "table-tennis-backend/internal/domain/tournament"

	"github.com/gofiber/fiber/v2"
)

type TournamentHandler struct {
	createUC      *tournament.CreateEventUseCase
	updateUC      *tournament.UpdateEventUseCase
	getByID       *tournament.GetEventByIDUseCase
	getAll        *tournament.GetAllEventsUseCase
	deleteUC      *tournament.DeleteEventUseCase
	divisionUC    *division.DivisionUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
	exportPdfUC   *event.ExportEventPdfUseCase
	getBoardUC    *tournament.GetBoardDataUseCase
}

func NewTournamentHandler(
	createUC *tournament.CreateEventUseCase,
	updateUC *tournament.UpdateEventUseCase,
	getByID *tournament.GetEventByIDUseCase,
	getAll *tournament.GetAllEventsUseCase,
	deleteUC *tournament.DeleteEventUseCase,
	divisionUC *division.DivisionUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	exportPdfUC *event.ExportEventPdfUseCase,
	getBoardUC *tournament.GetBoardDataUseCase,
) *TournamentHandler {
	return &TournamentHandler{
		createUC:      createUC,
		updateUC:      updateUC,
		getByID:       getByID,
		getAll:        getAll,
		deleteUC:      deleteUC,
		divisionUC:    divisionUC,
		leaderboardUC: leaderboardUC,
		exportPdfUC:   exportPdfUC,
		getBoardUC:    getBoardUC,
	}
}

func (h *TournamentHandler) Create(c *fiber.Ctx) error {
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

	parseCategoryConfig := func(catKey string, defaultFormat string) tournament.CategoryConfig {
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

		return tournament.CategoryConfig{
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
		return ErrorHandler(err)
	}

	var rowBuf strings.Builder
	if err := c.App().Config().Views.Render(&rowBuf, "admin/partials/tournament-row", e); err != nil {
		return ErrorHandler(err)
	}

	toastHTML := fmt.Sprintf(`
<td style="display:none;">
	<div id="toast-container" hx-swap-oob="beforeend">
		<div class="flex items-center gap-3 px-5 py-4 rounded-2xl bg-club-panel border border-white/10 shadow-2xl transition-all duration-500 max-w-sm toast-slide-in pointer-tournaments-auto">
			<svg class="w-5 h-5 text-emerald-400 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
			<span class="text-xs font-bold uppercase tracking-wider text-white/90">Grand Tournament "%s" initialized successfully!</span>
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

func (h *TournamentHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		tournament any
		err        error
		divisions  any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return ErrorHandler(res.err)
	}

	return c.Render("admin/tournament-detail", fiber.Map{
		"Tournament": res.tournament,
		"Divisions":  res.divisions,
	}, "layouts/admin")
}

func (h *TournamentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		if c.Get("HX-Current-URL") != "" && fmt.Sprintf("/admin/tournaments/%s", id) == c.Get("HX-Current-URL") {
			c.Set("HX-Redirect", "/admin/tournaments")
		}
		return c.SendString("")
	}
	return c.SendString("")
}

func (h *TournamentHandler) DeleteBulk(c *fiber.Ctx) error {
	var body struct {
		IDs []string `json:"ids" form:"ids"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	if len(body.IDs) > 0 {
		if err := h.deleteUC.ExecuteBulk(c.Context(), body.IDs); err != nil {
			return ErrorHandler(err)
		}
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Redirect", "/admin/tournaments")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect("/admin/tournaments")
}

func (h *TournamentHandler) PublicDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")

	type result struct {
		tournament any
		err        error
		divisions  any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return ErrorHandler(res.err)
	}

	return c.Render("public/tournament-detail", merge(tMap(lang), fiber.Map{
		"Tournament":   res.tournament,
		"Divisions":    res.divisions,
		"Type":         "Events", // highlight events tab in layout
		"OGImage":      c.BaseURL() + "/open_tdm.jpeg",
		"CanonicalURL": c.BaseURL() + c.Path(),
	}), "layouts/public")
}

func (h *TournamentHandler) ExportEventPDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pdfBytes, err := h.exportPdfUC.Execute(c.Context(), idStr)
	if err != nil {
		return ErrorHandler(err)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tournament_report_%s.pdf\"", idStr))
	return c.Send(pdfBytes)
}

func (h *TournamentHandler) getBoardData(c *fiber.Ctx, eventID string) (*eventDomain.Tournament, []*divisionDomain.Division, []event.BoardCard, []event.BoardCard, []event.BoardCard, error) {
	return h.getBoardUC.Execute(c.Context(), eventID)
}

func (h *TournamentHandler) AdminBoard(c *fiber.Ctx) error {
	lang := getLang(c)
	e, _, scheduled, inProgress, finished, err := h.getBoardData(c, c.Params("id"))
	if err != nil {
		return ErrorHandler(err)
	}

	// Build AllDivisions and AllCategories from the board cards
	uniqueDivsMap := make(map[string]bool)
	uniqueCatsMap := make(map[string]bool)
	for _, card := range scheduled {
		if card.DivisionName != "" {
			uniqueDivsMap[card.DivisionName] = true
		}
		if card.Category != "" {
			uniqueCatsMap[card.Category] = true
		}
	}
	for _, card := range inProgress {
		if card.DivisionName != "" {
			uniqueDivsMap[card.DivisionName] = true
		}
		if card.Category != "" {
			uniqueCatsMap[card.Category] = true
		}
	}
	for _, card := range finished {
		if card.DivisionName != "" {
			uniqueDivsMap[card.DivisionName] = true
		}
		if card.Category != "" {
			uniqueCatsMap[card.Category] = true
		}
	}
	var allDivs []string
	for d := range uniqueDivsMap {
		allDivs = append(allDivs, d)
	}
	var allCats []string
	for cat := range uniqueCatsMap {
		allCats = append(allCats, cat)
	}

	// Parse filter params
	q := strings.ToLower(c.Query("q"))
	var selectedDivs []string
	for _, d := range c.Request().URI().QueryArgs().PeekMulti("div") {
		selectedDivs = append(selectedDivs, string(d))
	}
	var selectedCats []string
	for _, cat := range c.Request().URI().QueryArgs().PeekMulti("cat") {
		selectedCats = append(selectedCats, string(cat))
	}

	if c.Query("q") != "" || len(selectedDivs) > 0 || len(selectedCats) > 0 {
		scheduled = FilterEventBoardCards(scheduled, q, selectedDivs, selectedCats)
		inProgress = FilterEventBoardCards(inProgress, q, selectedDivs, selectedCats)
		finished = FilterEventBoardCards(finished, q, selectedDivs, selectedCats)
	}
	selectedDivsMap := make(map[string]bool)
	for _, d := range selectedDivs {
		selectedDivsMap[d] = true
	}
	selectedCatsMap := make(map[string]bool)
	for _, cat := range selectedCats {
		selectedCatsMap[cat] = true
	}

	// Build tables from tournament NumTables
	tables := buildEventTables(e, inProgress)

	return c.Render("admin/tournament-board", merge(tMap(lang), fiber.Map{
		"Tournament":    e,
		"Scheduled":     scheduled,
		"InProgress":    inProgress,
		"Finished":      finished,
		"AllDivisions":  allDivs,
		"AllCategories": allCats,
		"QueryQ":        c.Query("q"),
		"SelectedDivs":  selectedDivsMap,
		"SelectedCats":  selectedCatsMap,
		"Tables":        tables,
	}), "layouts/admin")
}

func (h *TournamentHandler) PublicTVDashboard(c *fiber.Ctx) error {
	lang := getLang(c)
	e, _, scheduled, inProgress, finished, err := h.getBoardData(c, c.Params("id"))
	if err != nil {
		return ErrorHandler(err)
	}

	tables := buildEventTables(e, inProgress)

	return c.Render("public/tournament-board", merge(tMap(lang), fiber.Map{
		"Tournament": e,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
	}))
}

func (h *TournamentHandler) BoardColumns(c *fiber.Ctx) error {
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
	var selectedCats []string
	for _, cat := range c.Request().URI().QueryArgs().PeekMulti("cat") {
		selectedCats = append(selectedCats, string(cat))
	}

	if c.Query("q") != "" || len(selectedDivs) > 0 || len(selectedCats) > 0 {
		scheduled = FilterEventBoardCards(scheduled, q, selectedDivs, selectedCats)
		inProgress = FilterEventBoardCards(inProgress, q, selectedDivs, selectedCats)
		finished = FilterEventBoardCards(finished, q, selectedDivs, selectedCats)
	}

	tables := buildEventTables(e, inProgress)
	return c.Render("admin/partials/tournament-board-columns", merge(tMap(lang), fiber.Map{
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
	}))
}

func (h *TournamentHandler) TournamentHealth(c *fiber.Ctx) error {
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render("admin/tournament-health", fiber.Map{
		"Tournament": t,
	}, "layouts/admin")
}

func (h *TournamentHandler) TournamentHealthMetrics(c *fiber.Ctx) error {
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return ErrorHandler(err)
	}

	// Calculate overall metrics by summing up event metrics if available
	// Alternatively, if the metrics are aggregated on the frontend or backend,
	// we will calculate them from the underlying events.
	var overall domainEvent.TournamentMetrics
	var combinedDuration int
	var matchesCount int

	for _, e := range t.Events {
		if e.Metrics != nil {
			overall.TotalMatchesPlayed += e.Metrics.TotalMatchesPlayed
			overall.TotalSetsPlayed += e.Metrics.TotalSetsPlayed
			overall.TotalPointsScored += e.Metrics.TotalPointsScored
			overall.CleanSweeps += e.Metrics.CleanSweeps
			overall.DecidingSets += e.Metrics.DecidingSets
			overall.Walkovers += e.Metrics.Walkovers

			if e.Metrics.LongestMatchDurationSeconds > overall.LongestMatchDurationSeconds {
				overall.LongestMatchDurationSeconds = e.Metrics.LongestMatchDurationSeconds
				overall.LongestMatchID = e.Metrics.LongestMatchID
			}

			combinedDuration += e.Metrics.AverageMatchDurationSeconds * e.Metrics.TotalMatchesPlayed
			matchesCount += e.Metrics.TotalMatchesPlayed
		}
	}

	if matchesCount > 0 {
		overall.AverageMatchDurationSeconds = combinedDuration / matchesCount
		overall.AveragePointsPerMatch = float64(overall.TotalPointsScored) / float64(matchesCount)
		overall.AverageSetsPerMatch = float64(overall.TotalSetsPlayed) / float64(matchesCount)
	}

	// Set pointer to struct or nil if no matches
	var metricsPtr *domainEvent.TournamentMetrics
	if matchesCount > 0 {
		metricsPtr = &overall
	}

	avgMatchMinutes := 0
	if metricsPtr != nil {
		avgMatchMinutes = overall.AverageMatchDurationSeconds / 60
	}

	return c.Render("admin/partials/tournament-health-metrics", fiber.Map{
		"Metrics":         metricsPtr,
		"AvgMatchMinutes": avgMatchMinutes,
	})
}

// buildEventTables creates a TableVM slice from an tournament's NumTables + occupied tables.
func buildEventTables(e *eventDomain.Tournament, inProgress []event.BoardCard) []TableVM {
	var tables []TableVM
	if e == nil {
		return tables
	}

	numTables := e.NumTables
	for _, t := range e.Events {
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

type DivPriority struct {
	ID         string
	Name       string
	Priorities string
}

func buildDivPriorities(divs []*divisionDomain.Division, t *eventDomain.Tournament) []DivPriority {
	var divPriorities []DivPriority
	for _, d := range divs {
		var pStr string
		if p := t.TablePriorityFor(d.ID); len(p) > 0 {
			var pStrs []string
			for _, num := range p {
				pStrs = append(pStrs, strconv.Itoa(num))
			}
			pStr = strings.Join(pStrs, ",")
		}
		divPriorities = append(divPriorities, DivPriority{
			ID:         d.ID,
			Name:       d.Name,
			Priorities: pStr,
		})
	}
	return divPriorities
}

func (h *TournamentHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	e, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return ErrorHandler(err)
	}
	divs, _ := h.divisionUC.GetAll(c.Context())
	divPriorities := buildDivPriorities(divs, e)

	return c.Render("admin/partials/tournament-edit-form", fiber.Map{
		"Tournament":    e,
		"DivPriorities": divPriorities,
	})
}

func (h *TournamentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	name := c.FormValue("name")
	startDate := c.FormValue("startDate")
	endDate := c.FormValue("endDate")
	var numTables int
	fmt.Sscanf(c.FormValue("numTables"), "%d", &numTables)

	tablePriorities := make(map[string][]int)
	c.Request().PostArgs().VisitAll(func(key, value []byte) {
		k := string(key)
		if strings.HasPrefix(k, "priority_") {
			divID := strings.TrimPrefix(k, "priority_")
			valStr := string(value)
			var tables []int
			for _, part := range strings.Split(valStr, ",") {
				part = strings.TrimSpace(part)
				if part == "" {
					continue
				}
				if t, err := strconv.Atoi(part); err == nil {
					tables = append(tables, t)
				}
			}
			if len(tables) > 0 {
				tablePriorities[divID] = tables
			}
		}
	})

	e, err := h.updateUC.Execute(c.Context(), id, name, startDate, endDate, numTables, tablePriorities)
	if err != nil {
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "tournament-updated")
		divs, _ := h.divisionUC.GetAll(c.Context())
		divPriorities := buildDivPriorities(divs, e)

		return c.Render("admin/partials/tournament-edit-form", fiber.Map{
			"Tournament":    e,
			"DivPriorities": divPriorities,
			"Success":       true,
		})
	}
	return c.Redirect("/admin/tournaments/" + id)
}

func FilterEventBoardCards(cards []event.BoardCard, q string, divs []string, cats []string) []event.BoardCard {
	if q == "" && len(divs) == 0 && len(cats) == 0 {
		return cards
	}

	divMap := make(map[string]bool)
	for _, d := range divs {
		divMap[d] = true
	}

	catMap := make(map[string]bool)
	for _, cat := range cats {
		catMap[cat] = true
	}

	var filtered []event.BoardCard
	for _, card := range cards {
		matchesSearch := q == "" || strings.Contains(strings.ToLower(card.PlayerAName), q) ||
			strings.Contains(strings.ToLower(card.PlayerBName), q) ||
			strings.Contains(strings.ToLower(card.GroupName), q)
		matchesDiv := len(divMap) == 0 || divMap[card.DivisionName]
		matchesCat := len(catMap) == 0 || catMap[card.Category]

		if matchesSearch && matchesDiv && matchesCat {
			filtered = append(filtered, card)
		}
	}
	return filtered
}
