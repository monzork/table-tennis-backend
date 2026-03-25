package handler

import (
	"table-tennis-backend/internal/application/tournament"

	"github.com/gofiber/fiber/v2"
)

type TournamentHandler struct {
	createUC *tournament.CreateTournamentUseCase
}

func NewTournamentHandler(createUC *tournament.CreateTournamentUseCase) *TournamentHandler {
	return &TournamentHandler{createUC: createUC}
}

func (h *TournamentHandler) Create(c *fiber.Ctx) error {
	var body struct {
		Name           string   `json:"name" form:"name"`
		Type           string   `json:"type" form:"type"`
		Format         string   `json:"format" form:"format"`
		StartDate      string   `json:"startDate" form:"startDate"`
		EndDate        string   `json:"endDate" form:"endDate"`
		ParticipantIDs []string `json:"participant_ids" form:"participant_ids[]"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	// Extract new players from form manually since BodyParser doesn't handle multiple slices as structs well
	newPlayerFirstNames := c.FormValue("new_player_first_name")
	var newPlayers []tournament.NewPlayerData
	if newPlayerFirstNames != "" {
		// If there's at least one, we might have multiple. 
		// Fiber's FormValue only returns the first one. 
		// We should use context's internal form values for slices.
		form, err := c.MultipartForm()
		if err == nil {
			names := form.Value["new_player_first_name[]"]
			lasts := form.Value["new_player_last_name[]"]
			genders := form.Value["new_player_gender[]"]
			for i := 0; i < len(names); i++ {
				np := tournament.NewPlayerData{
					FirstName: names[i],
				}
				if i < len(lasts) {
					np.LastName = lasts[i]
				}
				if i < len(genders) {
					np.Gender = genders[i]
				}
				if np.FirstName != "" && np.LastName != "" {
					newPlayers = append(newPlayers, np)
				}
			}
		}
	}

	t, err := h.createUC.Execute(c.Context(), body.Name, body.Type, body.Format, body.StartDate, body.EndDate, body.ParticipantIDs, newPlayers)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	// Updated path for the partial
	return c.Render("admin/partials/tournament-row", t)
}
