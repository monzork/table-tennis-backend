package handler

import (
	"fmt"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/application/division"

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
}

func NewTournamentHandler(
	createUC *tournament.CreateTournamentUseCase,
	getByID *tournament.GetTournamentByIDUseCase,
	updateUC *tournament.UpdateTournamentUseCase,
	deleteUC *tournament.DeleteTournamentUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	divisionUC *division.DivisionUseCase,
	finishUC *tournament.FinishTournamentUseCase,
) *TournamentHandler {
	return &TournamentHandler{
		createUC:      createUC,
		getByID:       getByID,
		updateUC:      updateUC,
		deleteUC:      deleteUC,
		leaderboardUC: leaderboardUC,
		divisionUC:    divisionUC,
		finishUC:      finishUC,
	}
}


func (h *TournamentHandler) Create(c *fiber.Ctx) error {
	var body struct {
		Name      string `json:"name" form:"name"`
		Type      string `json:"type" form:"type"`
		Format    string `form:"format"`
		StartDate string `form:"startDate"`
		EndDate   string `form:"endDate"`
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

	t, err := h.createUC.Execute(c.Context(), body.Name, body.Type, body.Format, body.StartDate, body.EndDate, participantIDs, newPlayers)
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
	return c.Render("admin/tournament-detail", fiber.Map{
		"Tournament": t,
		"Players":    players,
		"Divisions":  divisions,
	}, "layouts/admin")
}

func (h *TournamentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Name      string `form:"name"`
		Type      string `form:"type"`
		Format    string `form:"format"`
		StartDate string `form:"startDate"`
		EndDate   string `form:"endDate"`
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
		c.Context(), id, body.Name, body.Type, body.Format, body.StartDate, body.EndDate,
		participantIDs, newPlayers, stageRules,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Render("admin/partials/tournament-row", t)
}

func (h *TournamentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
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
	return c.JSON(fiber.Map{"status": "finished"})
}
