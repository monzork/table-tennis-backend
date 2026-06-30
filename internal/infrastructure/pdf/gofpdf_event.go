package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"table-tennis-backend/internal/domain/event"

	"github.com/jung-kurt/gofpdf"
)

func (g *GoFpdfGenerator) GenerateEventReport(e *event.Event) ([]byte, error) {
	tournamentsList := e.Tournaments

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 52, 15)
	pdf.SetAutoPageBreak(true, 15)

	tr := pdf.UnicodeTranslatorFromDescriptor("")
	imagePath := findHeaderImage()

	pdf.SetHeaderFunc(func() {
		pdf.Image(imagePath, 15, 10, 25, 0, false, "", 0, "")
		pdf.SetY(17)
		pdf.SetX(48)
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(0, 10, tr("EVENTO TENIS DE MESA - "+strings.ToUpper(e.Name)), "", 1, "L", false, 0, "")
		pdf.SetDrawColor(200, 200, 200)
		w, _ := pdf.GetPageSize()
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

	pdf.CellFormat(60, 8, "Total Sub-Tournaments:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%d Tournaments", len(tournamentsList)), "", 1, "L", false, 0, "")

	for _, t := range tournamentsList {
		BuildTournamentPdfContent(pdf, t, tr)
	}

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
