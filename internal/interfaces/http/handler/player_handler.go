package handler

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"table-tennis-backend/internal/application/player"

	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

type PlayerHandler struct {
	registerPlayerUC *player.RegisterPlayerUseCase
	updatePlayerUC   *player.UpdatePlayerUseCase
	deletePlayerUC   *player.DeletePlayerUseCase
	importPlayersUC  *player.ImportPlayersUseCase
}

func NewPlayerHandler(
	uc *player.RegisterPlayerUseCase,
	uuc *player.UpdatePlayerUseCase,
	duc *player.DeletePlayerUseCase,
	iuc *player.ImportPlayersUseCase,
) *PlayerHandler {
	return &PlayerHandler{
		registerPlayerUC: uc,
		updatePlayerUC:   uuc,
		deletePlayerUC:   duc,
		importPlayersUC:  iuc,
	}
}

func (h *PlayerHandler) Register(c *fiber.Ctx) error {
	var body struct {
		FirstName      string `json:"firstName" form:"firstName"`
		LastName       string `json:"lastName" form:"lastName"`
		Birthdate      string `json:"birthdate" form:"birthdate"`
		Country        string `json:"country" form:"country"`
		Gender         string `json:"gender" form:"gender"`
		WhatsAppNumber string `json:"whatsAppNumber" form:"whatsAppNumber"`
		SinglesElo     int16  `json:"singlesElo" form:"singlesElo"`
		DoublesElo     int16  `json:"doublesElo" form:"doublesElo"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.registerPlayerUC.Execute(context.Background(), body.FirstName, body.LastName, body.Birthdate, body.Gender, body.Country, body.WhatsAppNumber, body.SinglesElo, body.DoublesElo)

	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("admin/partials/player-row", player)
}

func (h *PlayerHandler) Update(c *fiber.Ctx) error {
	id := c.Params("id")
	var body struct {
		FirstName      string `json:"firstName" form:"firstName"`
		LastName       string `json:"lastName" form:"lastName"`
		Birthdate      string `json:"birthdate" form:"birthdate"`
		Country        string `json:"country" form:"country"`
		Gender         string `json:"gender" form:"gender"`
		WhatsAppNumber string `json:"whatsAppNumber" form:"whatsAppNumber"`
		SinglesElo     int16  `json:"singlesElo" form:"singlesElo"`
		DoublesElo     int16  `json:"doublesElo" form:"doublesElo"`
	}

	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	player, err := h.updatePlayerUC.Execute(c.Context(), id, body.FirstName, body.LastName, body.Birthdate, body.Gender, body.Country, body.WhatsAppNumber, body.SinglesElo, body.DoublesElo)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.Render("admin/partials/player-row", player)
}

func (h *PlayerHandler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.deletePlayerUC.Execute(c.Context(), id); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusOK)
}

// ImportTemplate returns a downloadable player template.
// ?format=csv  → plain CSV
// (default)    → styled .xlsx with example data, column notes and Elo fields
func (h *PlayerHandler) ImportTemplate(c *fiber.Ctx) error {
	if c.Query("format") == "csv" {
		c.Set("Content-Type", "text/csv")
		c.Set("Content-Disposition", `attachment; filename="players_template.csv"`)
		return c.SendString(
			"first_name,last_name,birthdate,gender,country,singles_elo,doubles_elo\n" +
				"John,Doe,1995-06-15,M,MEX,1200,1150\n" +
				"Jane,Smith,1998-03-22,F,USA,1350,1300\n",
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
		Header  string
		Width   float64
		Optional bool
		Note    string
	}
	cols := []col{
		{"first_name", 16, false, ""},
		{"last_name", 16, false, ""},
		{"birthdate", 14, false, "YYYY-MM-DD"},
		{"gender", 10, false, "M or F"},
		{"country", 10, false, "3-letter code"},
		{"singles_elo", 14, true, "Default: 500"},
		{"doubles_elo", 14, true, "Default: 500"},
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
		{"John", "Doe", "1995-06-15", "M", "MEX", 1200, 1150},
		{"Jane", "Smith", "1998-03-22", "F", "USA", 1350, 1300},
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
	f.SetCellValue(legendSheet, "A8", "birthdate: YYYY-MM-DD or DD/MM/YYYY")
	f.SetCellValue(legendSheet, "A9", "singles_elo / doubles_elo: FFTT starting points (default 500)")
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
		return fiber.NewError(fiber.StatusBadRequest, "no file uploaded")
	}

	f, err := file.Open()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to open file")
	}
	defer f.Close()

	result, err := h.importPlayersUC.Execute(c.Context(), file.Filename, f)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	msg := fmt.Sprintf("Imported %d players", result.Imported)
	if result.Skipped > 0 {
		msg += fmt.Sprintf(", %d skipped", result.Skipped)
	}

	// Return an HTMX-friendly JSON summary; the UI will handle the toast + reload
	return c.JSON(fiber.Map{
		"imported": result.Imported,
		"skipped":  result.Skipped,
		"errors":   result.Errors,
		"message":  msg,
	})
}
