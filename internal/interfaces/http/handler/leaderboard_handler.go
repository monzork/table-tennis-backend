package handler

import (
	"fmt"
	"html/template"
	"strings"
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
)

// RenderRankMovement renders the public leaderboard's rank-change indicator
// -- how many places a player has moved, within their own gender, since the
// Elo snapshot before the single most recently finished tournament
// (leaderboard.RankedPlayer.RankDelta). A nil delta (player wasn't in that
// tournament) renders nothing.
func RenderRankMovement(delta *int) template.HTML {
	if delta == nil {
		return ""
	}
	switch {
	case *delta > 0:
		return template.HTML(fmt.Sprintf(`<span class="inline-flex items-center gap-0.5 text-emerald-400 font-bold text-xs whitespace-nowrap"><svg class="w-3 h-3 shrink-0" fill="currentColor" viewBox="0 0 24 24"><path d="M12 4l8 10h-6v6h-4v-6H4z"/></svg>%d</span>`, *delta))
	case *delta < 0:
		return template.HTML(fmt.Sprintf(`<span class="inline-flex items-center gap-0.5 text-red-400 font-bold text-xs whitespace-nowrap"><svg class="w-3 h-3 shrink-0" fill="currentColor" viewBox="0 0 24 24"><path d="M12 20l-8-10h6V4h4v6h6z"/></svg>%d</span>`, -*delta))
	default:
		return template.HTML(`<span class="text-gray-500 text-xs font-bold">–</span>`)
	}
}

type LeaderboardHandler struct {
	getUC             *leaderboard.GetLeaderboardUseCase
	divisionUC        *division.DivisionUseCase
	getDistributionUC *leaderboard.GetDivisionDistributionUseCase
	rankMovementUC    *leaderboard.GetRankMovementUseCase
}

func NewLeaderboardHandler(uc *leaderboard.GetLeaderboardUseCase, divUC *division.DivisionUseCase, distUC *leaderboard.GetDivisionDistributionUseCase, rankMovementUC *leaderboard.GetRankMovementUseCase) *LeaderboardHandler {
	return &LeaderboardHandler{getUC: uc, divisionUC: divUC, getDistributionUC: distUC, rankMovementUC: rankMovementUC}
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
	view := c.Query("view", "overall")
	genderFilter := strings.ToUpper(c.Query("gender", "M"))
	if genderFilter != "M" && genderFilter != "F" {
		genderFilter = "M"
	}

	var players []*player.Player
	var divisions []*divisionDomain.Division
	var previousElo map[string]int16
	var pErr, dErr error
	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		players, pErr = h.getUC.Execute(c.Context(), rankType)
	}()
	go func() {
		defer wg.Done()
		divisions, dErr = h.divisionUC.GetAll(c.Context())
	}()
	go func() {
		defer wg.Done()
		// Rank movement is a nice-to-have indicator, not core ranking data --
		// an error here is swallowed rather than failing the whole page.
		previousElo, _ = h.rankMovementUC.Execute(c.Context(), rankType)
	}()
	wg.Wait()

	if pErr != nil {
		return ErrorHandler(pErr)
	}
	if dErr != nil {
		return ErrorHandler(dErr)
	}

	rankingParams := leaderboard.RankingParams{
		RankType:       rankType,
		Query:          query,
		DivisionFilter: divFilter,
		SortOrder:      sortOrder,
		GenderFilter:   genderFilter,
		PreviousElo:    previousElo,
	}

	var result leaderboard.RankingResult
	if view == "gender" {
		result = leaderboard.BuildGenderRanking(players, divisions, rankingParams)
	} else {
		result = leaderboard.BuildRanking(players, divisions, rankingParams)
	}

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
		"View":         view,
		"Gender":       genderFilter,
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

	// The distribution chart only shows the gender-specific bands -- the
	// legacy gender-agnostic bands stay in the DB for existing tournaments'
	// bracket rendering, but are retired from every new-facing display.
	distSVG, _ := h.getDistributionUC.Execute(players, divisionDomain.OnlyGendered(divisions), rankType)
	data["DistributionSVG"] = template.HTML(distSVG)

	return c.Render("rankings", data, "layouts/public")
}

func (h *LeaderboardHandler) GetSingles(c *fiber.Ctx) error {
	return h.renderRanking(c, "singles", "Singles")
}

func (h *LeaderboardHandler) GetDoubles(c *fiber.Ctx) error {
	return h.renderRanking(c, "doubles", "Doubles")
}
