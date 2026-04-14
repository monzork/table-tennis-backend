package tournament

import (
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

type ExportTournamentPdfUseCase struct {
	tournamentRepo *bun.TournamentRepository
}

func NewExportTournamentPdfUseCase(tournamentRepo *bun.TournamentRepository) *ExportTournamentPdfUseCase {
	return &ExportTournamentPdfUseCase{tournamentRepo: tournamentRepo}
}

func (uc *ExportTournamentPdfUseCase) Execute(ctx context.Context, tournamentIDStr string) ([]byte, error) {
	id, err := uuid.Parse(tournamentIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid tournament id")
	}

	t, err := uc.tournamentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)

	// Helpers
	writeHeader := func(text string) {
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(0, 10, text, "", 1, "C", false, 0, "")
		pdf.Ln(5)
	}

	// 1. PARTICIPANT ASSOCIATIONS
	pdf.AddPage()
	writeHeader("PARTICIPANT ASSOCIATIONS")

	pdf.SetFont("Arial", "B", 12)
	pdf.CellFormat(0, 10, t.Name, "", 1, "C", false, 0, "")
	pdf.Ln(5)

	assocMap := make(map[string]bool)
	var assocs []string
	for _, p := range t.Participants {
		country := strings.TrimSpace(p.Country)
		if country == "" {
			country = "UNKNOWN"
		}
		if !assocMap[country] {
			assocMap[country] = true
			assocs = append(assocs, country)
		}
	}
	sort.Strings(assocs)

	pdf.SetFont("Arial", "B", 10)
	pdf.CellFormat(20, 8, "#", "1", 0, "C", false, 0, "")
	pdf.CellFormat(100, 8, "Association Name", "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 8, "ITTF Code", "1", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	codeMap := make(map[string]string)
	for i, a := range assocs {
		code := a
		if len(code) > 3 {
			code = strings.ToUpper(code[:3])
		}
		codeMap[a] = code
		pdf.CellFormat(20, 8, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(100, 8, a, "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 8, code, "1", 1, "C", false, 0, "")
	}

	// 2. ENTRIES LIST
	if len(t.Participants) > 0 {
		pdf.AddPage()
		writeHeader(fmt.Sprintf("ENTRIES LIST - %d PLAYERS", len(t.Participants)))

		var mens, womens []*player.Player
		for _, p := range t.Participants {
			if p.Gender == "M" {
				mens = append(mens, p)
			} else {
				womens = append(womens, p)
			}
		}

		printEntries := func(title string, list []*player.Player, startNr int) {
			if len(list) == 0 {
				return
			}
			pdf.Ln(5)
			pdf.SetFont("Arial", "B", 12)
			pdf.CellFormat(0, 8, fmt.Sprintf("%s (%d PLAYERS)", title, len(list)), "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(20, 8, "Nr", "1", 0, "C", false, 0, "")
			pdf.CellFormat(100, 8, "NAME", "1", 0, "C", false, 0, "")
			pdf.CellFormat(40, 8, "ASSOC.", "1", 1, "C", false, 0, "")

			pdf.SetFont("Arial", "", 10)
			for i, p := range list {
				country := strings.TrimSpace(p.Country)
				if country == "" {
					country = "UNKNOWN"
				}
				pdf.CellFormat(20, 8, fmt.Sprintf("%d", startNr+i), "1", 0, "C", false, 0, "")
				pdf.CellFormat(100, 8, strings.ToUpper(p.LastName)+" "+p.FirstName, "1", 0, "L", false, 0, "")
				pdf.CellFormat(40, 8, codeMap[country], "1", 1, "C", false, 0, "")
			}
		}

		printEntries("MENS ENTRIES", mens, 101)
		pdf.Ln(10)
		printEntries("WOMENS ENTRIES", womens, 301)
	}

	// 3. MATCH RESULTS
	if len(t.Matches) > 0 {
		pdf.AddPage()
		writeHeader("MATCH RESULTS")

		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(30, 8, "Stage", "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 8, "Team A", "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 8, "Team B", "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 8, "Score", "1", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 9)
		for _, m := range t.Matches {
			if m.Status != "finished" {
				continue
			}

			nameA := "Team A"
			if len(m.TeamA) > 0 {
				nameA = m.TeamA[0].LastName
				if len(m.TeamA) > 1 {
					nameA += "/" + m.TeamA[1].LastName
				}
			}
			nameB := "Team B"
			if len(m.TeamB) > 0 {
				nameB = m.TeamB[0].LastName
				if len(m.TeamB) > 1 {
					nameB += "/" + m.TeamB[1].LastName
				}
			}

			scoreStr := fmt.Sprintf("%d - %d", m.ScoreA(), m.ScoreB())

			var stagesStr []string
			for _, s := range m.Sets {
				stagesStr = append(stagesStr, fmt.Sprintf("%d-%d", s.ScoreA, s.ScoreB))
			}
			if len(stagesStr) > 0 {
				scoreStr += " (" + strings.Join(stagesStr, ", ") + ")"
			}

			pdf.CellFormat(30, 8, "Match", "1", 0, "C", false, 0, "")

			if m.WinnerTeam == "A" {
				pdf.SetFont("Arial", "B", 9)
			} else {
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(55, 8, truncateStr(nameA, 25), "1", 0, "L", false, 0, "")

			if m.WinnerTeam == "B" {
				pdf.SetFont("Arial", "B", 9)
			} else {
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(55, 8, truncateStr(nameB, 25), "1", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(40, 8, scoreStr, "1", 1, "C", false, 0, "")
		}
	}

	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func truncateStr(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
