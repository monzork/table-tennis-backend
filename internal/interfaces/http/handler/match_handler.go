package handler

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	appTournament "table-tennis-backend/internal/application/event"
	"table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/application/notification"
	"table-tennis-backend/internal/domain/event"

	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MatchHandler struct {
	playerRepo          *bun.PlayerRepository
	matchRepo           *bun.MatchRepository
	tournamentRepo      *bun.EventRepository
	containerRepo       *bun.TournamentRepository
	autoAssignTablesUC  *match.AutoAssignTablesUseCase
	createUC            *match.CreateMatchUseCase
	finishUC            *match.FinishMatchUseCase
	updateScoreUC       *match.UpdateMatchScoreUseCase
	finishTournamentUC  *appTournament.FinishTournamentUseCase
	broadcastPushUC     *notification.BroadcastPushNotificationUseCase
	teamMatchUC         *match.TeamMatchOrchestratorUseCase
	startMatchUC        *match.StartMatchUseCase
	scoreFormViewUC     *match.GetScoreFormViewUseCase
	teamMatchFormViewUC *match.GetTeamMatchFormViewUseCase
}

func NewMatchHandler(
	createUC *match.CreateMatchUseCase,
	finishUC *match.FinishMatchUseCase,
	updateScoreUC *match.UpdateMatchScoreUseCase,
	playerRepo *bun.PlayerRepository,
	matchRepo *bun.MatchRepository,
	tournamentRepo *bun.EventRepository,
	containerRepo *bun.TournamentRepository,
	finishTournamentUC *appTournament.FinishTournamentUseCase,
	broadcastPushUC *notification.BroadcastPushNotificationUseCase,
	teamMatchUC *match.TeamMatchOrchestratorUseCase,
	startMatchUC *match.StartMatchUseCase,
) *MatchHandler {
	return &MatchHandler{
		createUC:            createUC,
		finishUC:            finishUC,
		updateScoreUC:       updateScoreUC,
		playerRepo:          playerRepo,
		matchRepo:           matchRepo,
		tournamentRepo:      tournamentRepo,
		containerRepo:       containerRepo,
		autoAssignTablesUC:  match.NewAutoAssignTablesUseCase(matchRepo, containerRepo),
		finishTournamentUC:  finishTournamentUC,
		broadcastPushUC:     broadcastPushUC,
		teamMatchUC:         teamMatchUC,
		startMatchUC:        startMatchUC,
		scoreFormViewUC:     match.NewGetScoreFormViewUseCase(matchRepo, tournamentRepo, playerRepo, createUC, teamMatchUC),
		teamMatchFormViewUC: match.NewGetTeamMatchFormViewUseCase(matchRepo, tournamentRepo),
	}
}

func (h *MatchHandler) getOccupiedTables(ctx context.Context, t *event.Event) []int {
	var occupiedList []int
	if t != nil {
		if t.EventID != nil {
			occupiedList, _ = h.matchRepo.GetOccupiedTablesByEvent(ctx, *t.EventID)
		} else {
			occupiedList, _ = h.matchRepo.GetOccupiedTablesByTournament(ctx, t.ID)
		}
	}
	return occupiedList
}

func (h *MatchHandler) broadcastToTournamentOrEvent(c *fiber.Ctx, tournamentID string, eventData map[string]string) {
	ctx := c.Context()
	t, err := h.tournamentRepo.GetByID(ctx, tournamentID)

	var htmlStr string
	if err == nil {
		if matchID, ok := eventData["matchId"]; ok {
			var matched *event.Match
			for i := range t.Matches {
				if t.Matches[i].ID == matchID {
					matched = &t.Matches[i]
					break
				}
			}
			if matched != nil {
				var buf bytes.Buffer
				if err := c.App().Config().Views.Render(&buf, "admin/partials/match-row", matched, ""); err == nil {
					searchStr := fmt.Sprintf(`id="match-row-%s"`, matched.ID)
					replaceStr := fmt.Sprintf(`id="match-row-%s" hx-swap-oob="true"`, matched.ID)
					htmlStr = strings.Replace(buf.String(), searchStr, replaceStr, 1)
				}
			}
		}
	}

	broadcastFunc := func(tID string) {
		GlobalBracketHub.Broadcast(tID, eventData)
		if htmlStr != "" {
			GlobalBracketHub.BroadcastHTML(tID, htmlStr)
		}
	}

	if err == nil && t.EventID != nil {
		eventUUID, _ := uuid.Parse(*t.EventID)
		// Broadcast to the tournament dashboard
		broadcastFunc(fmt.Sprintf("tournament_%s", *t.EventID))
		if tourneys, err := h.tournamentRepo.GetByEventID(ctx, eventUUID, false); err == nil {
			for _, tourney := range tourneys {
				broadcastFunc(tourney.ID)
			}
			return
		}
	}
	broadcastFunc(tournamentID)
}

func (h *MatchHandler) Create(c *fiber.Ctx) error {
	var body struct {
		TournamentID   string   `json:"tournamentId" form:"tournamentId"`
		MatchType      string   `json:"matchType" form:"matchType"`
		TeamAPlayerIDs []string `json:"teamAPlayerIds" form:"teamAPlayerIds"`
		TeamBPlayerIDs []string `json:"teamBPlayerIds" form:"teamBPlayerIds"`
	}

	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	if _, err := uuid.Parse(body.TournamentID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid event id")
	}

	var teamA []string
	for _, idStr := range body.TeamAPlayerIDs {
		if _, err := uuid.Parse(idStr); err == nil {
			teamA = append(teamA, idStr)
		}
	}

	var teamB []string
	for _, idStr := range body.TeamBPlayerIDs {
		if _, err := uuid.Parse(idStr); err == nil {
			teamB = append(teamB, idStr)
		}
	}

	newMatch, err := h.createUC.Execute(c.Context(), body.TournamentID, body.MatchType, teamA, teamB)
	if err != nil {
		return ErrorHandler(err)
	}

	// Score modal requests come without HX-Request header — return JSON with match ID
	if c.Get("HX-Request") == "" {
		return c.JSON(fiber.Map{"id": newMatch.ID})
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
		return ErrorHandler(err)
	}

	mUUID, err := uuid.Parse(body.MatchID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid match ID")
	}

	mModel, err := h.matchRepo.GetModelByID(c.Context(), mUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "match not found")
	}

	t, err := h.tournamentRepo.GetByID(c.Context(), mModel.TournamentID.String())
	if err != nil {
		return ErrorHandler(err)
	}

	var matched *event.Match
	for i := range t.Matches {
		if t.Matches[i].ID == body.MatchID {
			matched = &t.Matches[i]
			break
		}
	}
	if matched == nil {
		return fiber.NewError(fiber.StatusNotFound, "match not found in event list")
	}

	// Delegate bracket advancement + team-match aggregation to the repository transaction
	if err := h.matchRepo.FinishMatch(c.Context(), event.FinishMatchCommand{
		MatchID:    body.MatchID,
		WinnerTeam: body.WinnerTeam,
	}); err != nil {
		return ErrorHandler(err)
	}

	// Apply Elo via domain service
	_ = h.finishUC.Execute(matched, body.WinnerTeam)

	var nameA, nameB string
	if len(matched.TeamA) > 0 {
		p := matched.TeamA[0]
		if len(matched.TeamA) > 1 {
			nameA = p.FirstName + " / " + matched.TeamA[1].FirstName
		} else {
			nameA = p.FirstName + " " + p.LastName
		}
	} else {
		nameA = "TBD"
	}
	if len(matched.TeamB) > 0 {
		p := matched.TeamB[0]
		if len(matched.TeamB) > 1 {
			nameB = p.FirstName + " / " + matched.TeamB[1].FirstName
		} else {
			nameB = p.FirstName + " " + p.LastName
		}
	} else {
		nameB = "TBD"
	}
	winStr := nameA + " vs " + nameB
	if body.WinnerTeam == "A" {
		winStr = nameA + " defeated " + nameB
	} else if body.WinnerTeam == "B" {
		winStr = nameB + " defeated " + nameA
	}

	h.broadcastToTournamentOrEvent(c, mModel.TournamentID.String(), map[string]string{
		"tournament":   "score_updated",
		"tournamentId": mModel.TournamentID.String(),
		"matchId":      body.MatchID,
		"matchStatus":  "finished",
		"message":      fmt.Sprintf("Match finished: %s", winStr),
	})

	// Re-fetch event to render updated row
	t, _ = h.tournamentRepo.GetByID(c.Context(), mModel.TournamentID.String())
	var updatedMatched *event.Match
	for i := range t.Matches {
		if t.Matches[i].ID == body.MatchID {
			updatedMatched = &t.Matches[i]
			break
		}
	}
	if updatedMatched == nil {
		return fiber.NewError(fiber.StatusNotFound, "match not found")
	}

	return c.Render("admin/partials/match-row", updatedMatched)
}

func (h *MatchHandler) ShowScoreForm(c *fiber.Ctx) error {
	return h.renderScoreFormInternal(c, "admin/partials/match-score-form")
}

func (h *MatchHandler) ShowPublicScoreForm(c *fiber.Ctx) error {
	return h.renderScoreFormInternal(c, "public/match-score-form")
}

func (h *MatchHandler) renderScoreFormInternal(c *fiber.Ctx, templateName string) error {
	matchID := c.Params("id")
	if matchID == "" {
		matchID = c.Query("matchId")
		if matchID == "" {
			matchID = c.FormValue("matchId")
		}
	}
	tID := c.Query("tournamentId")
	if tID == "" {
		tID = c.FormValue("tournamentId")
	}
	stage := c.Query("stage")
	if stage == "" {
		stage = c.FormValue("stage")
	}
	bestOf := c.QueryInt("bestOf", 0)
	if bestOf == 0 {
		if val := c.FormValue("bestOf"); val != "" {
			if parsed, err := strconv.Atoi(val); err == nil {
				bestOf = parsed
			}
		}
	}
	if bestOf == 0 {
		bestOf = 5
	}
	p1Id := c.Query("p1Id")
	if p1Id == "" {
		p1Id = c.FormValue("p1Id")
	}
	p2Id := c.Query("p2Id")
	if p2Id == "" {
		p2Id = c.FormValue("p2Id")
	}

	lang := getLang(c)
	tMap := i18n.PrecomputedMaps[lang]

	view, err := h.scoreFormViewUC.Execute(c.Context(), matchID, tID, stage, bestOf, p1Id, p2Id)
	if err != nil {
		return ErrorHandler(err)
	}

	// Team-match flow — delegate to existing team form renderer
	if view.IsTeams {
		tmplName := "admin/partials/team-match-score-form"
		if strings.HasPrefix(templateName, "public/") {
			tmplName = "public/team-match-score-form"
		}
		return h.renderTeamMatchFormInternal(c, view.MatchID, tID, stage, tmplName)
	}

	var tourney *event.Event
	if tID != "" {
		tourney, _ = h.tournamentRepo.GetByIDLite(c.Context(), tID)
	}

	return c.Render(templateName, fiber.Map{
		"MatchID":        view.MatchID,
		"TournamentID":   view.TournamentID,
		"Stage":          view.Stage,
		"BestOf":         view.BestOf,
		"PlayerA":        view.PlayerA,
		"PlayerB":        view.PlayerB,
		"Sets":           view.Sets,
		"P1Id":           view.P1Id,
		"P2Id":           view.P2Id,
		"IsSubMatch":     view.IsSubMatch,
		"IsDoubles":      view.IsDoubles,
		"PlayerANames":   view.PlayerANames,
		"PlayerBNames":   view.PlayerBNames,
		"Pin":            view.Pin,
		"RefereeID":      view.RefereeID,
		"TableNumber":    view.TableNumber,
		"TableNumberVal": view.TableNumberVal,
		"Status":         view.Status,
		"Participants":   view.Participants,
		"Tables":         appTournament.BuildTableVMs(tourney, view.MatchID, h.getOccupiedTables(c.Context(), tourney)),
		"T":              tMap,
	})
}

// UpdateScore accepts set scores via JSON/form and persists them, auto-resolving winner.
func (h *MatchHandler) UpdateScore(c *fiber.Ctx) error {
	if c.FormValue("action") == "update_squads" {
		matchID := c.Params("id")
		if matchID == "" {
			matchID = c.FormValue("matchId")
		}

		squadA := []string{c.FormValue("squad_a_p1"), c.FormValue("squad_a_p2"), c.FormValue("squad_a_p3")}
		squadB := []string{c.FormValue("squad_b_p1"), c.FormValue("squad_b_p2"), c.FormValue("squad_b_p3")}

		// Validate that all required players are selected
		for _, p := range squadA {
			if p == "" || p == uuid.Nil.String() {
				return fiber.NewError(fiber.StatusBadRequest, "All 3 players must be selected for each team")
			}
		}
		for _, p := range squadB {
			if p == "" || p == uuid.Nil.String() {
				return fiber.NewError(fiber.StatusBadRequest, "All 3 players must be selected for each team")
			}
		}

		parent, err := h.matchRepo.GetByID(c.Context(), matchID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "parent match not found: "+err.Error())
		}

		t, err := h.tournamentRepo.GetByID(c.Context(), parent.TournamentID)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "event not found: "+err.Error())
		}

		teamFormat := t.TeamFormat
		if teamFormat == "" {
			teamFormat = "olympic"
		}

		err = h.teamMatchUC.UpdateTeamSquads(c.Context(), matchID, squadA, squadB, teamFormat, parent.Stage)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}

		// Re-render the team matchup form in-place
		return h.renderTeamMatchForm(c, matchID, c.FormValue("tournamentId"), parent.Stage)
	}

	matchID := c.Params("id")
	var body struct {
		TournamentID string   `json:"tournamentId" form:"tournamentId"`
		MatchID      string   `json:"matchId" form:"matchId"`
		Stage        string   `json:"stage" form:"stage"`
		P1Id         string   `json:"p1Id" form:"p1Id"`
		P2Id         string   `json:"p2Id" form:"p2Id"`
		Scores       []string `json:"scores" form:"scores[]"`
	}
	if err := c.BodyParser(&body); err != nil {
		return ErrorHandler(err)
	}

	if matchID == "" {
		matchID = body.MatchID
	}

	// If still no matchID, look for existing match first, then create on the fly
	if matchID == "" && body.P1Id != "" && body.P2Id != "" {
		// Try to find existing match for these players
		existing, err := h.matchRepo.GetMatchByParticipants(c.Context(), body.TournamentID, body.P1Id, body.P2Id, body.Stage)
		if err == nil && existing != nil {
			matchID = existing.ID
		}

		if matchID == "" {
			matchType := "singles"
			if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
				switch t.Type {
				case "doubles", "mixed_doubles":
					matchType = "doubles"
				case "teams":
					matchType = "teams"
				}
			}

			m, err := h.createUC.Execute(c.Context(), body.TournamentID, matchType, []string{body.P1Id}, []string{body.P2Id}, body.Stage)
			if err == nil {
				matchID = m.ID
			} else {
				return fiber.NewError(fiber.StatusInternalServerError, "Failed to create match: "+err.Error())
			}
		}
	}
	// Also accept form multi-values
	if len(body.Scores) == 0 {
		for _, s := range c.Request().PostArgs().PeekMulti("scores[]") {
			body.Scores = append(body.Scores, string(s))
		}
	}

	// Support split A/B scores from HTMX form
	if len(body.Scores) == 0 {
		as := c.Request().PostArgs().PeekMulti("scores[]_a")
		bs := c.Request().PostArgs().PeekMulti("scores[]_b")
		for i := 0; i < len(as) && i < len(bs); i++ {
			aStr := string(as[i])
			bStr := string(bs[i])
			if aStr != "" && bStr != "" {
				body.Scores = append(body.Scores, aStr+"-"+bStr)
			}
		}
	}
	if body.Stage == "" {
		body.Stage = "group"
	}

	// Update referee and table number metadata if provided
	refereeIDStr := c.FormValue("refereeId")
	tableNumberStr := c.FormValue("tableNumber")
	if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
		var refereePtr *string
		if refereeIDStr != "" {
			refereePtr = &refereeIDStr
		}

		var tableNumPtr *int
		if tableNumberStr != "" {
			if tNum, err := strconv.Atoi(tableNumberStr); err == nil {
				// Check if another match in this event/tournament is currently in_progress on this table
				var occupiedList []int
				if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
					if t.EventID != nil {
						occupiedList, _ = h.matchRepo.GetOccupiedTablesByEvent(c.Context(), *t.EventID)
					} else {
						occupiedList, _ = h.matchRepo.GetOccupiedTablesByTournament(c.Context(), t.ID)
					}

					isOccupiedByOther := false
					for _, occ := range occupiedList {
						if occ == tNum {
							occupied, _ := h.matchRepo.IsTableOccupiedByOtherMatch(c.Context(), matchID, tNum)
							if occupied {
								isOccupiedByOther = true
							}
							break
						}
					}

					if isOccupiedByOther {
						c.Set("HX-Trigger", `{"show-toast": {"message": "Table is currently occupied by another match!", "type": "error"}}`)
						return fiber.NewError(fiber.StatusBadRequest, "Table is occupied")
					}
				}
				tableNumPtr = &tNum
			}
		}

		_ = h.matchRepo.UpdateMetadata(c.Context(), matchID, refereePtr, tableNumPtr)
	}

	var prevStatus string
	if mUUID, err := uuid.Parse(matchID); err == nil {
		if prevMatch, err := h.matchRepo.GetModelByID(c.Context(), mUUID); err == nil {
			prevStatus = prevMatch.Status
		}
	}

	if err := h.updateScoreUC.Execute(c.Context(), matchID, body.Scores, body.TournamentID, body.Stage); err != nil {
		return ErrorHandler(err)
	}

	// Broadcast real-time update to all bracket viewers for this event
	var nameA, nameB string
	var scored *bun.MatchModel
	if mUUID, err := uuid.Parse(matchID); err == nil {
		scored, _ = h.matchRepo.GetModelByID(c.Context(), mUUID)
	}

	if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
		for i := range t.Matches {
			if t.Matches[i].ID == matchID {
				matched := &t.Matches[i]
				if len(matched.TeamA) > 0 {
					p := matched.TeamA[0]
					if len(matched.TeamA) > 1 {
						nameA = p.FirstName + " / " + matched.TeamA[1].FirstName
					} else {
						nameA = p.FirstName + " " + p.LastName
					}
				} else {
					nameA = "TBD"
				}
				if len(matched.TeamB) > 0 {
					p := matched.TeamB[0]
					if len(matched.TeamB) > 1 {
						nameB = p.FirstName + " / " + matched.TeamB[1].FirstName
					} else {
						nameB = p.FirstName + " " + p.LastName
					}
				} else {
					nameB = "TBD"
				}
				break
			}
		}
	}
	matchName := nameA + " vs " + nameB

	broadcastData := map[string]string{
		"tournament":   "score_updated",
		"tournamentId": body.TournamentID,
		"matchId":      matchID,
	}
	if scored != nil && scored.Status == "finished" {
		broadcastData["matchStatus"] = "finished"
		if prevStatus != "finished" {
			broadcastData["message"] = fmt.Sprintf("Match finished: %s", matchName)
		}
	}

	h.broadcastToTournamentOrEvent(c, body.TournamentID, broadcastData)

	if scored != nil && scored.Status == "finished" && prevStatus != "finished" {
		if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
			if t.EventID != nil {
				assigned, err := h.autoAssignTablesUC.Execute(c.Context(), *t.EventID)
				if err == nil && len(assigned) > 0 {
					for _, m := range assigned {
						p1 := "TBD"
						p2 := "TBD"
						if len(m.TeamA) > 0 {
							p1 = m.TeamA[0].FirstName + " " + m.TeamA[0].LastName
						}
						if len(m.TeamB) > 0 {
							p2 = m.TeamB[0].FirstName + " " + m.TeamB[0].LastName
						}
						h.broadcastToTournamentOrEvent(c, body.TournamentID, map[string]string{
							"event":        "start_match",
							"tournamentId": m.TournamentID,
							"matchId":      m.ID,
							"tableNumber":  strconv.Itoa(*m.TableNumber),
							"p1":           p1,
							"p2":           p2,
						})
					}
				}
			}
			if scored.Stage == "group" {
				allDone := true
				hasGroup := false
				for _, tm := range t.Matches {
					if tm.Stage == "group" && tm.TeamMatchID == nil {
						hasGroup = true
						if tm.Status != "finished" {
							allDone = false
							break
						}
					}
				}
				if hasGroup && allDone {
					h.broadcastToTournamentOrEvent(c, body.TournamentID, map[string]string{
						"event": "group_stage_finished",
					})
				}
			}
		}
	}

	// If this was a sub-match, return to the team matchup form instead of refreshing
	mUUID, _ := uuid.Parse(matchID)
	if scored, err := h.matchRepo.GetModelByID(c.Context(), mUUID); err == nil && scored.TeamMatchID != nil {
		return h.renderTeamMatchForm(c, scored.TeamMatchID.String(), body.TournamentID, body.Stage)
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
	}
	return c.SendStatus(fiber.StatusOK)
}

// UpdatePublicScore accepts set scores via public form, validates PIN, and persists them, notifying admin if finished by referee.
func (h *MatchHandler) UpdatePublicScore(c *fiber.Ctx) error {
	refereeIDStr := c.FormValue("refereeId")
	isRefereeSubmission := refereeIDStr != ""

	if c.FormValue("action") == "update_squads" {
		matchID := c.Params("id")
		if matchID == "" {
			matchID = c.FormValue("matchId")
		}
		if matchID == "" || matchID == "nil" || matchID == "null" || matchID == "undefined" {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Match not found</div>")
		}
		parentUUID, _ := uuid.Parse(matchID)
		parent, err := h.matchRepo.GetModelByID(c.Context(), parentUUID)
		if err != nil {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Match not found: " + err.Error() + "</div>")
		}

		// Validate PIN against event participants and officials if provided (optional)
		submittedPin := c.FormValue("pin")
		var updaterPlayerID string
		if submittedPin != "" {
			updaterPlayerID, _ = h.tournamentRepo.GetParticipantOrOfficialByPIN(c.Context(), parent.TournamentID.String(), submittedPin)
		}

		squadA := []string{c.FormValue("squad_a_p1"), c.FormValue("squad_a_p2"), c.FormValue("squad_a_p3")}
		squadB := []string{c.FormValue("squad_b_p1"), c.FormValue("squad_b_p2"), c.FormValue("squad_b_p3")}

		// Validate that all required players are selected
		for _, p := range squadA {
			if p == "" || p == uuid.Nil.String() {
				return c.SendString("<div class='text-red-400 font-mono text-sm'>All 3 players must be selected for each team</div>")
			}
		}
		for _, p := range squadB {
			if p == "" || p == uuid.Nil.String() {
				return c.SendString("<div class='text-red-400 font-mono text-sm'>All 3 players must be selected for each team</div>")
			}
		}

		t, err := h.tournamentRepo.GetByID(c.Context(), parent.TournamentID.String())
		if err != nil {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Event not found: " + err.Error() + "</div>")
		}

		teamFormat := t.TeamFormat
		if teamFormat == "" {
			teamFormat = "olympic"
		}

		err = h.teamMatchUC.UpdateTeamSquads(c.Context(), matchID, squadA, squadB, teamFormat, parent.Stage)
		if err != nil {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Failed to update sub-match: " + err.Error() + "</div>")
		}

		// Update table number if provided in squad update form
		tableNumberStr := c.FormValue("tableNumber")
		if tableNumberStr != "" {
			if tNum, err := strconv.Atoi(tableNumberStr); err == nil {
				parent.TableNumber = &tNum
			}
		} else {
			parent.TableNumber = nil
		}
		if refereeIDStr != "" {
			if refUUID, err := uuid.Parse(refereeIDStr); err == nil {
				parent.RefereeID = &refUUID
			}
		} else if updaterPlayerID != "" {
			if refUUID, err := uuid.Parse(updaterPlayerID); err == nil {
				parent.RefereeID = &refUUID
			}
		}
		_, _ = h.matchRepo.DB().NewUpdate().Model(parent).WherePK().Column("referee_id", "table_number").Exec(c.Context())

		// Re-render the team matchup form in-place for public
		return h.renderTeamMatchFormInternal(c, matchID, c.FormValue("tournamentId"), parent.Stage, "public/team-match-score-form")
	}

	matchID := c.Params("id")
	var body struct {
		TournamentID string   `json:"tournamentId" form:"tournamentId"`
		MatchID      string   `json:"matchId" form:"matchId"`
		Stage        string   `json:"stage" form:"stage"`
		P1Id         string   `json:"p1Id" form:"p1Id"`
		P2Id         string   `json:"p2Id" form:"p2Id"`
		Scores       []string `json:"scores" form:"scores[]"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>Bad Request</div>")
	}

	if matchID == "" {
		matchID = body.MatchID
	}

	// If still no matchID, look for existing match first, then create on the fly
	if (matchID == "" || matchID == "nil" || matchID == "null" || matchID == "undefined") && body.P1Id != "" && body.P2Id != "" {
		tUUID, _ := uuid.Parse(body.TournamentID)
		p1UUID, _ := uuid.Parse(body.P1Id)
		p2UUID, _ := uuid.Parse(body.P2Id)

		var existing bun.MatchModel
		err := h.matchRepo.DB().NewSelect().Model(&existing).
			Where("event_id = ?", tUUID).
			Where("team_match_id IS NULL").
			Where("((team_a_player_1_id = ? AND team_b_player_1_id = ?) OR (team_a_player_1_id = ? AND team_b_player_1_id = ?))",
				p1UUID, p2UUID, p2UUID, p1UUID).
			Where("stage = ?", body.Stage).
			Scan(c.Context())
		if err == nil {
			matchID = existing.ID.String()
		}

		if matchID == "" || matchID == "nil" || matchID == "null" || matchID == "undefined" {
			matchType := "singles"
			if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
				switch t.Type {
				case "doubles", "mixed_doubles":
					matchType = "doubles"
				case "teams":
					matchType = "teams"
				}
			}

			m, err := h.createUC.Execute(c.Context(), body.TournamentID, matchType, []string{body.P1Id}, []string{body.P2Id}, body.Stage)
			if err == nil {
				matchID = m.ID
			} else {
				return c.SendString("<div class='text-red-400 font-mono text-sm'>Failed to create match: " + err.Error() + "</div>")
			}
		}
	}

	if matchID == "" || matchID == "nil" || matchID == "null" || matchID == "undefined" {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>Match not found</div>")
	}

	// Fetch match from DB
	mUUID, err := uuid.Parse(matchID)
	if err != nil {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>Invalid match ID</div>")
	}
	m, err := h.matchRepo.GetModelByID(c.Context(), mUUID)
	if err != nil {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>Match not found</div>")
	}

	// Update table number if provided
	tableNumberStr := c.FormValue("tableNumber")
	if tableNumberStr != "" {
		if tNum, err := strconv.Atoi(tableNumberStr); err == nil {
			// Check if another match in this event/tournament is currently in_progress on this table
			var occupiedList []int
			if t, err := h.tournamentRepo.GetByID(c.Context(), m.TournamentID.String()); err == nil {
				if t.EventID != nil {
					occupiedList, _ = h.matchRepo.GetOccupiedTablesByEvent(c.Context(), *t.EventID)
				} else {
					occupiedList, _ = h.matchRepo.GetOccupiedTablesByTournament(c.Context(), t.ID)
				}

				isOccupiedByOther := false
				for _, occ := range occupiedList {
					if occ == tNum {
						count, err := h.matchRepo.DB().NewSelect().Model((*bun.MatchModel)(nil)).
							Where("status = 'in_progress' AND table_number = ? AND id != ?", tNum, mUUID).
							Count(c.Context())
						if err == nil && count > 0 {
							isOccupiedByOther = true
						}
						break
					}
				}

				if isOccupiedByOther {
					return c.SendString("<div class='text-red-400 font-mono text-sm'>Table is currently occupied by another match!</div>")
				}
			}
			m.TableNumber = &tNum
		}
	} else {
		m.TableNumber = nil
	}

	// Update referee if provided
	if refereeIDStr != "" {
		refUUID, err := uuid.Parse(refereeIDStr)
		if err == nil {
			m.RefereeID = &refUUID
		}
	}

	// Persist referee and table number
	_, _ = h.matchRepo.DB().NewUpdate().Model(m).WherePK().Column("referee_id", "table_number").Exec(c.Context())

	// Support split A/B scores from form
	if len(body.Scores) == 0 {
		as := c.Request().PostArgs().PeekMulti("scores[]_a")
		bs := c.Request().PostArgs().PeekMulti("scores[]_b")
		for i := 0; i < len(as) && i < len(bs); i++ {
			aStr := string(as[i])
			bStr := string(bs[i])
			if aStr != "" && bStr != "" {
				body.Scores = append(body.Scores, aStr+"-"+bStr)
			}
		}
	}
	if body.Stage == "" {
		body.Stage = "group"
	}

	if err := h.updateScoreUC.Execute(c.Context(), matchID, body.Scores, body.TournamentID, body.Stage); err != nil {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>" + err.Error() + "</div>")
	}

	// Fetch updated status to check if finished
	updatedMatch, err := h.matchRepo.GetModelByID(c.Context(), m.ID)
	if err == nil && m.Status != "finished" && updatedMatch.Status == "finished" {
		// If it's a referee submission, notify admin
		if isRefereeSubmission && m.RefereeID != nil {
			refPlayer, _ := h.playerRepo.GetById(c.Context(), m.RefereeID.String())
			refName := "Referee"
			if refPlayer != nil {
				refName = refPlayer.FullName()
			}

			// Get player/team names for the message
			var pAName, pBName string
			if pA, err := h.playerRepo.GetById(c.Context(), m.TeamAPlayer1ID.String()); err == nil {
				pAName = pA.FullName()
			}
			if pB, err := h.playerRepo.GetById(c.Context(), m.TeamBPlayer1ID.String()); err == nil {
				pBName = pB.FullName()
			}

			tableInfo := ""
			if m.TableNumber != nil {
				tableInfo = fmt.Sprintf(" on Table %d", *m.TableNumber)
			}

			winStr := pAName + " vs " + pBName
			if updatedMatch.WinnerTeam != nil {
				if *updatedMatch.WinnerTeam == "A" {
					winStr = pAName + " defeated " + pBName
				} else if *updatedMatch.WinnerTeam == "B" {
					winStr = pBName + " defeated " + pAName
				}
			}

			h.broadcastToTournamentOrEvent(c, body.TournamentID, map[string]string{
				"tournament": "referee_notification",
				"message":    fmt.Sprintf("%s marked match finished%s: %s", refName, tableInfo, winStr),
			})
		}
	}

	// Broadcast real-time update to all bracket viewers for this event
	var nameA, nameB string
	if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
		for i := range t.Matches {
			if t.Matches[i].ID == matchID {
				matched := &t.Matches[i]
				if len(matched.TeamA) > 0 {
					p := matched.TeamA[0]
					if len(matched.TeamA) > 1 {
						nameA = p.FirstName + " / " + matched.TeamA[1].FirstName
					} else {
						nameA = p.FirstName + " " + p.LastName
					}
				} else {
					nameA = "TBD"
				}
				if len(matched.TeamB) > 0 {
					p := matched.TeamB[0]
					if len(matched.TeamB) > 1 {
						nameB = p.FirstName + " / " + matched.TeamB[1].FirstName
					} else {
						nameB = p.FirstName + " " + p.LastName
					}
				} else {
					nameB = "TBD"
				}
				break
			}
		}
	}
	matchName := nameA + " vs " + nameB

	broadcastData := map[string]string{
		"tournament":   "score_updated",
		"tournamentId": body.TournamentID,
		"matchId":      matchID,
	}
	if updatedMatch != nil && updatedMatch.Status == "finished" {
		broadcastData["matchStatus"] = "finished"
		winStr := matchName
		if updatedMatch.WinnerTeam != nil {
			if *updatedMatch.WinnerTeam == "A" {
				winStr = nameA + " defeated " + nameB
			} else if *updatedMatch.WinnerTeam == "B" {
				winStr = nameB + " defeated " + nameA
			}
		}
		if m.Status != "finished" {
			broadcastData["message"] = fmt.Sprintf("Match finished: %s", winStr)
		}
	}

	h.broadcastToTournamentOrEvent(c, body.TournamentID, broadcastData)

	if m.Status != "finished" && updatedMatch != nil && updatedMatch.Status == "finished" {
		if t, err := h.tournamentRepo.GetByID(c.Context(), body.TournamentID); err == nil {
			if t.EventID != nil {
				assigned, err := h.autoAssignTablesUC.Execute(c.Context(), *t.EventID)
				if err == nil && len(assigned) > 0 {
					for _, am := range assigned {
						p1 := "TBD"
						p2 := "TBD"
						if len(am.TeamA) > 0 {
							p1 = am.TeamA[0].FirstName + " " + am.TeamA[0].LastName
						}
						if len(am.TeamB) > 0 {
							p2 = am.TeamB[0].FirstName + " " + am.TeamB[0].LastName
						}
						h.broadcastToTournamentOrEvent(c, body.TournamentID, map[string]string{
							"event":        "start_match",
							"tournamentId": am.TournamentID,
							"matchId":      am.ID,
							"tableNumber":  strconv.Itoa(*am.TableNumber),
							"p1":           p1,
							"p2":           p2,
						})
					}
				}
			}
			if updatedMatch.Stage == "group" {
				allDone := true
				hasGroup := false
				for _, tm := range t.Matches {
					if tm.Stage == "group" && tm.TeamMatchID == nil {
						hasGroup = true
						if tm.Status != "finished" {
							allDone = false
							break
						}
					}
				}
				if hasGroup && allDone {
					h.broadcastToTournamentOrEvent(c, body.TournamentID, map[string]string{
						"event": "group_stage_finished",
					})
				}
			}
		}
	}

	if m.Status != "finished" && updatedMatch != nil && updatedMatch.Status == "finished" && h.broadcastPushUC != nil {
		winStr := broadcastData["message"] // "Match finished: nameA defeated nameB"
		go func() {
			_ = h.broadcastPushUC.Execute(notification.PushMessage{
				Title: "Match Finished!",
				Body:  winStr,
				URL:   "/events/" + body.TournamentID + "/tv",
			})
		}()
	}

	if c.Get("HX-Request") != "" {
		if updatedMatch != nil && updatedMatch.Status == "finished" {
			// Immediately replace the URL in the browser to prevent users from refreshing and getting the next match on the same table.
			c.Set("HX-Replace-Url", "/events/"+updatedMatch.TournamentID.String())

			if updatedMatch.TeamMatchID != nil {
				return c.SendString(`
				<div id="public-score-form" hx-swap-oob="outerHTML" class="text-center py-8 animate-fade-in">
					<div class="w-20 h-20 bg-green-500/10 rounded-full flex items-center justify-center mx-auto mb-6 shadow-[0_0_30px_rgba(34,197,94,0.2)]">
						<svg class="w-10 h-10 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
							<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
						</svg>
					</div>
					<h3 class="text-2xl font-black uppercase tracking-tight text-white mb-2">Set Finished!</h3>
					<p class="text-gray-400 text-sm font-mono mb-8">The set results have been successfully recorded.</p>
					<button type="button" onclick="document.getElementById('public-score-modal')?.classList.add('hidden')" class="bg-white hover:bg-gray-200 text-black font-black py-4 px-10 rounded-2xl transition-all uppercase tracking-widest text-xs shadow-lg">
						Return to Bracket
					</button>
				</div>`)
			}

			// Return a beautiful success component to replace the form out-of-band
			return c.SendString(`
			<div id="public-score-form" hx-swap-oob="outerHTML" class="text-center py-8 animate-fade-in">
				<div class="w-20 h-20 bg-green-500/10 rounded-full flex items-center justify-center mx-auto mb-6 shadow-[0_0_30px_rgba(34,197,94,0.2)]">
					<svg class="w-10 h-10 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
					</svg>
				</div>
				<h3 class="text-2xl font-black uppercase tracking-tight text-white mb-2">Match Finished!</h3>
				<p class="text-gray-400 text-sm font-mono mb-8">The match results have been successfully recorded and the bracket has been updated.</p>
				<button type="button" onclick="window.close(); window.location.replace('/events/` + updatedMatch.TournamentID.String() + `')" class="bg-white hover:bg-gray-200 text-black font-black py-4 px-10 rounded-2xl transition-all uppercase tracking-widest text-xs shadow-lg">
					Close / Return to Bracket
				</button>
				<script>
					setTimeout(() => {
						window.close();
						setTimeout(() => window.location.replace('/events/` + updatedMatch.TournamentID.String() + `'), 100);
					}, 3000);
				</script>
			</div>`)
		} else {
			// Match still in progress, just show an inline success message in the #public-score-result div
			c.Set("HX-Trigger", `{"show-toast": {"message": "Score saved successfully", "type": "success"}}`)
			return c.SendString(`<div class='text-green-400 font-mono text-sm'>Score saved! You can continue updating sets.</div>`)
		}
	}
	return c.SendStatus(fiber.StatusOK)
}

// renderTeamMatchForm re-renders the team match score form into the modal without a page reload.
func (h *MatchHandler) renderTeamMatchForm(c *fiber.Ctx, matchID, tournamentID, stage string) error {
	return h.renderTeamMatchFormInternal(c, matchID, tournamentID, stage, "admin/partials/team-match-score-form")
}

func (h *MatchHandler) renderTeamMatchFormInternal(c *fiber.Ctx, matchID, tournamentID, stage string, templateName string) error {
	view, err := h.teamMatchFormViewUC.Execute(c.Context(), matchID, tournamentID, stage)
	if err != nil {
		return ErrorHandler(err)
	}

	return c.Render(templateName, fiber.Map{
		"MatchID":      view.MatchID,
		"TournamentID": view.TournamentID,
		"Stage":        view.Stage,
		"BestOf":       view.BestOf,
		"TeamA":        view.TeamA,
		"TeamB":        view.TeamB,
		"TeamFormat":   view.TeamFormat,
		"SubMatches":   view.SubMatches,
		"SquadAP1":     view.SquadAP1,
		"SquadAP2":     view.SquadAP2,
		"SquadAP3":     view.SquadAP3,
		"SquadBP1":     view.SquadBP1,
		"SquadBP2":     view.SquadBP2,
		"SquadBP3":     view.SquadBP3,
		"Participants": view.Participants,
		"Pin":          view.Pin,
		"RefereeID":    view.RefereeID,
		"TableNumber":  view.TableNumber,
	})
}

// Start sets a match status to in_progress, persists to DB, broadcasts WS update, and renders updated row.
func (h *MatchHandler) Start(c *fiber.Ctx) error {
	matchID := c.Params("id")
	if matchID == "" {
		matchID = c.FormValue("matchId")
	}

	tID := c.Query("tournamentId")
	if tID == "" {
		tID = c.FormValue("tournamentId")
	}
	p1Id := c.Query("p1Id")
	if p1Id == "" {
		p1Id = c.FormValue("p1Id")
	}
	p2Id := c.Query("p2Id")
	if p2Id == "" {
		p2Id = c.FormValue("p2Id")
	}
	stage := c.Query("stage")
	if stage == "" {
		stage = c.FormValue("stage")
	}
	if stage == "" {
		stage = "group"
	}

	// Fetch fully loaded event to get table counts & division details
	var t *event.Event
	var err error

	if matchID == "" || matchID == "nil" || matchID == "null" || matchID == "undefined" {
		// Try to create from tournamentId, p1Id, p2Id, stage
		if tID != "" && p1Id != "" && p2Id != "" {
			matchType := "singles"
			if t, err = h.tournamentRepo.GetByID(c.Context(), tID); err == nil {
				switch t.Type {
				case "doubles", "mixed_doubles":
					matchType = "doubles"
				case "teams":
					matchType = "teams"
				}
			}
			newID, err := h.matchRepo.FindOrCreateMatch(c.Context(), tID, p1Id, p2Id, stage, matchType)
			if err != nil {
				return ErrorHandler(err)
			}
			matchID = newID
		}
	} else {
		mUUID, err := uuid.Parse(matchID)
		if err == nil {
			if m, err := h.matchRepo.GetModelByID(c.Context(), mUUID); err == nil {
				tID = m.TournamentID.String()
				stage = m.Stage
			}
		}
	}

	if matchID == "" || matchID == "nil" || matchID == "null" || matchID == "undefined" {
		return fiber.NewError(fiber.StatusNotFound, "Match not found")
	}

	if t == nil && tID != "" {
		t, err = h.tournamentRepo.GetByID(c.Context(), tID)
		if err != nil {
			return ErrorHandler(err)
		}
	}

	var tableNumber *int
	manualTableStr := c.FormValue("tableNumber")
	if manualTableStr != "" {
		if tNum, err := strconv.Atoi(manualTableStr); err == nil {
			tableNumber = &tNum
		}
	}

	var eventNumTables int
	if t != nil && t.EventID != nil {
		eventNumTables, _ = h.tournamentRepo.GetEventNumTables(c.Context(), *t.EventID)
	}
	totalTables := 4
	if t != nil && t.NumTables > 0 {
		totalTables = t.NumTables
	}
	if eventNumTables > 0 {
		totalTables = eventNumTables
	}

	isHighPriority := stage == "semifinal" || stage == "final"
	if t != nil {
		tNameLower := strings.ToLower(t.Name)
		if strings.Contains(tNameLower, "1st division") ||
			strings.Contains(tNameLower, "division 1") ||
			strings.Contains(tNameLower, "primera division") ||
			strings.Contains(tNameLower, "primera división") ||
			strings.Contains(tNameLower, "div 1") ||
			strings.Contains(tNameLower, "division i") ||
			strings.Contains(tNameLower, "1ra division") ||
			strings.Contains(tNameLower, "1ra división") {
			isHighPriority = true
		}
	}

	cmd := event.StartMatchCommand{
		MatchID:        matchID,
		TournamentID:   tID,
		TableNumber:    tableNumber,
		TotalTables:    totalTables,
		IsHighPriority: isHighPriority,
	}

	res, err := h.startMatchUC.Execute(c.Context(), cmd)
	if err != nil {
		if err == event.ErrTableOccupied {
			msg := "All tables are currently occupied!"
			if tableNumber != nil {
				msg = fmt.Sprintf("Table %d is currently occupied by another match!", *tableNumber)
			}
			c.Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"message": "%s", "type": "error"}}`, msg))
			var matched *event.Match
			if t != nil {
				for i := range t.Matches {
					if t.Matches[i].ID == matchID {
						matched = &t.Matches[i]
						break
					}
				}
			}
			if matched == nil {
				return fiber.NewError(fiber.StatusNotFound, "Match not found in event list")
			}
			return c.Render("admin/partials/match-row", matched)
		}
		return ErrorHandler(err)
	}

	// Broadcast WS
	h.broadcastToTournamentOrEvent(c, tID, map[string]string{
		"tournament":   "score_updated",
		"tournamentId": tID,
		"matchId":      matchID,
	})

	if h.broadcastPushUC != nil {
		tblStr := ""
		if res.TableNumber > 0 {
			tblStr = fmt.Sprintf("Table %d: ", res.TableNumber)
		}
		go func() {
			_ = h.broadcastPushUC.Execute(notification.PushMessage{
				Title: "Match Called to Table!",
				Body:  fmt.Sprintf("%s%s vs %s", tblStr, res.PlayerAName, res.PlayerBName),
				URL:   "/events/" + tID + "/tv",
			})
		}()

		tblNum := ""
		if res.TableNumber > 0 {
			tblNum = fmt.Sprintf("%d", res.TableNumber)
		}
		c.Append("HX-Trigger", fmt.Sprintf(`{"match-started": {"p1": %q, "p2": %q, "table": %q}}`, res.PlayerAName, res.PlayerBName, tblNum))
	}

	t, err = h.tournamentRepo.GetByID(c.Context(), tID)
	if err != nil {
		return ErrorHandler(err)
	}
	var matched *event.Match
	for i := range t.Matches {
		if t.Matches[i].ID == matchID {
			matched = &t.Matches[i]
			break
		}
	}
	if matched == nil {
		return fiber.NewError(fiber.StatusNotFound, "Match not found in event list")
	}
	return c.Render("admin/partials/match-row", matched)
}

// Reset reverts a match back to "scheduled", clearing all sets, winner, and table assignment.
func (h *MatchHandler) Reset(c *fiber.Ctx) error {
	mUUID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid match ID")
	}

	m, err := h.matchRepo.GetModelByID(c.Context(), mUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "match not found")
	}

	m.Status = "scheduled"
	m.WinnerTeam = nil
	m.TableNumber = nil
	now := time.Now()
	m.UpdatedAt = &now

	_, err = h.matchRepo.DB().NewDelete().TableExpr("match_sets").Where("match_id = ?", mUUID).Exec(c.Context())
	if err != nil {
		return ErrorHandler(err)
	}

	_, err = h.matchRepo.DB().NewUpdate().Model(m).WherePK().
		Column("status", "winner_team", "table_number", "updated_at").Exec(c.Context())
	if err != nil {
		return ErrorHandler(err)
	}

	h.broadcastToTournamentOrEvent(c, m.TournamentID.String(), map[string]string{
		"tournament":   "score_updated",
		"tournamentId": m.TournamentID.String(),
		"matchId":      m.ID.String(),
	})

	c.Set("HX-Trigger", "reload-bracket, reload-matches")
	return c.SendStatus(fiber.StatusOK)
}

// ShowMatchScorePage renders the standalone public score page for a match.
// Accessed via /score/:matchId (shareable QR-code URL).
// Renders the score form directly without requiring a PIN.
func (h *MatchHandler) ShowMatchScorePage(c *fiber.Ctx) error {
	matchIDStr := c.Params("matchId")
	if matchIDStr == "" {
		matchIDStr = c.Params("id")
	}
	mUUID, err := uuid.Parse(matchIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid match ID")
	}
	m, err := h.matchRepo.GetModelByID(c.Context(), mUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Match not found")
	}

	// Load player names
	playerAName, playerBName := "Player A", "Player B"
	if p, err := h.playerRepo.GetById(c.Context(), m.TeamAPlayer1ID.String()); err == nil {
		playerAName = p.FullName()
	}
	if p, err := h.playerRepo.GetById(c.Context(), m.TeamBPlayer1ID.String()); err == nil {
		playerBName = p.FullName()
	}

	lang := getLang(c)
	tMap := i18n.PrecomputedMaps[lang]

	// Load full match data for score form
	t, _ := h.tournamentRepo.GetByID(c.Context(), m.TournamentID.String())
	bestOf := 5
	if t != nil {
		bestOf = t.GetEffectiveStageRule(m.Stage, m.DivisionID).BestOf
	}

	type setVM struct {
		Number int
		ScoreA interface{}
		ScoreB interface{}
	}
	var sets []setVM
	existingScores := make(map[int]bun.MatchSetModel)
	if s, err := h.matchRepo.GetSets(c.Context(), matchIDStr); err == nil {
		for _, sm := range s {
			existingScores[sm.SetNumber] = sm
		}
	}
	for i := 1; i <= bestOf; i++ {
		valA, valB := interface{}(""), interface{}("")
		if sm, ok := existingScores[i]; ok {
			valA = sm.ScoreA
			valB = sm.ScoreB
		}
		sets = append(sets, setVM{Number: i, ScoreA: valA, ScoreB: valB})
	}

	tournamentID := ""
	if t != nil {
		tournamentID = t.ID
	}

	return c.Render("public/match-score-page", fiber.Map{
		"MatchID":      matchIDStr,
		"TournamentID": tournamentID,
		"Stage":        m.Stage,
		"BestOf":       bestOf,
		"PlayerA":      playerAName,
		"PlayerB":      playerBName,
		"Sets":         sets,
		"P1Id":         m.TeamAPlayer1ID.String(),
		"P2Id":         m.TeamBPlayer1ID.String(),
		"IsDoubles":    m.MatchType == "doubles",
		"TableNumber":  m.TableNumber,
		"T":            tMap,
		"Lang":         lang,
	})
}

// ShowTableScorePage renders the standalone public score page for a table.
// Accessed via /score/table/:tableNumber (shareable table QR-code URL).
// If an active match (in_progress) is found on the table, it renders the score form directly.
// Otherwise, it renders the table-no-match page.
func (h *MatchHandler) ShowTableScorePage(c *fiber.Ctx) error {
	tournamentIDStr := c.Params("tournamentId")
	eventIDStr := c.Params("eventId")

	var err error
	var tournamentUUID, eventUUID uuid.UUID

	if tournamentIDStr != "" {
		tournamentUUID, err = uuid.Parse(tournamentIDStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid event ID")
		}
	} else if eventIDStr != "" {
		eventUUID, err = uuid.Parse(eventIDStr)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, "Invalid tournament ID")
		}
	} else {
		return fiber.NewError(fiber.StatusBadRequest, "Missing event or tournament ID")
	}

	tableNumberStr := c.Params("tableNumber")
	tableNumber, err := strconv.Atoi(tableNumberStr)
	if err != nil || tableNumber <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid table number")
	}

	// Query for an in_progress match on this table within the specific event or tournament
	var m bun.MatchModel
	q := h.matchRepo.DB().NewSelect().Model(&m).
		Where("status = 'in_progress' AND table_number = ?", tableNumber)

	if tournamentIDStr != "" {
		q.Where("event_id = ?", tournamentUUID)
	} else if eventIDStr != "" {
		q.Where("event_id IN (SELECT id FROM events WHERE tournament_id = ?)", eventUUID)
	}

	err = q.Limit(1).Scan(c.Context())

	lang := getLang(c)
	tMap := i18n.PrecomputedMaps[lang]

	if err != nil {
		// No match is in progress on this table.
		return c.Render("public/table-no-match", fiber.Map{
			"TableNumber": tableNumber,
			"T":           tMap,
			"Lang":        lang,
		})
	}

	// Load player names
	playerAName, playerBName := "Player A", "Player B"
	if p, err := h.playerRepo.GetById(c.Context(), m.TeamAPlayer1ID.String()); err == nil {
		playerAName = p.FullName()
	}
	if p, err := h.playerRepo.GetById(c.Context(), m.TeamBPlayer1ID.String()); err == nil {
		playerBName = p.FullName()
	}

	matchIDStr := m.ID.String()

	// Load full match data for score form
	t, _ := h.tournamentRepo.GetByID(c.Context(), m.TournamentID.String())
	bestOf := 5
	if t != nil {
		bestOf = t.GetEffectiveStageRule(m.Stage, m.DivisionID).BestOf
	}

	type setVM struct {
		Number int
		ScoreA interface{}
		ScoreB interface{}
	}
	var sets []setVM
	existingScores := make(map[int]bun.MatchSetModel)
	if s, err := h.matchRepo.GetSets(c.Context(), matchIDStr); err == nil {
		for _, sm := range s {
			existingScores[sm.SetNumber] = sm
		}
	}
	for i := 1; i <= bestOf; i++ {
		valA, valB := interface{}(""), interface{}("")
		if sm, ok := existingScores[i]; ok {
			valA = sm.ScoreA
			valB = sm.ScoreB
		}
		sets = append(sets, setVM{Number: i, ScoreA: valA, ScoreB: valB})
	}

	tournamentID := ""
	if t != nil {
		tournamentID = t.ID
	}

	return c.Render("public/match-score-page", fiber.Map{
		"MatchID":      matchIDStr,
		"TournamentID": tournamentID,
		"Stage":        m.Stage,
		"BestOf":       bestOf,
		"PlayerA":      playerAName,
		"PlayerB":      playerBName,
		"Sets":         sets,
		"P1Id":         m.TeamAPlayer1ID.String(),
		"P2Id":         m.TeamBPlayer1ID.String(),
		"IsDoubles":    m.MatchType == "doubles",
		"TableNumber":  m.TableNumber,
		"T":            tMap,
		"Lang":         lang,
	})
}

// ValidateMatchPIN validates the PIN for a match and, if correct, returns the inline score form.
// POST /score/:matchId/verify
func (h *MatchHandler) ValidateMatchPIN(c *fiber.Ctx) error {
	matchIDStr := c.Params("matchId")
	if matchIDStr == "" {
		matchIDStr = c.Params("id")
	}
	mUUID, err := uuid.Parse(matchIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("<p class=\"text-red-400 font-mono text-sm text-center\">Invalid match ID</p>")
	}
	m, err := h.matchRepo.GetModelByID(c.Context(), mUUID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("<p class=\"text-red-400 font-mono text-sm text-center\">Match not found</p>")
	}

	submittedPin := c.FormValue("pin")
	_, err = h.tournamentRepo.GetParticipantOrOfficialByPIN(c.Context(), m.TournamentID.String(), submittedPin)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("<p class=\"text-red-400 font-mono text-sm text-center mt-4\">❌ Incorrect PIN. Please try again.</p>")
	}

	if m.Status != "in_progress" {
		return c.Status(fiber.StatusBadRequest).SendString("<p class=\"text-yellow-400 font-mono text-sm text-center mt-4\">⚠️ This match is not in progress yet.</p>")
	}

	// Load full match data for score form
	t, _ := h.tournamentRepo.GetByID(c.Context(), m.TournamentID.String())
	bestOf := 5
	if t != nil {
		bestOf = t.GetEffectiveStageRule(m.Stage, m.DivisionID).BestOf
	}

	playerAName, playerBName := "Player A", "Player B"
	if p, err := h.playerRepo.GetById(c.Context(), m.TeamAPlayer1ID.String()); err == nil {
		playerAName = p.FullName()
	}
	if p, err := h.playerRepo.GetById(c.Context(), m.TeamBPlayer1ID.String()); err == nil {
		playerBName = p.FullName()
	}

	type setVM struct {
		Number int
		ScoreA interface{}
		ScoreB interface{}
	}
	var sets []setVM
	existingScores := make(map[int]bun.MatchSetModel)
	if s, err := h.matchRepo.GetSets(c.Context(), matchIDStr); err == nil {
		for _, sm := range s {
			existingScores[sm.SetNumber] = sm
		}
	}
	for i := 1; i <= bestOf; i++ {
		valA, valB := interface{}(""), interface{}("")
		if sm, ok := existingScores[i]; ok {
			valA = sm.ScoreA
			valB = sm.ScoreB
		}
		sets = append(sets, setVM{Number: i, ScoreA: valA, ScoreB: valB})
	}

	tournamentID := ""
	if t != nil {
		tournamentID = t.ID
	}

	return c.Render("public/match-score-form", fiber.Map{
		"MatchID":      matchIDStr,
		"TournamentID": tournamentID,
		"Stage":        m.Stage,
		"BestOf":       bestOf,
		"PlayerA":      playerAName,
		"PlayerB":      playerBName,
		"Sets":         sets,
		"P1Id":         m.TeamAPlayer1ID.String(),
		"P2Id":         m.TeamBPlayer1ID.String(),
		"IsDoubles":    m.MatchType == "doubles",
		"TableNumber":  m.TableNumber,
		"Pin":          submittedPin, // pass validated PIN so form can re-submit
	})
}
