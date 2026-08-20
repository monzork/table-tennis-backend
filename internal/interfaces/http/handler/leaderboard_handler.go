package handler

import (
	"html/template"
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
)

type LeaderboardHandler struct {
	getUC             *leaderboard.GetLeaderboardUseCase
	divisionUC        *division.DivisionUseCase
	getDistributionUC *leaderboard.GetDivisionDistributionUseCase
}

func NewLeaderboardHandler(uc *leaderboard.GetLeaderboardUseCase, divUC *division.DivisionUseCase, distUC *leaderboard.GetDivisionDistributionUseCase) *LeaderboardHandler {
	return &LeaderboardHandler{getUC: uc, divisionUC: divUC, getDistributionUC: distUC}
}

type DivisionGroup struct {
	Division *divisionDomain.Division
	Players  []*player.Player
}

func (h *LeaderboardHandler) getGroupedPlayers(c *fiber.Ctx, rankType string) ([]DivisionGroup, error) {
	var players []*player.Player
	var divisions []*divisionDomain.Division
	var pErr, dErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		players, pErr = h.getUC.Execute(c.Context(), rankType)
	}()
	go func() {
		defer wg.Done()
		divisions, dErr = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if pErr != nil {
		return nil, pErr
	}
	if dErr != nil {
		return nil, dErr
	}

	var filteredDivisions []*divisionDomain.Division
	for _, d := range divisions {
		if d.ID != "none" && d.Name != "No Division" {
			filteredDivisions = append(filteredDivisions, d)
		}
	}
	divisions = filteredDivisions

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
	var players []*player.Player
	var divisions []*divisionDomain.Division
	var pErr, dErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		players, pErr = h.getUC.ExecuteByGender(c.Context(), rankType, gender)
	}()
	go func() {
		defer wg.Done()
		divisions, dErr = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if pErr != nil {
		return nil, pErr
	}
	if dErr != nil {
		return nil, dErr
	}

	var filteredDivisions []*divisionDomain.Division
	for _, d := range divisions {
		if d.ID != "none" && d.Name != "No Division" {
			filteredDivisions = append(filteredDivisions, d)
		}
	}
	divisions = filteredDivisions

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

func (h *LeaderboardHandler) renderRanking(c *fiber.Ctx, rankType string, title string) error {
	query := c.Query("q")
	divFilter := c.Query("division")
	sortOrder := c.Query("sort", "points_desc")

	var players []*player.Player
	var divisions []*divisionDomain.Division
	var pErr, dErr error
	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		players, pErr = h.getUC.Execute(c.Context(), rankType)
	}()
	go func() {
		defer wg.Done()
		divisions, dErr = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if pErr != nil {
		return ErrorHandler(pErr)
	}
	if dErr != nil {
		return ErrorHandler(dErr)
	}

	result := leaderboard.BuildRanking(players, divisions, leaderboard.RankingParams{
		RankType:       rankType,
		Query:          query,
		DivisionFilter: divFilter,
		SortOrder:      sortOrder,
	})

	lang := getLang(c)
	tMap := i18n.PrecomputedMaps[lang]

	data := fiber.Map{
		"Groups":       result.Groups,
		"Type":         title,
		"RankType":     rankType,
		"ActiveTab":    rankType,
		"Query":        query,
		"Division":     divFilter,
		"Sort":         sortOrder,
		"IsDivisional": result.IsDivisional,
		"Divisions":    divisions,
		"CurrentPath":  c.Path(),
		"T":            tMap,
		"Lang":         lang,
		"Title":        title,
	}

	if c.Get("HX-Request") == "true" && c.Get("HX-Boosted") != "true" {
		// Fragment responses (search/filter/sort keystrokes) never compute the
		// distribution chart -- it would otherwise be recomputed on every
		// request instead of once per full page load.
		return c.Render("partials/rankings-container", data)
	}

	distSVG, _ := h.getDistributionUC.Execute(players, divisions, rankType)
	data["DistributionSVG"] = template.HTML(distSVG)

	return c.Render("rankings", data, "layouts/public")
}

func (h *LeaderboardHandler) GetSingles(c *fiber.Ctx) error {
	return h.renderRanking(c, "singles", "Singles")
}

func (h *LeaderboardHandler) GetDoubles(c *fiber.Ctx) error {
	return h.renderRanking(c, "doubles", "Doubles")
}
