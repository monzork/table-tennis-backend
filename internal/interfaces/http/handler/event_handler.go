package handler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
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

func (h *TournamentHandler) getBoardData(c *fiber.Ctx, eventID string) (*eventDomain.Tournament, []*divisionDomain.Division, []BoardCard, []BoardCard, []BoardCard, error) {
	ctx := c.Context()

	// Ensure deep fetch of the tournament so all events and matches are loaded.
	// Actually, getByID uses GetByIDDeep behind the scenes which includes events and matches.
	e, err := h.getByID.Execute(ctx, eventID)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	divs, _ := h.divisionUC.GetAll(ctx)

	var inProgress []BoardCard
	var finished []BoardCard

	type activePool struct {
		id             string
		tournamentID   string
		divisionName   string
		tournamentName string
		pool           []BoardCard
	}
	var activePools []activePool

	divPlayerCounts := make(map[string]int)

	// Global tracker for initial player availability based on finished matches
	lastActivity := make(map[string]time.Time)

	for _, t := range e.Events {
		s, i, f := BuildBoardCards(t, divs)

		// Build the view model to get player counts per division
		vm := bracket.BuildBracket(t, divs, nil)
		for _, dv := range vm.Divisions {
			divPlayerCounts[dv.Name] += len(dv.Players)
		}

		// Inject TournamentName and TournamentID into cards
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

		// Group scheduled cards by division
		scheduledByDiv := make(map[string][]BoardCard)
		for _, card := range s {
			scheduledByDiv[card.DivisionName] = append(scheduledByDiv[card.DivisionName], card)
		}

		// Group in-progress cards by division
		inProgressByDiv := make(map[string][]BoardCard)
		for _, card := range i {
			inProgressByDiv[card.DivisionName] = append(inProgressByDiv[card.DivisionName], card)
		}

		// Collect all active divisions in this event
		allDivNamesMap := make(map[string]bool)
		for _, card := range s {
			allDivNamesMap[card.DivisionName] = true
		}
		for _, card := range i {
			allDivNamesMap[card.DivisionName] = true
		}

		for divName := range allDivNamesMap {
			var u []BoardCard
			var v []BoardCard
			for _, card := range scheduledByDiv[divName] {
				if card.MatchID == "" {
					v = append(v, card)
				} else {
					u = append(u, card)
				}
			}

			pool := append(u, v...)
			poolKey := fmt.Sprintf("%s:%s", t.ID, divName)

			activePools = append(activePools, activePool{
				id:             poolKey,
				tournamentID:   t.ID,
				divisionName:   divName,
				tournamentName: t.Name,
				pool:           pool,
			})
		}

		inProgress = append(inProgress, i...)
		finished = append(finished, f...)

		// Track last activity based on database finished matches
		for _, m := range t.Matches {
			if m.Status == "finished" {
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

	// Calculate unique active divisions across all events
	activeDivisionsMap := make(map[string]bool)
	for _, ap := range activePools {
		activeDivisionsMap[ap.divisionName] = true
	}

	type activeDivInfo struct {
		name        string
		playerCount int
	}
	var activeDivs []activeDivInfo
	totalPlayers := 0

	for divName := range activeDivisionsMap {
		pCount := divPlayerCounts[divName]
		if pCount <= 0 {
			pCount = 1
		}
		activeDivs = append(activeDivs, activeDivInfo{
			name:        divName,
			playerCount: pCount,
		})
		totalPlayers += pCount
	}

	// 1. Proportional Table Allocation (Hamilton Method) globally by Division Name
	maxTables := e.NumTables
	if maxTables <= 0 {
		maxTables = 6 // default fallback
	}

	allocated := make(map[string]int)
	remainingTables := maxTables
	numActiveDivs := len(activeDivs)

	if numActiveDivs > 0 {
		if remainingTables >= numActiveDivs {
			for _, ad := range activeDivs {
				allocated[ad.name] = 1
				remainingTables--
			}
		} else {
			// Sort active divisions by player count descending
			sort.Slice(activeDivs, func(i, j int) bool {
				return activeDivs[i].playerCount > activeDivs[j].playerCount
			})
			for _, ad := range activeDivs {
				if remainingTables > 0 {
					allocated[ad.name] = 1
					remainingTables--
				} else {
					allocated[ad.name] = 0
				}
			}
		}

		if remainingTables > 0 && totalPlayers > 0 {
			type remainderInfo struct {
				name      string
				remainder float64
			}
			var remainders []remainderInfo
			allocatedSum := 0

			for _, ad := range activeDivs {
				share := float64(ad.playerCount) / float64(totalPlayers) * float64(remainingTables)
				intShare := int(share)
				allocated[ad.name] += intShare
				allocatedSum += intShare
				remainders = append(remainders, remainderInfo{
					name:      ad.name,
					remainder: share - float64(intShare),
				})
			}

			leftover := remainingTables - allocatedSum
			sort.Slice(remainders, func(i, j int) bool {
				return remainders[i].remainder > remainders[j].remainder
			})
			for i := 0; i < leftover && i < len(remainders); i++ {
				allocated[remainders[i].name]++
			}
		}
	}

	// 2. Tournament-Driven queue scheduling simulation
	pools := make(map[string][]BoardCard)
	for _, ap := range activePools {
		pools[ap.id] = ap.pool
	}

	var globalScheduled []BoardCard
	type runningMatch struct {
		divisionName string
		endTime      time.Time
	}
	var runningMatches []runningMatch

	simClock := time.Now()
	availableTime := make(map[string]time.Time)
	scheduledCounts := make(map[string]int)

	// Pre-fill player availableTime from database finished matches
	for pID, tAct := range lastActivity {
		availableTime[pID] = tAct.Add(10 * time.Minute) // 10 minutes of rest after their last finished match
	}

	// Initialize in-progress matches
	for _, c := range inProgress {
		avail := time.Now().Add(25 * time.Minute) // 15 mins play + 10 mins rest
		if c.P1Id != "" {
			availableTime[c.P1Id] = avail
		}
		if c.P2Id != "" {
			availableTime[c.P2Id] = avail
		}
		runningMatches = append(runningMatches, runningMatch{
			divisionName: c.DivisionName,
			endTime:      time.Now().Add(15 * time.Minute),
		})
	}

	getPlayerAvail := func(playerID string) time.Time {
		if playerID == "" {
			return time.Time{}
		}
		if t, ok := availableTime[playerID]; ok {
			return t
		}
		return time.Time{}
	}

	for {
		// Clean up finished matches at current simClock
		var nextRunning []runningMatch
		activeCount := make(map[string]int)
		for _, rm := range runningMatches {
			if rm.endTime.After(simClock) {
				nextRunning = append(nextRunning, rm)
				activeCount[rm.divisionName]++
			}
		}
		runningMatches = nextRunning

		// Identify candidates
		type candidate struct {
			card         BoardCard
			maxAvail     time.Time
			ratio        float64
			poolIdx      string
			divisionName string
		}
		var candidates []candidate

		for id, pool := range pools {
			if len(pool) == 0 {
				continue
			}
			c := pool[0]
			t1 := getPlayerAvail(c.P1Id)
			t2 := getPlayerAvail(c.P2Id)
			maxAvail := t1
			if t2.After(t1) {
				maxAvail = t2
			}

			divName := c.DivisionName
			alloc := allocated[divName]
			var ratio float64
			if alloc > 0 {
				ratio = float64(activeCount[divName]) / float64(alloc)
			} else {
				ratio = float64(activeCount[divName]) / 0.5
			}

			candidates = append(candidates, candidate{
				card:         c,
				maxAvail:     maxAvail,
				ratio:        ratio,
				poolIdx:      id,
				divisionName: divName,
			})
		}

		if len(candidates) == 0 {
			break
		}

		// Check available candidates
		var availableCandidates []candidate
		for _, cand := range candidates {
			if !cand.maxAvail.After(simClock) {
				availableCandidates = append(availableCandidates, cand)
			}
		}

		var chosen candidate
		if len(availableCandidates) > 0 {
			sort.Slice(availableCandidates, func(i, j int) bool {
				c1, c2 := availableCandidates[i], availableCandidates[j]
				u1 := c1.ratio < 1.0
				u2 := c2.ratio < 1.0
				if u1 != u2 {
					return u1
				}
				if c1.ratio != c2.ratio {
					return c1.ratio < c2.ratio
				}
				// Prioritize the division with the larger player count (allocated tables)
				alloc1 := allocated[c1.divisionName]
				alloc2 := allocated[c2.divisionName]
				if alloc1 != alloc2 {
					return alloc1 > alloc2 // descending
				}
				// Alternate between events of the same division to prevent category starvation
				count1 := scheduledCounts[c1.poolIdx]
				count2 := scheduledCounts[c2.poolIdx]
				if count1 != count2 {
					return count1 < count2
				}
				if !c1.maxAvail.Equal(c2.maxAvail) {
					return c1.maxAvail.Before(c2.maxAvail)
				}
				return c1.poolIdx < c2.poolIdx
			})
			chosen = availableCandidates[0]
		} else {
			// Time warp
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].maxAvail.Before(candidates[j].maxAvail)
			})
			simClock = candidates[0].maxAvail
			continue
		}

		globalScheduled = append(globalScheduled, chosen.card)
		scheduledCounts[chosen.poolIdx]++

		// Occupy table globally by division name
		runningMatches = append(runningMatches, runningMatch{
			divisionName: chosen.divisionName,
			endTime:      simClock.Add(20 * time.Minute),
		})

		// Lock players
		avail := simClock.Add(30 * time.Minute)
		if chosen.card.P1Id != "" {
			availableTime[chosen.card.P1Id] = avail
		}
		if chosen.card.P2Id != "" {
			availableTime[chosen.card.P2Id] = avail
		}

		// Advance pool
		pools[chosen.poolIdx] = pools[chosen.poolIdx][1:]
	}

	// Mark players currently in a match across the entire tournament and set global QueuePosition
	inMatchPlayers := make(map[string]bool)
	for _, c := range inProgress {
		if c.P1Id != "" {
			inMatchPlayers[c.P1Id] = true
		}
		if c.P2Id != "" {
			inMatchPlayers[c.P2Id] = true
		}
	}
	for i := range globalScheduled {
		if globalScheduled[i].P1Id != "" && inMatchPlayers[globalScheduled[i].P1Id] {
			globalScheduled[i].P1InMatch = true
		} else {
			globalScheduled[i].P1InMatch = false
		}
		if globalScheduled[i].P2Id != "" && inMatchPlayers[globalScheduled[i].P2Id] {
			globalScheduled[i].P2InMatch = true
		} else {
			globalScheduled[i].P2InMatch = false
		}
		globalScheduled[i].QueuePosition = i + 1
	}

	return e, divs, globalScheduled, inProgress, finished, nil
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

	// Build tables from tournament NumTables
	tables := buildEventTables(e, inProgress)

	return c.Render("admin/partials/tournament-board-columns", fiber.Map{
		"Tournament": e,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
		"T":          tMap(lang)["T"],
	})
}

// buildEventTables creates a TableVM slice from an tournament's NumTables + occupied tables.
func buildEventTables(e *eventDomain.Tournament, inProgress []BoardCard) []TableVM {
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

func (h *TournamentHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	e, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return ErrorHandler(err)
	}
	divs, _ := h.divisionUC.GetAll(c.Context())

	type DivPriority struct {
		ID         string
		Name       string
		Priorities string
	}
	var divPriorities []DivPriority
	for _, d := range divs {
		var pStr string
		if e.TablePriorities != nil {
			if p, ok := e.TablePriorities[d.ID]; ok {
				var pStrs []string
				for _, num := range p {
					pStrs = append(pStrs, strconv.Itoa(num))
				}
				pStr = strings.Join(pStrs, ",")
			}
		}
		divPriorities = append(divPriorities, DivPriority{
			ID:         d.ID,
			Name:       d.Name,
			Priorities: pStr,
		})
	}

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

		type DivPriority struct {
			ID         string
			Name       string
			Priorities string
		}
		var divPriorities []DivPriority
		for _, d := range divs {
			var pStr string
			if e.TablePriorities != nil {
				if p, ok := e.TablePriorities[d.ID]; ok {
					var pStrs []string
					for _, num := range p {
						pStrs = append(pStrs, strconv.Itoa(num))
					}
					pStr = strings.Join(pStrs, ",")
				}
			}
			divPriorities = append(divPriorities, DivPriority{
				ID:         d.ID,
				Name:       d.Name,
				Priorities: pStr,
			})
		}

		return c.Render("admin/partials/tournament-edit-form", fiber.Map{
			"Tournament":    e,
			"DivPriorities": divPriorities,
			"Success":       true,
		})
	}
	return c.Redirect("/admin/tournaments/" + id)
}

func FilterEventBoardCards(cards []BoardCard, q string, divs []string, cats []string) []BoardCard {
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

	var filtered []BoardCard
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
