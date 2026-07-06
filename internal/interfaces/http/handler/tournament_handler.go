package handler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"table-tennis-backend/internal/application/division"
	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/application/tournament"
	divisionDomain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
	"time"

	"github.com/gofiber/fiber/v2"
)

type TournamentHandler struct {
	createUC               *tournament.CreateTournamentUseCase
	getByID                *tournament.GetTournamentByIDUseCase
	updateUC               *tournament.UpdateTournamentUseCase
	deleteUC               *tournament.DeleteTournamentUseCase
	leaderboardUC          *leaderboard.GetLeaderboardUseCase
	divisionUC             *division.DivisionUseCase
	finishUC               *tournament.FinishTournamentUseCase
	exportUC               *tournament.ExportTournamentReportUseCase
	exportPdfUC            *tournament.ExportTournamentPdfUseCase
	movePlayerUC           *tournament.MovePlayerUseCase
	createTeamUC           *tournament.CreateTeamUseCase
	deleteTeamUC           *tournament.DeleteTeamUseCase
	assignPlayerToTeamUC   *tournament.AssignPlayerToTeamUseCase
	removePlayerFromTeamUC *tournament.RemovePlayerFromTeamUseCase
	getTournamentsUC       *tournament.GetTournamentsUseCase
	getOccupiedTablesUC    *tournament.GetOccupiedTablesUseCase
	regenerateSeedsUC      *tournament.RegenerateGroupSeedsUseCase
	updateParticipantEloUC *tournament.UpdateParticipantEloBeforeUseCase
	removeParticipantUC    *tournament.RemoveParticipantUseCase
}

func NewTournamentHandler(
	createUC *tournament.CreateTournamentUseCase,
	getByID *tournament.GetTournamentByIDUseCase,
	updateUC *tournament.UpdateTournamentUseCase,
	deleteUC *tournament.DeleteTournamentUseCase,
	leaderboardUC *leaderboard.GetLeaderboardUseCase,
	divisionUC *division.DivisionUseCase,
	finishUC *tournament.FinishTournamentUseCase,
	exportUC *tournament.ExportTournamentReportUseCase,
	exportPdfUC *tournament.ExportTournamentPdfUseCase,
	movePlayerUC *tournament.MovePlayerUseCase,
	createTeamUC *tournament.CreateTeamUseCase,
	deleteTeamUC *tournament.DeleteTeamUseCase,
	assignPlayerToTeamUC *tournament.AssignPlayerToTeamUseCase,
	removePlayerFromTeamUC *tournament.RemovePlayerFromTeamUseCase,
	getTournamentsUC *tournament.GetTournamentsUseCase,
	getOccupiedTablesUC *tournament.GetOccupiedTablesUseCase,
	regenerateSeedsUC *tournament.RegenerateGroupSeedsUseCase,
	updateParticipantEloUC *tournament.UpdateParticipantEloBeforeUseCase,
	removeParticipantUC *tournament.RemoveParticipantUseCase,
) *TournamentHandler {
	return &TournamentHandler{
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
	}
}

func (h *TournamentHandler) Create(c *fiber.Ctx) error {
	var body struct {
		Name           string `json:"name" form:"name"`
		Type           string `json:"type" form:"type"`
		Format         string `form:"format"`
		EventCategory  string `form:"eventCategory"`
		StartDate      string `form:"startDate"`
		EndDate        string `form:"endDate"`
		GroupPassCount int    `form:"groupPassCount"`
		TeamFormat     string `form:"teamFormat"`
		NumTables      int    `form:"numTables" json:"numTables"`
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

	// Parse per-stage rule overrides
	createStages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	var stageRules []tournament.StageRuleOverride
	for _, stage := range createStages {
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

	// Parse division-specific rules (stage-based)
	var divisionRules []tournamentDomain.DivisionRule
	divisionStages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	divisionIDs := c.Request().PostArgs().PeekMulti("division_rule[division_id][]")
	for _, divIDBytes := range divisionIDs {
		divID := string(divIDBytes)
		if divID == "" {
			continue
		}
		for _, stage := range divisionStages {
			boStr := string(c.Request().PostArgs().Peek("division_rule[" + divID + "][" + stage + "][best_of]"))
			ptStr := string(c.Request().PostArgs().Peek("division_rule[" + divID + "][" + stage + "][points_to_win]"))
			pmStr := string(c.Request().PostArgs().Peek("division_rule[" + divID + "][" + stage + "][points_margin]"))
			if boStr != "" {
				bo := 5
				pt := 11
				pm := 2
				fmt.Sscanf(boStr, "%d", &bo)
				fmt.Sscanf(ptStr, "%d", &pt)
				fmt.Sscanf(pmStr, "%d", &pm)
				divisionRules = append(divisionRules, tournamentDomain.DivisionRule{
					DivisionID:   divID,
					Stage:        stage,
					BestOf:       bo,
					PointsToWin:  pt,
					PointsMargin: pm,
				})
			}
		}
	}

	skipElo := c.FormValue("skipElo") == "on"
	hasThirdPlaceMatch := c.FormValue("hasThirdPlaceMatch") == "on"
	var eventID *string
	if eIDStr := c.FormValue("eventId"); eIDStr != "" {
		eventID = &eIDStr
	}

	divisionFormats := make(map[string]string)
	for _, divIDBytes := range divisionIDs {
		divID := string(divIDBytes)
		if divID == "" {
			continue
		}
		dfStr := string(c.Request().PostArgs().Peek("division_formats[" + divID + "]"))
		if dfStr != "" {
			divisionFormats[divID] = dfStr
		}
	}

	t, err := h.createUC.Execute(
		c.Context(), body.Name, body.Type, body.Format, body.EventCategory, body.StartDate, body.EndDate,
		participantIDs, newPlayers, body.GroupPassCount, stageRules, divisionRules, skipElo, eventID,
		body.TeamFormat, body.NumTables, hasThirdPlaceMatch, divisionFormats,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("admin/partials/tournament-row", t)
}

func (h *TournamentHandler) Detail(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		tournament *tournamentDomain.Tournament
		err        error
		players    any
		divisions  []*divisionDomain.Division
		snapshots  []tournamentDomain.ParticipantSnapshot
		officials  []tournamentDomain.ParticipantSnapshot
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(5)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.players, _ = h.leaderboardUC.ExecuteSingles(c.Context())
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	go func() {
		defer wg.Done()
		res.snapshots, _ = h.getByID.GetSnapshots(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		// Since we don't have a specific usecase for officials yet, we use a quick workaround.
		// A cleaner architecture would be to inject the repo into getByID or create a new usecase.
		// I will create getByID.GetOfficials() below.
		res.officials, _ = h.getByID.GetOfficials(c.Context(), id)
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}
	t := res.tournament
	players := res.players
	divisions := res.divisions
	snapshots := res.snapshots

	statusFilter := c.Query("status", "all")
	if statusFilter != "all" {
		var filtered []tournamentDomain.Match
		for _, m := range t.Matches {
			if m.Status == statusFilter {
				filtered = append(filtered, m)
			}
		}
		t.Matches = filtered
	}

	// Build the view model for the bracket rendering
	tmap, _ := c.Locals("T").(map[string]string)
	vm := BuildTournamentViewModel(t, divisions, tmap)

	// Calculate available participants (those not in any team)
	var availableParticipants []*player.Player
	assignedMap := make(map[string]bool)
	for _, team := range t.Teams {
		for _, p := range team.Players {
			assignedMap[p.ID] = true
		}
	}
	for _, p := range t.Participants {
		if !assignedMap[p.ID] {
			availableParticipants = append(availableParticipants, p)
		}
	}

	// Fetch Participant PINs
	playerPins := make(map[string]string)
	for _, snap := range snapshots {
		playerPins[snap.PlayerID] = snap.Pin
	}

	// Build a sorted ParticipantRow slice: seed#, group name, division
	type ParticipantRow struct {
		Player    *player.Player
		Seed      int
		GroupName string
		DivName   string
		Pin       string
		Elo       int16
	}

	// Build player→group map from tournament.Groups
	playerGroupMap := make(map[string]string)
	for _, g := range t.Groups {
		gDisplayName := g.Name
		// Strip "Division - " prefix if present
		if idx := strings.Index(g.Name, " - "); idx != -1 {
			gDisplayName = g.Name[idx+3:]
		}
		for _, p := range g.Players {
			playerGroupMap[p.ID] = gDisplayName
		}
	}

	// Build player→division map using the same Elo-band logic as BuildBoardCards
	playerDivMap := make(map[string]string)
	for _, p := range t.Participants {
		elo := p.SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			elo = p.DoublesElo
		}
		found := false
		for _, d := range divisions {
			if d.MinElo == 0 && d.MaxElo == nil {
				continue
			}
			if (d.Category == "both" || d.Category == t.Type) && elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
				playerDivMap[p.ID] = d.Name
				found = true
				break
			}
		}
		if !found && !t.SkipElo {
			playerDivMap[p.ID] = "Open Bracket"
		}
	}

	// Sort participants by Elo descending to assign seed numbers
	sortedParts := make([]*player.Player, len(t.Participants))
	copy(sortedParts, t.Participants)
	sort.Slice(sortedParts, func(i, j int) bool {
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			return sortedParts[i].DoublesElo > sortedParts[j].DoublesElo
		}
		return sortedParts[i].SinglesElo > sortedParts[j].SinglesElo
	})
	seedMap := make(map[string]int, len(sortedParts))
	for i, p := range sortedParts {
		seedMap[p.ID] = i + 1
	}

	rows := make([]ParticipantRow, len(sortedParts))
	for i, p := range sortedParts {
		elo := p.SinglesElo
		if t.Type == "doubles" || t.Type == "mixed_doubles" {
			elo = p.DoublesElo
		}
		rows[i] = ParticipantRow{
			Player:    p,
			Seed:      seedMap[p.ID],
			GroupName: playerGroupMap[p.ID],
			DivName:   playerDivMap[p.ID],
			Pin:       playerPins[p.ID],
			Elo:       elo,
		}
	}

	return c.Render("admin/tournament-detail", fiber.Map{
		"Tournament":            t,
		"Players":               players,
		"Divisions":             divisions,
		"BracketViewModel":      vm,
		"AvailableParticipants": availableParticipants,
		"StatusFilter":          statusFilter,
		"PlayerPins":            playerPins,
		"Officials":             res.officials,
		"ParticipantRows":       rows,
	}, "layouts/admin")
}

func (h *TournamentHandler) AddOfficial(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := h.getByID.AddOfficial(c.Context(), tournamentID, body.PlayerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) RemoveOfficial(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	playerID := c.Params("playerId")
	if err := h.getByID.RemoveOfficial(c.Context(), tournamentID, playerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) RemoveParticipant(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	playerID := c.Params("playerId")
	if err := h.removeParticipantUC.Execute(c.Context(), tournamentID, playerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		tournament *tournamentDomain.Tournament
		err        error
		players    any
		divisions  []*divisionDomain.Division
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.players, _ = h.leaderboardUC.ExecuteSingles(c.Context())
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}
	return c.Render("admin/partials/tournament-edit-form", fiber.Map{
		"Tournament": res.tournament,
		"Players":    res.players,
		"Divisions":  res.divisions,
	})
}

func (h *TournamentHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		Name             string `form:"name"`
		Type             string `form:"type"`
		Format           string `form:"format"`
		EventCategory    string `form:"eventCategory"`
		StartDate        string `form:"startDate"`
		EndDate          string `form:"endDate"`
		GroupPassCount   int    `form:"groupPassCount"`
		RegistrationOpen bool   `form:"registrationOpen"`
		TeamFormat       string `form:"teamFormat"`
		NumTables        int    `form:"numTables" json:"numTables"`
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

	// Parse division-specific rules (stage-based)
	var divisionRules []tournamentDomain.DivisionRule
	divisionStages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	divisionIDs := c.Request().PostArgs().PeekMulti("division_rule[division_id][]")
	for _, divIDBytes := range divisionIDs {
		divID := string(divIDBytes)
		if divID == "" {
			continue
		}
		for _, stage := range divisionStages {
			boStr := string(c.Request().PostArgs().Peek("division_rule[" + divID + "][" + stage + "][best_of]"))
			ptStr := string(c.Request().PostArgs().Peek("division_rule[" + divID + "][" + stage + "][points_to_win]"))
			pmStr := string(c.Request().PostArgs().Peek("division_rule[" + divID + "][" + stage + "][points_margin]"))
			if boStr != "" {
				bo := 5
				pt := 11
				pm := 2
				fmt.Sscanf(boStr, "%d", &bo)
				fmt.Sscanf(ptStr, "%d", &pt)
				fmt.Sscanf(pmStr, "%d", &pm)
				divisionRules = append(divisionRules, tournamentDomain.DivisionRule{
					DivisionID:   divID,
					Stage:        stage,
					BestOf:       bo,
					PointsToWin:  pt,
					PointsMargin: pm,
				})
			}
		}
	}

	skipElo := c.FormValue("skipElo") == "on"
	hasThirdPlaceMatch := c.FormValue("hasThirdPlaceMatch") == "on"
	var eventID *string
	if eIDStr := c.FormValue("eventId"); eIDStr != "" {
		eventID = &eIDStr
	}

	divisionFormats := make(map[string]string)
	for _, divIDBytes := range divisionIDs {
		divID := string(divIDBytes)
		if divID == "" {
			continue
		}
		dfStr := string(c.Request().PostArgs().Peek("division_formats[" + divID + "]"))
		if dfStr != "" {
			divisionFormats[divID] = dfStr
		}
	}

	t, err := h.updateUC.Execute(
		c.Context(), id, body.Name, body.Type, body.Format, body.EventCategory, body.StartDate, body.EndDate,
		body.RegistrationOpen, participantIDs, newPlayers, stageRules, divisionRules, body.GroupPassCount,
		skipElo, eventID, body.TeamFormat, body.NumTables, hasThirdPlaceMatch, divisionFormats,
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Render("admin/partials/tournament-row", t)
}

func (h *TournamentHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deleteUC.Execute(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		if c.Get("HX-Current-URL") != "" && fmt.Sprintf("/admin/tournaments/%s", id) == c.Get("HX-Current-URL") {
			c.Set("HX-Redirect", "/admin/tournaments")
		}
		return c.SendString("")
	}
	return c.SendString("")
}

func (h *TournamentHandler) Finish(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.finishUC.Execute(c.Context(), idStr); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "finished"})
}

func (h *TournamentHandler) RegenerateGroupSeeds(c *fiber.Ctx) error {
	idStr := c.Params("id")
	if err := h.regenerateSeedsUC.Execute(c.Context(), idStr); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "regenerated"})
}

func (h *TournamentHandler) UpdateParticipantEloBefore(c *fiber.Ctx) error {
	idStr := c.Params("id")
	var body struct {
		PlayerID   string `form:"playerId"`
		SinglesElo int16  `form:"singlesElo"`
		DoublesElo int16  `form:"doublesElo"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if body.PlayerID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "playerId is required")
	}
	if err := h.updateParticipantEloUC.Execute(c.Context(), idStr, body.PlayerID, body.SinglesElo, body.DoublesElo); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.JSON(fiber.Map{"status": "updated"})
}

func (h *TournamentHandler) Export(c *fiber.Ctx) error {
	idStr := c.Params("id")
	csvBytes, err := h.exportUC.Execute(c.Context(), idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "text/csv")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tournament_report_%s.csv\"", idStr))
	return c.Send(csvBytes)
}

func (h *TournamentHandler) ExportPDF(c *fiber.Ctx) error {
	idStr := c.Params("id")
	pdfBytes, err := h.exportPdfUC.Execute(c.Context(), idStr)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	c.Set("Content-Type", "application/pdf")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"tournament_report_%s.pdf\"", idStr))
	return c.Send(pdfBytes)
}

func (h *TournamentHandler) MovePlayer(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		PlayerID      string `json:"playerId" form:"playerId"`
		TargetGroupID string `json:"targetGroupId" form:"targetGroupId"`
		TargetIndex   *int   `json:"targetIndex" form:"targetIndex"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	targetIndex := -1
	if body.TargetIndex != nil {
		targetIndex = *body.TargetIndex
	}

	if err := h.movePlayerUC.Execute(c.Context(), id, body.PlayerID, body.TargetGroupID, targetIndex); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	if c.Get("HX-Request") != "" {
		c.Set("HX-Trigger", "reload-bracket, reload-matches")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.SendString("OK")
}

func (h *TournamentHandler) CreateTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	var body struct {
		Name string `form:"name"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if _, err := h.createTeamUC.Execute(c.Context(), tournamentID, body.Name); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) DeleteTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	if err := h.deleteTeamUC.Execute(c.Context(), teamID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) AssignPlayerToTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	var body struct {
		PlayerID string `form:"playerId"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := h.assignPlayerToTeamUC.Execute(c.Context(), teamID, body.PlayerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) RemovePlayerFromTeam(c *fiber.Ctx) error {
	tournamentID := c.Params("id")
	teamID := c.Params("teamId")
	playerID := c.Params("playerId")
	if err := h.removePlayerFromTeamUC.Execute(c.Context(), teamID, playerID); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if c.Get("HX-Request") != "" {
		c.Set("HX-Refresh", "true")
		return c.SendStatus(fiber.StatusOK)
	}
	return c.Redirect(fmt.Sprintf("/admin/tournaments/%s", tournamentID))
}

func (h *TournamentHandler) PublicList(c *fiber.Ctx) error {
	lang := getLang(c)
	tournaments, err := h.getTournamentsUC.Execute(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.Render("public/tournaments", merge(tMap(lang), fiber.Map{
		"Tournaments":  tournaments,
		"Type":         "Tournaments",
		"OGImage":      c.BaseURL() + "/open_tdm.jpeg",
		"Title":        "Tournaments",
		"CanonicalURL": c.BaseURL() + c.Path(),
	}), "layouts/public")
}

func (h *TournamentHandler) PublicDetail(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")

	type result struct {
		tournament *tournamentDomain.Tournament
		err        error
		divisions  []*divisionDomain.Division
		players    any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	go func() {
		defer wg.Done()
		res.players, _ = h.leaderboardUC.ExecuteSingles(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}
	t := res.tournament
	divisions := res.divisions

	statusFilter := c.Query("status", "all")
	if statusFilter != "all" {
		var filtered []tournamentDomain.Match
		for _, m := range t.Matches {
			if m.Status == statusFilter {
				filtered = append(filtered, m)
			}
		}
		t.Matches = filtered
	}

	tmap, _ := c.Locals("T").(map[string]string)
	vm := BuildTournamentViewModel(t, divisions, tmap)
	vm.IsPublic = true

	// Build a map of Referee IDs to Names
	refereeNames := make(map[string]string)
	if allPlayers, ok := res.players.([]*player.Player); ok {
		for _, m := range t.Matches {
			if m.RefereeID != nil {
				for _, p := range allPlayers {
					if p.ID == *m.RefereeID {
						refereeNames[*m.RefereeID] = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
						break
					}
				}
			}
		}
	}

	// SEO additions
	canonicalURL := c.BaseURL() + c.Path()
	// Quick escape for JSON string (just in case there are quotes in name)
	safeName := strings.ReplaceAll(t.Name, `"`, `\"`)
	jsonLD := fmt.Sprintf(`{
  "@context": "https://schema.org",
  "@type": "SportsEvent",
  "name": "%s",
  "startDate": "%s",
  "endDate": "%s",
  "sport": "Table Tennis",
  "url": "%s",
  "location": {
    "@type": "Place",
    "name": "Nicaragua"
  }
}`, safeName, t.StartDate.Format(time.RFC3339), t.EndDate.Format(time.RFC3339), canonicalURL)

	return c.Render("public/tournament-detail", merge(tMap(lang), fiber.Map{
		"Tournament":       t,
		"Divisions":        divisions,
		"BracketViewModel": vm,
		"Type":             "Tournaments",
		"StatusFilter":     statusFilter,
		"RefereeNames":     refereeNames,
		"CanonicalURL":     canonicalURL,
		"OGImage":          c.BaseURL() + "/open_tdm.jpeg",
		"JSONLD":           jsonLD,
		"Title":            t.Name,
		"Description":      fmt.Sprintf("%s Tournament. Register and view live bracket.", t.Name),
	}), "layouts/public")
}

func (h *TournamentHandler) PublicTVDashboard(c *fiber.Ctx) error {
	lang := getLang(c)
	id := c.Params("id")

	type result struct {
		tournament *tournamentDomain.Tournament
		err        error
		divisions  []*divisionDomain.Division
		players    any
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	go func() {
		defer wg.Done()
		res.players, _ = h.leaderboardUC.ExecuteSingles(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}

	tmap, _ := c.Locals("T").(map[string]string)
	vm := BuildTournamentViewModel(res.tournament, res.divisions, tmap)
	vm.IsPublic = true

	scheduled, inProgress, finished := BuildBoardCards(res.tournament, res.divisions)
	tables := buildTables(res.tournament, "", h.getOccupiedTables(c.Context(), res.tournament))

	return c.Render("public/tv-dashboard", merge(tMap(lang), fiber.Map{
		"Tournament":       res.tournament,
		"Divisions":        res.divisions,
		"BracketViewModel": vm,
		"Scheduled":        scheduled,
		"InProgress":       inProgress,
		"Finished":         finished,
		"Tables":           tables,
	})) // No layout for TV
}

// BoardCard is a flattened match representation used by the kanban board.
type BoardCard struct {
	MatchID        string
	Status         string
	Stage          string
	BestOf         int
	PlayerAName    string
	PlayerBName    string
	P1Id           string
	P2Id           string
	TableNumber    *int
	ScoreA         int
	ScoreB         int
	Pin            string
	GroupName      string
	DivisionName   string
	P1InMatch      bool
	P2InMatch      bool
	TournamentID   string
	TournamentName string
}

type TableVM struct {
	Number int
	IsUsed bool
}

func (h *TournamentHandler) getOccupiedTables(ctx context.Context, t *tournamentDomain.Tournament) []int {
	occupiedList, _ := h.getOccupiedTablesUC.Execute(ctx, t)
	return occupiedList
}

func buildTables(t *tournamentDomain.Tournament, excludeMatchID string, globalOccupied []int) []TableVM {
	var tables []TableVM
	if t == nil || t.NumTables <= 0 {
		return tables
	}
	used := make(map[int]bool)
	for _, m := range t.Matches {
		if m.Status == "in_progress" && m.TableNumber != nil {
			if m.ID != excludeMatchID {
				used[*m.TableNumber] = true
			}
		}
	}
	for _, occ := range globalOccupied {
		used[occ] = true
	}
	for i := 1; i <= t.NumTables; i++ {
		tables = append(tables, TableVM{
			Number: i,
			IsUsed: used[i],
		})
	}
	return tables
}

func FilterBoardCards(cards []BoardCard, q string, divs []string) []BoardCard {
	if q == "" && len(divs) == 0 {
		return cards
	}

	divMap := make(map[string]bool)
	for _, d := range divs {
		divMap[d] = true
	}

	var filtered []BoardCard
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

func BuildBoardCards(t *tournamentDomain.Tournament, divs []*divisionDomain.Division) (scheduled, inProgress, finished []BoardCard) {
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
		card := BoardCard{
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
	vm := BuildTournamentViewModel(t, divs, nil)
	for _, dv := range vm.Divisions {
		if vm.Format == "round_robin" {
			for _, mv := range dv.RoundRobinMatches {
				if mv.Player1 != nil && mv.Player2 != nil {
					if !matchExists(t.Matches, mv.Player1.ID, mv.Player2.ID) {
						groupName := findGroupName(mv.Player1.ID)
						scheduled = append(scheduled, BoardCard{
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
						})
					}
				}
			}
		} else if vm.Format == "groups_elimination" {
			for _, g := range dv.Groups {
				for _, mv := range g.Matches {
					if mv.Player1 != nil && mv.Player2 != nil {
						if !matchExists(t.Matches, mv.Player1.ID, mv.Player2.ID) {
							scheduled = append(scheduled, BoardCard{
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
							})
						}
					}
				}
			}
			if dv.AllGroupsFinished {
				for _, round := range dv.KnockoutRounds {
					for _, bmv := range round.Matches {
						if bmv.Player1 != nil && bmv.Player2 != nil && bmv.Player1.Player != nil && bmv.Player2.Player != nil {
							if !matchExists(t.Matches, bmv.Player1.Player.ID, bmv.Player2.Player.ID) {
								scheduled = append(scheduled, BoardCard{
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
						if !matchExists(t.Matches, bmv.Player1.Player.ID, bmv.Player2.Player.ID) {
							scheduled = append(scheduled, BoardCard{
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
	var reordered []BoardCard
	var unstarted []BoardCard
	var virtualScheduled []BoardCard

	// Virtual matches (MatchID == "") are handled after real matches
	for _, c := range scheduled {
		if c.MatchID == "" {
			virtualScheduled = append(virtualScheduled, c)
		} else {
			unstarted = append(unstarted, c)
		}
	}

	simClock := time.Now().Add(24 * time.Hour) // start in future to override past matches
	
	scheduleMatchGreedy := func(pool *[]BoardCard) {
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
	}

	return
}

func matchExists(matches []tournamentDomain.Match, p1ID, p2ID string) bool {
	for _, m := range matches {
		if m.TeamMatchID != nil {
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

func (h *TournamentHandler) Board(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		tournament *tournamentDomain.Tournament
		err        error
		divisions  []*divisionDomain.Division
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}
	t := res.tournament
	divs := res.divisions
	scheduled, inProgress, finished := BuildBoardCards(t, divs)
	tables := buildTables(t, "", h.getOccupiedTables(c.Context(), t))

	uniqueDivsMap := make(map[string]bool)
	for _, c := range scheduled {
		if c.DivisionName != "" {
			uniqueDivsMap[c.DivisionName] = true
		}
	}
	for _, c := range inProgress {
		if c.DivisionName != "" {
			uniqueDivsMap[c.DivisionName] = true
		}
	}
	for _, c := range finished {
		if c.DivisionName != "" {
			uniqueDivsMap[c.DivisionName] = true
		}
	}
	var allDivs []string
	for d := range uniqueDivsMap {
		allDivs = append(allDivs, d)
	}
	sort.Strings(allDivs)

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

	selectedDivsMap := make(map[string]bool)
	for _, d := range selectedDivs {
		selectedDivsMap[d] = true
	}

	return c.Render("admin/tournament-board", fiber.Map{
		"Tournament":   t,
		"Scheduled":    scheduled,
		"InProgress":   inProgress,
		"Finished":     finished,
		"Tables":       tables,
		"AllDivisions": allDivs,
		"QueryQ":       c.Query("q"),
		"SelectedDivs": selectedDivsMap,
	}, "layouts/admin")
}

func (h *TournamentHandler) BoardColumns(c *fiber.Ctx) error {
	id := c.Params("id")

	type result struct {
		tournament *tournamentDomain.Tournament
		err        error
		divisions  []*divisionDomain.Division
	}
	var res result
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		res.tournament, res.err = h.getByID.Execute(c.Context(), id)
	}()
	go func() {
		defer wg.Done()
		res.divisions, _ = h.divisionUC.GetAll(c.Context())
	}()
	wg.Wait()

	if res.err != nil {
		return fiber.NewError(fiber.StatusNotFound, res.err.Error())
	}
	t := res.tournament
	divs := res.divisions
	scheduled, inProgress, finished := BuildBoardCards(t, divs)
	tables := buildTables(t, "", h.getOccupiedTables(c.Context(), t))

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
		"Tournament": t,
		"Scheduled":  scheduled,
		"InProgress": inProgress,
		"Finished":   finished,
		"Tables":     tables,
		"T":          c.Locals("T"),
		"Lang":       c.Locals("Lang"),
	})
}


