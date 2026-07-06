package handler

import (
	"sort"
	"strings"
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/interfaces/http/i18n"

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

func groupPlayers(players []RankedPlayer, rankType string, isDivisional bool, divisions []*divisionDomain.Division) []DivisionGroupView {
	var groups []DivisionGroupView
	if isDivisional {
		for _, div := range divisions {
			var divPlayers []RankedPlayer
			for _, rp := range players {
				elo := rp.SinglesElo
				if rankType == "doubles" {
					elo = rp.DoublesElo
				}
				if div.ContainsElo(elo) {
					divPlayers = append(divPlayers, rp)
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
		groups = []DivisionGroupView{{
			Division: nil,
			Players:  players,
		}}
	}
	return groups
}

func (h *LeaderboardHandler) renderRanking(c *fiber.Ctx, rankType string, gender string, title string) error {
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
		players, pErr = h.getUC.ExecuteByGender(c.Context(), rankType, gender)
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

	var filteredDivisions []*divisionDomain.Division
	for _, d := range divisions {
		if d.ID != "none" && d.Name != "No Division" {
			filteredDivisions = append(filteredDivisions, d)
		}
	}
	divisions = filteredDivisions

	isMixed := gender == ""

	// 0. Pre-rank all players by absolute Elo
	var preRankedPlayers []RankedPlayer
	if isMixed {
		var men, women []*player.Player
		for _, p := range players {
			if strings.ToUpper(p.Gender) == "M" {
				men = append(men, p)
			} else if strings.ToUpper(p.Gender) == "F" {
				women = append(women, p)
			}
		}
		sort.Slice(men, func(i, j int) bool {
			if rankType == "doubles" {
				return men[i].DoublesElo > men[j].DoublesElo
			}
			return men[i].SinglesElo > men[j].SinglesElo
		})
		for i, p := range men {
			preRankedPlayers = append(preRankedPlayers, RankedPlayer{Player: p, Rank: i + 1})
		}
		sort.Slice(women, func(i, j int) bool {
			if rankType == "doubles" {
				return women[i].DoublesElo > women[j].DoublesElo
			}
			return women[i].SinglesElo > women[j].SinglesElo
		})
		for i, p := range women {
			preRankedPlayers = append(preRankedPlayers, RankedPlayer{Player: p, Rank: i + 1})
		}
	} else {
		sort.Slice(players, func(i, j int) bool {
			if rankType == "doubles" {
				return players[i].DoublesElo > players[j].DoublesElo
			}
			return players[i].SinglesElo > players[j].SinglesElo
		})
		for i, p := range players {
			preRankedPlayers = append(preRankedPlayers, RankedPlayer{Player: p, Rank: i + 1})
		}
	}

	// 1. Filter by Search Query (Name, Country, or Department)
	var filteredPlayers []RankedPlayer
	if query != "" {
		qUpper := strings.ToUpper(query)
		for _, rp := range preRankedPlayers {
			fullName := strings.ToUpper(rp.FirstName + " " + rp.LastName)
			country := strings.ToUpper(rp.Country)
			dept := strings.ToUpper(rp.Department)
			if strings.Contains(fullName, qUpper) || strings.Contains(country, qUpper) || strings.Contains(dept, qUpper) {
				filteredPlayers = append(filteredPlayers, rp)
			}
		}
	} else {
		filteredPlayers = preRankedPlayers
	}

	// 2. Filter by Division
	var finalPlayers []RankedPlayer
	if divFilter != "" && divFilter != "all" {
		var targetDiv *divisionDomain.Division
		for _, d := range divisions {
			if d.Name == divFilter {
				targetDiv = d
				break
			}
		}
		if targetDiv != nil {
			for _, rp := range filteredPlayers {
				elo := rp.SinglesElo
				if rankType == "doubles" {
					elo = rp.DoublesElo
				}
				if targetDiv.ContainsElo(elo) {
					finalPlayers = append(finalPlayers, rp)
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
		rpA, rpB := finalPlayers[i], finalPlayers[j]
		if sortOrder == "name_asc" {
			return (rpA.FirstName + rpA.LastName) < (rpB.FirstName + rpB.LastName)
		}
		ptsA := rpA.SinglesElo
		ptsB := rpB.SinglesElo
		if rankType == "doubles" {
			ptsA = rpA.DoublesElo
			ptsB = rpB.DoublesElo
		}
		if ptsA == ptsB {
			if sortOrder == "points_asc" {
				return rpA.Rank > rpB.Rank
			}
			return rpA.Rank < rpB.Rank
		}
		if sortOrder == "points_asc" {
			return ptsA < ptsB
		}
		return ptsA > ptsB // Default points_desc
	})

	// 4. Group
	isDivisional := sortOrder == "points_desc" && query == "" && (divFilter == "" || divFilter == "all") && len(divisions) > 0

	var menGroups []DivisionGroupView
	var womenGroups []DivisionGroupView
	var groups []DivisionGroupView

	if isMixed {
		var menPlayers []RankedPlayer
		var womenPlayers []RankedPlayer
		for _, rp := range finalPlayers {
			if strings.ToUpper(rp.Gender) == "M" {
				menPlayers = append(menPlayers, rp)
			} else if strings.ToUpper(rp.Gender) == "F" {
				womenPlayers = append(womenPlayers, rp)
			}
		}
		menGroups = groupPlayers(menPlayers, rankType, isDivisional, divisions)
		womenGroups = groupPlayers(womenPlayers, rankType, isDivisional, divisions)
	} else {
		groups = groupPlayers(finalPlayers, rankType, isDivisional, divisions)
	}

	lang := getLang(c)
	tMap := i18n.PrecomputedMaps[lang]

	data := fiber.Map{
		"Groups":       groups,
		"MenGroups":    menGroups,
		"WomenGroups":  womenGroups,
		"IsMixed":      isMixed,
		"Type":         title,
		"RankType":     rankType,
		"Gender":       gender,
		"ActiveTab":    gender + "-" + rankType,
		"Query":        query,
		"Division":     divFilter,
		"Sort":         sortOrder,
		"IsDivisional": isDivisional,
		"Divisions":    divisions,
		"CurrentPath":  c.Path(),
		"T":            tMap,
		"Lang":         lang,
		"Title":        title,
	}

	if c.Get("HX-Request") == "true" && c.Get("HX-Boosted") != "true" {
		return c.Render("partials/rankings-container", data)
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
