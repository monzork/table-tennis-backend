package tournament

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jung-kurt/gofpdf"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type ExportTournamentPdfUseCase struct {
	tournamentRepo tournamentDomain.Repository
}

func NewExportTournamentPdfUseCase(tournamentRepo tournamentDomain.Repository) *ExportTournamentPdfUseCase {
	return &ExportTournamentPdfUseCase{tournamentRepo: tournamentRepo}
}

func (uc *ExportTournamentPdfUseCase) Execute(ctx context.Context, tournamentIDStr string) ([]byte, error) {
	t, err := uc.tournamentRepo.GetByID(ctx, tournamentIDStr)
	if err != nil {
		return nil, err
	}

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 42, 15)
	pdf.SetAutoPageBreak(true, 15)

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Locate header image dynamically so that tests run from subdirectories can find it.
	imagePath := findHeaderImage()

	// Header setup: printed on every page
	pdf.SetHeaderFunc(func() {
		pdf.Image(imagePath, 15, 10, 25, 0, false, "", 0, "")
		pdf.SetY(17)
		pdf.SetX(45)
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(0, 10, tr("TORNEO TENIS DE MESA - "+strings.ToUpper(t.Name)), "", 1, "L", false, 0, "")
		pdf.SetDrawColor(200, 200, 200)
		pdf.Line(15, 38, 195, 38)
	})

	// Helpers
	writeHeader := func(text string) {
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 8, tr(text), "", 1, "L", false, 0, "")
		pdf.Ln(3)
	}

	getMatchPlayerNames := func(m tournamentDomain.Match) (string, string) {
		nameA := "Team A"
		if len(m.TeamA) > 0 {
			nameA = m.TeamA[0].FirstName + " " + m.TeamA[0].LastName
			if len(m.TeamA) > 1 {
				nameA += "/" + m.TeamA[1].FirstName + " " + m.TeamA[1].LastName
			}
		}
		nameB := "Team B"
		if len(m.TeamB) > 0 {
			nameB = m.TeamB[0].FirstName + " " + m.TeamB[0].LastName
			if len(m.TeamB) > 1 {
				nameB += "/" + m.TeamB[1].FirstName + " " + m.TeamB[1].LastName
			}
		}
		return nameA, nameB
	}

	// 1. PARTICIPANT ASSOCIATIONS
	pdf.AddPage()
	writeHeader("PARTICIPANT ASSOCIATIONS")

	if t.Status == "finished" {
		first, second, third := getTournamentPlaces(t)
		if first != "" || second != "" || third != "" {
			pdf.SetFillColor(245, 247, 250) // clean light grey background
			pdf.SetFont("Arial", "B", 10)
			pdf.CellFormat(0, 8, tr("  FINAL STANDINGS / PLACINGS"), "1", 1, "L", true, 0, "")

			pdf.SetFont("Arial", "", 9)
			if first != "" {
				pdf.CellFormat(45, 7, tr("  1st Place (Champion):"), "1", 0, "L", false, 0, "")
				pdf.SetFont("Arial", "B", 9)
				pdf.CellFormat(0, 7, tr("  "+strings.ToUpper(first)), "1", 1, "L", false, 0, "")
				pdf.SetFont("Arial", "", 9)
			}
			if second != "" {
				pdf.CellFormat(45, 7, tr("  2nd Place:"), "1", 0, "L", false, 0, "")
				pdf.CellFormat(0, 7, tr("  "+strings.ToUpper(second)), "1", 1, "L", false, 0, "")
			}
			if third != "" {
				pdf.CellFormat(45, 7, tr("  3rd Place:"), "1", 0, "L", false, 0, "")
				pdf.CellFormat(0, 7, tr("  "+strings.ToUpper(third)), "1", 1, "L", false, 0, "")
			}
			pdf.Ln(10)
		}
	}

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
	pdf.CellFormat(100, 8, tr("Association Name"), "1", 0, "C", false, 0, "")
	pdf.CellFormat(40, 8, tr("ITTF Code"), "1", 1, "C", false, 0, "")

	pdf.SetFont("Arial", "", 10)
	codeMap := make(map[string]string)
	for i, a := range assocs {
		code := a
		if len(code) > 3 {
			runes := []rune(code)
			if len(runes) > 3 {
				code = string(runes[:3])
			}
			code = strings.ToUpper(code)
		}
		codeMap[a] = code
		pdf.CellFormat(20, 8, fmt.Sprintf("%d", i+1), "1", 0, "C", false, 0, "")
		pdf.CellFormat(100, 8, tr(a), "1", 0, "L", false, 0, "")
		pdf.CellFormat(40, 8, tr(code), "1", 1, "C", false, 0, "")
	}

	// 2. PARTICIPANTS LIST (SINGLE TABLE)
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

		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(20, 8, "Nr", "1", 0, "C", false, 0, "")
		pdf.CellFormat(110, 8, "NAME", "1", 0, "C", false, 0, "")
		pdf.CellFormat(50, 8, tr("ASSOC."), "1", 1, "C", false, 0, "")

		// Men List
		if len(mens) > 0 {
			pdf.SetFillColor(240, 240, 240)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(180, 8, tr("  MALE PARTICIPANTS / VARONES"), "1", 1, "L", true, 0, "")

			pdf.SetFont("Arial", "", 10)
			for i, p := range mens {
				country := strings.TrimSpace(p.Country)
				if country == "" {
					country = "UNKNOWN"
				}
				pdf.CellFormat(20, 8, fmt.Sprintf("%d", 101+i), "1", 0, "C", false, 0, "")
				pdf.CellFormat(110, 8, tr(p.FirstName+" "+p.LastName), "1", 0, "L", false, 0, "")
				pdf.CellFormat(50, 8, tr(codeMap[country]), "1", 1, "C", false, 0, "")
			}
		}

		// Women List
		if len(womens) > 0 {
			pdf.SetFillColor(240, 240, 240)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(180, 8, tr("  FEMALE PARTICIPANTS / DAMAS"), "1", 1, "L", true, 0, "")

			pdf.SetFont("Arial", "", 10)
			for i, p := range womens {
				country := strings.TrimSpace(p.Country)
				if country == "" {
					country = "UNKNOWN"
				}
				pdf.CellFormat(20, 8, fmt.Sprintf("%d", 301+i), "1", 0, "C", false, 0, "")
				pdf.CellFormat(110, 8, tr(p.FirstName+" "+p.LastName), "1", 0, "L", false, 0, "")
				pdf.CellFormat(50, 8, tr(codeMap[country]), "1", 1, "C", false, 0, "")
			}
		}
	}

	// 3. GROUP STAGE AND KNOCKOUT TABLES
	var groupMatches []tournamentDomain.Match
	var drawMatches []tournamentDomain.Match
	for _, m := range t.Matches {
		if m.TeamMatchID != nil {
			continue
		}
		if strings.ToLower(m.Stage) == "group" {
			groupMatches = append(groupMatches, m)
		} else {
			drawMatches = append(drawMatches, m)
		}
	}

	// Sort draw matches by stage order
	stagePriority := map[string]int{
		"r32":          1,
		"r16":          2,
		"quarterfinal": 3,
		"semifinal":    4,
		"final":        5,
	}
	sort.Slice(drawMatches, func(i, j int) bool {
		pI := stagePriority[strings.ToLower(drawMatches[i].Stage)]
		pJ := stagePriority[strings.ToLower(drawMatches[j].Stage)]
		if pI == 0 {
			pI = 99
		}
		if pJ == 0 {
			pJ = 99
		}
		if pI != pJ {
			return pI < pJ
		}
		return drawMatches[i].ID < drawMatches[j].ID
	})

	if len(groupMatches) > 0 {
		pdf.AddPage()
		writeHeader("GROUP STAGE MATCHES")

		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(30, 8, tr("Stage"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 8, tr("Team A"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 8, tr("Team B"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 8, tr("Score"), "1", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 9)
		for _, m := range groupMatches {
			nameA, nameB := getMatchPlayerNames(m)
			scoreStr := tr("Scheduled")
			if m.Status == "finished" {
				scoreStr = fmt.Sprintf("%d - %d", m.ScoreA(), m.ScoreB())
			} else if m.Status == "in_progress" {
				scoreStr = tr("In Progress")
			}

			pdf.CellFormat(30, 8, tr("Group Stage"), "1", 0, "C", false, 0, "")

			if m.Status == "finished" && m.WinnerTeam == "A" {
				pdf.SetFont("Arial", "B", 9)
			} else {
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(55, 8, tr(truncateStr(nameA, 25)), "1", 0, "L", false, 0, "")

			if m.Status == "finished" && m.WinnerTeam == "B" {
				pdf.SetFont("Arial", "B", 9)
			} else {
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(55, 8, tr(truncateStr(nameB, 25)), "1", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(40, 8, scoreStr, "1", 1, "C", false, 0, "")
		}
	}

	if len(drawMatches) > 0 {
		pdf.AddPage()
		writeHeader("THE DRAW (KNOCKOUT)")

		pdf.SetFont("Arial", "B", 10)
		pdf.CellFormat(30, 8, tr("Stage"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 8, tr("Team A"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(55, 8, tr("Team B"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(40, 8, tr("Score"), "1", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 9)
		for _, m := range drawMatches {
			nameA, nameB := getMatchPlayerNames(m)
			scoreStr := tr("Scheduled")
			if m.Status == "finished" {
				scoreStr = fmt.Sprintf("%d - %d", m.ScoreA(), m.ScoreB())
			} else if m.Status == "in_progress" {
				scoreStr = tr("In Progress")
			}

			stageText := getTournamentStageHeader(strings.ToLower(m.Stage))

			pdf.CellFormat(30, 8, tr(stageText), "1", 0, "C", false, 0, "")

			if m.Status == "finished" && m.WinnerTeam == "A" {
				pdf.SetFont("Arial", "B", 9)
			} else {
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(55, 8, tr(truncateStr(nameA, 25)), "1", 0, "L", false, 0, "")

			if m.Status == "finished" && m.WinnerTeam == "B" {
				pdf.SetFont("Arial", "B", 9)
			} else {
				pdf.SetFont("Arial", "", 9)
			}
			pdf.CellFormat(55, 8, tr(truncateStr(nameB, 25)), "1", 0, "L", false, 0, "")

			pdf.SetFont("Arial", "", 9)
			pdf.CellFormat(40, 8, scoreStr, "1", 1, "C", false, 0, "")
		}
	}

	// 4. DETAILED MATCH RESULTS
	if len(t.Matches) > 0 {
		pdf.AddPage()
		writeHeader("DETAILED MATCH RESULTS")

		pdf.SetFont("Arial", "B", 8)
		pdf.CellFormat(60, 8, tr("Matchup (PlayerA vs PlayerB)"), "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "1st Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "2nd Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "3rd Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "4th Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "5th Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "6th Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(13, 8, "7th Set", "1", 0, "C", false, 0, "")
		pdf.CellFormat(29, 8, tr("Result"), "1", 1, "C", false, 0, "")

		pdf.SetFont("Arial", "", 8)
		for _, m := range t.Matches {
			if m.TeamMatchID != nil {
				continue
			}
			nameA, nameB := getMatchPlayerNames(m)
			matchupStr := truncateStr(nameA, 15) + " vs " + truncateStr(nameB, 15)

			pdf.CellFormat(60, 8, tr(matchupStr), "1", 0, "L", false, 0, "")

			// Render sets
			for setNum := 1; setNum <= 7; setNum++ {
				var setScoreStr = "—"
				for _, set := range m.Sets {
					if set.Number == setNum {
						setScoreStr = fmt.Sprintf("%d-%d", set.ScoreA, set.ScoreB)
						break
					}
				}
				pdf.CellFormat(13, 8, setScoreStr, "1", 0, "C", false, 0, "")
			}

			// Final result
			resStr := ""
			if m.Status == "finished" {
				resStr = fmt.Sprintf("%d - %d", m.ScoreA(), m.ScoreB())
			} else if m.Status == "in_progress" {
				resStr = tr("In Progress")
			} else {
				resStr = tr("Scheduled")
			}
			pdf.CellFormat(29, 8, resStr, "1", 1, "C", false, 0, "")
		}
	}

	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func getTournamentStageHeader(stage string) string {
	switch stage {
	case "group":
		return "Group Stage"
	case "r32":
		return "Round of 32"
	case "r16":
		return "Round of 16"
	case "quarterfinal":
		return "Quarter-Finals"
	case "semifinal":
		return "Semi-Finals"
	case "final":
		return "Final"
	default:
		return strings.ToUpper(stage)
	}
}

func truncateStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max])
	}
	return s
}

func findHeaderImage() string {
	dir, err := os.Getwd()
	if err != nil {
		return "open_tdm.jpeg"
	}
	for {
		target := filepath.Join(dir, "open_tdm.jpeg")
		if _, err := os.Stat(target); err == nil {
			return target
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "open_tdm.jpeg"
}


