package handler

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"table-tennis-backend/internal/application/match"
	appTournament "table-tennis-backend/internal/application/tournament"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/domain/tournament"

	"table-tennis-backend/internal/infrastructure/persistence/bun"
	"table-tennis-backend/internal/interfaces/http/i18n"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type MatchHandler struct {
	createUC           *match.CreateMatchUseCase
	finishUC           *match.FinishMatchUseCase
	updateScoreUC      *match.UpdateMatchScoreUseCase
	playerRepo         *bun.PlayerRepository
	matchRepo          *bun.MatchRepository
	tournamentRepo     *bun.TournamentRepository
	finishTournamentUC *appTournament.FinishTournamentUseCase
}

func NewMatchHandler(
	createUC *match.CreateMatchUseCase,
	finishUC *match.FinishMatchUseCase,
	updateScoreUC *match.UpdateMatchScoreUseCase,
	playerRepo *bun.PlayerRepository,
	matchRepo *bun.MatchRepository,
	tournamentRepo *bun.TournamentRepository,
	finishTournamentUC *appTournament.FinishTournamentUseCase,
) *MatchHandler {
	return &MatchHandler{
		createUC:           createUC,
		finishUC:           finishUC,
		updateScoreUC:      updateScoreUC,
		playerRepo:         playerRepo,
		matchRepo:          matchRepo,
		tournamentRepo:     tournamentRepo,
		finishTournamentUC: finishTournamentUC,
	}
}

func (h *MatchHandler) getOccupiedTables(ctx context.Context, t *tournament.Tournament) []int {
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

func (h *MatchHandler) broadcastToTournamentOrEvent(ctx context.Context, tournamentID string, eventData map[string]string) {
	t, err := h.tournamentRepo.GetByID(ctx, tournamentID)
	if err == nil && t.EventID != nil {
		eventUUID, _ := uuid.Parse(*t.EventID)
		if tourneys, err := h.tournamentRepo.GetByEventID(ctx, eventUUID, false); err == nil {
			for _, tourney := range tourneys {
				GlobalBracketHub.Broadcast(tourney.ID, eventData)
			}
			return
		}
	}
	GlobalBracketHub.Broadcast(tournamentID, eventData)
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

	if _, err := uuid.Parse(body.TournamentID); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid tournament id")
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
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
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
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	mUUID, err := uuid.Parse(body.MatchID)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid match ID")
	}

	mModel, err := h.matchRepo.GetByID(c.Context(), mUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "match not found")
	}

	t, err := h.tournamentRepo.GetByID(c.Context(), mModel.TournamentID.String())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var matched *tournament.Match
	for i := range t.Matches {
		if t.Matches[i].ID == body.MatchID {
			matched = &t.Matches[i]
			break
		}
	}

	if matched == nil {
		return fiber.NewError(fiber.StatusNotFound, "match not found in tournament list")
	}

	tx, err := h.matchRepo.DB().BeginTx(c.Context(), nil)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	defer tx.Rollback()

	// Update match to finished
	mModel.Status = "finished"
	mModel.WinnerTeam = &body.WinnerTeam
	now := time.Now()
	mModel.UpdatedAt = &now

	_, err = tx.NewUpdate().Model(mModel).WherePK().Column("status", "winner_team", "updated_at").Exec(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Advance winner to next match slot if configured
	if mModel.NextMatchID != nil {
		nextID, _ := uuid.Parse(*mModel.NextMatchID)
		winnedPlayerID := mModel.TeamAPlayer1ID
		if body.WinnerTeam == "B" {
			winnedPlayerID = mModel.TeamBPlayer1ID
		}
		if mModel.NextMatchSlot == "A" {
			_, _ = tx.NewUpdate().TableExpr("matches").Set("team_a_player_1_id = ?, status = 'scheduled'", winnedPlayerID).Where("id = ? AND status = 'scheduled'", nextID).Exec(c.Context())
		} else {
			_, _ = tx.NewUpdate().TableExpr("matches").Set("team_b_player_1_id = ?, status = 'scheduled'", winnedPlayerID).Where("id = ? AND status = 'scheduled'", nextID).Exec(c.Context())
		}
	}

	// If this was a sub-match of a team match, update parent team match status
	if mModel.TeamMatchID != nil {
		var siblingMatches []bun.MatchModel
		_ = tx.NewSelect().Model(&siblingMatches).Where("team_match_id = ?", mModel.TeamMatchID).Scan(c.Context())

		subWinsA, subWinsB := 0, 0
		for _, sm := range siblingMatches {
			if sm.ID == mModel.ID {
				if body.WinnerTeam == "A" {
					subWinsA++
				} else {
					subWinsB++
				}
				continue
			}
			if sm.Status == "finished" && sm.WinnerTeam != nil {
				if *sm.WinnerTeam == "A" {
					subWinsA++
				} else if *sm.WinnerTeam == "B" {
					subWinsB++
				}
			}
		}

		parentMatch := new(bun.MatchModel)
		if err := tx.NewSelect().Model(parentMatch).Where("id = ?", mModel.TeamMatchID).Scan(c.Context()); err == nil {
			if subWinsA >= 3 {
				w := "A"
				parentMatch.WinnerTeam = &w
				parentMatch.Status = "finished"
			} else if subWinsB >= 3 {
				w := "B"
				parentMatch.WinnerTeam = &w
				parentMatch.Status = "finished"
			} else {
				parentMatch.Status = "in_progress"
			}
			pNow := time.Now()
			parentMatch.UpdatedAt = &pNow
			_, _ = tx.NewUpdate().Model(parentMatch).WherePK().Column("status", "winner_team", "updated_at").Exec(c.Context())

			if parentMatch.Status == "finished" {
				_, _ = tx.NewUpdate().TableExpr("matches").
					Set("status = 'scheduled'").
					Where("team_match_id = ? AND status = 'in_progress' AND id != ?", mModel.TeamMatchID, mModel.ID).
					Exec(c.Context())
			}

			if parentMatch.Status == "finished" && parentMatch.NextMatchID != nil {
				nextID, _ := uuid.Parse(*parentMatch.NextMatchID)
				winnedTeamID := parentMatch.TeamAPlayer1ID
				if *parentMatch.WinnerTeam == "B" {
					winnedTeamID = parentMatch.TeamBPlayer1ID
				}
				if parentMatch.NextMatchSlot == "A" {
					_, _ = tx.NewUpdate().TableExpr("matches").Set("team_a_player_1_id = ?, status = 'scheduled'", winnedTeamID).Where("id = ? AND status = 'scheduled'", nextID).Exec(c.Context())
				} else {
					_, _ = tx.NewUpdate().TableExpr("matches").Set("team_b_player_1_id = ?, status = 'scheduled'", winnedTeamID).Where("id = ? AND status = 'scheduled'", nextID).Exec(c.Context())
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Apply Elo
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
	matchName := nameA + " vs " + nameB
	winStr := matchName
	if mModel.WinnerTeam != nil {
		if *mModel.WinnerTeam == "A" {
			winStr = nameA + " defeated " + nameB
		} else if *mModel.WinnerTeam == "B" {
			winStr = nameB + " defeated " + nameA
		}
	}

	h.broadcastToTournamentOrEvent(c.Context(), mModel.TournamentID.String(), map[string]string{
		"event":        "score_updated",
		"tournamentId": mModel.TournamentID.String(),
		"matchId":      body.MatchID,
		"matchStatus":  "finished",
		"message":      fmt.Sprintf("Match finished: %s", winStr),
	})

	// Re-fetch tournament to render updated row
	t, _ = h.tournamentRepo.GetByID(c.Context(), mModel.TournamentID.String())
	var updatedMatched *tournament.Match
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

	// Load match metadata (pin, referee, table number, status) first
	var matchPin string
	var matchRefereeID *uuid.UUID
	var matchTableNumber *int
	matchStatus := "scheduled"
	if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
		mUUID, _ := uuid.Parse(matchID)
		if mModel, err := h.matchRepo.GetByID(c.Context(), mUUID); err == nil {
			matchPin = mModel.Pin
			matchRefereeID = mModel.RefereeID
			matchTableNumber = mModel.TableNumber
			matchStatus = mModel.Status
		}
	}
	var refereeIDStr string
	if matchRefereeID != nil {
		refereeIDStr = matchRefereeID.String()
	}

	// Fetch tournament if tournamentId is provided
	var tourney *tournament.Tournament
	if tID != "" {
		if t, err := h.tournamentRepo.GetByID(c.Context(), tID); err == nil {
			tourney = t
		}
	}

	// Check if tournament is teams
	var isTeams bool
	var isSubMatch bool
	var teamA, teamB *tournament.Team
	var subMatches []bun.MatchModel
	var teamFormat string
	if tourney != nil && tourney.Type == "teams" {
		tUUID, _ := uuid.Parse(tID)
		t := tourney
		// If matchID refers to a sub-match (has team_match_id), treat as regular singles/doubles
		if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
			mUUID, _ := uuid.Parse(matchID)
			if existingMatch, err := h.matchRepo.GetByID(c.Context(), mUUID); err == nil && existingMatch.TeamMatchID != nil {
				isSubMatch = true
			}
		}

		if !isSubMatch {
			isTeams = true
			teamFormat = t.TeamFormat
			if teamFormat == "" {
				teamFormat = "olympic"
			}
			for _, team := range t.Teams {
				if team.ID == p1Id {
					teamA = team
				}
				if team.ID == p2Id {
					teamB = team
				}
			}

			// Render team matchup view
			if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
				parentUUID, _ := uuid.Parse(matchID)
				_ = h.matchRepo.DB().NewSelect().Model(&subMatches).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())
			}

			if len(subMatches) == 0 {
				// Check if we need to auto-create parent match
				var parentUUID uuid.UUID
				if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
					parentUUID, _ = uuid.Parse(matchID)
				} else {
					parentUUID = uuid.New()
					matchID = parentUUID.String()

					// Get player IDs for team A and team B
					var teamAPlayerIDs, teamBPlayerIDs []string
					if teamA != nil {
						for _, p := range teamA.Players {
							teamAPlayerIDs = append(teamAPlayerIDs, p.ID)
						}
					}
					if teamB != nil {
						for _, p := range teamB.Players {
							teamBPlayerIDs = append(teamBPlayerIDs, p.ID)
						}
					}

					if len(teamAPlayerIDs) > 0 && len(teamBPlayerIDs) > 0 {
						_, _ = h.createUC.Execute(c.Context(), tID, "teams", teamAPlayerIDs, teamBPlayerIDs, stage)
					}
				}

				// Only create sub-matches if both teams have at least one player (FK constraint)
				if len(subMatches) == 0 && teamA != nil && len(teamA.Players) > 0 && teamB != nil && len(teamB.Players) > 0 {
					for order := 1; order <= 5; order++ {
						subID := uuid.New()
						matchType := "singles"
						if teamFormat == "olympic" && order == 1 {
							matchType = "doubles"
						}

						p1ID, _ := uuid.Parse(teamA.Players[0].ID)
						p2ID, _ := uuid.Parse(teamB.Players[0].ID)

						subModel := &bun.MatchModel{
							ID:             subID,
							TournamentID:   tUUID,
							MatchType:      matchType,
							TeamAPlayer1ID: p1ID,
							TeamBPlayer1ID: p2ID,
							Status:         "scheduled",
							Stage:          stage,
							RoundNumber:    order,
							TeamMatchID:    &parentUUID,
							Pin:            h.matchRepo.GenerateUniquePin(c.Context()),
						}
						_, _ = h.matchRepo.DB().NewInsert().Model(subModel).Exec(c.Context())
					}
					_ = h.matchRepo.DB().NewSelect().Model(&subMatches).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())
				}
			}
		}
	}

	lang := getLang(c)
	tMap := i18n.PrecomputedMaps[lang]

	if isTeams {
		playerNames := make(map[string]string)
		var playerModels []bun.PlayerModel
		_ = h.matchRepo.DB().NewSelect().Model(&playerModels).Scan(c.Context())
		for _, p := range playerModels {
			playerNames[p.ID.String()] = p.FullName()
		}

		var matchesVM []map[string]interface{}
		for _, sm := range subMatches {
			var pAName, pBName string
			teamAP2 := ""
			teamBP2 := ""

			if sm.MatchType == "singles" {
				pAName = playerNames[sm.TeamAPlayer1ID.String()]
				pBName = playerNames[sm.TeamBPlayer1ID.String()]
			} else {
				pAName = "Doubles"
				pBName = "Doubles"
				if sm.TeamAPlayer2ID != nil {
					pAName = playerNames[sm.TeamAPlayer1ID.String()] + " & " + playerNames[sm.TeamAPlayer2ID.String()]
					teamAP2 = sm.TeamAPlayer2ID.String()
				} else {
					pAName = playerNames[sm.TeamAPlayer1ID.String()]
				}
				if sm.TeamBPlayer2ID != nil {
					pBName = playerNames[sm.TeamBPlayer1ID.String()] + " & " + playerNames[sm.TeamBPlayer2ID.String()]
					teamBP2 = sm.TeamBPlayer2ID.String()
				} else {
					pBName = playerNames[sm.TeamBPlayer1ID.String()]
				}
			}

			// Calculate wins count exactly like renderTeamMatchFormInternal
			var setModels []bun.MatchSetModel
			_ = h.matchRepo.DB().NewSelect().Model(&setModels).Where("match_id = ?", sm.ID).Scan(c.Context())

			winsA, winsB := 0, 0
			for _, set := range setModels {
				if set.ScoreA > set.ScoreB {
					winsA++
				} else if set.ScoreB > set.ScoreA {
					winsB++
				}
			}

			wt := ""
			if sm.WinnerTeam != nil {
				wt = *sm.WinnerTeam
			}

			alignA, alignB := getSubMatchAlignments(sm.RoundNumber, teamFormat)

			matchesVM = append(matchesVM, map[string]interface{}{
				"ID":             sm.ID.String(),
				"RoundNumber":    sm.RoundNumber,
				"MatchType":      sm.MatchType,
				"TeamAPlayer1ID": sm.TeamAPlayer1ID.String(),
				"TeamAPlayer2ID": teamAP2,
				"TeamBPlayer1ID": sm.TeamBPlayer1ID.String(),
				"TeamBPlayer2ID": teamBP2,
				"Status":         sm.Status,
				"PlayerAName":    pAName,
				"PlayerBName":    pBName,
				"AlignmentA":     alignA,
				"AlignmentB":     alignB,
				"ScoreA":         winsA,
				"ScoreB":         winsB,
				"WinnerTeam":     wt,
			})
		}

		squadAP1, squadAP2, squadAP3 := "", "", ""
		squadBP1, squadBP2, squadBP3 := "", "", ""
		if len(subMatches) > 0 {
			// Prepopulate dropdown selections using standard olympic round 1 & 2
			for _, sm := range subMatches {
				if sm.RoundNumber == 1 {
					squadAP1 = sm.TeamAPlayer1ID.String()
					squadBP1 = sm.TeamBPlayer1ID.String()
					if sm.TeamAPlayer2ID != nil {
						squadAP2 = sm.TeamAPlayer2ID.String()
					}
					if sm.TeamBPlayer2ID != nil {
						squadBP2 = sm.TeamBPlayer2ID.String()
					}
				}
				if sm.RoundNumber == 2 {
					squadAP3 = sm.TeamAPlayer1ID.String()
					squadBP3 = sm.TeamBPlayer1ID.String()
				}
			}
		}

		var participants []*player.Player
		if tID != "" {
			if t, err := h.tournamentRepo.GetByID(c.Context(), tID); err == nil {
				participants = t.Participants
			}
		}

		var teamMatchTemplateName string
		if strings.HasPrefix(templateName, "public/") {
			teamMatchTemplateName = "public/team-match-score-form"
		} else {
			teamMatchTemplateName = "admin/partials/team-match-score-form"
		}

		return c.Render(teamMatchTemplateName, fiber.Map{
			"MatchID":      matchID,
			"TournamentID": tID,
			"Stage":        stage,
			"TeamA":        teamA,
			"TeamB":        teamB,
			"SubMatches":   matchesVM,
			"SquadAP1":     squadAP1,
			"SquadAP2":     squadAP2,
			"SquadAP3":     squadAP3,
			"SquadBP1":     squadBP1,
			"SquadBP2":     squadBP2,
			"SquadBP3":     squadBP3,
			"Participants": participants,
			"Pin":          matchPin,
			"RefereeID":    refereeIDStr,
			"TableNumber":  matchTableNumber,
			"TableNumberVal": func() int {
				if matchTableNumber != nil {
					return *matchTableNumber
				}
				return 0
			}(),
			"Tables": buildTables(tourney, matchID, h.getOccupiedTables(c.Context(), tourney)),
			"T":      tMap,
		})
	}

	// tourney is already fetched above

	// Fetch existing match if matchID is provided
	var existingMatch *bun.MatchModel
	if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
		mUUID, _ := uuid.Parse(matchID)
		if em, err := h.matchRepo.GetByID(c.Context(), mUUID); err == nil {
			existingMatch = em
		}
	}

	// Determine match type / doubles status
	isDoubles := false
	if tourney != nil && (tourney.Type == "doubles" || tourney.Type == "mixed_doubles") {
		isDoubles = true
	} else if existingMatch != nil && existingMatch.MatchType == "doubles" {
		isDoubles = true
	}

	playerAName := "Player 1"
	playerBName := "Player 2"
	var playerANames, playerBNames string

	if isDoubles {
		var p1A, p2A, p1B, p2B *player.Player

		if existingMatch != nil {
			p1UUID := existingMatch.TeamAPlayer1ID
			p1B_UUID := existingMatch.TeamBPlayer1ID
			var p2UUID, p2B_UUID *uuid.UUID
			if existingMatch.TeamAPlayer2ID != nil {
				p2UUID = existingMatch.TeamAPlayer2ID
			}
			if existingMatch.TeamBPlayer2ID != nil {
				p2B_UUID = existingMatch.TeamBPlayer2ID
			}

			if p, err := h.playerRepo.GetById(c.Context(), p1UUID.String()); err == nil {
				p1A = p
			}
			if p2UUID != nil {
				if p, err := h.playerRepo.GetById(c.Context(), p2UUID.String()); err == nil {
					p2A = p
				}
			}
			if p, err := h.playerRepo.GetById(c.Context(), p1B_UUID.String()); err == nil {
				p1B = p
			}
			if p2B_UUID != nil {
				if p, err := h.playerRepo.GetById(c.Context(), p2B_UUID.String()); err == nil {
					p2B = p
				}
			}
		} else {
			if tourney != nil {
				for _, team := range tourney.Teams {
					if team.ID == p1Id {
						if len(team.Players) > 0 {
							p1A = team.Players[0]
						}
						if len(team.Players) > 1 {
							p2A = team.Players[1]
						}
					}
					if team.ID == p2Id {
						if len(team.Players) > 0 {
							p1B = team.Players[0]
						}
						if len(team.Players) > 1 {
							p2B = team.Players[1]
						}
					}
				}
			}
		}

		// Look up team names
		var teamAName, teamBName string
		if tourney != nil {
			if p1A != nil {
				for _, team := range tourney.Teams {
					for _, tp := range team.Players {
						if tp.ID == p1A.ID {
							teamAName = team.Name
							break
						}
					}
					if teamAName != "" {
						break
					}
				}
			}
			if p1B != nil {
				for _, team := range tourney.Teams {
					for _, tp := range team.Players {
						if tp.ID == p1B.ID {
							teamBName = team.Name
							break
						}
					}
					if teamBName != "" {
						break
					}
				}
			}
		}

		// Fallbacks & combining player names
		if p1A != nil {
			playerANames = p1A.FirstNameWithSecond() + " " + p1A.LastNameWithSecond()
			if p2A != nil {
				playerANames += " & " + p2A.FirstNameWithSecond() + " " + p2A.LastNameWithSecond()
			}
		}
		if p1B != nil {
			playerBNames = p1B.FirstNameWithSecond() + " " + p1B.LastNameWithSecond()
			if p2B != nil {
				playerBNames += " & " + p2B.FirstNameWithSecond() + " " + p2B.LastNameWithSecond()
			}
		}

		if teamAName != "" {
			playerAName = teamAName
		} else if playerANames != "" {
			playerAName = playerANames
		}

		if teamBName != "" {
			playerBName = teamBName
		} else if playerBNames != "" {
			playerBName = playerBNames
		}
	} else {
		// Singles flow
		if p1Id != "" {
			if p, err := h.playerRepo.GetById(c.Context(), p1Id); err == nil {
				playerAName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		} else if existingMatch != nil {
			if p, err := h.playerRepo.GetById(c.Context(), existingMatch.TeamAPlayer1ID.String()); err == nil {
				playerAName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		}

		if p2Id != "" {
			if p, err := h.playerRepo.GetById(c.Context(), p2Id); err == nil {
				playerBName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		} else if existingMatch != nil {
			if p, err := h.playerRepo.GetById(c.Context(), existingMatch.TeamBPlayer1ID.String()); err == nil {
				playerBName = p.FirstNameWithSecond() + " " + p.LastNameWithSecond()
			}
		}
	}

	type setVM struct {
		Number int
		ScoreA interface{}
		ScoreB interface{}
	}
	var sets []setVM

	// Load existing scores if matchID is present
	existingScores := make(map[int]bun.MatchSetModel)
	if matchID != "" && matchID != "nil" && matchID != "null" && matchID != "undefined" {
		if s, err := h.matchRepo.GetSets(c.Context(), matchID); err == nil {
			for _, sm := range s {
				existingScores[sm.SetNumber] = sm
			}
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

	var participants []*player.Player
	if tID != "" {
		if t, err := h.tournamentRepo.GetByID(c.Context(), tID); err == nil {
			participants = t.Participants
		}
	}

	return c.Render(templateName, fiber.Map{
		"MatchID":      matchID,
		"TournamentID": tID,
		"Stage":        stage,
		"BestOf":       bestOf,
		"PlayerA":      playerAName,
		"PlayerB":      playerBName,
		"Sets":         sets,
		"P1Id":         p1Id,
		"P2Id":         p2Id,
		"IsSubMatch":   isSubMatch,
		"IsDoubles":    isDoubles,
		"PlayerANames": playerANames,
		"PlayerBNames": playerBNames,
		"Pin":          matchPin,
		"RefereeID":    refereeIDStr,
		"TableNumber":  matchTableNumber,
		"TableNumberVal": func() int {
			if matchTableNumber != nil {
				return *matchTableNumber
			}
			return 0
		}(),
		"Status":       matchStatus,
		"Participants": participants,
		"Tables":       buildTables(tourney, matchID, h.getOccupiedTables(c.Context(), tourney)),
		"T":            tMap,
	})
}

// UpdateScore accepts set scores via JSON/form and persists them, auto-resolving winner.
func (h *MatchHandler) UpdateScore(c *fiber.Ctx) error {
	if c.FormValue("action") == "update_squads" {
		matchID := c.Params("id")
		if matchID == "" {
			matchID = c.FormValue("matchId")
		}
		parentUUID, _ := uuid.Parse(matchID)

		p1A, _ := uuid.Parse(c.FormValue("squad_a_p1"))
		p2A, _ := uuid.Parse(c.FormValue("squad_a_p2"))
		p3A, _ := uuid.Parse(c.FormValue("squad_a_p3"))

		p1B, _ := uuid.Parse(c.FormValue("squad_b_p1"))
		p2B, _ := uuid.Parse(c.FormValue("squad_b_p2"))
		p3B, _ := uuid.Parse(c.FormValue("squad_b_p3"))

		// Validate that all required players are selected
		if p1A == uuid.Nil || p2A == uuid.Nil || p3A == uuid.Nil || p1B == uuid.Nil || p2B == uuid.Nil || p3B == uuid.Nil {
			return fiber.NewError(fiber.StatusBadRequest, "All 3 players must be selected for each team")
		}

		parent, err := h.matchRepo.GetByID(c.Context(), parentUUID)
		if err != nil {
			return fiber.NewError(fiber.StatusNotFound, "parent match not found: "+err.Error())
		}

		t, err := h.tournamentRepo.GetByID(c.Context(), parent.TournamentID.String())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "tournament not found: "+err.Error())
		}

		teamFormat := t.TeamFormat
		if teamFormat == "" {
			teamFormat = "olympic"
		}

		var subs []bun.MatchModel
		_ = h.matchRepo.DB().NewSelect().Model(&subs).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())

		// Create sub-matches if they don't exist yet
		if len(subs) == 0 {
			for order := 1; order <= 5; order++ {
				subID := uuid.New()
				matchType := "singles"
				if teamFormat == "olympic" && order == 1 {
					matchType = "doubles"
				}
				subModel := &bun.MatchModel{
					ID:             subID,
					TournamentID:   parent.TournamentID,
					MatchType:      matchType,
					TeamAPlayer1ID: p1A,
					TeamBPlayer1ID: p1B,
					Status:         "scheduled",
					Stage:          parent.Stage,
					RoundNumber:    order,
					TeamMatchID:    &parentUUID,
					Pin:            h.matchRepo.GenerateUniquePin(c.Context()),
				}
				if _, err := h.matchRepo.DB().NewInsert().Model(subModel).Exec(c.Context()); err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "failed to create sub-match: "+err.Error())
				}
			}
			_ = h.matchRepo.DB().NewSelect().Model(&subs).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())
		}

		for _, sub := range subs {
			var teamAP1, teamAP2, teamBP1, teamBP2 uuid.UUID
			if teamFormat == "olympic" {
				switch sub.RoundNumber {
				case 1:
					teamAP1, teamAP2 = p1A, p2A
					teamBP1, teamBP2 = p1B, p2B
				case 2:
					teamAP1 = p3A
					teamBP1 = p3B
				case 3:
					teamAP1 = p1A
					teamBP1 = p1B
				case 4:
					teamAP1 = p2A
					teamBP1 = p2B
				case 5:
					teamAP1 = p3A
					teamBP1 = p1B
				}
			} else {
				switch sub.RoundNumber {
				case 1:
					teamAP1 = p1A
					teamBP1 = p1B
				case 2:
					teamAP1 = p2A
					teamBP1 = p2B
				case 3:
					teamAP1 = p3A
					teamBP1 = p3B
				case 4:
					teamAP1 = p1A
					teamBP1 = p2B
				case 5:
					teamAP1 = p2A
					teamBP1 = p1B
				}
			}

			var teamAP2Ptr, teamBP2Ptr *uuid.UUID
			if teamAP2 != uuid.Nil {
				teamAP2Ptr = &teamAP2
			}
			if teamBP2 != uuid.Nil {
				teamBP2Ptr = &teamBP2
			}

			sub.TeamAPlayer1ID = teamAP1
			sub.TeamAPlayer2ID = teamAP2Ptr
			sub.TeamBPlayer1ID = teamBP1
			sub.TeamBPlayer2ID = teamBP2Ptr
			if _, err := h.matchRepo.DB().NewUpdate().Model(&sub).WherePK().Column("team_a_player_1_id", "team_a_player_2_id", "team_b_player_1_id", "team_b_player_2_id").Exec(c.Context()); err != nil {
				return fiber.NewError(fiber.StatusInternalServerError, "failed to update sub-match: "+err.Error())
			}
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
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	if matchID == "" {
		matchID = body.MatchID
	}

	// If still no matchID, look for existing match first, then create on the fly
	if matchID == "" && body.P1Id != "" && body.P2Id != "" {
		tUUID, _ := uuid.Parse(body.TournamentID)
		p1UUID, _ := uuid.Parse(body.P1Id)
		p2UUID, _ := uuid.Parse(body.P2Id)

		// Try to find existing match for these players
		var existing bun.MatchModel
		err := h.matchRepo.DB().NewSelect().Model(&existing).
			Where("tournament_id = ?", tUUID).
			Where("team_match_id IS NULL").
			Where("((team_a_player_1_id = ? AND team_b_player_1_id = ?) OR (team_a_player_1_id = ? AND team_b_player_1_id = ?))",
				p1UUID, p2UUID, p2UUID, p1UUID).
			Scan(c.Context())
		if err == nil {
			matchID = existing.ID.String()
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
		if mUUID, err := uuid.Parse(matchID); err == nil {
			if m, err := h.matchRepo.GetByID(c.Context(), mUUID); err == nil {
				if refereeIDStr != "" {
					if refUUID, err := uuid.Parse(refereeIDStr); err == nil {
						m.RefereeID = &refUUID
					}
				} else {
					m.RefereeID = nil
				}

				if tableNumberStr != "" {
					if tNum, err := strconv.Atoi(tableNumberStr); err == nil {
						// Check if another match in this tournament/event is currently in_progress on this table
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
								c.Set("HX-Trigger", `{"show-toast": {"message": "Table is currently occupied by another match!", "type": "error"}}`)
								return fiber.NewError(fiber.StatusBadRequest, "Table is occupied")
							}
						}
						m.TableNumber = &tNum
					}
				} else {
					m.TableNumber = nil
				}

				_, _ = h.matchRepo.DB().NewUpdate().Model(m).WherePK().Column("referee_id", "table_number").Exec(c.Context())
			}
		}
	}

	if err := h.updateScoreUC.Execute(c.Context(), matchID, body.Scores, body.TournamentID, body.Stage); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Broadcast real-time update to all bracket viewers for this tournament
	var nameA, nameB string
	var scored *bun.MatchModel
	if mUUID, err := uuid.Parse(matchID); err == nil {
		scored, _ = h.matchRepo.GetByID(c.Context(), mUUID)
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
		"event":        "score_updated",
		"tournamentId": body.TournamentID,
		"matchId":      matchID,
	}
	if scored != nil && scored.Status == "finished" {
		broadcastData["matchStatus"] = "finished"
		broadcastData["message"] = fmt.Sprintf("Match finished: %s", matchName)
	}

	h.broadcastToTournamentOrEvent(c.Context(), body.TournamentID, broadcastData)

	// If this was a sub-match, return to the team matchup form instead of refreshing
	mUUID, _ := uuid.Parse(matchID)
	if scored, err := h.matchRepo.GetByID(c.Context(), mUUID); err == nil && scored.TeamMatchID != nil {
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
		parent, err := h.matchRepo.GetByID(c.Context(), parentUUID)
		if err != nil {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Match not found: " + err.Error() + "</div>")
		}

		// Validate PIN against tournament participants and officials
		submittedPin := c.FormValue("pin")
		updaterPlayerID, err := h.tournamentRepo.GetParticipantOrOfficialByPIN(c.Context(), parent.TournamentID.String(), submittedPin)
		if err != nil || updaterPlayerID == "" {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Invalid PIN. Please try again.</div>")
		}

		p1A, _ := uuid.Parse(c.FormValue("squad_a_p1"))
		p2A, _ := uuid.Parse(c.FormValue("squad_a_p2"))
		p3A, _ := uuid.Parse(c.FormValue("squad_a_p3"))

		p1B, _ := uuid.Parse(c.FormValue("squad_b_p1"))
		p2B, _ := uuid.Parse(c.FormValue("squad_b_p2"))
		p3B, _ := uuid.Parse(c.FormValue("squad_b_p3"))

		// Validate that all required players are selected
		if p1A == uuid.Nil || p2A == uuid.Nil || p3A == uuid.Nil || p1B == uuid.Nil || p2B == uuid.Nil || p3B == uuid.Nil {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>All 3 players must be selected for each team</div>")
		}

		t, err := h.tournamentRepo.GetByID(c.Context(), parent.TournamentID.String())
		if err != nil {
			return c.SendString("<div class='text-red-400 font-mono text-sm'>Tournament not found: " + err.Error() + "</div>")
		}

		teamFormat := t.TeamFormat
		if teamFormat == "" {
			teamFormat = "olympic"
		}

		var subs []bun.MatchModel
		_ = h.matchRepo.DB().NewSelect().Model(&subs).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())

		// Create sub-matches if they don't exist yet
		if len(subs) == 0 {
			for order := 1; order <= 5; order++ {
				subID := uuid.New()
				matchType := "singles"
				if teamFormat == "olympic" && order == 1 {
					matchType = "doubles"
				}
				subModel := &bun.MatchModel{
					ID:             subID,
					TournamentID:   parent.TournamentID,
					MatchType:      matchType,
					TeamAPlayer1ID: p1A,
					TeamBPlayer1ID: p1B,
					Status:         "scheduled",
					Stage:          parent.Stage,
					RoundNumber:    order,
					TeamMatchID:    &parentUUID,
					Pin:            h.matchRepo.GenerateUniquePin(c.Context()),
				}
				if _, err := h.matchRepo.DB().NewInsert().Model(subModel).Exec(c.Context()); err != nil {
					return c.SendString("<div class='text-red-400 font-mono text-sm'>Failed to create sub-match: " + err.Error() + "</div>")
				}
			}
			_ = h.matchRepo.DB().NewSelect().Model(&subs).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())
		}

		for _, sub := range subs {
			var teamAP1, teamAP2, teamBP1, teamBP2 uuid.UUID
			if teamFormat == "olympic" {
				switch sub.RoundNumber {
				case 1:
					teamAP1, teamAP2 = p1A, p2A
					teamBP1, teamBP2 = p1B, p2B
				case 2:
					teamAP1 = p3A
					teamBP1 = p3B
				case 3:
					teamAP1 = p1A
					teamBP1 = p1B
				case 4:
					teamAP1 = p2A
					teamBP1 = p2B
				case 5:
					teamAP1 = p3A
					teamBP1 = p1B
				}
			} else {
				switch sub.RoundNumber {
				case 1:
					teamAP1 = p1A
					teamBP1 = p1B
				case 2:
					teamAP1 = p2A
					teamBP1 = p2B
				case 3:
					teamAP1 = p3A
					teamBP1 = p3B
				case 4:
					teamAP1 = p1A
					teamBP1 = p2B
				case 5:
					teamAP1 = p2A
					teamBP1 = p1B
				}
			}

			var teamAP2Ptr, teamBP2Ptr *uuid.UUID
			if teamAP2 != uuid.Nil {
				teamAP2Ptr = &teamAP2
			}
			if teamBP2 != uuid.Nil {
				teamBP2Ptr = &teamBP2
			}

			sub.TeamAPlayer1ID = teamAP1
			sub.TeamAPlayer2ID = teamAP2Ptr
			sub.TeamBPlayer1ID = teamBP1
			sub.TeamBPlayer2ID = teamBP2Ptr
			if _, err := h.matchRepo.DB().NewUpdate().Model(&sub).WherePK().Column("team_a_player_1_id", "team_a_player_2_id", "team_b_player_1_id", "team_b_player_2_id").Exec(c.Context()); err != nil {
				return c.SendString("<div class='text-red-400 font-mono text-sm'>Failed to update sub-match: " + err.Error() + "</div>")
			}
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
			refUUID, err := uuid.Parse(refereeIDStr)
			if err == nil {
				parent.RefereeID = &refUUID
			}
		} else {
			refUUID, _ := uuid.Parse(updaterPlayerID)
			parent.RefereeID = &refUUID
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
			Where("tournament_id = ?", tUUID).
			Where("team_match_id IS NULL").
			Where("((team_a_player_1_id = ? AND team_b_player_1_id = ?) OR (team_a_player_1_id = ? AND team_b_player_1_id = ?))",
				p1UUID, p2UUID, p2UUID, p1UUID).
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
	m, err := h.matchRepo.GetByID(c.Context(), mUUID)
	if err != nil {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>Match not found</div>")
	}

	submittedPin := c.FormValue("pin")

	// Validate PIN against tournament participants and officials
	updaterPlayerID, err := h.tournamentRepo.GetParticipantOrOfficialByPIN(c.Context(), m.TournamentID.String(), submittedPin)
	if err != nil || updaterPlayerID == "" {
		return c.SendString("<div class='text-red-400 font-mono text-sm'>Invalid Verification PIN. Please try again.</div>")
	}

	// Update table number if provided
	tableNumberStr := c.FormValue("tableNumber")
	if tableNumberStr != "" {
		if tNum, err := strconv.Atoi(tableNumberStr); err == nil {
			// Check if another match in this tournament/event is currently in_progress on this table
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

	// Update referee if provided, otherwise default to the PIN owner
	if refereeIDStr != "" {
		refUUID, err := uuid.Parse(refereeIDStr)
		if err == nil {
			m.RefereeID = &refUUID
		}
	} else {
		refUUID, _ := uuid.Parse(updaterPlayerID)
		m.RefereeID = &refUUID
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
	updatedMatch, err := h.matchRepo.GetByID(c.Context(), m.ID)
	if err == nil && updatedMatch.Status == "finished" {
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

			h.broadcastToTournamentOrEvent(c.Context(), body.TournamentID, map[string]string{
				"event":   "referee_notification",
				"message": fmt.Sprintf("%s marked match finished%s: %s", refName, tableInfo, winStr),
			})
		}
	}

	// Broadcast real-time update to all bracket viewers for this tournament
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
		"event":        "score_updated",
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
		broadcastData["message"] = fmt.Sprintf("Match finished: %s", winStr)
	}

	h.broadcastToTournamentOrEvent(c.Context(), body.TournamentID, broadcastData)

	if c.Get("HX-Request") != "" {
		// Return a beautiful success component to replace the form out-of-band
		return c.SendString(`
		<div id="public-score-form" hx-swap-oob="outerHTML" class="text-center py-8 animate-fade-in">
			<div class="w-20 h-20 bg-green-500/10 rounded-full flex items-center justify-center mx-auto mb-6 shadow-[0_0_30px_rgba(34,197,94,0.2)]">
				<svg class="w-10 h-10 text-green-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path>
				</svg>
			</div>
			<h3 class="text-2xl font-black uppercase tracking-tight text-white mb-2">Score Confirmed!</h3>
			<p class="text-gray-400 text-sm font-mono mb-8">The match results have been successfully recorded and the bracket has been updated.</p>
			<button type="button" onclick="document.getElementById('public-score-modal').classList.add('hidden')" class="bg-white hover:bg-gray-200 text-black font-black py-4 px-10 rounded-2xl transition-all uppercase tracking-widest text-xs shadow-lg">
				Return to Bracket
			</button>
		</div>`)
	}
	return c.SendStatus(fiber.StatusOK)
}

// renderTeamMatchForm re-renders the team match score form into the modal without a page reload.
func (h *MatchHandler) renderTeamMatchForm(c *fiber.Ctx, matchID, tournamentID, stage string) error {
	return h.renderTeamMatchFormInternal(c, matchID, tournamentID, stage, "admin/partials/team-match-score-form")
}

func (h *MatchHandler) renderTeamMatchFormInternal(c *fiber.Ctx, matchID, tournamentID, stage string, templateName string) error {
	parentUUID, _ := uuid.Parse(matchID)
	tUUID, _ := uuid.Parse(tournamentID)

	t, err := h.tournamentRepo.GetByID(c.Context(), tournamentID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	parent, err := h.matchRepo.GetByID(c.Context(), parentUUID)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	}

	bestOf := 5
	if stageRule, err := bun.GetStageRule(c.Context(), h.matchRepo.DB(), tUUID, stage); err == nil {
		bestOf = stageRule.BestOf
	}

	teamFormat := t.TeamFormat
	if teamFormat == "" {
		teamFormat = "olympic"
	}

	// Find teams by looking up which team contains the parent match players
	var teamA, teamB *tournament.Team
	for _, team := range t.Teams {
		if team.ID == parent.TeamAPlayer1ID.String() {
			teamA = team
		} else {
			for _, p := range team.Players {
				if p.ID == parent.TeamAPlayer1ID.String() {
					teamA = team
					break
				}
			}
		}
		if team.ID == parent.TeamBPlayer1ID.String() {
			teamB = team
		} else {
			for _, p := range team.Players {
				if p.ID == parent.TeamBPlayer1ID.String() {
					teamB = team
					break
				}
			}
		}
	}

	var subMatches []bun.MatchModel
	_ = h.matchRepo.DB().NewSelect().Model(&subMatches).Where("team_match_id = ?", parentUUID).Order("round_number ASC").Scan(c.Context())

	// Build player name cache
	playerNames := make(map[string]string)
	var playerModels []bun.PlayerModel
	_ = h.matchRepo.DB().NewSelect().Model(&playerModels).Scan(c.Context())
	for _, pm := range playerModels {
		playerNames[pm.ID.String()] = pm.FullName()
	}

	// Derive squad selections from sub-matches
	var squadAP1, squadAP2, squadAP3 string
	var squadBP1, squadBP2, squadBP3 string
	for _, sm := range subMatches {
		if teamFormat == "olympic" {
			switch sm.RoundNumber {
			case 3:
				squadAP1 = sm.TeamAPlayer1ID.String()
				squadBP1 = sm.TeamBPlayer1ID.String()
			case 4:
				squadAP2 = sm.TeamAPlayer1ID.String()
				squadBP2 = sm.TeamBPlayer1ID.String()
			case 2:
				squadAP3 = sm.TeamAPlayer1ID.String()
				squadBP3 = sm.TeamBPlayer1ID.String()
			}
		} else {
			switch sm.RoundNumber {
			case 1:
				squadAP1 = sm.TeamAPlayer1ID.String()
				squadBP1 = sm.TeamBPlayer1ID.String()
			case 2:
				squadAP2 = sm.TeamAPlayer1ID.String()
				squadBP2 = sm.TeamBPlayer1ID.String()
			case 3:
				squadAP3 = sm.TeamAPlayer1ID.String()
				squadBP3 = sm.TeamBPlayer1ID.String()
			}
		}
	}

	// Check if squads are swapped (Team B is playing as ABC, Team A as XYZ)
	if squadAP1 != "" && squadAP1 != "00000000-0000-0000-0000-000000000000" && teamB != nil {
		isSwapped := false
		for _, p := range teamB.Players {
			if p.ID == squadAP1 {
				isSwapped = true
				break
			}
		}
		if isSwapped {
			teamA, teamB = teamB, teamA
		}
	}

	if squadAP1 == "00000000-0000-0000-0000-000000000000" && teamA != nil && len(teamA.Players) > 0 {
		squadAP1 = teamA.Players[0].ID
		if len(teamA.Players) > 1 {
			squadAP2 = teamA.Players[1].ID
		}
		if len(teamA.Players) > 2 {
			squadAP3 = teamA.Players[2].ID
		}
	}
	if squadBP1 == "00000000-0000-0000-0000-000000000000" && teamB != nil && len(teamB.Players) > 0 {
		squadBP1 = teamB.Players[0].ID
		if len(teamB.Players) > 1 {
			squadBP2 = teamB.Players[1].ID
		}
		if len(teamB.Players) > 2 {
			squadBP3 = teamB.Players[2].ID
		}
	}

	type SubMatchVM struct {
		ID             string
		MatchType      string
		RoundNumber    int
		TeamAPlayer1ID string
		TeamAPlayer2ID string
		TeamBPlayer1ID string
		TeamBPlayer2ID string
		PlayerAName    string
		PlayerBName    string
		AlignmentA     string
		AlignmentB     string
		ScoreA         int
		ScoreB         int
		Status         string
		WinnerTeam     string
	}

	var subMatchVMs []SubMatchVM
	for _, sm := range subMatches {
		teamAP2Str := ""
		teamBP2Str := ""
		if sm.TeamAPlayer2ID != nil {
			teamAP2Str = sm.TeamAPlayer2ID.String()
		}
		if sm.TeamBPlayer2ID != nil {
			teamBP2Str = sm.TeamBPlayer2ID.String()
		}

		var pAName, pBName string
		if sm.MatchType == "doubles" {
			pAName = playerNames[sm.TeamAPlayer1ID.String()]
			if teamAP2Str != "" {
				pAName += " & " + playerNames[teamAP2Str]
			}
			pBName = playerNames[sm.TeamBPlayer1ID.String()]
			if teamBP2Str != "" {
				pBName += " & " + playerNames[teamBP2Str]
			}
		} else {
			pAName = playerNames[sm.TeamAPlayer1ID.String()]
			pBName = playerNames[sm.TeamBPlayer1ID.String()]
		}

		var setModels []bun.MatchSetModel
		_ = h.matchRepo.DB().NewSelect().Model(&setModels).Where("match_id = ?", sm.ID).Scan(c.Context())

		winsA, winsB := 0, 0
		for _, set := range setModels {
			if set.ScoreA > set.ScoreB {
				winsA++
			} else if set.ScoreB > set.ScoreA {
				winsB++
			}
		}

		wt := ""
		if sm.WinnerTeam != nil {
			wt = *sm.WinnerTeam
		}

		alignA, alignB := getSubMatchAlignments(sm.RoundNumber, teamFormat)

		subMatchVMs = append(subMatchVMs, SubMatchVM{
			ID:             sm.ID.String(),
			MatchType:      sm.MatchType,
			RoundNumber:    sm.RoundNumber,
			TeamAPlayer1ID: sm.TeamAPlayer1ID.String(),
			TeamAPlayer2ID: teamAP2Str,
			TeamBPlayer1ID: sm.TeamBPlayer1ID.String(),
			TeamBPlayer2ID: teamBP2Str,
			PlayerAName:    pAName,
			PlayerBName:    pBName,
			AlignmentA:     alignA,
			AlignmentB:     alignB,
			ScoreA:         winsA,
			ScoreB:         winsB,
			Status:         sm.Status,
			WinnerTeam:     wt,
		})
	}

	var refereeIDStr string
	if parent.RefereeID != nil {
		refereeIDStr = parent.RefereeID.String()
	}

	return c.Render(templateName, fiber.Map{
		"MatchID":      matchID,
		"TournamentID": tournamentID,
		"Stage":        stage,
		"BestOf":       bestOf,
		"TeamA":        teamA,
		"TeamB":        teamB,
		"TeamFormat":   teamFormat,
		"SubMatches":   subMatchVMs,
		"SquadAP1":     squadAP1,
		"SquadAP2":     squadAP2,
		"SquadAP3":     squadAP3,
		"SquadBP1":     squadBP1,
		"SquadBP2":     squadBP2,
		"SquadBP3":     squadBP3,
		"Participants": t.Participants,
		"Pin":          parent.Pin,
		"RefereeID":    refereeIDStr,
		"TableNumber":  parent.TableNumber,
	})
}

// Start sets a match status to in_progress, persists to DB, broadcasts WS update, and renders updated row.
func (h *MatchHandler) Start(c *fiber.Ctx) error {
	matchIDStr := c.Params("id")
	if matchIDStr == "" {
		matchIDStr = c.FormValue("matchId")
	}

	var m *bun.MatchModel
	var matchID string
	var err error

	if matchIDStr == "" || matchIDStr == "nil" || matchIDStr == "null" || matchIDStr == "undefined" {
		// Try to create from tournamentId, p1Id, p2Id, stage
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

		if tID != "" && p1Id != "" && p2Id != "" {
			tUUID, _ := uuid.Parse(tID)
			p1UUID, _ := uuid.Parse(p1Id)
			p2UUID, _ := uuid.Parse(p2Id)

			// Check if already exists
			var existing bun.MatchModel
			err = h.matchRepo.DB().NewSelect().Model(&existing).
				Where("tournament_id = ?", tUUID).
				Where("team_match_id IS NULL").
				Where("((team_a_player_1_id = ? AND team_b_player_1_id = ?) OR (team_a_player_1_id = ? AND team_b_player_1_id = ?))",
					p1UUID, p2UUID, p2UUID, p1UUID).
				Scan(c.Context())
			if err == nil {
				m = &existing
				matchID = m.ID.String()
			} else {
				// Create new match
				matchType := "singles"
				if t, err := h.tournamentRepo.GetByID(c.Context(), tID); err == nil {
					switch t.Type {
					case "doubles", "mixed_doubles":
						matchType = "doubles"
					case "teams":
						matchType = "teams"
					}
				}
				created, err := h.createUC.Execute(c.Context(), tID, matchType, []string{p1Id}, []string{p2Id}, stage)
				if err == nil {
					cUUID, _ := uuid.Parse(created.ID)
					m, _ = h.matchRepo.GetByID(c.Context(), cUUID)
					matchID = created.ID
				}
			}
		}
	} else {
		matchID = matchIDStr
		mUUID, err := uuid.Parse(matchIDStr)
		if err == nil {
			m, _ = h.matchRepo.GetByID(c.Context(), mUUID)
		}
	}

	if m == nil {
		return fiber.NewError(fiber.StatusNotFound, "Match not found")
	}

	// Fetch fully loaded tournament to get table counts & division details
	t, err := h.tournamentRepo.GetByID(c.Context(), m.TournamentID.String())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Override table number if provided manually in the form
	manualTableStr := c.FormValue("tableNumber")
	if manualTableStr != "" {
		if tNum, err := strconv.Atoi(manualTableStr); err == nil {
			m.TableNumber = &tNum
		}
	}

	// Auto-assign table if not already assigned or validate occupied table
	var eventNumTables int
	if t.EventID != nil {
		eventNumTables, _ = h.tournamentRepo.GetEventNumTables(c.Context(), *t.EventID)
	}

	totalTables := 4
	if t.NumTables > 0 {
		totalTables = t.NumTables
	}
	if eventNumTables > 0 {
		totalTables = eventNumTables
	}

	// Find occupied tables across the event/tournament
	var occupiedList []int
	if t.EventID != nil {
		occupiedList, _ = h.matchRepo.GetOccupiedTablesByEvent(c.Context(), *t.EventID)
	} else {
		occupiedList, _ = h.matchRepo.GetOccupiedTablesByTournament(c.Context(), t.ID)
	}

	occupiedMap := make(map[int]bool)
	for _, num := range occupiedList {
		occupiedMap[num] = true
	}

	if m.TableNumber == nil {
		// Find all available tables within totalTables range
		var availableTables []int
		for i := 1; i <= totalTables; i++ {
			if !occupiedMap[i] {
				availableTables = append(availableTables, i)
			}
		}

		if len(availableTables) == 0 {
			// No tables available!
			c.Set("HX-Trigger", `{"show-toast": {"message": "All tables are currently occupied!", "type": "error"}}`)

			var matched *tournament.Match
			for i := range t.Matches {
				if t.Matches[i].ID == matchID {
					matched = &t.Matches[i]
					break
				}
			}
			if matched == nil {
				return fiber.NewError(fiber.StatusNotFound, "Match not found in tournament list")
			}
			return c.Render("admin/partials/match-row", matched)
		}

		// Heuristic logic:
		isFirstDivision := false
		tNameLower := strings.ToLower(t.Name)
		if strings.Contains(tNameLower, "1st division") ||
			strings.Contains(tNameLower, "division 1") ||
			strings.Contains(tNameLower, "primera division") ||
			strings.Contains(tNameLower, "primera división") ||
			strings.Contains(tNameLower, "div 1") ||
			strings.Contains(tNameLower, "division i") ||
			strings.Contains(tNameLower, "1ra division") ||
			strings.Contains(tNameLower, "1ra división") {
			isFirstDivision = true
		}

		isHighPriority := isFirstDivision || m.Stage == "semifinal" || m.Stage == "final"

		assignedTable := availableTables[0]
		if isHighPriority {
			if !occupiedMap[1] {
				assignedTable = 1
			} else if !occupiedMap[2] && totalTables >= 2 {
				assignedTable = 2
			} else {
				assignedTable = availableTables[0]
			}
		} else {
			found := false
			for _, tbl := range availableTables {
				if tbl >= 3 {
					if !found || tbl > assignedTable {
						assignedTable = tbl
						found = true
					}
				}
			}
			if !found {
				if !occupiedMap[2] && totalTables >= 2 {
					assignedTable = 2
				} else {
					assignedTable = availableTables[0]
				}
			}
		}
		m.TableNumber = &assignedTable
	} else {
		// Table was already assigned, check if it's currently occupied by another match
		isOccupiedByOther := false
		for _, occ := range occupiedList {
			if occ == *m.TableNumber {
				mUUID := m.ID
				count, err := h.matchRepo.DB().NewSelect().Model((*bun.MatchModel)(nil)).
					Where("status = 'in_progress' AND table_number = ? AND id != ?", *m.TableNumber, mUUID).
					Count(c.Context())
				if err == nil && count > 0 {
					isOccupiedByOther = true
				}
				break
			}
		}

		if isOccupiedByOther {
			c.Set("HX-Trigger", fmt.Sprintf(`{"show-toast": {"message": "Table %d is currently occupied by another match!", "type": "error"}}`, *m.TableNumber))
			var matched *tournament.Match
			for i := range t.Matches {
				if t.Matches[i].ID == matchID {
					matched = &t.Matches[i]
					break
				}
			}
			if matched == nil {
				return fiber.NewError(fiber.StatusNotFound, "Match not found in tournament list")
			}
			return c.Render("admin/partials/match-row", matched)
		}
	}

	// Update status to in_progress
	m.Status = "in_progress"
	now := time.Now()
	m.UpdatedAt = &now

	// Generate PIN if missing
	if m.Pin == "" {
		m.Pin = h.matchRepo.GenerateUniquePin(c.Context())
	}

	_, err = h.matchRepo.DB().NewUpdate().Model(m).WherePK().Column("status", "updated_at", "table_number", "pin").Exec(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Broadcast real-time update to all bracket viewers for this tournament
	h.broadcastToTournamentOrEvent(c.Context(), m.TournamentID.String(), map[string]string{
		"event":        "score_updated",
		"tournamentId": m.TournamentID.String(),
		"matchId":      m.ID.String(),
	})

	// Refresh tournament to get updated matches
	t, err = h.tournamentRepo.GetByID(c.Context(), m.TournamentID.String())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	var matched *tournament.Match
	for i := range t.Matches {
		if t.Matches[i].ID == matchID {
			matched = &t.Matches[i]
			break
		}
	}

	if matched == nil {
		return fiber.NewError(fiber.StatusNotFound, "Match not found in tournament list")
	}

	return c.Render("admin/partials/match-row", matched)
}

// Reset reverts a match back to "scheduled", clearing all sets, winner, and table assignment.
func (h *MatchHandler) Reset(c *fiber.Ctx) error {
	mUUID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid match ID")
	}

	m, err := h.matchRepo.GetByID(c.Context(), mUUID)
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
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	_, err = h.matchRepo.DB().NewUpdate().Model(m).WherePK().
		Column("status", "winner_team", "table_number", "updated_at").Exec(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	h.broadcastToTournamentOrEvent(c.Context(), m.TournamentID.String(), map[string]string{
		"event":        "score_updated",
		"tournamentId": m.TournamentID.String(),
		"matchId":      m.ID.String(),
	})

	c.Set("HX-Trigger", "reload-bracket, reload-matches")
	return c.SendStatus(fiber.StatusOK)
}

// ShowMatchScorePage renders the standalone public score page for a match.
// Accessed via /score/:matchId (shareable QR-code URL).
// Step 1: show PIN entry. Step 2 (POST): validate PIN, then show the score form inline.
func (h *MatchHandler) ShowMatchScorePage(c *fiber.Ctx) error {
	matchIDStr := c.Params("matchId")
	if matchIDStr == "" {
		matchIDStr = c.Params("id")
	}
	mUUID, err := uuid.Parse(matchIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid match ID")
	}
	m, err := h.matchRepo.GetByID(c.Context(), mUUID)
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

	return c.Render("public/match-pin-entry", fiber.Map{
		"MatchID":     matchIDStr,
		"PlayerA":     playerAName,
		"PlayerB":     playerBName,
		"TableNumber": m.TableNumber,
		"Status":      m.Status,
		"T":           tMap,
		"Lang":        lang,
	})
}

// ShowTableScorePage renders the standalone public score page for a table.
// Accessed via /score/table/:tableNumber (shareable table QR-code URL).
// If an active match (in_progress) is found on the table, it renders the PIN entry page.
// Otherwise, it renders the table-no-match page.
func (h *MatchHandler) ShowTableScorePage(c *fiber.Ctx) error {
	tournamentIDStr := c.Params("tournamentId")
	tournamentUUID, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid tournament ID")
	}

	tableNumberStr := c.Params("tableNumber")
	tableNumber, err := strconv.Atoi(tableNumberStr)
	if err != nil || tableNumber <= 0 {
		return fiber.NewError(fiber.StatusBadRequest, "Invalid table number")
	}

	// Query for an in_progress match on this table within the specific tournament
	var m bun.MatchModel
	err = h.matchRepo.DB().NewSelect().Model(&m).
		Where("tournament_id = ? AND status = 'in_progress' AND table_number = ?", tournamentUUID, tableNumber).
		Limit(1).
		Scan(c.Context())

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

	return c.Render("public/match-pin-entry", fiber.Map{
		"MatchID":     m.ID.String(),
		"PlayerA":     playerAName,
		"PlayerB":     playerBName,
		"TableNumber": m.TableNumber,
		"Status":      m.Status,
		"T":           tMap,
		"Lang":        lang,
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
	m, err := h.matchRepo.GetByID(c.Context(), mUUID)
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
		for _, sr := range t.StageRules {
			if sr.Stage == m.Stage {
				bestOf = sr.BestOf
				break
			}
		}
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

func getSubMatchAlignments(roundNumber int, teamFormat string) (string, string) {
	if teamFormat == "" {
		teamFormat = "olympic"
	}
	if teamFormat == "olympic" {
		switch roundNumber {
		case 1:
			return "A & B", "X & Y"
		case 2:
			return "C", "Z"
		case 3:
			return "A", "X"
		case 4:
			return "B", "Y"
		case 5:
			return "C", "X"
		}
	} else {
		// Corbillon or other format
		switch roundNumber {
		case 1:
			return "A", "X"
		case 2:
			return "B", "Y"
		case 3:
			return "C", "Z"
		case 4:
			return "A", "Y"
		case 5:
			return "B", "X"
		}
	}
	return "", ""
}
