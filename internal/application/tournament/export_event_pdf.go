package tournament

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jung-kurt/gofpdf"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
	eventDomain "table-tennis-backend/internal/domain/event"
)

type ExportEventPdfUseCase struct {
	tournamentRepo *bun.TournamentRepository
	eventRepo      eventDomain.Repository
}

func NewExportEventPdfUseCase(tournamentRepo *bun.TournamentRepository, eventRepo eventDomain.Repository) *ExportEventPdfUseCase {
	return &ExportEventPdfUseCase{tournamentRepo: tournamentRepo, eventRepo: eventRepo}
}

func (uc *ExportEventPdfUseCase) Execute(ctx context.Context, eventID string) ([]byte, error) {
	e, err := uc.eventRepo.GetByIDDeep(ctx, eventID)
	if err != nil {
		return nil, err
	}
	tournamentsList := e.Tournaments

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 15)

	// --- 1. COVER PAGE ---
	pdf.AddPage()
	pdf.SetFillColor(24, 24, 27) // sleek dark background accents
	
	// Draw a beautiful dark modern title header bar
	pdf.Rect(0, 0, 210, 60, "F")
	
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 24)
	pdf.Ln(10)
	pdf.CellFormat(0, 12, strings.ToUpper(e.Name), "", 1, "C", false, 0, "")
	
	pdf.SetFont("Arial", "I", 12)
	pdf.CellFormat(0, 8, "Grand Event & Sub-Tournament Records", "", 1, "C", false, 0, "")

	// Date and Summary Info
	pdf.SetTextColor(50, 50, 50)
	pdf.Ln(25)
	
	pdf.SetFont("Arial", "B", 14)
	pdf.CellFormat(0, 10, "REPORT SUMMARY", "B", 1, "L", false, 0, "")
	pdf.Ln(5)

	pdf.SetFont("Arial", "", 11)
	pdf.CellFormat(60, 8, "Date Generated:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, time.Now().Format("January 02, 2006 at 15:04 PM"), "", 1, "L", false, 0, "")

	pdf.CellFormat(60, 8, "Total Sub-Tournaments:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%d Tournaments", len(tournamentsList)), "", 1, "L", false, 0, "")

	var activeCount, finishedCount int
	for _, t := range tournamentsList {
		if t.Status == "finished" {
			finishedCount++
		} else {
			activeCount++
		}
	}
	pdf.CellFormat(60, 8, "Completed Tournaments:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%d", finishedCount), "", 1, "L", false, 0, "")

	pdf.CellFormat(60, 8, "Active/Ongoing Tournaments:", "", 0, "L", false, 0, "")
	pdf.CellFormat(0, 8, fmt.Sprintf("%d", activeCount), "", 1, "L", false, 0, "")

	pdf.Ln(40)
	pdf.SetFont("Arial", "I", 10)
	pdf.SetTextColor(120, 120, 120)
	pdf.CellFormat(0, 10, "Table Tennis Tournament Management System • Admin Reports", "", 1, "C", false, 0, "")

	// --- 2. INDIVIDUAL TOURNAMENT SECTIONS ---
	pdf.SetTextColor(0, 0, 0)
	
	for _, t := range tournamentsList {
		pdf.AddPage()

		// Tournament Title Block
		pdf.SetFont("Arial", "B", 16)
		pdf.SetFillColor(240, 240, 240)
		pdf.CellFormat(0, 12, strings.ToUpper(t.Name), "1", 1, "C", true, 0, "")
		pdf.Ln(4)

		// Metadata Table
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(30, 7, "Game Type:", "1", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(60, 7, strings.ToUpper(t.Type), "1", 0, "L", false, 0, "")
		
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(30, 7, "Status:", "1", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		statusText := strings.ToUpper(t.Status)
		pdf.CellFormat(0, 7, statusText, "1", 1, "L", false, 0, "")

		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(30, 7, "Format:", "1", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		pdf.CellFormat(60, 7, strings.ToUpper(t.Format), "1", 0, "L", false, 0, "")
		
		pdf.SetFont("Arial", "B", 9)
		pdf.CellFormat(30, 7, "Dates:", "1", 0, "L", false, 0, "")
		pdf.SetFont("Arial", "", 9)
		dateStr := fmt.Sprintf("%s - %s", t.StartDate.Format("Jan 02, 2006"), t.EndDate.Format("Jan 02, 2006"))
		pdf.CellFormat(0, 7, dateStr, "1", 1, "L", false, 0, "")

		pdf.Ln(4)

		// Display Final Places/Standings if finished
		if t.Status == "finished" {
			first, second, third := getTournamentPlaces(t)
			if first != "" || second != "" || third != "" {
				pdf.SetFillColor(245, 247, 250) // clean light grey background
				pdf.SetFont("Arial", "B", 10)
				pdf.CellFormat(0, 8, "  FINAL STANDINGS / PLACINGS", "1", 1, "L", true, 0, "")

				pdf.SetFont("Arial", "", 9)
				if first != "" {
					pdf.CellFormat(40, 7, "  1st Place (Champion):", "1", 0, "L", false, 0, "")
					pdf.SetFont("Arial", "B", 9)
					pdf.CellFormat(0, 7, "  " + strings.ToUpper(first), "1", 1, "L", false, 0, "")
					pdf.SetFont("Arial", "", 9)
				}
				if second != "" {
					pdf.CellFormat(40, 7, "  2nd Place:", "1", 0, "L", false, 0, "")
					pdf.CellFormat(0, 7, "  " + strings.ToUpper(second), "1", 1, "L", false, 0, "")
				}
				if third != "" {
					pdf.CellFormat(40, 7, "  3rd Place:", "1", 0, "L", false, 0, "")
					pdf.CellFormat(0, 7, "  " + strings.ToUpper(third), "1", 1, "L", false, 0, "")
				}
				pdf.Ln(4)
			}
		}

		// Players / Teams Section
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(0, 8, "PARTICIPANTS", "B", 1, "L", false, 0, "")
		pdf.Ln(2)

		if t.Type == "teams" {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(15, 7, "No.", "1", 0, "C", false, 0, "")
			pdf.CellFormat(165, 7, "Team Name", "1", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "", 9)
			for idx, team := range t.Teams {
				pdf.CellFormat(15, 7, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
				pdf.CellFormat(165, 7, team.Name, "1", 1, "L", false, 0, "")
			}
		} else {
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(15, 7, "No.", "1", 0, "C", false, 0, "")
			pdf.CellFormat(100, 7, "Player Name", "1", 0, "L", false, 0, "")
			pdf.CellFormat(30, 7, "Gender", "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 7, "Rating (Elo)", "1", 1, "C", false, 0, "")

			pdf.SetFont("Arial", "", 9)
			for idx, p := range t.Participants {
				fullName := p.FirstName + " " + p.LastName
				pdf.CellFormat(15, 7, fmt.Sprintf("%d", idx+1), "1", 0, "C", false, 0, "")
				pdf.CellFormat(100, 7, fullName, "1", 0, "L", false, 0, "")
				pdf.CellFormat(30, 7, p.Gender, "1", 0, "C", false, 0, "")
				
				rating := p.SinglesElo
				if t.Type == "doubles" || t.Type == "mixed_doubles" {
					rating = p.DoublesElo
				}
				pdf.CellFormat(35, 7, fmt.Sprintf("%d", rating), "1", 1, "C", false, 0, "")
			}
		}

		pdf.Ln(6)

		// Brackets & Match Results Section
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(0, 8, "BRACKETS & MATCH PLAY", "B", 1, "L", false, 0, "")
		pdf.Ln(2)

		if len(t.Matches) == 0 {
			pdf.SetFont("Arial", "I", 9)
			pdf.CellFormat(0, 7, "No matches generated yet.", "", 1, "L", false, 0, "")
		} else {
			// Group matches by their stages/rounds
			stageOrder := []string{"group", "r32", "r16", "quarterfinal", "semifinal", "final"}
			stageMap := make(map[string][]MatchData)

			for _, m := range t.Matches {
				// We skip sub-matches of team matches in global listing to avoid cluttering
				if m.TeamMatchID != nil {
					continue
				}

				nameA := getTeamDisplayName(m.TeamA, t.Type)
				if (t.Type == "doubles" || t.Type == "mixed_doubles") && len(m.TeamA) >= 2 {
					nameA = m.TeamA[0].LastName + "/" + m.TeamA[1].LastName
				}
				nameB := getTeamDisplayName(m.TeamB, t.Type)
				if (t.Type == "doubles" || t.Type == "mixed_doubles") && len(m.TeamB) >= 2 {
					nameB = m.TeamB[0].LastName + "/" + m.TeamB[1].LastName
				}
				
				scoreStr := fmt.Sprintf("%d - %d", m.ScoreA(), m.ScoreB())
				var setsList []string
				for _, set := range m.Sets {
					setsList = append(setsList, fmt.Sprintf("%d-%d", set.ScoreA, set.ScoreB))
				}
				if len(setsList) > 0 {
					scoreStr += " (" + strings.Join(setsList, ", ") + ")"
				}

				stage := strings.ToLower(m.Stage)
				stageMap[stage] = append(stageMap[stage], MatchData{
					PlayerA:    nameA,
					PlayerB:    nameB,
					Score:      scoreStr,
					WinnerTeam: m.WinnerTeam,
					Status:     m.Status,
				})
			}

			// Render each stage in order
			hasRenderedMatch := false
			for _, stgKey := range stageOrder {
				matches, exists := stageMap[stgKey]
				if !exists || len(matches) == 0 {
					continue
				}

				hasRenderedMatch = true
				stageTitle := getStageHeader(stgKey)

				pdf.SetFont("Arial", "B", 10)
				pdf.SetFillColor(245, 245, 245)
				pdf.CellFormat(0, 8, fmt.Sprintf(" %s", stageTitle), "1", 1, "L", true, 0, "")
				
				pdf.SetFont("Arial", "B", 8)
				pdf.CellFormat(70, 7, "Side A", "1", 0, "C", false, 0, "")
				pdf.CellFormat(70, 7, "Side B", "1", 0, "C", false, 0, "")
				pdf.CellFormat(40, 7, "Result/Score", "1", 1, "C", false, 0, "")

				for _, mData := range matches {
					// Left side (A)
					if mData.Status == "finished" && mData.WinnerTeam == "A" {
						pdf.SetFont("Arial", "B", 8)
					} else {
						pdf.SetFont("Arial", "", 8)
					}
					pdf.CellFormat(70, 7, truncateStr(mData.PlayerA, 35), "1", 0, "L", false, 0, "")

					// Right side (B)
					if mData.Status == "finished" && mData.WinnerTeam == "B" {
						pdf.SetFont("Arial", "B", 8)
					} else {
						pdf.SetFont("Arial", "", 8)
					}
					pdf.CellFormat(70, 7, truncateStr(mData.PlayerB, 35), "1", 0, "L", false, 0, "")

					// Score / Status
					pdf.SetFont("Arial", "", 8)
					dispScore := mData.Score
					if mData.Status != "finished" {
						dispScore = "SCHEDULED"
					}
					pdf.CellFormat(40, 7, dispScore, "1", 1, "C", false, 0, "")
				}
				pdf.Ln(3)
			}

			if !hasRenderedMatch {
				pdf.SetFont("Arial", "I", 9)
				pdf.CellFormat(0, 7, "No matches generated yet.", "", 1, "L", false, 0, "")
			}
		}
	}

	var buf bytes.Buffer
	err = pdf.Output(&buf)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

type MatchData struct {
	PlayerA    string
	PlayerB    string
	Score      string
	WinnerTeam string
	Status     string
}

func getStageHeader(stage string) string {
	switch stage {
	case "group":
		return "Group Stage Round-Robin"
	case "r32":
		return "Round of 32 (Knockout)"
	case "r16":
		return "Round of 16 (Knockout)"
	case "quarterfinal":
		return "Quarter-Finals"
	case "semifinal":
		return "Semi-Finals"
	case "final":
		return "Grand Final"
	default:
		return strings.ToUpper(stage)
	}
}
