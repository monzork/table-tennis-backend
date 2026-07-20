package handler

import (
	"fmt"
	"table-tennis-backend/internal/application/event"
	tournamentDomain "table-tennis-backend/internal/domain/event"

	"github.com/gofiber/fiber/v2"
)

func parseCreateEventCommand(c *fiber.Ctx) (event.CreateEventCommand, error) {
	var body struct {
		Name           string `json:"name" form:"name"`
		Type           string `json:"type" form:"type"`
		Format         string `form:"format"`
		EventCategory  string `form:"eventCategory"`
		StartDate      string `form:"startDate"`
		EndDate        string `form:"endDate"`
		GroupPassCount int    `form:"groupPassCount"`
		TeamFormat            string `form:"teamFormat"`
		NumTables             int    `form:"numTables" json:"numTables"`
		KnockoutBracketsCount int    `form:"knockoutBracketsCount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return event.CreateEventCommand{}, err
	}

	var participantIDs []string
	for _, id := range c.Request().PostArgs().PeekMulti("participant_ids[]") {
		participantIDs = append(participantIDs, string(id))
	}

	var newPlayers []event.NewPlayerData
	firstNames := c.Request().PostArgs().PeekMulti("new_player_first_name[]")
	lastNames := c.Request().PostArgs().PeekMulti("new_player_last_name[]")
	genders := c.Request().PostArgs().PeekMulti("new_player_gender[]")

	for i := 0; i < len(firstNames); i++ {
		np := event.NewPlayerData{FirstName: string(firstNames[i])}
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

	createStages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	var stageRules []event.StageRuleOverride
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
			stageRules = append(stageRules, event.StageRuleOverride{
				Stage:        stage,
				BestOf:       bo,
				PointsToWin:  pt,
				PointsMargin: pm,
			})
		}
	}

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
	divisionGroupPassCounts := make(map[string]int)
	divisionGroupCounts := make(map[string]int)
	for _, divIDBytes := range divisionIDs {
		divID := string(divIDBytes)
		if divID == "" {
			continue
		}
		dfStr := string(c.Request().PostArgs().Peek("division_formats[" + divID + "]"))
		if dfStr != "" {
			divisionFormats[divID] = dfStr
		}
		dgpcStr := string(c.Request().PostArgs().Peek("division_group_pass_counts[" + divID + "]"))
		if dgpcStr != "" {
			dgpc := 0
			fmt.Sscanf(dgpcStr, "%d", &dgpc)
			if dgpc > 0 {
				divisionGroupPassCounts[divID] = dgpc
			}
		}
		dgcStr := string(c.Request().PostArgs().Peek("division_group_counts[" + divID + "]"))
		if dgcStr != "" {
			dgc := 0
			fmt.Sscanf(dgcStr, "%d", &dgc)
			if dgc > 0 {
				divisionGroupCounts[divID] = dgc
			}
		}
	}

	return event.CreateEventCommand{
		Name:                    body.Name,
		Type:                    body.Type,
		Format:                  body.Format,
		Category:                body.EventCategory,
		StartDate:               body.StartDate,
		EndDate:                 body.EndDate,
		ParticipantIDs:          participantIDs,
		NewPlayers:              newPlayers,
		GroupPassCount:          body.GroupPassCount,
		StageRuleOverrides:      stageRules,
		DivisionRules:           divisionRules,
		SkipElo:                 skipElo,
		EventID:                 eventID,
		TeamFormat:              body.TeamFormat,
		NumTables:               body.NumTables,
		HasThirdPlaceMatch:      hasThirdPlaceMatch,
		DivisionFormats:         divisionFormats,
		DivisionGroupPassCounts: divisionGroupPassCounts,
		DivisionGroupCounts:     divisionGroupCounts,
		KnockoutBracketsCount:   body.KnockoutBracketsCount,
	}, nil
}

func parseUpdateEventCommand(c *fiber.Ctx) (event.UpdateEventCommand, error) {
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
		TeamFormat            string `form:"teamFormat"`
		NumTables             int    `form:"numTables" json:"numTables"`
		KnockoutBracketsCount int    `form:"knockoutBracketsCount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return event.UpdateEventCommand{}, err
	}

	var participantIDs []string
	for _, pid := range c.Request().PostArgs().PeekMulti("participant_ids[]") {
		participantIDs = append(participantIDs, string(pid))
	}

	var newPlayers []event.NewPlayerData
	firstNames := c.Request().PostArgs().PeekMulti("new_player_first_name[]")
	lastNames := c.Request().PostArgs().PeekMulti("new_player_last_name[]")
	genders := c.Request().PostArgs().PeekMulti("new_player_gender[]")

	for i := 0; i < len(firstNames); i++ {
		np := event.NewPlayerData{FirstName: string(firstNames[i])}
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

	stages := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
	var stageRules []event.StageRuleOverride
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
			stageRules = append(stageRules, event.StageRuleOverride{
				Stage:        stage,
				BestOf:       bo,
				PointsToWin:  pt,
				PointsMargin: pm,
			})
		}
	}

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
	divisionGroupPassCounts := make(map[string]int)
	divisionGroupCounts := make(map[string]int)
	for _, divIDBytes := range divisionIDs {
		divID := string(divIDBytes)
		if divID == "" {
			continue
		}
		dfStr := string(c.Request().PostArgs().Peek("division_formats[" + divID + "]"))
		if dfStr != "" {
			divisionFormats[divID] = dfStr
		}
		dgpcStr := string(c.Request().PostArgs().Peek("division_group_pass_counts[" + divID + "]"))
		if dgpcStr != "" {
			dgpc := 0
			fmt.Sscanf(dgpcStr, "%d", &dgpc)
			if dgpc > 0 {
				divisionGroupPassCounts[divID] = dgpc
			}
		}
		dgcStr := string(c.Request().PostArgs().Peek("division_group_counts[" + divID + "]"))
		if dgcStr != "" {
			dgc := 0
			fmt.Sscanf(dgcStr, "%d", &dgc)
			if dgc > 0 {
				divisionGroupCounts[divID] = dgc
			}
		}
	}

	return event.UpdateEventCommand{
		ID:                      id,
		Name:                    body.Name,
		Type:                    body.Type,
		Format:                  body.Format,
		Category:                body.EventCategory,
		StartDate:               body.StartDate,
		EndDate:                 body.EndDate,
		RegistrationOpen:        body.RegistrationOpen,
		ParticipantIDs:          participantIDs,
		NewPlayers:              newPlayers,
		GroupPassCount:          body.GroupPassCount,
		StageRuleOverrides:      stageRules,
		DivisionRules:           divisionRules,
		SkipElo:                 skipElo,
		EventID:                 eventID,
		TeamFormat:              body.TeamFormat,
		NumTables:               body.NumTables,
		HasThirdPlaceMatch:      hasThirdPlaceMatch,
		DivisionFormats:         divisionFormats,
		DivisionGroupPassCounts: divisionGroupPassCounts,
		DivisionGroupCounts:     divisionGroupCounts,
		KnockoutBracketsCount:   body.KnockoutBracketsCount,
	}, nil
}
