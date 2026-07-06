package handler

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/application/tournament"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

type PlayerHandler struct {
	registerPlayerUC        *player.RegisterPlayerUseCase
	updatePlayerUC          *player.UpdatePlayerUseCase
	deletePlayerUC          *player.DeletePlayerUseCase
	getPlayerByIDUC         *player.GetPlayerByIDUseCase
	searchPlayerUC          *player.SearchPlayersUseCase
	searchPlayerSelectionUC *player.SearchPlayersForSelectionUseCase
	importPlayersUC         *player.ImportPlayersUseCase
	enrollPlayerUC          *tournament.EnrollPlayerUseCase
	getTournamentsUC        *tournament.GetTournamentsUseCase
}

func NewPlayerHandler(
	uc *player.RegisterPlayerUseCase,
	uuc *player.UpdatePlayerUseCase,
	duc *player.DeletePlayerUseCase,
	giuc *player.GetPlayerByIDUseCase,
	siuc *player.SearchPlayersUseCase,
	ssuc *player.SearchPlayersForSelectionUseCase,
	iuc *player.ImportPlayersUseCase,
	enrollUC *tournament.EnrollPlayerUseCase,
	gtuc *tournament.GetTournamentsUseCase,
) *PlayerHandler {
	return &PlayerHandler{
		registerPlayerUC:        uc,
		updatePlayerUC:          uuc,
		deletePlayerUC:          duc,
		getPlayerByIDUC:         giuc,
		searchPlayerUC:          siuc,
		searchPlayerSelectionUC: ssuc,
		importPlayersUC:         iuc,
		enrollPlayerUC:          enrollUC,
		getTournamentsUC:        gtuc,
	}
}

func (h *PlayerHandler) Register(c *fiber.Ctx) error {
	var body struct {
		FirstName      string `json:"firstName" form:"firstName"`
		SecondName     string `json:"secondName" form:"secondName"`
		LastName       string `json:"lastName" form:"lastName"`
		SecondLastName string `json:"secondLastName" form:"secondLastName"`
		Birthdate      string `json:"birthdate" form:"birthdate"`
		Country        string `json:"country" form:"country"`
		Department     string `json:"department" form:"department"`
		Gender         string `json:"gender" form:"gender"`
		WhatsAppNumber string `json:"whatsAppNumber" form:"whatsAppNumber"`
		NationalID     string `json:"nationalID" form:"nationalID"`
		SinglesElo     int16  `json:"singlesElo" form:"singlesElo"`
		DoublesElo     int16  `json:"doublesElo" form:"doublesElo"`
		TournamentID   string `json:"tournamentId" form:"tournamentId"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("admin/partials/error-alert", "Invalid request body")
	}

	player, err := h.registerPlayerUC.Execute(c.Context(), body.FirstName, body.SecondName, body.LastName, body.SecondLastName, body.Birthdate, body.Gender, body.Country, body.Department, body.WhatsAppNumber, body.NationalID, body.SinglesElo, body.DoublesElo)

	if err != nil {
		return c.Status(fiber.StatusBadRequest).Render("admin/partials/error-alert", err.Error())
	}

	if body.TournamentID != "" {
		if err := h.enrollPlayerUC.Execute(c.Context(), body.TournamentID, player.ID, player.SinglesElo, player.DoublesElo); err != nil {
			slog.Warn("failed to enroll newly created player into tournament", "playerID", player.ID, "tournamentID", body.TournamentID, "err", err)
		}
	}

	return c.Render("admin/partials/player-row", player)
}

func (h *PlayerHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		FirstName      string `json:"firstName" form:"firstName"`
		SecondName     string `json:"secondName" form:"secondName"`
		LastName       string `json:"lastName" form:"lastName"`
		SecondLastName string `json:"secondLastName" form:"secondLastName"`
		Birthdate      string `json:"birthdate" form:"birthdate"`
		Country        string `json:"country" form:"country"`
		Department     string `json:"department" form:"department"`
		Gender         string `json:"gender" form:"gender"`
		WhatsAppNumber string `json:"whatsAppNumber" form:"whatsAppNumber"`
		NationalID     string `json:"nationalID" form:"nationalID"`
		SinglesElo     int16  `json:"singlesElo" form:"singlesElo"`
		DoublesElo     int16  `json:"doublesElo" form:"doublesElo"`
		TournamentID   string `json:"tournamentId" form:"tournamentId"`
	}

	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("admin/partials/error-alert", "Invalid request body")
	}

	player, err := h.updatePlayerUC.Execute(c.Context(), id, body.FirstName, body.SecondName, body.LastName, body.SecondLastName, body.Birthdate, body.Gender, body.Country, body.Department, body.WhatsAppNumber, body.NationalID, body.SinglesElo, body.DoublesElo)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).Render("admin/partials/error-alert", err.Error())
	}

	if body.TournamentID != "" {
		if err := h.enrollPlayerUC.Execute(c.Context(), body.TournamentID, player.ID, player.SinglesElo, player.DoublesElo); err != nil {
			slog.Warn("failed to enroll updated player into tournament", "playerID", player.ID, "tournamentID", body.TournamentID, "err", err)
		}
	}

	return c.Render("admin/partials/player-row", player)
}

func (h *PlayerHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deletePlayerUC.Execute(c.Context(), id); err != nil {
		return c.Status(fiber.StatusBadRequest).Render("admin/partials/error-alert", err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}

func (h *PlayerHandler) ShowEditForm(c *fiber.Ctx) error {
	id := c.Params("id")
	p, err := h.getPlayerByIDUC.Execute(c.Context(), id)
	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "Player not found")
	}
	
	var activeTournaments []any
	tournaments, _ := h.getTournamentsUC.Execute(c.Context())
	if tournaments != nil {
		for _, t := range tournaments {
			if t.Status != "finished" {
				activeTournaments = append(activeTournaments, t)
			}
		}
	}

	return c.Render("admin/partials/player-edit-form", fiber.Map{
		"Player":      p,
		"Tournaments": activeTournaments,
	})
}

func (h *PlayerHandler) Search(c *fiber.Ctx) error {
	query := c.Query("q")
	players, err := h.searchPlayerUC.Execute(c.Context(), query)
	if err != nil {
		return ErrorHandler(err)
	}
	return c.Render("admin/partials/player-list-rows", fiber.Map{
		"Players": players,
	})
}

func (h *PlayerHandler) SearchSelectionCards(c *fiber.Ctx) error {
	query := c.Query("q")
	gender := c.Query("gender") // optional: "M" or "F"
	category := c.Query("eventCategory")
	if category == "men" {
		gender = "M"
	} else if category == "women" {
		gender = "F"
	}
	selectAll := c.Query("selectAll") == "true" // if true, mark all returned players as checked

	players, err := h.searchPlayerSelectionUC.Execute(c.Context(), query, gender)
	if err != nil {
		return ErrorHandler(err)
	}

	// Build selected map: preserve existing selections OR select all returned
	selectedMap := make(map[string]bool)
	if selectAll {
		for _, p := range players {
			selectedMap[p.ID] = true
		}
	} else {
		for _, id := range c.Request().URI().QueryArgs().PeekMulti("participant_ids[]") {
			selectedMap[string(id)] = true
		}
	}

	return c.Render("admin/partials/player-selection-cards", fiber.Map{
		"Players":     players,
		"SelectedIDs": selectedMap,
	})
}

// ImportTemplate returns a downloadable player template.
// ?format=csv  → plain CSV
// (default)    → styled .xlsx with example data, column notes and Elo fields
func (h *PlayerHandler) ImportTemplate(c *fiber.Ctx) error {
	if c.Query("format") == "csv" {
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", `attachment; filename="players_template.csv"`)
		return c.SendString(
			"first_name,second_name,last_name,second_last_name,national_id,birthdate,gender,country,department,singles_elo,doubles_elo,whatsapp_number,pin\n" +
				"John,Carlos,Doe,Gomez,001-150695-0000A,1995-06-15,M,MEX,IT,1200,1150,+5212345678,1234\n" +
				"Jane,,Smith,,002-220398-0000B,1998-03-22,F,USA,HR,1350,1300,+11234567890,4321\n",
		)
	}

	// ── Generate .xlsx inline ────────────────────────────────────────────────
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Players"
	f.SetSheetName("Sheet1", sheet)

	// Header style: dark background, bold white text, centred
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"1a1a2e"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center", WrapText: true},
		Border: []excelize.Border{
			{Type: "bottom", Color: "4a4a8a", Style: 2},
		},
	})

	// Required field style: light yellow background
	requiredStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1a1a2e", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"fff3cd"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Optional field style: light blue background
	optionalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "1a1a2e", Size: 11},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"cfe2ff"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Example row style
	exampleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Color: "555555"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"f8f9fa"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})

	type col struct {
		Header   string
		Width    float64
		Optional bool
		Note     string
	}
	cols := []col{
		{"first_name", 16, false, ""},
		{"second_name", 16, true, "Second Name"},
		{"last_name", 16, false, ""},
		{"second_last_name", 16, true, "Second Last Name"},
		{"national_id", 20, true, "National ID / Cédula"},
		{"birthdate", 14, false, "YYYY-MM-DD"},
		{"gender", 10, false, "M or F"},
		{"country", 10, false, "3-letter code"},
		{"department", 16, true, "e.g. IT, HR, Sales"},
		{"singles_elo", 14, true, "Default: 500"},
		{"doubles_elo", 14, true, "Default: 500"},
		{"whatsapp_number", 18, true, "e.g. +50588888888"},
		{"pin", 10, true, "e.g. 1234"},
	}

	// Write header row (row 1)
	for i, c := range cols {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, c.Header)
		if c.Optional {
			f.SetCellStyle(sheet, cell, cell, optionalStyle)
		} else {
			f.SetCellStyle(sheet, cell, cell, requiredStyle)
		}
	}

	// Notes row (row 2)
	noteStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Italic: true, Color: "888888", Size: 9},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"f1f1f1"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	for i, c := range cols {
		if c.Note != "" {
			cell, _ := excelize.CoordinatesToCellName(i+1, 2)
			f.SetCellValue(sheet, cell, c.Note)
			f.SetCellStyle(sheet, cell, cell, noteStyle)
		}
	}

	// Header row (row 3) title
	headerRowCell, _ := excelize.CoordinatesToCellName(1, 3)
	headerRowEnd, _ := excelize.CoordinatesToCellName(len(cols), 3)
	f.SetCellValue(sheet, headerRowCell, "↓ Your data starts here")
	f.SetCellStyle(sheet, headerRowCell, headerRowEnd, headerStyle)

	// Example data rows (4 and 5)
	examples := [][]interface{}{
		{"John", "Carlos", "Doe", "Gomez", "001-150695-0000A", "1995-06-15", "M", "MEX", "IT", 1200, 1150, "+5212345678", "1234"},
		{"Jane", "", "Smith", "", "002-220398-0000B", "1998-03-22", "F", "USA", "HR", 1350, 1300, "+11234567890", "4321"},
	}
	for r, row := range examples {
		for c, val := range row {
			cell, _ := excelize.CoordinatesToCellName(c+1, r+4)
			f.SetCellValue(sheet, cell, val)
			f.SetCellStyle(sheet, cell, cell, exampleStyle)
		}
	}

	// Column widths
	for i, c := range cols {
		colLetter, _ := excelize.ColumnNumberToName(i + 1)
		f.SetColWidth(sheet, colLetter, colLetter, c.Width)
	}

	// Row heights
	f.SetRowHeight(sheet, 1, 22)
	f.SetRowHeight(sheet, 2, 16)
	f.SetRowHeight(sheet, 3, 18)

	// Freeze the header + note rows
	f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      3,
		TopLeftCell: "A4",
		ActivePane:  "bottomLeft",
	})

	// Legend: yellow = required, blue = optional
	legendSheet := "Legend"
	f.NewSheet(legendSheet)
	f.SetCellValue(legendSheet, "A1", "Column Guide")
	f.SetCellValue(legendSheet, "A3", "🟡 Yellow = Required")
	f.SetCellValue(legendSheet, "A4", "🔵 Blue = Optional (defaults to 500 if blank)")
	f.SetCellValue(legendSheet, "A6", "gender: M = Male, F = Female")
	f.SetCellValue(legendSheet, "A7", "country: use 3-letter ISO code (MEX, USA, CHN...)")
	f.SetCellValue(legendSheet, "A8", "department: player department (e.g. IT, HR, Sales, etc.)")
	f.SetCellValue(legendSheet, "A9", "birthdate: YYYY-MM-DD or DD/MM/YYYY")
	f.SetCellValue(legendSheet, "A10", "singles_elo / doubles_elo: FFTT starting points (default 500)")
	f.SetCellValue(legendSheet, "A11", "whatsapp_number: WhatsApp phone number (optional)")
	f.SetCellValue(legendSheet, "A12", "pin: 4-digit verification PIN (optional, default 1234)")
	f.SetColWidth(legendSheet, "A", "A", 55)

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate template")
	}

	c.Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Set("Content-Disposition", `attachment; filename="players_template.xlsx"`)
	c.Set("Content-Length", strconv.Itoa(buf.Len()))
	return c.Send(buf.Bytes())
}

// Import handles multipart file upload of CSV/XLSX and bulk-inserts players.
func (h *PlayerHandler) Import(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(
			`<p class="text-red-400 font-bold">✗ No file uploaded</p>`,
		)
	}

	f, err := file.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString(
			`<p class="text-red-400 font-bold">✗ Failed to open file</p>`,
		)
	}
	defer f.Close()

	result, err := h.importPlayersUC.Execute(c.Context(), file.Filename, f)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).SendString(
			fmt.Sprintf(`<p class="text-red-400 font-bold">✗ %s</p>`, err.Error()),
		)
	}

	msg := fmt.Sprintf("Imported %d players", result.Imported)
	if result.Skipped > 0 {
		msg += fmt.Sprintf(", %d skipped", result.Skipped)
	}

	// Build error list HTML
	errHTML := ""
	if len(result.Errors) > 0 {
		errHTML = `<ul class="text-yellow-400 mt-2 space-y-1 text-xs">`
		for _, e := range result.Errors {
			errHTML += fmt.Sprintf(`<li>⚠ %s</li>`, e)
		}
		errHTML += `</ul>`
	}

	// Signal HTMX to refresh the player list after a short delay
	c.Set("HX-Trigger-After-Settle", `{"refreshPlayerList": true}`)

	return c.SendString(fmt.Sprintf(
		`<p class="text-green-400 font-bold mb-1">✓ %s</p>%s`,
		msg, errHTML,
	))
}
