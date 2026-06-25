package handler

import (
	"fmt"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"

	"github.com/gofiber/fiber/v2"
)

type TournamentHandler struct {
	createUC      *tournament.CreateTournamentUseCase
	getByID       *tournament.GetTournamentByIDUseCase
	updateUC      *tournament.UpdateTournamentUseCase
	deleteUC      *tournament.DeleteTournamentUseCase
	leaderboardUC *leaderboard.GetLeaderboardUseCase
	divisionUC    *division.DivisionUseCase
	finishUC      *tournament.FinishTournamentUseCase
	exportUC      *tournament.ExportTournamentReportUseCase
	exportPdfUC   *tournament.ExportTournamentPdfUseCase
	exportAllPdfUC *tournament.ExportAllTournamentsPdfUseCase
	movePlayerUC  *tournament.MovePlayerUseCase
	createTeamUC  *tournament.CreateTeamUseCase
	deleteTeamUC  *tournament.DeleteTeamUseCase
	assignPlayerToTeamUC *tournament.AssignPlayerToTeamUseCase
	removePlayerFromTeamUC *tournament.RemovePlayerFromTeamUseCase
	getTournamentsUC *tournament.GetTournamentsUseCase
}

func NewTournamentHandler(
	createUC *tournament.CreateTournamentUseCase,
	getByID *tournament.GetTournamentByIDUseCase,
	updateUC *tournament.UpdateTournamentUseCase,
	deleteUC *tournament.DeleteTournamentUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	divisionUC *division.DivisionUseCase,
	finishUC *tournament.FinishTournamentUseCase,
	exportUC *tournament.ExportTournamentReportUseCase,
	exportPdfUC *tournament.ExportTournamentPdfUseCase,
	exportAllPdfUC *tournament.ExportAllTournamentsPdfUseCase,
	movePlayerUC *tournament.MovePlayerUseCase,
	createTeamUC *tournament.CreateTeamUseCase,
	deleteTeamUC *tournament.DeleteTeamUseCase,
	assignPlayerToTeamUC *tournament.AssignPlayerToTeamUseCase,
	removePlayerFromTeamUC *tournament.RemovePlayerFromTeamUseCase,
	getTournamentsUC *tournament.GetTournamentsUseCase,
) *TournamentHandler {
	return &TournamentHandler{
		createUC:      createUC,
		getByID:       getByID,
		updateUC:      updateUC,
		deleteUC:      deleteUC,
		leaderboardUC: leaderboardUC,
		divisionUC:    divisionUC,
		finishUC:      finishUC,
		exportUC:      exportUC,
		exportPdfUC:   exportPdfUC,
		exportAllPdfUC: exportAllPdfUC,
		movePlayerUC:  movePlayerUC,
		createTeamUC:  createTeamUC,
		deleteTeamUC:  deleteTeamUC,
		assignPlayerToTeamUC: assignPlayerToTeamUC,
		removePlayerFromTeamUC: removePlayerFromTeamUC,
		getTournamentsUC: getTournamentsUC,
	}
}

func (h *TournamentHandler) Create(c *fiber.Ctx) error {
	var body struct {
		Name           string `json:"name" form:"name"`
		Type           string `json:"type" form:"type"`
		Format         string `form:"format"`
		EventCategory  string `form:"eventCategory"`
		StartDate      string `form:"startDate"`
		EndDate        string `form:"endDate"`
		GroupPassCount int    `form:"groupPassCount"`
		TeamFormat     string `form:"teamFormat"`
		NumTables      int    `form:"numTables" json:"numTables"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Parse arrays directly from PostArgs since the form is application/x-www-form-urlencoded
	var participantIDs []string
	for _, id := range c.Request().PostArgs().PeekMulti("participant_ids[]") {
		participantIDs = append(participantIDs, string(id))
	}

	var newPlayers []tournament.NewPlayerData
	firstNames := c.Request().PostArgs().PeekMulti("new_player_first_name[]")
	lastNames := c.Request().PostArgs().PeekMulti("new_player_last_name[]")
	genders := c.Request().PostArgs().PeekMulti("new_player_gender[]")

	for i := 0; i < len(firstNames); i++ {
		np := tournament.NewPlayerData{FirstName: string(firstNames[i])}
		if i < len(lastNames) {
			np.LastName = string(lastNames[i])
		}
		if i < len(genders) {
			np.Gender = string(genders[i])
		}
		if np.FirstName != "" && np.LastName != "" {
			newPlayers = append(newPlayers, np)
		}
	}

	// Parse per-stage rule overrides
	createStages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	var stageRules []tournament.StageRuleOverride
	for _, stage := range createStages {
		boStr := string(c.Request().PostArgs().Peek("stage_rule[" + stage + "][best_of]"))
		ptStr := string(c.Request().PostArgs().Peek("stage_rule[" + stage + "][points_to_win]"))
		pmStr := string(c.Request().PostArgs().Peek("stage_rule[" + stage + "][points_margin]"))
		if boStr != "" {
			bo := 5
			pt := 11
			pm := 2
			fmt.Sscanf(boStr, "%d", &bo)
			fmt.Sscanf(ptStr, "%d", &pt)
			fmt.Sscanf(pmStr, "%d", &pm)
			stageRules = append(stageRules, tournament.StageRuleOverride{
				Stage:        stage,
				BestOf:       bo,
				PointsToWin:  pt,
				PointsMargin: pm,
			})
		}
	}

	skipElo := c.FormValue("skipElo") == "on"
	var eventID *string
	if eIDStr := c.FormValue("eventId"); eIDStr != "" {
		eventID = &eIDStr
	}

	t, err := h.createUC.Execute(c.Context(), body.Name, body.Type, body.Format, body.EventCategory, body.StartDate, body.EndDate, participantIDs, newPlayers, body.GroupPassCount, stageRules, skipElo, eventID, body.TeamFormat, body.NumTables)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("admin/partials/tournament-row", t)
}

func (h *TournamentHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	statusFilter := c.Query("status", "all")
	if statusFilter != "all" {
		var filtered []tournamentDomain.Match
		for _, m := range t.Matches {
			if m.Status == statusFilter {
				filtered = append(filtered, m)
			}
		}
		t.Matches = filtered
	}

	players, _ := h.leaderboardUC.ExecuteSingles(c.Context())
	divisions, _ := h.divisionUC.GetAll(c.Context())

	// Build the view model for the bracket rendering
	vm := BuildTournamentViewModel(t, divisions)

	// Calculate available participants (those not in any team)
	var availableParticipants []*player.Player
	assignedMap := make(map[string]bool)
	for _, team := range t.Teams {
		for _, p := range team.Players {
			assignedMap[p.ID] = true
		}
	}
	for _, p := range t.Participants {
		if !assignedMap[p.ID] {
			availableParticipants = append(availableParticipants, p)
		}
	}

	return c.Render("admin/tournament-detail", fiber.Map{
		"Tournament":       t,
		"Players":          players,
		"Divisions":        divisions,
		"BracketViewModel": vm,
		"AvailableParticipants": availableParticipants,
		"StatusFilter":     statusFilter,
	}, "layouts/admin")
}

func (h *TournamentHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	players, _ := h.leaderboardUC.ExecuteSingles(c.Context())
	return c.Render("admin/partials/tournament-edit-form", fiber.Map{
		"Tournament": t,
		"Players":    players,
	})
}

func (h *TournamentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Name             string `form:"name"`
		Type             string `form:"type"`
		Format           string `form:"format"`
		EventCategory    string `form:"eventCategory"`
		StartDate        string `form:"startDate"`
		EndDate          string `form:"endDate"`
		GroupPassCount   int    `form:"groupPassCount"`
		RegistrationOpen bool   `form:"registrationOpen"`
		TeamFormat       string `form:"teamFormat"`
		NumTables        int    `form:"numTables" json:"numTables"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	var participantIDs []string
	for _, pid := range c.Request().PostArgs().PeekMulti("participant_ids[]") {
		participantIDs = append(participantIDs, string(pid))
	}

	var newPlayers []tournament.NewPlayerData
	firstNames := c.Request().PostArgs().PeekMulti("new_player_first_name[]")
	lastNames := c.Request().PostArgs().PeekMulti("new_player_last_name[]")
	genders := c.Request().PostArgs().PeekMulti("new_player_gender[]")

	for i := 0; i < len(firstNames); i++ {
		np := tournament.NewPlayerData{FirstName: string(firstNames[i])}
		if i < len(lastNames) {
			np.LastName = string(lastNames[i])
		}
		if i < len(genders) {
			np.Gender = string(genders[i])
		}
		if np.FirstName != "" && np.LastName != "" {
			newPlayers = append(newPlayers, np)
		}
	}

	// Parse per-stage rule overrides (sent as stage_rule[group][best_of]=5 etc.)
	stages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	var stageRules []tournament.StageRuleOverride
	for _, stage := range stages {
		boStr := string(c.Request().PostArgs().Peek("stage_rule[" + stage + "][best_of]"))
		ptStr := string(c.Request().PostArgs().Peek("stage_rule[" + stage + "][points_to_win]"))
		pmStr := string(c.Request().PostArgs().Peek("stage_rule[" + stage + "][points_margin]"))
		if boStr != "" {
			bo := 5
			pt := 11
			pm := 2
			fmt.Sscanf(boStr, "%d", &bo)
			fmt.Sscanf(ptStr, "%d", &pt)
			fmt.Sscanf(pmStr, "%d", &pm)
			stageRules = append(stageRules, tournament.StageRuleOverride{
				Stage:        stage,
				BestOf:       bo,
				PointsToWin:  pt,
				PointsMargin: pm,
			})
		}
	}

	skipElo := c.FormValue("skipElo") == "on"
	var eventID *string
	if eIDStr := c.FormValue("eventId"); eIDStr != "" {
		eventID = &eIDStr
	}

	t, err := h.updateUC.Execute(
		c.Context(), id, body.Name, body.Type, body.Format, body.EventCategory, body.StartDate, body.EndDate,
		body.RegistrationOpen, participantIDs, newPlayers, stageRules, body.GroupPassCount,
		skipElo, eventID, body.TeamFormat, body.NumTables,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Render("admin/partials/tournament-row", t)
}

func (h *TournamentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		if c.Get("HX-Current-URL") != "" && fmt.Sprintf("/admin/tournaments/%s", id) == c.Get("HX-Current-URL") {
			c.Set("HX-Redirect", "/admin/tournaments")
		}
		return c.SendString("")
	}
	return c.SendString("")
}

func (h *TournamentHandler) Finish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.finishUC.Execute(c.Context(), idStr); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "finished"})
}

func (h *TournamentHandler) Export(c *fiber.Ctx) error {
	idStr := c.Params("id")
	csvBytes, err := h.exportUC.Execute(c.Context(), idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tournament_report_%s.csv\"", idStr))
	return c.Send(csvBytes)
}

func (h *TournamentHandler) ExportPDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pdfBytes, err := h.exportPdfUC.Execute(c.Context(), idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tournament_report_%s.pdf\"", idStr))
	return c.Send(pdfBytes)
}

func (h *TournamentHandler) ExportAllPDF(c *fiber.Ctx) error {
	pdfBytes, err := h.exportAllPdfUC.Execute(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", "attachment; filename=\"all_tournaments_report.pdf\"")
	return c.Send(pdfBytes)
}

func (h *TournamentHandler) MovePlayer(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		PlayerID      string `json:"playerId" form:"playerId"`
		TargetGroupID string `json:"targetGroupId" form:"targetGroupId"`
		TargetIndex   *int   `json:"targetIndex" form:"targetIndex"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	targetIndex := -1
	if body.TargetIndex != nil {
		targetIndex = *body.TargetIndex
	}

	if err := h.movePlayerUC.Execute(c.Context(), id, body.PlayerID, body.TargetGroupID, targetIndex); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *TournamentHandler) CreateTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		Name string `form:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if _, err := h.createTeamUC.Execute(c.Context(), tournamentID, body.Name); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) DeleteTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	if err := h.deleteTeamUC.Execute(c.Context(), teamID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) AssignPlayerToTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := h.assignPlayerToTeamUC.Execute(c.Context(), teamID, body.PlayerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) RemovePlayerFromTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	playerID := c.Params("playerId")
	if err := h.removePlayerFromTeamUC.Execute(c.Context(), teamID, playerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) PublicList(c *fiber.Ctx) error {
	lang := getLang(c)
	tournaments, err := h.getTournamentsUC.Execute(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Render("public/tournaments", merge(tMap(lang), fiber.Map{
		"Tournaments": tournaments,
		"Type":        "Tournaments",
	}), "layouts/public")
}

func (h *TournamentHandler) PublicDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	statusFilter := c.Query("status", "all")
	if statusFilter != "all" {
		var filtered []tournamentDomain.Match
		for _, m := range t.Matches {
			if m.Status == statusFilter {
				filtered = append(filtered, m)
			}
		}
		t.Matches = filtered
	}

	divisions, _ := h.divisionUC.GetAll(c.Context())

	vm := BuildTournamentViewModel(t, divisions)
	vm.IsPublic = true

	return c.Render("public/tournament-detail", merge(tMap(lang), fiber.Map{
		"Tournament":       t,
		"Divisions":        divisions,
		"BracketViewModel": vm,
		"Type":             "Tournaments",
		"StatusFilter":     statusFilter,
	}), "layouts/public")
}

// BoardCard is a flattened match representation used by the kanban board.
type BoardCard struct {
	MatchID     string
	Status      string
	Stage       string
	BestOf      int
	PlayerAName string
	PlayerBName string
	P1Id        string
	P2Id        string
	TableNumber *int
	ScoreA      int
	ScoreB      int
	Pin         string
}

func buildBoardCards(t *tournamentDomain.Tournament) (scheduled, inProgress, finished []BoardCard) {
	bestOfForStage := func(stage string) int {
		for _, r := range t.StageRules {
			if r.Stage == stage {
				return r.BestOf
			}
		}
		return 5
	}
	nameOf := func(players []*player.Player) string {
		if len(players) == 0 {
			return "TBD"
		}
		p := players[0]
		if len(players) > 1 {
			return p.FirstNameWithSecond() + " / " + players[1].FirstNameWithSecond()
		}
		return p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
	}
	idOf := func(players []*player.Player) string {
		if len(players) == 0 {
			return ""
		}
		return players[0].ID
	}

	for i := range t.Matches {
		m := &t.Matches[i]
		if m.TeamMatchID != nil { // skip sub-matches
			continue
		}
		card := BoardCard{
			MatchID:     m.ID,
			Status:      m.Status,
			Stage:       m.Stage,
			BestOf:      bestOfForStage(m.Stage),
			PlayerAName: nameOf(m.TeamA),
			PlayerBName: nameOf(m.TeamB),
			P1Id:        idOf(m.TeamA),
			P2Id:        idOf(m.TeamB),
			TableNumber: m.TableNumber,
			ScoreA:      m.ScoreA(),
			ScoreB:      m.ScoreB(),
			Pin:         m.Pin,
		}
		switch m.Status {
		case "in_progress":
			inProgress = append(inProgress, card)
		case "finished":
			finished = append(finished, card)
		default:
			scheduled = append(scheduled, card)
		}
	}
	return
}

func (h *TournamentHandler) Board(c *fiber.Ctx) error {
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	scheduled, inProgress, finished := buildBoardCards(t)
	return c.Render("admin/tournament-board", fiber.Map{
		"Tournament": t,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
	}, "layouts/admin")
}

func (h *TournamentHandler) BoardColumns(c *fiber.Ctx) error {
	id := c.Params("id")
	t, err := h.getByID.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}
	scheduled, inProgress, finished := buildBoardCards(t)
	return c.Render("admin/partials/board-columns", fiber.Map{
		"Tournament": t,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
	})
}
