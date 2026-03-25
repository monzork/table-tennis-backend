package handler

import (
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MatchHandler struct {
	createUC      *match.CreateMatchUseCase
	finishUC      *match.FinishMatchUseCase
	updateScoreUC *match.UpdateMatchScoreUseCase
}

func NewMatchHandler(createUC *match.CreateMatchUseCase, finishUC *match.FinishMatchUseCase, updateScoreUC *match.UpdateMatchScoreUseCase) *MatchHandler {
	return &MatchHandler{
		finishUC:      finishUC,
		createUC:      createUC,
		updateScoreUC: updateScoreUC,
	}
}

func (h *MatchHandler) Create(c *fiber.Ctx) error {
	var body struct {
		TournamentID   string   `json:"tournamentId" form:"tournamentId"`
		MatchType      string   `json:"matchType" form:"matchType"`
		TeamAPlayerIDs []string `json:"teamAPlayerIds" form:"teamAPlayerIds"`
		TeamBPlayerIDs []string `json:"teamBPlayerIds" form:"teamBPlayerIds"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	tID, err := uuid.Parse(body.TournamentID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tournament id")
	}

	var teamA []uuid.UUID
	for _, idStr := range body.TeamAPlayerIDs {
		id, err := uuid.Parse(idStr)
		if err == nil {
			teamA = append(teamA, id)
		}
	}

	var teamB []uuid.UUID
	for _, idStr := range body.TeamBPlayerIDs {
		id, err := uuid.Parse(idStr)
		if err == nil {
			teamB = append(teamB, id)
		}
	}

	newMatch, err := h.createUC.Execute(c.Context(), tID, body.MatchType, teamA, teamB)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Score modal requests come without HX-Request header — return JSON with match ID
	if c.Get("HX-Request") == "" {
		return c.JSON(fiber.Map{"id": newMatch.ID.String()})
	}
	// HTMX requests get the rendered row
	return c.Render("admin/partials/match-row", newMatch)
}

func (h *MatchHandler) Finish(c *fiber.Ctx) error {
	var body struct {
		MatchID    string `json:"matchId" form:"matchId"`
		WinnerTeam string `json:"winnerTeam" form:"winnerTeam"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	matchID, _ := uuid.Parse(body.MatchID)

	// In a real application, we would fetch the match from the repository here.
	// Since this handler creates a dummy match. Let's assume we fetch it so the compiler doesn't complain.
	// For now we just instantiate a mock one to satisfy the finishUC.
	m := &tournament.Match{
		ID:         matchID,
		TeamA:      []*player.Player{},
		TeamB:      []*player.Player{},
		Status:     "in_progress",
		WinnerTeam: "",
	}

	if err := h.finishUC.Execute(m, body.WinnerTeam); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(m)
}
// UpdateScore accepts set scores via JSON/form and persists them, auto-resolving winner.
func (h *MatchHandler) UpdateScore(c *fiber.Ctx) error {
	matchID := c.Params("id")
	var body struct {
		TournamentID string   `json:"tournamentId" form:"tournamentId"`
		Stage        string   `json:"stage" form:"stage"`
		Scores       []string `json:"scores" form:"scores[]"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	// Also accept form multi-values
	if len(body.Scores) == 0 {
		for _, s := range c.Request().PostArgs().PeekMulti("scores[]") {
			body.Scores = append(body.Scores, string(s))
		}
	}
	if body.Stage == "" {
		body.Stage = "group"
	}
	if err := h.updateScoreUC.Execute(c.Context(), matchID, body.Scores, body.TournamentID, body.Stage); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}
