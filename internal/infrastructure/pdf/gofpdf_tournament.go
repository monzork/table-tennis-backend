package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/tournament"

	"github.com/jung-kurt/gofpdf"
)

func (g *GoFpdfGenerator) GenerateEventReport(e *tournament.Tournament, divs []*division.Division) ([]byte, error) {
	tournamentsList := e.Events

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 52, 15)
	pdf.SetAutoPageBreak(true, 15)

	tr := pdf.UnicodeTranslatorFromDescriptor("")
	imagePath := findHeaderImage()

	pdf.SetHeaderFunc(func() {
		pdf.Image(imagePath, 15, 10, 25, 0, false, "", 0, "")
		pdf.SetY(17)
		pdf.SetX(48)

		text := tr("EVENTO TENIS DE MESA - " + strings.ToUpper(e.Name))
		w, _ := pdf.GetPageSize()
		maxWidth := w - 48 - 15

		fontSize := 14.0
		pdf.SetFont("Arial", "B", fontSize)
		for pdf.GetStringWidth(text) > maxWidth && fontSize > 8.0 {
			fontSize -= 0.5
			pdf.SetFont("Arial", "B", fontSize)
		}

		pdf.CellFormat(0, 10, text, "", 1, "L", false, 0, "")
		pdf.SetDrawColor(200, 200, 200)
		w, _ = pdf.GetPageSize()
		pdf.Line(15, 45, w-15, 45)
		pdf.SetY(52)
	})

	// --- 1. COVER PAGE ---
	pdf.AddPage()

	// Date and Summary Info
	pdf.SetTextColor(50, 50, 50)
	pdf.Ln(5)

	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "REPORT SUMMARY", "B", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(60, 8, "Date Generated:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, time.Now().Format("January 02, 2006 at 15:04 PM"), "", 1, "L", false, 0, "")

	pdf.CellFormat(60, 8, "Total Sub-Events:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%d Events", len(tournamentsList)), "", 1, "L", false, 0, "")

	for _, t := range tournamentsList {
		BuildTournamentPdfContent(pdf, t, divs, tr)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
