package handler

import (
	"sort"
	"strings"
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

type RankedPlayer struct {
	*player.Player
	Rank int
}

type DivisionGroupView struct {
	Division *divisionDomain.Division
	Players  []RankedPlayer
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
	query := c.Query("q")
	divFilter := c.Query("division")
	sortOrder := c.Query("sort", "points_desc")

	players, err := h.getUC.ExecuteByGender(c.Context(), rankType, gender)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	divisions, err := h.divisionUC.GetAll(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// 1. Filter by Search Query (Name or Country)
	var filteredPlayers []*player.Player
	if query != "" {
		qUpper := strings.ToUpper(query)
		for _, p := range players {
			fullName := strings.ToUpper(p.FirstName + " " + p.LastName)
			country := strings.ToUpper(p.Country)
			if strings.Contains(fullName, qUpper) || strings.Contains(country, qUpper) {
				filteredPlayers = append(filteredPlayers, p)
			}
		}
	} else {
		filteredPlayers = players
	}

	// 2. Filter by Division
	var finalPlayers []*player.Player
	if divFilter != "" && divFilter != "all" {
		var targetDiv *divisionDomain.Division
		for _, d := range divisions {
			if d.Name == divFilter {
				targetDiv = d
				break
			}
		}
		if targetDiv != nil {
			for _, p := range filteredPlayers {
				elo := p.SinglesElo
				if rankType == "doubles" {
					elo = p.DoublesElo
				}
				if targetDiv.ContainsElo(elo) {
					finalPlayers = append(finalPlayers, p)
				}
			}
		} else {
			finalPlayers = filteredPlayers
		}
	} else {
		finalPlayers = filteredPlayers
	}

	// 3. Sort
	sort.Slice(finalPlayers, func(i, j int) bool {
		pA, pB := finalPlayers[i], finalPlayers[j]
		if sortOrder == "name_asc" {
			return (pA.FirstName + pA.LastName) < (pB.FirstName + pB.LastName)
		}
		ptsA := pA.SinglesElo
		ptsB := pB.SinglesElo
		if rankType == "doubles" {
			ptsA = pA.DoublesElo
			ptsB = pB.DoublesElo
		}
		if sortOrder == "points_asc" {
			return ptsA < ptsB
		}
		return ptsA > ptsB // Default points_desc
	})

	// 4. Group and Rank
	isDivisional := sortOrder == "points_desc" && query == "" && (divFilter == "" || divFilter == "all")
	var groups []DivisionGroupView
	
	globalRank := 1
	if isDivisional {
		for _, div := range divisions {
			var divPlayers []RankedPlayer
			for _, p := range finalPlayers {
				elo := p.SinglesElo
				if rankType == "doubles" {
					elo = p.DoublesElo
				}
				if div.ContainsElo(elo) {
					divPlayers = append(divPlayers, RankedPlayer{Player: p, Rank: globalRank})
					globalRank++
				}
			}
			if len(divPlayers) > 0 {
				groups = append(groups, DivisionGroupView{
					Division: div,
					Players:  divPlayers,
				})
			}
		}
	} else {
		// Flat list
		var rankedPlayers []RankedPlayer
		for _, p := range finalPlayers {
			rankedPlayers = append(rankedPlayers, RankedPlayer{Player: p, Rank: globalRank})
			globalRank++
		}
		groups = []DivisionGroupView{{
			Division: nil,
			Players:  rankedPlayers,
		}}
	}

	data := fiber.Map{
		"Groups":        groups,
		"Type":          title,
		"RankType":      rankType,
		"Gender":        gender,
		"ActiveTab":     gender + "-" + rankType,
		"Query":         query,
		"Division":      divFilter,
		"Sort":          sortOrder,
		"IsDivisional":  isDivisional,
		"Divisions":     divisions,
		"CurrentPath":   c.Path(),
	}

	if c.Get("HX-Request") == "true" {
		return c.Render("partials/rankings-table", data)
	}

	return c.Render("rankings", data, "layouts/public")
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
