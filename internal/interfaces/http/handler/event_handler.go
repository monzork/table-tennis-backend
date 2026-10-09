package handler

import (
	"context"
	"fmt"

	"strings"
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	tournamentApp "table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
)

type EventHandler struct {
	createUC               *event.CreateTournamentUseCase
	getByID                *event.GetTournamentByIDUseCase
	updateUC               *event.UpdateTournamentUseCase
	deleteUC               *event.DeleteTournamentUseCase
	leaderboardUC          *leaderboard.GetLeaderboardUseCase
	divisionUC             *division.DivisionUseCase
	finishUC               *event.FinishTournamentUseCase
	exportUC               *event.ExportTournamentReportUseCase
	exportPdfUC            *event.ExportTournamentPdfUseCase
	movePlayerUC           *event.MovePlayerUseCase
	createTeamUC           *event.CreateTeamUseCase
	deleteTeamUC           *event.DeleteTeamUseCase
	assignPlayerToTeamUC   *event.AssignPlayerToTeamUseCase
	removePlayerFromTeamUC *event.RemovePlayerFromTeamUseCase
	getTournamentsUC       *event.GetTournamentsUseCase
	getOccupiedTablesUC    *event.GetOccupiedTablesUseCase
	regenerateSeedsUC      *event.RegenerateGroupSeedsUseCase
	updateParticipantEloUC *event.UpdateParticipantEloBeforeUseCase
	removeParticipantUC    *event.RemoveParticipantUseCase
	saveKnockoutSeedsUC    *event.SaveKnockoutSeedsUseCase
	toggleSeedingLockUC    *event.ToggleSeedingLockUseCase
	addGroupUC             *event.AddGroupUseCase
	recalculateEloUC       *event.RecalculateTournamentEloUseCase
	startKnockoutUC        *event.StartKnockoutStageUseCase
	getDetailViewUC        *event.GetEventDetailViewUseCase
	getPublicDetailViewUC  *event.GetPublicEventDetailViewUseCase
	tvDashboardUC          *event.GetPublicTVDashboardViewUseCase
	boardViewUC            *event.GetBoardViewUseCase
	editFormViewUC         *event.GetEditFormViewUseCase
	inactivityDecayUC      *tournamentApp.ApplyInactivityDecayUseCase
}

// WithInactivityDecay wires the optional inactivity-decay pass, run after an
// event finishes on the chance it's the last child event of its parent
// tournament. Left unset, Finish just skips it.
func (h *EventHandler) WithInactivityDecay(uc *tournamentApp.ApplyInactivityDecayUseCase) *EventHandler {
	h.inactivityDecayUC = uc
	return h
}

func NewEventHandler(
	createUC *event.CreateTournamentUseCase,
	getByID *event.GetTournamentByIDUseCase,
	updateUC *event.UpdateTournamentUseCase,
	deleteUC *event.DeleteTournamentUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	divisionUC *division.DivisionUseCase,
	finishUC *event.FinishTournamentUseCase,
	exportUC *event.ExportTournamentReportUseCase,
	exportPdfUC *event.ExportTournamentPdfUseCase,
	movePlayerUC *event.MovePlayerUseCase,
	createTeamUC *event.CreateTeamUseCase,
	deleteTeamUC *event.DeleteTeamUseCase,
	assignPlayerToTeamUC *event.AssignPlayerToTeamUseCase,
	removePlayerFromTeamUC *event.RemovePlayerFromTeamUseCase,
	getTournamentsUC *event.GetTournamentsUseCase,
	getOccupiedTablesUC *event.GetOccupiedTablesUseCase,
	regenerateSeedsUC *event.RegenerateGroupSeedsUseCase,
	updateParticipantEloUC *event.UpdateParticipantEloBeforeUseCase,
	removeParticipantUC *event.RemoveParticipantUseCase,
	saveKnockoutSeedsUC *event.SaveKnockoutSeedsUseCase,
	toggleSeedingLockUC *event.ToggleSeedingLockUseCase,
	addGroupUC *event.AddGroupUseCase,
	recalculateEloUC *event.RecalculateTournamentEloUseCase,
	startKnockoutUC *event.StartKnockoutStageUseCase,
	getDetailViewUC *event.GetEventDetailViewUseCase,
	getPublicDetailViewUC *event.GetPublicEventDetailViewUseCase,
	tvDashboardUC *event.GetPublicTVDashboardViewUseCase,
	boardViewUC *event.GetBoardViewUseCase,
	editFormViewUC *event.GetEditFormViewUseCase,
) *EventHandler {
	return &EventHandler{
		createUC:               createUC,
		getByID:                getByID,
		updateUC:               updateUC,
		deleteUC:               deleteUC,
		leaderboardUC:          leaderboardUC,
		divisionUC:             divisionUC,
		finishUC:               finishUC,
		exportUC:               exportUC,
		exportPdfUC:            exportPdfUC,
		movePlayerUC:           movePlayerUC,
		createTeamUC:           createTeamUC,
		deleteTeamUC:           deleteTeamUC,
		assignPlayerToTeamUC:   assignPlayerToTeamUC,
		removePlayerFromTeamUC: removePlayerFromTeamUC,
		getTournamentsUC:       getTournamentsUC,
		getOccupiedTablesUC:    getOccupiedTablesUC,
		regenerateSeedsUC:      regenerateSeedsUC,
		updateParticipantEloUC: updateParticipantEloUC,
		removeParticipantUC:    removeParticipantUC,
		saveKnockoutSeedsUC:    saveKnockoutSeedsUC,
		toggleSeedingLockUC:    toggleSeedingLockUC,
		addGroupUC:             addGroupUC,
		recalculateEloUC:       recalculateEloUC,
		startKnockoutUC:        startKnockoutUC,
		getDetailViewUC:        getDetailViewUC,
		getPublicDetailViewUC:  getPublicDetailViewUC,
		tvDashboardUC:          tvDashboardUC,
		boardViewUC:            boardViewUC,
		editFormViewUC:         editFormViewUC,
	}
}

func (h *EventHandler) StartKnockout(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	divID := c.Params("divId")

	err := h.startKnockoutUC.Execute(c.Context(), tournamentID, divID)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	c.Set("HX-Trigger", `{"show-toast": {"message": "Knockout matches created and scheduled!", "type": "success"}, "reload-bracket": true}`)
	return c.SendString("")
}

func (h *EventHandler) Create(c *fiber.Ctx) error {
	cmd, err := parseCreateEventCommand(c)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	t, err := h.createUC.Execute(c.Context(), cmd)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	lang := getLang(c)
	return c.Render("admin/partials/event-row", merge(tMap(lang), fiber.Map{"Event": t}))
}

func (h *EventHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")
	statusFilter := c.Query("status", "all")
	playerSearch := c.Query("player_search", "")

	lang := getLang(c)
	view, err := h.getDetailViewUC.Execute(c.Context(), id, statusFilter, playerSearch, i18n.PrecomputedMaps[lang])
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	return c.Render("admin/event-detail", merge(tMap(lang), fiber.Map{
		"Event":                 view.Event,
		"Players":               view.Players,
		"Divisions":             view.Divisions,
		"BracketViewModel":      view.BracketViewModel,
		"AvailableParticipants": view.AvailableParticipants,
		"StatusFilter":          statusFilter,
		"PlayerSearch":          playerSearch,
		"PlayerPins":            view.PlayerPins,
		"Officials":             view.Officials,
		"ParticipantRows":       view.ParticipantRows,
	}), "layouts/admin")
}

func (h *EventHandler) AddOfficial(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if err := h.getByID.AddOfficial(c.Context(), tournamentID, body.PlayerID); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) RemoveOfficial(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	playerID := c.Params("playerId")
	if err := h.getByID.RemoveOfficial(c.Context(), tournamentID, playerID); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) RemoveParticipant(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	playerID := c.Params("playerId")
	if err := h.removeParticipantUC.Execute(c.Context(), tournamentID, playerID); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	view, err := h.editFormViewUC.Execute(c.Context(), id)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	lang := getLang(c)
	return c.Render("admin/partials/event-edit-form", merge(tMap(lang), fiber.Map{
		"Event":     view.Event,
		"Players":   view.Players,
		"Divisions": view.Divisions,
	}))
}

func (h *EventHandler) Update(c *fiber.Ctx) error {
	cmd, err := parseUpdateEventCommand(c)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	t, err := h.updateUC.Execute(c.Context(), cmd)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	lang := getLang(c)
	return c.Render("admin/partials/event-row", merge(tMap(lang), fiber.Map{"Event": t}))
}

func (h *EventHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		if c.Get("HX-Current-URL") != "" && fmt.Sprintf("/admin/events/%s", id) == c.Get("HX-Current-URL") {
			c.Set("HX-Redirect", "/admin/events")
		}
		return c.SendString("")
	}
	return c.SendString("")
}

func (h *EventHandler) Finish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.finishUC.Execute(c.Context(), idStr); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if h.inactivityDecayUC != nil {
		if ev, err := h.getByID.Execute(c.Context(), idStr); err == nil {
			// Best-effort: a failure here shouldn't undo the event having
			// finished, so it's logged rather than surfaced to the caller.
			if err := h.inactivityDecayUC.ExecuteForEvent(c.Context(), ev.TournamentID); err != nil {
				fmt.Println("inactivity decay:", err)
			}
		}
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "finished"})
}

func (h *EventHandler) RegenerateGroupSeeds(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.regenerateSeedsUC.Execute(c.Context(), idStr); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "regenerated"})
}

func (h *EventHandler) UpdateParticipantEloBefore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var body struct {
		PlayerID   string `form:"playerId"`
		SinglesElo int16  `form:"singlesElo"`
		DoublesElo int16  `form:"doublesElo"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if body.PlayerID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "playerId is required")
	}
	if err := h.updateParticipantEloUC.Execute(c.Context(), idStr, body.PlayerID, body.SinglesElo, body.DoublesElo); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

func (h *EventHandler) Export(c *fiber.Ctx) error {
	idStr := c.Params("id")
	csvBytes, err := h.exportUC.Execute(c.Context(), idStr)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_report_%s.csv\"", idStr))
	return c.Send(csvBytes)
}

func (h *EventHandler) ExportPDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pdfBytes, err := h.exportPdfUC.Execute(c.Context(), idStr, getLang(c))
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_report_%s.pdf\"", idStr))
	return c.Send(pdfBytes)
}

func (h *EventHandler) MovePlayer(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		PlayerID      string `json:"playerId" form:"playerId"`
		TargetGroupID string `json:"targetGroupId" form:"targetGroupId"`
		TargetIndex   *int   `json:"targetIndex" form:"targetIndex"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	targetIndex := -1
	if body.TargetIndex != nil {
		targetIndex = *body.TargetIndex
	}

	if err := h.movePlayerUC.Execute(c.Context(), id, body.PlayerID, body.TargetGroupID, targetIndex); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *EventHandler) SaveKnockoutSeeds(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		DivID     string `json:"divId" form:"divId"`
		PlayerIDs string `json:"playerIds" form:"playerIds"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	if err := h.saveKnockoutSeedsUC.Execute(c.Context(), id, body.DivID, body.PlayerIDs); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *EventHandler) AddGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		DivisionName string `json:"divisionName" form:"divisionName"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	if err := h.addGroupUC.Execute(c.Context(), id, body.DivisionName); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *EventHandler) CreateTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		Name string `form:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if _, err := h.createTeamUC.Execute(c.Context(), tournamentID, body.Name); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) DeleteTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	if err := h.deleteTeamUC.Execute(c.Context(), teamID); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) AssignPlayerToTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if err := h.assignPlayerToTeamUC.Execute(c.Context(), teamID, body.PlayerID); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) RemovePlayerFromTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	playerID := c.Params("playerId")
	if err := h.removePlayerFromTeamUC.Execute(c.Context(), teamID, playerID); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) PublicList(c *fiber.Ctx) error {
	lang := getLang(c)
	events, err := h.getTournamentsUC.Execute(c.Context())
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	return c.Render("public/events", merge(tMap(lang), fiber.Map{
		"Events":       events,
		"Type":         "Events",
		"OGImage":      c.BaseURL() + "/open_tdm.jpeg",
		"Title":        "Events",
		"CanonicalURL": c.BaseURL() + c.Path(),
	}), "layouts/public")
}

// PublicRedirectToTournament sends old per-category public links (/events/:id) to
// the parent tournament page; the category bracket itself now lives at /categories/:id.
func (h *EventHandler) PublicRedirectToTournament(c *fiber.Ctx) error {
	id := c.Params("id")
	ev, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return ErrorHandler(err)
	}
	if ev.TournamentID == nil {
		return h.PublicDetail(c)
	}
	return c.Redirect("/tournaments/" + *ev.TournamentID)
}

func (h *EventHandler) PublicDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")
	statusFilter := c.Query("status", "all")
	playerSearch := c.Query("player_search", "")
	canonicalURL := c.BaseURL() + c.Path()
	tmap, _ := c.Locals("T").(map[string]string)

	view, err := h.getPublicDetailViewUC.Execute(
		c.Context(), id, statusFilter, playerSearch, canonicalURL, BuildBoardCards, tmap,
	)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	return c.Render("public/event-detail", merge(tMap(lang), fiber.Map{
		"Event":            view.Event,
		"Divisions":        view.Divisions,
		"BracketViewModel": view.BracketViewModel,
		"ParticipantRows":  view.ParticipantRows,
		"Type":             "Events",
		"StatusFilter":     statusFilter,
		"PlayerSearch":     playerSearch,
		"RefereeNames":     view.RefereeNames,
		"CanonicalURL":     canonicalURL,
		"OGImage":          c.BaseURL() + "/open_tdm.jpeg",
		"JSONLD":           view.JSONLD,
		"Title":            view.Event.Name,
		"Description":      fmt.Sprintf("%s Event. Register and view live bracket.", view.Event.Name),
	}), "layouts/public")
}

func (h *EventHandler) PublicTVDashboard(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")
	playerSearch := c.Query("player_search", "")
	tmap, _ := c.Locals("T").(map[string]string)

	view, err := h.tvDashboardUC.Execute(c.Context(), id, playerSearch, BuildBoardCards, tmap)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	tables := event.BuildTableVMs(view.Event, "", h.getOccupiedTables(c.Context(), view.Event))

	return c.Render("public/tv-dashboard", merge(tMap(lang), fiber.Map{
		"Event":            view.Event,
		"Divisions":        view.Divisions,
		"BracketViewModel": view.BracketViewModel,
		"Scheduled":        view.Scheduled,
		"InProgress":       view.InProgress,
		"Finished":         view.Finished,
		"Tables":           tables,
	})) // No layout for TV
}

// TableVM is a view model for a table's status. Defined in application/event/board_helpers.go.
// This alias keeps templates working without change.
type TableVM = event.TableVM

func (h *EventHandler) getOccupiedTables(ctx context.Context, t *tournamentDomain.Event) []int {
	occupiedList, _ := h.getOccupiedTablesUC.Execute(ctx, t)
	return occupiedList
}

func FilterBoardCards(cards []event.BoardCard, q string, divs []string) []event.BoardCard {
	if q == "" && len(divs) == 0 {
		return cards
	}

	divMap := make(map[string]bool)
	for _, d := range divs {
		divMap[d] = true
	}

	var filtered []event.BoardCard
	for _, card := range cards {
		matchesSearch := q == "" || strings.Contains(strings.ToLower(card.PlayerAName), q) ||
			strings.Contains(strings.ToLower(card.PlayerBName), q) ||
			strings.Contains(strings.ToLower(card.GroupName), q)
		matchesDiv := len(divMap) == 0 || divMap[card.DivisionName]

		if matchesSearch && matchesDiv {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

// BuildBoardCards delegates to the application layer so every board shares one implementation.
var BuildBoardCards = event.BuildBoardCards

func (h *EventHandler) Board(c *fiber.Ctx) error {
	id := c.Params("id")
	q := c.Query("q", "")
	var selectedDivs []string
	for _, d := range c.Request().URI().QueryArgs().PeekMulti("div") {
		selectedDivs = append(selectedDivs, string(d))
	}

	view, err := h.boardViewUC.Execute(c.Context(), id, q, selectedDivs, BuildBoardCards, FilterBoardCards)
	if err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}

	tables := event.BuildTableVMs(view.Event, "", h.getOccupiedTables(c.Context(), view.Event))

	lang := getLang(c)
	return c.Render("admin/event-board", merge(tMap(lang), fiber.Map{
		"Event":        view.Event,
		"Scheduled":    view.Scheduled,
		"InProgress":   view.InProgress,
		"Finished":     view.Finished,
		"AllDivisions": view.AllDivs,
		"SelectedDivs": selectedDivs,
		"Query":        q,
		"Tables":       tables,
	}), "layouts/admin")
}

func (h *EventHandler) BoardColumns(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		event     *tournamentDomain.Event
		err       error
		divisions []*divisionDomain.Division
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
		return ErrorHandler(res.err)
	}
	t := res.event
	divs := res.divisions
	scheduled, inProgress, finished := BuildBoardCards(t, divs)
	tables := event.BuildTableVMs(t, "", h.getOccupiedTables(c.Context(), t))

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

	lang := getLang(c)
	return c.Render("admin/partials/board-columns", merge(tMap(lang), fiber.Map{
		"Event":      t,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
	}))
}

func (h *EventHandler) ToggleSeedingLock(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.toggleSeedingLockUC.Execute(c.Context(), id); err != nil {
		return c.Status(500).SendString("Failed to toggle seeding lock")
	}

	c.Set("HX-Trigger", "reload-bracket")
	return c.SendStatus(200)
}

func (h *EventHandler) RecalculateElo(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.recalculateEloUC.Execute(c.Context(), id); err != nil {
		fmt.Println(err)
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "recalculated"})
}
