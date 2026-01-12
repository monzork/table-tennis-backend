package handler

import (
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MatchHandler struct {
	createUC *match.CreateMatchUseCase
	finishUC *match.FinishMatchUseCase
}

func NewMatchHandler(createUC *match.CreateMatchUseCase, finishUC *match.FinishMatchUseCase) *MatchHandler {
	return &MatchHandler{
		finishUC: finishUC,
		createUC: createUC,
	}
}

func (h *MatchHandler) Create(c *fiber.Ctx) error {
	var body struct {
		TournamentID string `json:"tournamentId"`
		PlayerAID    string `json:"playerAId"`
		PlayerBID    string `json:"playerBId"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	tID, err := uuid.Parse(body.TournamentID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tournament id")
	}
	pAID, err := uuid.Parse(body.PlayerAID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid player A id")
	}
	pBID, err := uuid.Parse(body.PlayerBID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid player B id")
	}

	newMatch, err := h.createUC.Execute(c.Context(), tID, pAID, pBID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Return rendered match row for HTMX
	return c.Render("partials/match-row", newMatch)
}

func (h *MatchHandler) Finish(c *fiber.Ctx) error {
	var body struct {
		MatchID  string `json:"matchId"`
		WinnerID string `json:"winnerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	matchID, _ := uuid.Parse(body.MatchID)
	winnerID, _ := uuid.Parse(body.WinnerID)

	// fetch match from repo (omitted for brevity, assume in-memory or DB)
	m := &tournament.Match{
		ID:      matchID,
		Players: []*player.Player{},
		Status:  "in_progress",
	}

	if err := h.finishUC.Execute(m, winnerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(m)
}
