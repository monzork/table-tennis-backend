package handler

import (
	"fmt"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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

	t, err := h.createUC.Execute(c.Context(), body.Name, body.Type, body.Format, body.EventCategory, body.StartDate, body.EndDate, participantIDs, newPlayers, body.GroupPassCount)
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
	players, _ := h.leaderboardUC.ExecuteSingles(c.Context())
	divisions, _ := h.divisionUC.GetAll(c.Context())

	// Build the view model for the bracket rendering
	vm := BuildTournamentViewModel(t, divisions)

	return c.Render("admin/tournament-detail", fiber.Map{
		"Tournament":       t,
		"Players":          players,
		"Divisions":        divisions,
		"BracketViewModel": vm,
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

	t, err := h.updateUC.Execute(
		c.Context(), id, body.Name, body.Type, body.Format, body.EventCategory, body.StartDate, body.EndDate,
		body.RegistrationOpen, participantIDs, newPlayers, stageRules, body.GroupPassCount,
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
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tournament id")
	}
	if err := h.finishUC.Execute(c.Context(), id); err != nil {
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
