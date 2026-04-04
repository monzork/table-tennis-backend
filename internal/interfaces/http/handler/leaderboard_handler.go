package handler

import (
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"

	"github.com/gofiber/fiber/v2"
)

type LeaderboardHandler struct {
	getUC      *leaderboard.GetLeaderboardUseCase
	divisionUC *division.DivisionUseCase
}

func NewLeaderboardHandler(uc *leaderboard.GetLeaderboardUseCase, divUC *division.DivisionUseCase) *LeaderboardHandler {
	return &LeaderboardHandler{getUC: uc, divisionUC: divUC}
}

type DivisionGroup struct {
	Division *divisionDomain.Division
	Players  []*player.Player
}

func (h *LeaderboardHandler) getGroupedPlayers(c *fiber.Ctx, rankType string) ([]DivisionGroup, error) {
	players, err := h.getUC.Execute(c.Context(), rankType)
	if err != nil {
		return nil, err
	}

	divisions, err := h.divisionUC.GetAll(c.Context())
	if err != nil {
		return nil, err
	}

	var groups []DivisionGroup
	
	for _, div := range divisions {
		var divPlayers []*player.Player
		for _, p := range players {
			elo := p.SinglesElo
			if rankType == "doubles" {
				elo = p.DoublesElo
			}
			if div.ContainsElo(elo) {
				divPlayers = append(divPlayers, p)
			}
		}
		
		if len(divPlayers) > 0 {
			groups = append(groups, DivisionGroup{
				Division: div,
				Players:  divPlayers,
			})
		}
	}

	return groups, nil
}

func (h *LeaderboardHandler) getGroupedPlayersByGender(c *fiber.Ctx, rankType string, gender string) ([]DivisionGroup, error) {
	players, err := h.getUC.ExecuteByGender(c.Context(), rankType, gender)
	if err != nil {
		return nil, err
	}

	divisions, err := h.divisionUC.GetAll(c.Context())
	if err != nil {
		return nil, err
	}

	var groups []DivisionGroup
	for _, div := range divisions {
		var divPlayers []*player.Player
		for _, p := range players {
			elo := p.SinglesElo
			if rankType == "doubles" {
				elo = p.DoublesElo
			}
			if div.ContainsElo(elo) {
				divPlayers = append(divPlayers, p)
			}
		}
		if len(divPlayers) > 0 {
			groups = append(groups, DivisionGroup{
				Division: div,
				Players:  divPlayers,
			})
		}
	}
	return groups, nil
}

func (h *LeaderboardHandler) renderRanking(c *fiber.Ctx, rankType string, gender string, title string) error {
	groups, err := h.getGroupedPlayersByGender(c, rankType, gender)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Render("rankings", fiber.Map{
		"Groups":        groups,
		"Type":          title,
		"RankType":      rankType,
		"Gender":        gender,
		"ActiveTab":     gender + "-" + rankType,
	}, "layouts/public")
}

func (h *LeaderboardHandler) GetSingles(c *fiber.Ctx) error {
	return h.renderRanking(c, "singles", "", "Singles")
}

func (h *LeaderboardHandler) GetDoubles(c *fiber.Ctx) error {
	return h.renderRanking(c, "doubles", "", "Doubles")
}

func (h *LeaderboardHandler) GetMensSingles(c *fiber.Ctx) error {
	return h.renderRanking(c, "singles", "M", "Men's Singles")
}

func (h *LeaderboardHandler) GetWomensSingles(c *fiber.Ctx) error {
	return h.renderRanking(c, "singles", "F", "Women's Singles")
}

func (h *LeaderboardHandler) GetMensDoubles(c *fiber.Ctx) error {
	return h.renderRanking(c, "doubles", "M", "Men's Doubles")
}

func (h *LeaderboardHandler) GetWomensDoubles(c *fiber.Ctx) error {
	return h.renderRanking(c, "doubles", "F", "Women's Doubles")
}

func (h *LeaderboardHandler) GetMixedDoubles(c *fiber.Ctx) error {
	return h.renderRanking(c, "doubles", "", "Mixed Doubles")
}
