package handler

import (
	"context"
	"fmt"

	"strings"
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/bracket"
	divisionDomain "table-tennis-backend/internal/domain/division"
	tournamentDomain "table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	"time"

	"github.com/gofiber/fiber/v2"
)

type EventHandler struct {
	createUC               *event.CreateTournamentUseCase
	getByID                *event.GetTournamentByIDUseCase
	updateUC               *event.UpdateTournamentUseCase
	deleteUC               *event.DeleteTournamentUseCase
	leaderboardUC          *leaderboard.GetLeaderboardUseCase
	divisionUC             *division.DivisionUseCase
	finishUC               *event.FinishTournamentUseCase
	exportUC               *event.ExportTournamentReportUseCase
	exportPdfUC            *event.ExportTournamentPdfUseCase
	movePlayerUC           *event.MovePlayerUseCase
	createTeamUC           *event.CreateTeamUseCase
	deleteTeamUC           *event.DeleteTeamUseCase
	assignPlayerToTeamUC   *event.AssignPlayerToTeamUseCase
	removePlayerFromTeamUC *event.RemovePlayerFromTeamUseCase
	getTournamentsUC       *event.GetTournamentsUseCase
	getOccupiedTablesUC    *event.GetOccupiedTablesUseCase
	regenerateSeedsUC      *event.RegenerateGroupSeedsUseCase
	updateParticipantEloUC *event.UpdateParticipantEloBeforeUseCase
	removeParticipantUC    *event.RemoveParticipantUseCase
	saveKnockoutSeedsUC    *event.SaveKnockoutSeedsUseCase
	toggleSeedingLockUC    *event.ToggleSeedingLockUseCase
	addGroupUC             *event.AddGroupUseCase
	recalculateEloUC       *event.RecalculateTournamentEloUseCase
	startKnockoutUC        *event.StartKnockoutStageUseCase
	getDetailViewUC        *event.GetEventDetailViewUseCase
	getPublicDetailViewUC  *event.GetPublicEventDetailViewUseCase
	tvDashboardUC          *event.GetPublicTVDashboardViewUseCase
	boardViewUC            *event.GetBoardViewUseCase
	editFormViewUC         *event.GetEditFormViewUseCase
}

func NewEventHandler(
	createUC *event.CreateTournamentUseCase,
	getByID *event.GetTournamentByIDUseCase,
	updateUC *event.UpdateTournamentUseCase,
	deleteUC *event.DeleteTournamentUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	divisionUC *division.DivisionUseCase,
	finishUC *event.FinishTournamentUseCase,
	exportUC *event.ExportTournamentReportUseCase,
	exportPdfUC *event.ExportTournamentPdfUseCase,
	movePlayerUC *event.MovePlayerUseCase,
	createTeamUC *event.CreateTeamUseCase,
	deleteTeamUC *event.DeleteTeamUseCase,
	assignPlayerToTeamUC *event.AssignPlayerToTeamUseCase,
	removePlayerFromTeamUC *event.RemovePlayerFromTeamUseCase,
	getTournamentsUC *event.GetTournamentsUseCase,
	getOccupiedTablesUC *event.GetOccupiedTablesUseCase,
	regenerateSeedsUC *event.RegenerateGroupSeedsUseCase,
	updateParticipantEloUC *event.UpdateParticipantEloBeforeUseCase,
	removeParticipantUC *event.RemoveParticipantUseCase,
	saveKnockoutSeedsUC *event.SaveKnockoutSeedsUseCase,
	toggleSeedingLockUC *event.ToggleSeedingLockUseCase,
	addGroupUC *event.AddGroupUseCase,
	recalculateEloUC *event.RecalculateTournamentEloUseCase,
	startKnockoutUC *event.StartKnockoutStageUseCase,
	getDetailViewUC *event.GetEventDetailViewUseCase,
	getPublicDetailViewUC *event.GetPublicEventDetailViewUseCase,
	tvDashboardUC *event.GetPublicTVDashboardViewUseCase,
	boardViewUC *event.GetBoardViewUseCase,
	editFormViewUC *event.GetEditFormViewUseCase,
) *EventHandler {
	return &EventHandler{
		createUC:               createUC,
		getByID:                getByID,
		updateUC:               updateUC,
		deleteUC:               deleteUC,
		leaderboardUC:          leaderboardUC,
		divisionUC:             divisionUC,
		finishUC:               finishUC,
		exportUC:               exportUC,
		exportPdfUC:            exportPdfUC,
		movePlayerUC:           movePlayerUC,
		createTeamUC:           createTeamUC,
		deleteTeamUC:           deleteTeamUC,
		assignPlayerToTeamUC:   assignPlayerToTeamUC,
		removePlayerFromTeamUC: removePlayerFromTeamUC,
		getTournamentsUC:       getTournamentsUC,
		getOccupiedTablesUC:    getOccupiedTablesUC,
		regenerateSeedsUC:      regenerateSeedsUC,
		updateParticipantEloUC: updateParticipantEloUC,
		removeParticipantUC:    removeParticipantUC,
		saveKnockoutSeedsUC:    saveKnockoutSeedsUC,
		toggleSeedingLockUC:    toggleSeedingLockUC,
		addGroupUC:             addGroupUC,
		recalculateEloUC:       recalculateEloUC,
		startKnockoutUC:        startKnockoutUC,
		getDetailViewUC:        getDetailViewUC,
		getPublicDetailViewUC:  getPublicDetailViewUC,
		tvDashboardUC:          tvDashboardUC,
		boardViewUC:            boardViewUC,
		editFormViewUC:         editFormViewUC,
	}
}

func (h *EventHandler) StartKnockout(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	divID := c.Params("divId")

	err := h.startKnockoutUC.Execute(c.Context(), tournamentID, divID)
	if err != nil {
		return ErrorHandler(err)
	}

	c.Set("HX-Trigger", `{"show-toast": {"message": "Knockout matches created and scheduled!", "type": "success"}}`)
	return c.SendString("")
}


func (h *EventHandler) Create(c *fiber.Ctx) error {
	cmd, err := parseCreateEventCommand(c)
	if err != nil {
		return ErrorHandler(err)
	}

	t, err := h.createUC.Execute(c.Context(), cmd)
	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render("admin/partials/event-row", t)
}

func (h *EventHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")
	statusFilter := c.Query("status", "all")
	playerSearch := c.Query("player_search", "")

	view, err := h.getDetailViewUC.Execute(c.Context(), id, statusFilter, playerSearch)
	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render("admin/event-detail", fiber.Map{
		"Event":                 view.Event,
		"Players":               view.Players,
		"Divisions":             view.Divisions,
		"BracketViewModel":      view.BracketViewModel,
		"AvailableParticipants": view.AvailableParticipants,
		"StatusFilter":          statusFilter,
		"PlayerSearch":          playerSearch,
		"PlayerPins":            view.PlayerPins,
		"Officials":             view.Officials,
		"ParticipantRows":       view.ParticipantRows,
	}, "layouts/admin")
}

func (h *EventHandler) AddOfficial(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}
	if err := h.getByID.AddOfficial(c.Context(), tournamentID, body.PlayerID); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) RemoveOfficial(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	playerID := c.Params("playerId")
	if err := h.getByID.RemoveOfficial(c.Context(), tournamentID, playerID); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) RemoveParticipant(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	playerID := c.Params("playerId")
	if err := h.removeParticipantUC.Execute(c.Context(), tournamentID, playerID); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	view, err := h.editFormViewUC.Execute(c.Context(), id)
	if err != nil {
		return ErrorHandler(err)
	}
	return c.Render("admin/partials/event-edit-form", fiber.Map{
		"Event":     view.Event,
		"Players":   view.Players,
		"Divisions": view.Divisions,
	})
}


func (h *EventHandler) Update(c *fiber.Ctx) error {
	cmd, err := parseUpdateEventCommand(c)
	if err != nil {
		return ErrorHandler(err)
	}
	t, err := h.updateUC.Execute(c.Context(), cmd)
	if err != nil {
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Render("admin/partials/event-row", t)
}

func (h *EventHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		if c.Get("HX-Current-URL") != "" && fmt.Sprintf("/admin/events/%s", id) == c.Get("HX-Current-URL") {
			c.Set("HX-Redirect", "/admin/events")
		}
		return c.SendString("")
	}
	return c.SendString("")
}

func (h *EventHandler) Finish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.finishUC.Execute(c.Context(), idStr); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "finished"})
}

func (h *EventHandler) RegenerateGroupSeeds(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.regenerateSeedsUC.Execute(c.Context(), idStr); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "regenerated"})
}

func (h *EventHandler) UpdateParticipantEloBefore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var body struct {
		PlayerID   string `form:"playerId"`
		SinglesElo int16  `form:"singlesElo"`
		DoublesElo int16  `form:"doublesElo"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}
	if body.PlayerID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "playerId is required")
	}
	if err := h.updateParticipantEloUC.Execute(c.Context(), idStr, body.PlayerID, body.SinglesElo, body.DoublesElo); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

func (h *EventHandler) Export(c *fiber.Ctx) error {
	idStr := c.Params("id")
	csvBytes, err := h.exportUC.Execute(c.Context(), idStr)
	if err != nil {
		return ErrorHandler(err)
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_report_%s.csv\"", idStr))
	return c.Send(csvBytes)
}

func (h *EventHandler) ExportPDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pdfBytes, err := h.exportPdfUC.Execute(c.Context(), idStr)
	if err != nil {
		return ErrorHandler(err)
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"event_report_%s.pdf\"", idStr))
	return c.Send(pdfBytes)
}

func (h *EventHandler) MovePlayer(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		PlayerID      string `json:"playerId" form:"playerId"`
		TargetGroupID string `json:"targetGroupId" form:"targetGroupId"`
		TargetIndex   *int   `json:"targetIndex" form:"targetIndex"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	targetIndex := -1
	if body.TargetIndex != nil {
		targetIndex = *body.TargetIndex
	}

	if err := h.movePlayerUC.Execute(c.Context(), id, body.PlayerID, body.TargetGroupID, targetIndex); err != nil {
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *EventHandler) SaveKnockoutSeeds(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		DivID     string `json:"divId" form:"divId"`
		PlayerIDs string `json:"playerIds" form:"playerIds"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	if err := h.saveKnockoutSeedsUC.Execute(c.Context(), id, body.DivID, body.PlayerIDs); err != nil {
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *EventHandler) AddGroup(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		DivisionName string `json:"divisionName" form:"divisionName"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	if err := h.addGroupUC.Execute(c.Context(), id, body.DivisionName); err != nil {
		return ErrorHandler(err)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *EventHandler) CreateTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		Name string `form:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}
	if _, err := h.createTeamUC.Execute(c.Context(), tournamentID, body.Name); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) DeleteTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	if err := h.deleteTeamUC.Execute(c.Context(), teamID); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) AssignPlayerToTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}
	if err := h.assignPlayerToTeamUC.Execute(c.Context(), teamID, body.PlayerID); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) RemovePlayerFromTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	playerID := c.Params("playerId")
	if err := h.removePlayerFromTeamUC.Execute(c.Context(), teamID, playerID); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/events/%s", tournamentID))
}

func (h *EventHandler) PublicList(c *fiber.Ctx) error {
	lang := getLang(c)
	events, err := h.getTournamentsUC.Execute(c.Context())
	if err != nil {
		return ErrorHandler(err)
	}
	return c.Render("public/events", merge(tMap(lang), fiber.Map{
		"Events":       events,
		"Type":         "Events",
		"OGImage":      c.BaseURL() + "/open_tdm.jpeg",
		"Title":        "Events",
		"CanonicalURL": c.BaseURL() + c.Path(),
	}), "layouts/public")
}

func (h *EventHandler) PublicDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")
	statusFilter := c.Query("status", "all")
	playerSearch := c.Query("player_search", "")
	canonicalURL := c.BaseURL() + c.Path()
	tmap, _ := c.Locals("T").(map[string]string)

	view, err := h.getPublicDetailViewUC.Execute(
		c.Context(), id, statusFilter, playerSearch, canonicalURL, BuildBoardCards, tmap,
	)
	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render("public/event-detail", merge(tMap(lang), fiber.Map{
		"Event":            view.Event,
		"Divisions":        view.Divisions,
		"BracketViewModel": view.BracketViewModel,
		"Type":             "Events",
		"StatusFilter":     statusFilter,
		"PlayerSearch":     playerSearch,
		"RefereeNames":     view.RefereeNames,
		"CanonicalURL":     canonicalURL,
		"OGImage":          c.BaseURL() + "/open_tdm.jpeg",
		"JSONLD":           view.JSONLD,
		"Title":            view.Event.Name,
		"Description":      fmt.Sprintf("%s Event. Register and view live bracket.", view.Event.Name),
	}), "layouts/public")
}

func (h *EventHandler) PublicTVDashboard(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")
	playerSearch := c.Query("player_search", "")
	tmap, _ := c.Locals("T").(map[string]string)

	view, err := h.tvDashboardUC.Execute(c.Context(), id, playerSearch, BuildBoardCards, tmap)
	if err != nil {
		return ErrorHandler(err)
	}

	tables := event.BuildTableVMs(view.Event, "", h.getOccupiedTables(c.Context(), view.Event))

	return c.Render("public/tv-dashboard", merge(tMap(lang), fiber.Map{
		"Event":            view.Event,
		"Divisions":        view.Divisions,
		"BracketViewModel": view.BracketViewModel,
		"Scheduled":        view.Scheduled,
		"InProgress":       view.InProgress,
		"Finished":         view.Finished,
		"Tables":           tables,
	})) // No layout for TV
}

// TableVM is a view model for a table's status. Defined in application/event/board_helpers.go.
// This alias keeps templates working without change.
type TableVM = event.TableVM

func (h *EventHandler) getOccupiedTables(ctx context.Context, t *tournamentDomain.Event) []int {
	occupiedList, _ := h.getOccupiedTablesUC.Execute(ctx, t)
	return occupiedList
}


func FilterBoardCards(cards []event.BoardCard, q string, divs []string) []event.BoardCard {
	if q == "" && len(divs) == 0 {
		return cards
	}

	divMap := make(map[string]bool)
	for _, d := range divs {
		divMap[d] = true
	}

	var filtered []event.BoardCard
	for _, card := range cards {
		matchesSearch := q == "" || strings.Contains(strings.ToLower(card.PlayerAName), q) ||
			strings.Contains(strings.ToLower(card.PlayerBName), q) ||
			strings.Contains(strings.ToLower(card.GroupName), q)
		matchesDiv := len(divMap) == 0 || divMap[card.DivisionName]

		if matchesSearch && matchesDiv {
			filtered = append(filtered, card)
		}
	}
	return filtered
}

func BuildBoardCards(t *tournamentDomain.Event, divs []*divisionDomain.Division) (scheduled, inProgress, finished []event.BoardCard) {
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

	findGroupName := func(playerID string) string {
		for _, g := range t.Groups {
			for _, p := range g.Players {
				if p.ID == playerID {
					name := g.Name
					if idx := strings.Index(g.Name, " - "); idx != -1 {
						name = g.Name[idx+3:]
					}
					return name
				}
			}
		}
		return ""
	}

	findDivisionName := func(playerID string) string {
		if playerID == "" {
			return ""
		}
		var targetPlayer *player.Player
		for _, p := range t.Participants {
			if p.ID == playerID {
				targetPlayer = p
				break
			}
		}
		if targetPlayer == nil {
			return ""
		}
		elo := targetPlayer.SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			elo = targetPlayer.DoublesElo
		}
		for _, d := range divs {
			if d.MinElo == 0 && d.MaxElo == nil {
				continue
			}
			if (d.Category == "both" || d.Category == t.Type) && elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
				return d.Name
			}
		}
		return "Open Bracket"
	}

	// 1. Process actual matches in database
	for i := range t.Matches {
		m := &t.Matches[i]
		if m.TeamMatchID != nil { // skip sub-matches
			continue
		}
		card := event.BoardCard{
			MatchID:     m.ID,
			Status:      m.Status,
			Stage:       m.Stage,
			BestOf:      t.GetEffectiveStageRule(m.Stage, m.DivisionID).BestOf,
			PlayerAName: nameOf(m.TeamA),
			PlayerBName: nameOf(m.TeamB),
			P1Id:        idOf(m.TeamA),
			P2Id:        idOf(m.TeamB),
			TableNumber: m.TableNumber,
			ScoreA:      m.ScoreA(),
			ScoreB:      m.ScoreB(),
			Pin:         m.Pin,
			RoundNumber: m.RoundNumber,
			Category:    t.EventCategory,
			GroupName: func() string {
				if len(m.TeamA) > 0 {
					return findGroupName(m.TeamA[0].ID)
				}
				return ""
			}(),
			DivisionName: func() string {
				if len(m.TeamA) > 0 {
					return findDivisionName(m.TeamA[0].ID)
				}
				return ""
			}(),
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

	// 2. Identify virtual matches that should be scheduled based on the format
	vm := bracket.BuildBracket(t, divs, nil)
	for _, dv := range vm.Divisions {
		if vm.Format == "round_robin" {
			for _, mv := range dv.RoundRobinMatches {
				if mv.Player1 != nil && mv.Player2 != nil {
					if !matchExists(t.Matches, mv.Player1.ID, mv.Player2.ID, mv.Stage) {
						groupName := findGroupName(mv.Player1.ID)
						scheduled = append(scheduled, event.BoardCard{
							MatchID:      "",
							Status:       "scheduled",
							Stage:        mv.Stage,
							BestOf:       mv.BestOf,
							PlayerAName:  mv.Player1.FirstNameWithSecond() + " " + mv.Player1.LastNameWithSecond(),
							PlayerBName:  mv.Player2.FirstNameWithSecond() + " " + mv.Player2.LastNameWithSecond(),
							P1Id:         mv.Player1.ID,
							P2Id:         mv.Player2.ID,
							TableNumber:  nil,
							ScoreA:       0,
							ScoreB:       0,
							Pin:          "",
							GroupName:    groupName,
							DivisionName: dv.Name,
							Category:     t.EventCategory,
						})
					}
				}
			}
		} else if vm.Format == "groups_elimination" {
			for _, g := range dv.Groups {
				for _, mv := range g.Matches {
					if mv.Player1 != nil && mv.Player2 != nil {
						if !matchExists(t.Matches, mv.Player1.ID, mv.Player2.ID, mv.Stage) {
							scheduled = append(scheduled, event.BoardCard{
								MatchID:      "",
								Status:       "scheduled",
								Stage:        mv.Stage,
								BestOf:       mv.BestOf,
								PlayerAName:  mv.Player1.FirstNameWithSecond() + " " + mv.Player1.LastNameWithSecond(),
								PlayerBName:  mv.Player2.FirstNameWithSecond() + " " + mv.Player2.LastNameWithSecond(),
								P1Id:         mv.Player1.ID,
								P2Id:         mv.Player2.ID,
								TableNumber:  nil,
								ScoreA:       0,
								ScoreB:       0,
								Pin:          "",
								GroupName:    g.Name,
								DivisionName: dv.Name,
								Category:     t.EventCategory,
							})
						}
					}
				}
			}
			if dv.AllGroupsFinished {
				for _, round := range dv.KnockoutRounds {
					for _, bmv := range round.Matches {
						if bmv.Player1 != nil && bmv.Player2 != nil && bmv.Player1.Player != nil && bmv.Player2.Player != nil {
							if !matchExists(t.Matches, bmv.Player1.Player.ID, bmv.Player2.Player.ID, bmv.Stage) {
								scheduled = append(scheduled, event.BoardCard{
									MatchID:      "",
									Status:       "scheduled",
									Stage:        bmv.Stage,
									BestOf:       bmv.BestOf,
									PlayerAName:  bmv.Player1.Player.FirstNameWithSecond() + " " + bmv.Player1.Player.LastNameWithSecond(),
									PlayerBName:  bmv.Player2.Player.FirstNameWithSecond() + " " + bmv.Player2.Player.LastNameWithSecond(),
									P1Id:         bmv.Player1.Player.ID,
									P2Id:         bmv.Player2.Player.ID,
									TableNumber:  nil,
									ScoreA:       0,
									ScoreB:       0,
									Pin:          "",
									GroupName:    "",
									DivisionName: dv.Name,
									Category:     t.EventCategory,
								})
							}
						}
					}
				}
			}
		} else if vm.Format == "elimination" {
			for _, round := range dv.KnockoutRounds {
				for _, bmv := range round.Matches {
					if bmv.Player1 != nil && bmv.Player2 != nil && bmv.Player1.Player != nil && bmv.Player2.Player != nil {
						if !matchExists(t.Matches, bmv.Player1.Player.ID, bmv.Player2.Player.ID, bmv.Stage) {
							scheduled = append(scheduled, event.BoardCard{
								MatchID:      "",
								Status:       "scheduled",
								Stage:        bmv.Stage,
								BestOf:       bmv.BestOf,
								PlayerAName:  bmv.Player1.Player.FirstNameWithSecond() + " " + bmv.Player1.Player.LastNameWithSecond(),
								PlayerBName:  bmv.Player2.Player.FirstNameWithSecond() + " " + bmv.Player2.Player.LastNameWithSecond(),
								P1Id:         bmv.Player1.Player.ID,
								P2Id:         bmv.Player2.Player.ID,
								TableNumber:  nil,
								ScoreA:       0,
								ScoreB:       0,
								Pin:          "",
								GroupName:    "",
								DivisionName: dv.Name,
								Category:     t.EventCategory,
							})
						}
					}
				}
			}
		}
	}

	// Track last activity to allow players to rest
	lastActivity := make(map[string]time.Time)
	for _, m := range t.Matches {
		if m.Status == "in_progress" || m.Status == "finished" {
			tAct := time.Time{}
			if m.UpdatedAt != nil {
				tAct = *m.UpdatedAt
			}
			for _, p := range m.TeamA {
				if lastActivity[p.ID].Before(tAct) {
					lastActivity[p.ID] = tAct
				}
			}
			for _, p := range m.TeamB {
				if lastActivity[p.ID].Before(tAct) {
					lastActivity[p.ID] = tAct
				}
			}
		}
	}

	// We'll simulate a clock starting from a baseline to pick the best next match
	// for maximum rest interleaving (especially interleaving different groups).
	var reordered []event.BoardCard
	var unstarted []event.BoardCard
	var virtualScheduled []event.BoardCard

	// Virtual matches (MatchID == "") are handled after real matches
	for _, c := range scheduled {
		if c.MatchID == "" {
			virtualScheduled = append(virtualScheduled, c)
		} else {
			unstarted = append(unstarted, c)
		}
	}

	simClock := time.Now().Add(24 * time.Hour) // start in future to override past matches

	scheduleMatchGreedy := func(pool *[]event.BoardCard) {
		for len(*pool) > 0 {
			bestIdx := -1
			var bestPenalty time.Time
			var bestSum int64

			for i, c := range *pool {
				t1 := lastActivity[c.P1Id]
				t2 := lastActivity[c.P2Id]

				penalty := t1
				if t2.After(t1) {
					penalty = t2
				}

				sum := t1.UnixNano() + t2.UnixNano()

				if bestIdx == -1 || penalty.Before(bestPenalty) {
					bestIdx = i
					bestPenalty = penalty
					bestSum = sum
				} else if penalty.Equal(bestPenalty) {
					if sum < bestSum {
						bestIdx = i
						bestPenalty = penalty
						bestSum = sum
					} else if sum == bestSum {
						if (*pool)[i].MatchID < (*pool)[bestIdx].MatchID {
							bestIdx = i
							bestPenalty = penalty
							bestSum = sum
						}
					}
				}
			}

			picked := (*pool)[bestIdx]
			reordered = append(reordered, picked)

			simClock = simClock.Add(time.Second) // advance simulated time
			if picked.P1Id != "" {
				lastActivity[picked.P1Id] = simClock
			}
			if picked.P2Id != "" {
				lastActivity[picked.P2Id] = simClock
			}

			// Remove from pool
			*pool = append((*pool)[:bestIdx], (*pool)[bestIdx+1:]...)
		}
	}

	// Schedule real matches first, then virtual matches
	scheduleMatchGreedy(&unstarted)
	scheduleMatchGreedy(&virtualScheduled)

	scheduled = reordered

	// Mark players currently in a match
	inMatchPlayers := make(map[string]bool)
	for _, c := range inProgress {
		if c.P1Id != "" {
			inMatchPlayers[c.P1Id] = true
		}
		if c.P2Id != "" {
			inMatchPlayers[c.P2Id] = true
		}
	}
	for i := range scheduled {
		if scheduled[i].P1Id != "" && inMatchPlayers[scheduled[i].P1Id] {
			scheduled[i].P1InMatch = true
		}
		if scheduled[i].P2Id != "" && inMatchPlayers[scheduled[i].P2Id] {
			scheduled[i].P2InMatch = true
		}
		scheduled[i].QueuePosition = i + 1
	}

	return
}

func matchExists(matches []tournamentDomain.Match, p1ID, p2ID string, stage string) bool {
	for _, m := range matches {
		if m.TeamMatchID != nil {
			continue
		}
		if m.Stage != stage {
			continue
		}
		if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
			if (m.TeamA[0].ID == p1ID && m.TeamB[0].ID == p2ID) || (m.TeamA[0].ID == p2ID && m.TeamB[0].ID == p1ID) {
				return true
			}
		}
	}
	return false
}

func (h *EventHandler) Board(c *fiber.Ctx) error {
	id := c.Params("id")
	q := c.Query("q", "")
	var selectedDivs []string
	for _, d := range c.Request().URI().QueryArgs().PeekMulti("div") {
		selectedDivs = append(selectedDivs, string(d))
	}

	view, err := h.boardViewUC.Execute(c.Context(), id, q, selectedDivs, BuildBoardCards, FilterBoardCards)
	if err != nil {
		return ErrorHandler(err)
	}

	tables := event.BuildTableVMs(view.Event, "", h.getOccupiedTables(c.Context(), view.Event))

	return c.Render("admin/board", fiber.Map{
		"Event":        view.Event,
		"Scheduled":    view.Scheduled,
		"InProgress":   view.InProgress,
		"Finished":     view.Finished,
		"AllDivisions": view.AllDivs,
		"SelectedDivs": selectedDivs,
		"Query":        q,
		"Tables":       tables,
	}, "layouts/admin")
}

func (h *EventHandler) BoardColumns(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		event     *tournamentDomain.Event
		err       error
		divisions []*divisionDomain.Division
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.event, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return ErrorHandler(res.err)
	}
	t := res.event
	divs := res.divisions
	scheduled, inProgress, finished := BuildBoardCards(t, divs)
	tables := event.BuildTableVMs(t, "", h.getOccupiedTables(c.Context(), t))

	q := strings.ToLower(c.Query("q"))
	var selectedDivs []string
	for _, d := range c.Request().URI().QueryArgs().PeekMulti("div") {
		selectedDivs = append(selectedDivs, string(d))
	}

	if c.Query("q") != "" || len(selectedDivs) > 0 {
		scheduled = FilterBoardCards(scheduled, q, selectedDivs)
		inProgress = FilterBoardCards(inProgress, q, selectedDivs)
		finished = FilterBoardCards(finished, q, selectedDivs)
	}

	return c.Render("admin/partials/board-columns", fiber.Map{
		"Event":      t,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
		"T":          c.Locals("T"),
		"Lang":       c.Locals("Lang"),
	})
}

func (h *EventHandler) ToggleSeedingLock(c *fiber.Ctx) error {
	id := c.Params("id")

	if err := h.toggleSeedingLockUC.Execute(c.Context(), id); err != nil {
		return c.Status(500).SendString("Failed to toggle seeding lock")
	}

	c.Set("HX-Trigger", "reload-board")
	return c.SendStatus(200)
}

func (h *EventHandler) RecalculateElo(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.recalculateEloUC.Execute(c.Context(), id); err != nil {
		return ErrorHandler(err)
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "recalculated"})
}
