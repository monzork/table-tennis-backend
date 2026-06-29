package tournament

import (
	"bytes"
	"context"
	"fmt"
	"math"
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
	pdf.SetMargins(15, 52, 15)
	pdf.SetAutoPageBreak(true, 15)

	// Build player bib/dorsal number map
	playerNumberMap := make(map[string]int)
	var mens, womens []*player.Player
	for _, p := range t.Participants {
		if p.Gender == "M" {
			mens = append(mens, p)
		} else {
			womens = append(womens, p)
		}
	}
	for i, p := range mens {
		playerNumberMap[p.ID] = 101 + i
	}
	for i, p := range womens {
		playerNumberMap[p.ID] = 301 + i
	}

	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// Locate header image dynamically so that tests run from subdirectories can find it.
	imagePath := findHeaderImage()

	// Header setup: printed on every page
	pdf.SetHeaderFunc(func() {
		pdf.Image(imagePath, 15, 10, 25, 0, false, "", 0, "")
		pdf.SetY(17)
		pdf.SetX(48)
		pdf.SetFont("Arial", "B", 14)
		pdf.CellFormat(0, 10, tr("TORNEO TENIS DE MESA - "+strings.ToUpper(t.Name)), "", 1, "L", false, 0, "")
		pdf.SetDrawColor(200, 200, 200)
		w, _ := pdf.GetPageSize()
		pdf.Line(15, 45, w-15, 45)
	})

	// Helpers
	writeHeader := func(text string) {
		pdf.Ln(5)
		pdf.SetFont("Arial", "B", 12)
		pdf.CellFormat(0, 8, tr(text), "", 1, "L", false, 0, "")
		pdf.Ln(3)
	}

	formatPlayerName := func(p *player.Player) string {
		if p == nil {
			return ""
		}
		lastName := strings.TrimSpace(p.LastName)
		if lastName == " (Team)" || lastName == "" {
			return p.FirstName
		}
		return p.FirstName + " " + p.LastName
	}

	getMatchPlayerNames := func(m tournamentDomain.Match) (string, string) {
		nameA := "Team A"
		if len(m.TeamA) > 0 {
			nameA = formatPlayerName(m.TeamA[0])
			if len(m.TeamA) > 1 {
				nameA += "/" + formatPlayerName(m.TeamA[1])
			}
		}
		nameB := "Team B"
		if len(m.TeamB) > 0 {
			nameB = formatPlayerName(m.TeamB[0])
			if len(m.TeamB) > 1 {
				nameB += "/" + formatPlayerName(m.TeamB[1])
			}
		}
		return nameA, nameB
	}

	getCountryCode := func(country string) string {
		c := strings.TrimSpace(country)
		if c == "" {
			return "UNK"
		}
		runes := []rune(c)
		if len(runes) > 3 {
			c = string(runes[:3])
		}
		return strings.ToUpper(c)
	}

	// 1. FINAL STANDINGS / PLACINGS
	if t.Status == "finished" {
		first, second, third := getTournamentPlaces(t)
		if first != "" || second != "" || third != "" {
			pdf.AddPage()
			writeHeader("FINAL STANDINGS / PLACINGS")

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
		}
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
				pdf.CellFormat(110, 8, tr(formatPlayerName(p)), "1", 0, "L", false, 0, "")
				pdf.CellFormat(50, 8, tr(getCountryCode(country)), "1", 1, "C", false, 0, "")
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
				pdf.CellFormat(110, 8, tr(formatPlayerName(p)), "1", 0, "L", false, 0, "")
				pdf.CellFormat(50, 8, tr(getCountryCode(country)), "1", 1, "C", false, 0, "")
			}
		}
	}

	// 2.5 GROUP STANDINGS / POOLS
	if t.Format == "round_robin" || t.Format == "groups_elimination" {
		type groupStandings struct {
			DivisionName string
			GroupName    string
			Players      []*player.Player
			Standings    []tournamentDomain.PlayerStanding
		}
		var groupStages []groupStandings
		for i := range t.Groups {
			g := &t.Groups[i]
			if !strings.HasSuffix(g.Name, " - Bracket Draw") {
				divName := "Open Division"
				grpName := g.Name
				if idx := strings.Index(g.Name, " - "); idx != -1 {
					divName = g.Name[:idx]
					grpName = g.Name[idx+3:]
				}
				// Filter matches that are in this group and have valid teams
				var gMatches []tournamentDomain.Match
				for _, m := range t.Matches {
					if m.TeamMatchID != nil {
						continue
					}
					if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
						p1InGroup, p2InGroup := false, false
						for _, gp := range g.Players {
							if gp.ID == m.TeamA[0].ID {
								p1InGroup = true
							}
							if gp.ID == m.TeamB[0].ID {
								p2InGroup = true
							}
						}
						if p1InGroup && p2InGroup && strings.ToLower(m.Stage) == "group" {
							gMatches = append(gMatches, m)
						}
					}
				}
				st := tournamentDomain.BuildStandings(g.Players, gMatches)
				groupStages = append(groupStages, groupStandings{
					DivisionName: divName,
					GroupName:    grpName,
					Players:      g.Players,
					Standings:    st,
				})
			}
		}

		if len(groupStages) > 0 {
			pdf.AddPage()
			writeHeader("GROUP STANDINGS / TABLAS DE POSICIONES DE GRUPOS")

			for _, gs := range groupStages {
				pdf.SetFont("Arial", "B", 10)
				pdf.CellFormat(0, 8, tr(strings.ToUpper(gs.DivisionName)+" - "+strings.ToUpper(gs.GroupName)), "", 1, "L", false, 0, "")
				pdf.Ln(2)

				// Find matches in this group
				var gMatches []tournamentDomain.Match
				for _, m := range t.Matches {
					if m.TeamMatchID != nil {
						continue
					}
					if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
						p1InGroup, p2InGroup := false, false
						for _, gp := range gs.Players {
							if gp.ID == m.TeamA[0].ID {
								p1InGroup = true
							}
							if gp.ID == m.TeamB[0].ID {
								p2InGroup = true
							}
						}
						if p1InGroup && p2InGroup && strings.ToLower(m.Stage) == "group" {
							gMatches = append(gMatches, m)
						}
					}
				}

				// Build standing map (playerID -> rank)
				standingMap := make(map[string]int)
				for idx, std := range gs.Standings {
					standingMap[std.Player.ID] = idx + 1
				}

				startY := pdf.GetY()

				// --- PART A: Match Schedule (Left, width 42mm) ---
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212) // yellow header
				pdf.CellFormat(12, 5, "Day", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 5, "Time", "1", 0, "C", true, 0, "")
				pdf.CellFormat(9, 5, "Table", "1", 0, "C", true, 0, "")
				pdf.CellFormat(10, 5, "Match", "1", 1, "C", true, 0, "")

				pdf.SetFont("Arial", "", 7)
				for i := 0; i < len(gs.Players); i++ {
					// Draw up to 6 matches or len(gMatches) rows
					if i < len(gMatches) {
						m := gMatches[i]
						dStr := t.StartDate.Format("02-Jan")
						tStr := "10:00"
						if m.UpdatedAt != nil {
							dStr = m.UpdatedAt.Format("02-Jan")
							tStr = m.UpdatedAt.Format("15:04")
						}
						tblStr := "-"
						if m.TableNumber != nil {
							tblStr = fmt.Sprintf("%d", *m.TableNumber)
						}

						idxA, idxB := -1, -1
						for idx, gp := range gs.Players {
							if gp.ID == m.TeamA[0].ID { idxA = idx + 1 }
							if gp.ID == m.TeamB[0].ID { idxB = idx + 1 }
						}
						matchIdxStr := fmt.Sprintf("%d-%d", idxA, idxB)

						pdf.SetTextColor(0, 0, 0)
						pdf.CellFormat(12, 5, dStr, "1", 0, "C", false, 0, "")
						pdf.CellFormat(11, 5, tStr, "1", 0, "C", false, 0, "")
						pdf.SetTextColor(30, 80, 220) // blue
						pdf.CellFormat(9, 5, tblStr, "1", 0, "C", false, 0, "")
						pdf.CellFormat(10, 5, matchIdxStr, "1", 1, "C", false, 0, "")
					} else {
						// Empty padding rows to align heights
						pdf.CellFormat(12, 5, "", "1", 0, "C", false, 0, "")
						pdf.CellFormat(11, 5, "", "1", 0, "C", false, 0, "")
						pdf.CellFormat(9, 5, "", "1", 0, "C", false, 0, "")
						pdf.CellFormat(10, 5, "", "1", 1, "C", false, 0, "")
					}
				}
				pdf.SetTextColor(0, 0, 0)

				// --- PART B: Cross-Table Matrix (Middle, width 55 + 8*N mm) ---
				n := len(gs.Players)
				pdf.SetXY(15 + 42 + 5, startY)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212)
				pdf.CellFormat(55, 5, tr("   ") + tr(strings.ToUpper(gs.GroupName)), "1", 0, "L", true, 0, "")
				for col := 1; col <= n; col++ {
					pdf.CellFormat(8, 5, fmt.Sprintf("%d", col), "1", 0, "C", true, 0, "")
				}
				pdf.Ln(5)

				for rowIdx, p1 := range gs.Players {
					pdf.SetX(15 + 42 + 5)
					// Draw player/team info cell
					startX, currY := pdf.GetXY()
					pdf.CellFormat(55, 5, "", "1", 0, "L", false, 0, "")
					
					// Draw custom colored texts inside the cell
					pdf.SetXY(startX + 2, currY + 1)
					pdf.SetFont("Arial", "B", 7)
					pdf.SetTextColor(30, 80, 220) // blue index
					pdf.Text(pdf.GetX(), pdf.GetY() + 2.5, fmt.Sprintf("%d", rowIdx+1))
					
					pdf.SetX(startX + 6)
					pdf.SetTextColor(220, 100, 0) // orange bib
					bibStr := ""
					if num, ok := playerNumberMap[p1.ID]; ok {
						bibStr = fmt.Sprintf("%d", num)
					}
					pdf.Text(pdf.GetX(), pdf.GetY() + 2.5, bibStr)

					pdf.SetX(startX + 12)
					pdf.SetTextColor(0, 0, 0) // black name
					pdf.Text(pdf.GetX(), pdf.GetY() + 2.5, tr(truncateStr(formatPlayerName(p1), 18)))

					pdf.SetX(startX + 45)
					pdf.SetTextColor(120, 120, 120) // gray country
					pdf.Text(pdf.GetX(), pdf.GetY() + 2.5, getCountryCode(p1.Country))

					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(startX + 55, currY)

					// Draw columns
					pdf.SetFont("Arial", "", 7)
					for colIdx, p2 := range gs.Players {
						if rowIdx == colIdx {
							pdf.SetFillColor(220, 220, 220) // gray diagonal
							pdf.CellFormat(8, 5, "", "1", 0, "C", true, 0, "")
						} else {
							// Find match between p1 and p2
							var mVal = "-"
							for _, m := range gMatches {
								if (m.TeamA[0].ID == p1.ID && m.TeamB[0].ID == p2.ID) || (m.TeamA[0].ID == p2.ID && m.TeamB[0].ID == p1.ID) {
									if m.Status == "finished" {
										if m.TeamA[0].ID == p1.ID {
											mVal = fmt.Sprintf("%d-%d", m.ScoreA(), m.ScoreB())
										} else {
											mVal = fmt.Sprintf("%d-%d", m.ScoreB(), m.ScoreA())
										}
									}
									break
								}
							}
							pdf.CellFormat(8, 5, mVal, "1", 0, "C", false, 0, "")
						}
					}
					pdf.Ln(5)
				}

				// --- PART C: Points & Positions (Right, width 21mm) ---
				pdf.SetXY(15 + 42 + 5 + 55 + float64(n)*8 + 5, startY)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212)
				pdf.CellFormat(11, 5, "Points", "1", 0, "C", true, 0, "")
				pdf.CellFormat(10, 5, "Pos.", "1", 1, "C", true, 0, "")

				for _, p := range gs.Players {
					pdf.SetX(15 + 42 + 5 + 55 + float64(n)*8 + 5)
					
					var wins, losses int
					for _, std := range gs.Standings {
						if std.Player.ID == p.ID {
							wins = std.Wins
							losses = std.Losses
							break
						}
					}
					pts := wins*2 + losses
					posVal := standingMap[p.ID]

					pdf.SetFont("Arial", "", 7)
					pdf.CellFormat(11, 5, fmt.Sprintf("%d", pts), "1", 0, "C", false, 0, "")
					pdf.SetFont("Arial", "B", 7)
					pdf.CellFormat(10, 5, fmt.Sprintf("%d", posVal), "1", 1, "C", false, 0, "")
				}

				pdf.SetXY(15, startY + float64(n+1)*5 + 3)
				pdf.Ln(6)
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
			if m.TableNumber != nil {
				scoreStr += fmt.Sprintf(" (Table %d)", *m.TableNumber)
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
			if m.TableNumber != nil {
				scoreStr += fmt.Sprintf(" (Table %d)", *m.TableNumber)
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

	// 3.5 VISUAL BRACKET DRAW
	if t.Format == "elimination" || t.Format == "groups_elimination" {
		type divisionBracket struct {
			Name   string
			Group  *tournamentDomain.Group
			Rounds []pdfRoundView
		}
		var brackets []divisionBracket

		for i := range t.Groups {
			g := &t.Groups[i]
			if strings.HasSuffix(g.Name, " - Bracket Draw") {
				divName := g.Name[:len(g.Name)-15]
				var rounds []pdfRoundView
				if t.Format == "groups_elimination" {
					var divRRGroups []*tournamentDomain.Group
					for j := range t.Groups {
						rg := &t.Groups[j]
						if !strings.HasSuffix(rg.Name, " - Bracket Draw") && strings.HasPrefix(rg.Name, divName+" - ") {
							divRRGroups = append(divRRGroups, rg)
						}
					}
					sort.Slice(divRRGroups, func(a, b int) bool {
						return divRRGroups[a].Name < divRRGroups[b].Name
					})

					var advancing []*player.Player
					take := t.GroupPassCount
					if take == 0 {
						take = 2
					}
					for _, rg := range divRRGroups {
						var rgMatches []tournamentDomain.Match
						for _, m := range t.Matches {
							if m.TeamMatchID != nil {
								continue
							}
							if len(m.TeamA) > 0 && len(m.TeamB) > 0 {
								p1InGroup, p2InGroup := false, false
								for _, gp := range rg.Players {
									if gp.ID == m.TeamA[0].ID {
										p1InGroup = true
									}
									if gp.ID == m.TeamB[0].ID {
										p2InGroup = true
									}
								}
								if p1InGroup && p2InGroup && strings.ToLower(m.Stage) == "group" {
									rgMatches = append(rgMatches, m)
								}
							}
						}
						st := tournamentDomain.BuildStandings(rg.Players, rgMatches)
						limit := int(take)
						if limit > len(st) {
							limit = len(st)
						}
						for k := 0; k < limit; k++ {
							advancing = append(advancing, st[k].Player)
						}
					}
					sort.Slice(advancing, func(a, b int) bool {
						ea := advancing[a].SinglesElo
						eb := advancing[b].SinglesElo
						if t.Type == "doubles" || t.Type == "mixed_doubles" {
							ea = advancing[a].DoublesElo
							eb = advancing[b].DoublesElo
						}
						return ea > eb
					})
					rounds = buildPdfBracketRounds(t, advancing)
				} else {
					rounds = buildPdfBracketRounds(t, g.Players)
				}

				if len(rounds) > 0 {
					brackets = append(brackets, divisionBracket{
						Name:   divName,
						Group:  g,
						Rounds: rounds,
					})
				}
			}
		}

		for _, br := range brackets {
			pdf.AddPageFormat("L", gofpdf.SizeType{Wd: 210, Ht: 297})
			pdf.SetMargins(15, 52, 15)

			pdf.SetFont("Arial", "B", 12)
			pdf.CellFormat(0, 8, tr("VISUAL BRACKET - "+strings.ToUpper(br.Name)), "", 1, "C", false, 0, "")
			pdf.Ln(4)

			w, h := pdf.GetPageSize()
			marginL, marginT, marginR, marginB := 15.0, 52.0, 15.0, 15.0
			printableW := w - marginL - marginR
			printableH := h - marginT - marginB

			rounds := br.Rounds
			numRounds := len(rounds)
			if numRounds == 0 {
				continue
			}

			colW := printableW / float64(numRounds)
			boxW := colW - 8.0
			if boxW > 45.0 {
				boxW = 45.0
			}
			boxH := 12.0

			// Pre-calculate Y centers for all match boxes to avoid overlaps and layout constraints
			centers := make([][]float64, numRounds)
			for r := range rounds {
				centers[r] = make([]float64, len(rounds[r].Matches))
			}

			// Round 0 is spread uniformly
			k0 := len(rounds[0].Matches)
			if k0 == 1 {
				centers[0][0] = marginT + printableH/2
			} else if k0 > 1 {
				spacing := (printableH - boxH) / float64(k0-1)
				for j := 0; j < k0; j++ {
					centers[0][j] = marginT + boxH/2 + float64(j)*spacing
				}
			}

			// Subsequent rounds are calculated as midpoints of their children
			for r := 1; r < numRounds; r++ {
				for j := range rounds[r].Matches {
					if rounds[r].Name == "Champion" {
						centers[r][0] = centers[r-1][0]
					} else {
						c1 := 2 * j
						c2 := 2*j + 1
						if c1 < len(centers[r-1]) && c2 < len(centers[r-1]) {
							centers[r][j] = (centers[r-1][c1] + centers[r-1][c2]) / 2
						} else if c1 < len(centers[r-1]) {
							centers[r][j] = centers[r-1][c1]
						} else {
							centers[r][j] = marginT + printableH/2
						}
					}
				}
			}

			// Draw Round Headers
			pdf.SetFont("Arial", "B", 8)
			pdf.SetTextColor(100, 100, 100)
			for r, round := range rounds {
				colStartX := marginL + float64(r)*colW
				textX := colStartX + (colW-boxW)/2
				pdf.Text(textX, marginT - 3, tr(round.Name))
			}
			pdf.SetTextColor(0, 0, 0)

			getBracketPlayerText := func(sp *pdfMatchSlot) (string, string, string) {
				if sp == nil || sp.Player == nil {
					return "", "TBD", ""
				}
				p := sp.Player
				numStr := ""
				if num, ok := playerNumberMap[p.ID]; ok {
					numStr = fmt.Sprintf("%d", num)
				}
				name := strings.ToUpper(formatPlayerName(p))
				cc := getCountryCode(p.Country)
				return numStr, name, cc
			}

			for r, round := range rounds {
				colStartX := marginL + float64(r)*colW
				x := colStartX + (colW-boxW)/2
				numMatches := len(round.Matches)

				for j, m := range round.Matches {
					y := centers[r][j] - boxH/2

					if round.Name == "Champion" {
						pdf.SetFillColor(254, 254, 212) // yellow
						pdf.Rect(x, y+boxH/4, boxW, boxH/2, "FD")

						// Find the final match to get its score
						var finalScore string
						for _, tm := range t.Matches {
							if tm.Stage == "final" && tm.Status == "finished" && tm.TeamMatchID == nil {
								finalScore = fmt.Sprintf("(%d-%d)", tm.ScoreA(), tm.ScoreB())
								break
							}
						}

						pdf.SetFont("Arial", "B", 6)
						pdf.SetTextColor(0, 0, 0)
						champName := "TBD"
						var champNum, champCC string
						if m.Player1 != nil && m.Player1.Player != nil {
							champName = strings.ToUpper(formatPlayerName(m.Player1.Player))
							if num, ok := playerNumberMap[m.Player1.Player.ID]; ok {
								champNum = fmt.Sprintf("%d", num)
							}
							champCC = getCountryCode(m.Player1.Player.Country)
						}
						
						// Print champion text
						pdf.SetTextColor(220, 100, 0) // orange for bib
						pdf.Text(x+2, y+boxH/2+1, champNum)
						pdf.SetTextColor(0, 0, 0) // black name
						pdf.Text(x+8, y+boxH/2+1, tr("🏆 "+truncateStr(champName, 14)))
						pdf.SetTextColor(120, 120, 120) // gray country
						pdf.Text(x+boxW-8, y+boxH/2+1, champCC)

						if finalScore != "" {
							pdf.SetFont("Arial", "", 5)
							pdf.SetTextColor(0, 0, 0)
							pdf.Text(x+boxW/2-4, y+boxH/2+4, finalScore)
						}
						continue
					}

					// Draw Player 1 box (top half)
					pdf.SetFillColor(254, 254, 212) // yellow
					pdf.Rect(x, y, boxW, boxH/2, "FD")

					// Draw Player 2 box (bottom half)
					pdf.Rect(x, y+boxH/2, boxW, boxH/2, "FD")

					p1Bold, p2Bold := "", ""
					if m.Match != nil && m.Match.Status == "finished" {
						if m.Match.WinnerTeam == "A" {
							p1Bold = "B"
						} else if m.Match.WinnerTeam == "B" {
							p2Bold = "B"
						}
					}

					p1Num, p1Name, p1CC := getBracketPlayerText(m.Player1)
					p2Num, p2Name, p2CC := getBracketPlayerText(m.Player2)

					// Print Player 1 text
					pdf.SetFont("Arial", p1Bold, 6)
					pdf.SetTextColor(220, 100, 0) // orange for bib
					pdf.Text(x+2, y+4, p1Num)
					pdf.SetTextColor(0, 0, 0) // black
					pdf.Text(x+8, y+4, tr(truncateStr(p1Name, 15)))
					pdf.SetTextColor(120, 120, 120) // gray for country
					pdf.Text(x+boxW-8, y+4, p1CC)

					// Print Player 2 text
					pdf.SetFont("Arial", p2Bold, 6)
					pdf.SetTextColor(220, 100, 0) // orange for bib
					pdf.Text(x+2, y+10, p2Num)
					pdf.SetTextColor(0, 0, 0) // black
					pdf.Text(x+8, y+10, tr(truncateStr(p2Name, 15)))
					pdf.SetTextColor(120, 120, 120) // gray for country
					pdf.Text(x+boxW-8, y+10, p2CC)
				}

				if r < numRounds-1 {
					nextNumMatches := len(rounds[r+1].Matches)
					if nextNumMatches > 0 && rounds[r+1].Name != "Champion" {
						for j := 0; j < numMatches; j++ {
							currentMidY := centers[r][j]
							nextJ := j / 2
							nextMidY := centers[r+1][nextJ]

							lineX1 := x + boxW
							lineX2 := x + boxW + (colW-boxW)/2

							pdf.SetDrawColor(180, 180, 180)
							pdf.Line(lineX1, currentMidY, lineX2, currentMidY)

							// Print match details above and score below the line
							mForDetails := round.Matches[j]
							if mForDetails.Match != nil {
								dStr := t.StartDate.Format("02-Jan")
								tStr := "16:00"
								if mForDetails.Match.UpdatedAt != nil {
									dStr = mForDetails.Match.UpdatedAt.Format("02-Jan")
									tStr = mForDetails.Match.UpdatedAt.Format("15:04")
								}
								tblStr := ""
								if mForDetails.Match.TableNumber != nil {
									tblStr = fmt.Sprintf(" - Table %d", *mForDetails.Match.TableNumber)
								}
								matchDetails := fmt.Sprintf("%s - %sh%s", dStr, tStr, tblStr)

								pdf.SetFont("Arial", "", 5)
								pdf.SetTextColor(30, 80, 220) // blue
								pdf.Text(lineX1 + 1, currentMidY - 1, tr(matchDetails))

								if mForDetails.Match.Status == "finished" {
									scoreStr := fmt.Sprintf("(%d-%d)", mForDetails.Match.ScoreA(), mForDetails.Match.ScoreB())
									pdf.SetFont("Arial", "B", 5)
									pdf.SetTextColor(0, 0, 0) // black
									pdf.Text(lineX1 + 1, currentMidY + 3, scoreStr)
								}
							}
							pdf.SetTextColor(0, 0, 0)

							if j%2 == 0 && j+1 < numMatches {
								siblingMidY := centers[r][j+1]
								pdf.Line(lineX2, currentMidY, lineX2, siblingMidY)

								nextColStartX := marginL + float64(r+1)*colW
								nextColBoxX := nextColStartX + (colW-boxW)/2
								pdf.Line(lineX2, nextMidY, nextColBoxX, nextMidY)
							}
						}
					}
				}
			}
		}
	}

	// 4. DETAILED MATCH RESULTS
	if len(t.Matches) > 0 {
		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: 210, Ht: 297})
		pdf.SetMargins(15, 52, 15)
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

		if t.Type == "teams" {
			// Build player bib/dorsal number map
			playerNumberMap := make(map[string]int)
			var mens, womens []*player.Player
			for _, p := range t.Participants {
				if p.Gender == "M" {
					mens = append(mens, p)
				} else {
					womens = append(womens, p)
				}
			}
			for i, p := range mens {
				playerNumberMap[p.ID] = 101 + i
			}
			for i, p := range womens {
				playerNumberMap[p.ID] = 301 + i
			}

			for _, m := range t.Matches {
				if m.TeamMatchID != nil {
					continue
				}

				// We are rendering a single Team Match Tie (like the image)
				// Let's get the team names
				teamAName := "Team A"
				if len(m.TeamA) > 0 {
					teamAName = m.TeamA[0].FirstName
				}
				teamBName := "Team B"
				if len(m.TeamB) > 0 {
					teamBName = m.TeamB[0].FirstName
				}

				// Date & Time from m.UpdatedAt or tournament dates
				dateStr := t.StartDate.Format("02-Jan")
				timeStr := "10:00" // Default or placeholder
				if m.UpdatedAt != nil {
					dateStr = m.UpdatedAt.Format("02-Jan")
					timeStr = m.UpdatedAt.Format("15:04")
				}

				tableNumStr := "-"
				if m.TableNumber != nil {
					tableNumStr = fmt.Sprintf("%d", *m.TableNumber)
				}

				// --- 1. HEADER ROW ---
				// Event | Stage | Day | Time | Table
				pdf.SetFont("Arial", "B", 8)
				pdf.SetFillColor(254, 254, 212) // beautiful light yellow/cream

				pdf.CellFormat(50, 6, tr(" Event: ") + tr(truncateStr(t.Name, 22)), "1", 0, "L", true, 0, "")
				stageText := getTournamentStageHeader(strings.ToLower(m.Stage))
				pdf.CellFormat(55, 6, tr(" Stage: ") + tr(truncateStr(stageText, 25)), "1", 0, "L", true, 0, "")
				pdf.CellFormat(25, 6, tr(" Day: ") + dateStr, "1", 0, "L", true, 0, "")
				pdf.CellFormat(25, 6, tr(" Time: ") + timeStr, "1", 0, "L", true, 0, "")
				pdf.CellFormat(25, 6, tr(" Table: ") + tableNumStr, "1", 1, "L", true, 0, "")

				// --- 2. TEAMS & SETS HEADER ROW ---
				// Left: RUSIA   3 - 0   GRENADA
				// Right: 1st | 2nd | 3rd | 4th | 5th | Res. | Total
				pdf.SetFont("Arial", "B", 8)
				pdf.SetFillColor(245, 245, 245) // light gray for headers

				// We want the team matchup centered in the 103mm cell
				matchupHeaderStr := fmt.Sprintf("%s   %d - %d   %s", strings.ToUpper(teamAName), m.ScoreA(), m.ScoreB(), strings.ToUpper(teamBName))
				pdf.CellFormat(103, 6, tr(matchupHeaderStr), "1", 0, "C", true, 0, "")

				pdf.CellFormat(11, 6, "1st", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 6, "2nd", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 6, "3rd", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 6, "4th", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 6, "5th", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 6, "Res.", "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 6, "Total", "1", 1, "C", true, 0, "")

				// --- 3. SUB-MATCHES ---
				// Let's gather the sub-matches and sort them by RoundNumber
				var subMatches []tournamentDomain.Match
				for _, sub := range t.Matches {
					if sub.TeamMatchID != nil && *sub.TeamMatchID == m.ID {
						subMatches = append(subMatches, sub)
					}
				}
				sort.Slice(subMatches, func(i, j int) bool {
					return subMatches[i].RoundNumber < subMatches[j].RoundNumber
				})

				// Derive squad player IDs to map to A, B, C / X, Y, Z
				var squadAP1, squadAP2, squadAP3 string
				var squadBP1, squadBP2, squadBP3 string
				for _, sub := range subMatches {
					if len(sub.TeamA) == 0 || len(sub.TeamB) == 0 {
						continue
					}
					if t.TeamFormat == "olympic" || t.TeamFormat == "" {
						switch sub.RoundNumber {
						case 3:
							squadAP1 = sub.TeamA[0].ID
							squadBP1 = sub.TeamB[0].ID
						case 4:
							squadAP2 = sub.TeamA[0].ID
							squadBP2 = sub.TeamB[0].ID
						case 2:
							squadAP3 = sub.TeamA[0].ID
							squadBP3 = sub.TeamB[0].ID
						}
					} else {
						switch sub.RoundNumber {
						case 1:
							squadAP1 = sub.TeamA[0].ID
							squadBP1 = sub.TeamB[0].ID
						case 2:
							squadAP2 = sub.TeamA[0].ID
							squadBP2 = sub.TeamB[0].ID
						case 3:
							squadAP3 = sub.TeamA[0].ID
							squadBP3 = sub.TeamB[0].ID
						}
					}
				}

				getPlayerLetter := func(p *player.Player, isTeamB bool) string {
					if p == nil {
						return ""
					}
					if isTeamB {
						if squadBP1 != "" && p.ID == squadBP1 { return "X" }
						if squadBP2 != "" && p.ID == squadBP2 { return "Y" }
						if squadBP3 != "" && p.ID == squadBP3 { return "Z" }
						// Fallback to team players list order
						var actualTeam *tournamentDomain.Team
						for _, team := range t.Teams {
							if len(m.TeamB) > 0 && team.ID == m.TeamB[0].ID {
								actualTeam = team
								break
							}
						}
						if actualTeam != nil {
							for idx, tp := range actualTeam.Players {
								if tp.ID == p.ID {
									switch idx {
									case 0: return "X"
									case 1: return "Y"
									case 2: return "Z"
									}
								}
							}
						}
					} else {
						if squadAP1 != "" && p.ID == squadAP1 { return "A" }
						if squadAP2 != "" && p.ID == squadAP2 { return "B" }
						if squadAP3 != "" && p.ID == squadAP3 { return "C" }
						// Fallback to team players list order
						var actualTeam *tournamentDomain.Team
						for _, team := range t.Teams {
							if len(m.TeamA) > 0 && team.ID == m.TeamA[0].ID {
								actualTeam = team
								break
							}
						}
						if actualTeam != nil {
							for idx, tp := range actualTeam.Players {
								if tp.ID == p.ID {
									switch idx {
									case 0: return "A"
									case 1: return "B"
									case 2: return "C"
									}
								}
							}
						}
					}
					return ""
				}

				runningScoreA, runningScoreB := 0, 0

				for _, sub := range subMatches {
					isDoubles := sub.MatchType == "doubles"

					// Let's get Player Names, Alignment Letters, and Bib Numbers
					var pA1Name, pA2Name string
					var pB1Name, pB2Name string
					var pA1Let, pA2Let string
					var pB1Let, pB2Let string
					var pA1Num, pA2Num string
					var pB1Num, pB2Num string

					if len(sub.TeamA) > 0 {
						pA1Name = formatPlayerName(sub.TeamA[0])
						pA1Let = getPlayerLetter(sub.TeamA[0], false)
						if num, ok := playerNumberMap[sub.TeamA[0].ID]; ok {
							pA1Num = fmt.Sprintf("%d", num)
						}
						if len(sub.TeamA) > 1 {
							pA2Name = formatPlayerName(sub.TeamA[1])
							pA2Let = getPlayerLetter(sub.TeamA[1], false)
							if num, ok := playerNumberMap[sub.TeamA[1].ID]; ok {
								pA2Num = fmt.Sprintf("%d", num)
							}
						}
					}
					if len(sub.TeamB) > 0 {
						pB1Name = formatPlayerName(sub.TeamB[0])
						pB1Let = getPlayerLetter(sub.TeamB[0], true)
						if num, ok := playerNumberMap[sub.TeamB[0].ID]; ok {
							pB1Num = fmt.Sprintf("%d", num)
						}
						if len(sub.TeamB) > 1 {
							pB2Name = formatPlayerName(sub.TeamB[1])
							pB2Let = getPlayerLetter(sub.TeamB[1], true)
							if num, ok := playerNumberMap[sub.TeamB[1].ID]; ok {
								pB2Num = fmt.Sprintf("%d", num)
							}
						}
					}

					// Score calculation for this sub-match
					var setScores [5]string
					for i := 0; i < 5; i++ {
						setScores[i] = "-"
					}
					for _, set := range sub.Sets {
						if set.Number >= 1 && set.Number <= 5 {
							setScores[set.Number-1] = fmt.Sprintf("%d - %d", set.ScoreA, set.ScoreB)
						}
					}

					resStr := "0 - 0"
					if sub.Status == "finished" {
						resStr = fmt.Sprintf("%d - %d", sub.ScoreA(), sub.ScoreB())
						if sub.WinnerTeam == "A" {
							runningScoreA++
						} else if sub.WinnerTeam == "B" {
							runningScoreB++
						}
					} else if sub.Status == "in_progress" {
						resStr = tr("In Progress")
					}

					runningScoreStr := "-"
					if sub.Status == "finished" {
						runningScoreStr = fmt.Sprintf("%d - %d", runningScoreA, runningScoreB)
					}

					pdf.SetFont("Arial", "", 8)
					startX, startY := pdf.GetXY()

					if isDoubles {
						// Doubles Match: spans 2 rows of height 5.5mm each = 11mm total
						// Line 1
						pdf.CellFormat(8, 5.5, pA1Let, "LTR", 0, "C", false, 0, "")
						pdf.CellFormat(10, 5.5, pA1Num, "LTR", 0, "C", false, 0, "")
						pdf.CellFormat(33, 5.5, tr(truncateStr(pA1Name, 18)), "LTR", 0, "L", false, 0, "")
						pdf.CellFormat(2, 5.5, "", "LTR", 0, "C", false, 0, "")
						pdf.CellFormat(8, 5.5, pB1Let, "LTR", 0, "C", false, 0, "")
						pdf.CellFormat(10, 5.5, pB1Num, "LTR", 0, "C", false, 0, "")
						pdf.CellFormat(32, 5.5, tr(truncateStr(pB1Name, 18)), "LTR", 1, "L", false, 0, "")

						// Line 2
						pdf.SetX(startX)
						pdf.CellFormat(8, 5.5, pA2Let, "LBR", 0, "C", false, 0, "")
						pdf.CellFormat(10, 5.5, pA2Num, "LBR", 0, "C", false, 0, "")
						pdf.CellFormat(33, 5.5, tr(truncateStr(pA2Name, 18)), "LBR", 0, "L", false, 0, "")
						pdf.CellFormat(2, 5.5, "", "LBR", 0, "C", false, 0, "")
						pdf.CellFormat(8, 5.5, pB2Let, "LBR", 0, "C", false, 0, "")
						pdf.CellFormat(10, 5.5, pB2Num, "LBR", 0, "C", false, 0, "")
						pdf.CellFormat(32, 5.5, tr(truncateStr(pB2Name, 18)), "LBR", 1, "L", false, 0, "")

						// Set XY to right side columns at startY
						pdf.SetXY(startX+103, startY)
						for i := 0; i < 5; i++ {
							pdf.CellFormat(11, 11, setScores[i], "1", 0, "C", false, 0, "")
						}
						pdf.CellFormat(11, 11, resStr, "1", 0, "C", false, 0, "")
						pdf.SetFont("Arial", "B", 8)
						pdf.CellFormat(11, 11, runningScoreStr, "1", 1, "C", false, 0, "")
					} else {
						// Singles Match: height 8mm
						pdf.CellFormat(8, 8, pA1Let, "1", 0, "C", false, 0, "")
						pdf.CellFormat(10, 8, pA1Num, "1", 0, "C", false, 0, "")
						pdf.CellFormat(33, 8, tr(truncateStr(pA1Name, 18)), "1", 0, "L", false, 0, "")
						pdf.CellFormat(2, 8, "", "1", 0, "C", false, 0, "")
						pdf.CellFormat(8, 8, pB1Let, "1", 0, "C", false, 0, "")
						pdf.CellFormat(10, 8, pB1Num, "1", 0, "C", false, 0, "")
						pdf.CellFormat(32, 8, tr(truncateStr(pB1Name, 18)), "1", 0, "L", false, 0, "")

						for i := 0; i < 5; i++ {
							pdf.CellFormat(11, 8, setScores[i], "1", 0, "C", false, 0, "")
						}
						pdf.CellFormat(11, 8, resStr, "1", 0, "C", false, 0, "")
						pdf.SetFont("Arial", "B", 8)
						pdf.CellFormat(11, 8, runningScoreStr, "1", 1, "C", false, 0, "")
					}
				}
				pdf.Ln(6) // spacing between team ties
			}
		} else {
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
					var setScoreStr = "-"
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
				if m.TableNumber != nil {
					resStr += fmt.Sprintf(" (T. %d)", *m.TableNumber)
				}
				pdf.CellFormat(29, 8, resStr, "1", 1, "C", false, 0, "")
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

type pdfMatchSlot struct {
	Seed   int
	Player *player.Player
}

type pdfBracketMatchView struct {
	Player1 *pdfMatchSlot
	Player2 *pdfMatchSlot
	Match   *tournamentDomain.Match
	Stage   string
	BestOf  int
}

type pdfRoundView struct {
	Name    string
	Matches []pdfBracketMatchView
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p *= 2
	}
	return p
}

func getSeedingArrangement(size int) []int {
	rounds := int(math.Log2(float64(size)))
	if rounds == 0 {
		return []int{1}
	}
	bracket := []int{1, 2}
	for r := 2; r <= rounds; r++ {
		var newBracket []int
		sum := int(math.Pow(2, float64(r))) + 1
		for i, seed := range bracket {
			if i%2 == 0 {
				newBracket = append(newBracket, seed, sum-seed)
			} else {
				newBracket = append(newBracket, sum-seed, seed)
			}
		}
		bracket = newBracket
	}
	return bracket
}

func buildPdfBracketRounds(t *tournamentDomain.Tournament, players []*player.Player) []pdfRoundView {
	if len(players) == 0 {
		return nil
	}
	unresolvedSlot := &pdfMatchSlot{Seed: 0, Player: nil}
	size := nextPow2(len(players))
	if size < 2 {
		size = 2
	}
	arrangement := getSeedingArrangement(size)

	type Pair struct {
		P1 *pdfMatchSlot
		P2 *pdfMatchSlot
	}

	var current []Pair
	for i := 0; i < len(arrangement); i += 2 {
		s1 := arrangement[i] - 1
		s2 := -1
		if i+1 < len(arrangement) {
			s2 = arrangement[i+1] - 1
		}
		
		var p1, p2 *pdfMatchSlot
		if s1 >= 0 && s1 < len(players) {
			p1 = &pdfMatchSlot{Seed: s1 + 1, Player: players[s1]}
		}
		if s2 >= 0 && s2 < len(players) {
			p2 = &pdfMatchSlot{Seed: s2 + 1, Player: players[s2]}
		}
		current = append(current, Pair{P1: p1, P2: p2})
	}

	var rounds []pdfRoundView
	
	bestOfForStage := func(stage string) int {
		for _, r := range t.StageRules {
			if r.Stage == stage {
				return r.BestOf
			}
		}
		return 5
	}

	for len(current) > 1 {
		var next []Pair
		var rvMatches []pdfBracketMatchView
		
		stageNameCurrent := "r32"
		rem := len(current)
		if rem == 8 {
			stageNameCurrent = "r16"
		} else if rem == 4 {
			stageNameCurrent = "quarterfinal"
		} else if rem == 2 {
			stageNameCurrent = "semifinal"
		} else if rem == 1 {
			stageNameCurrent = "final"
		}
		
		for i := 0; i < len(current); i += 2 {
			mLeft := current[i]
			mRight := current[i+1]

			getWinner := func(m Pair) *pdfMatchSlot {
				if m.P1 == unresolvedSlot || m.P2 == unresolvedSlot {
					return unresolvedSlot
				}

				v1 := m.P1 != nil && m.P1.Player != nil
				v2 := m.P2 != nil && m.P2.Player != nil

				if !v1 && !v2 { return nil }
				if v1 && !v2 { return m.P1 }
				if !v1 && v2 { return m.P2 }

				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Status == "finished" && len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if tm.TeamA[0].ID == m.P1.Player.ID && tm.TeamB[0].ID == m.P2.Player.ID {
							if tm.WinnerTeam == "A" { return m.P1 } else { return m.P2 }
						}
						if tm.TeamA[0].ID == m.P2.Player.ID && tm.TeamB[0].ID == m.P1.Player.ID {
							if tm.WinnerTeam == "A" { return m.P2 } else { return m.P1 }
						}
					}
				}
				return unresolvedSlot
			}

			next = append(next, Pair{P1: getWinner(mLeft), P2: getWinner(mRight)})
		}

		for i := 0; i < len(current); i++ {
			p1 := current[i].P1
			p2 := current[i].P2
			var foundMatch *tournamentDomain.Match
			if p1 != nil && p2 != nil && p1.Player != nil && p2.Player != nil {
				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
							foundMatch = &t.Matches[k]
							break
						}
					}
				}
			}
			
			rvMatches = append(rvMatches, pdfBracketMatchView{
				Player1: p1,
				Player2: p2,
				Match: foundMatch,
				Stage: stageNameCurrent,
				BestOf: bestOfForStage(stageNameCurrent),
			})
		}

		name := fmt.Sprintf("Round of %d", len(current)*2)
		if len(current) == 4 { name = "Quarter-Finals" } else if len(current) == 2 { name = "Semi-Finals" } else if len(current) == 1 { name = "Final" }
		
		rounds = append(rounds, pdfRoundView{Name: name, Matches: rvMatches})
		
		current = next
	}

	if len(current) > 0 {
		var finalMatch *tournamentDomain.Match
		p1 := current[0].P1
		p2 := current[0].P2
		var champion *pdfMatchSlot

		bothFinalistsKnown := p1 != nil && p1.Player != nil && p2 != nil && p2.Player != nil

		if bothFinalistsKnown {
			for k := range t.Matches {
				tm := t.Matches[k]
				if tm.TeamMatchID != nil {
					continue
				}
				if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
					if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
						finalMatch = &t.Matches[k]
						if tm.Status == "finished" {
							if tm.WinnerTeam == "A" {
								if tm.TeamA[0].ID == p1.Player.ID { champion = p1 } else { champion = p2 }
							} else {
								if tm.TeamB[0].ID == p1.Player.ID { champion = p1 } else { champion = p2 }
							}
						}
						break
					}
				}
			}
		}

		rounds = append(rounds, pdfRoundView{
			Name: "🏆 Final",
			Matches: []pdfBracketMatchView{
				{
					Player1: p1,
					Player2: p2,
					Match: finalMatch,
					Stage: "final",
					BestOf: bestOfForStage("final"),
				},
			},
		})

		if champion != nil {
			rounds = append(rounds, pdfRoundView{
				Name: "Champion",
				Matches: []pdfBracketMatchView{
					{Player1: champion, Player2: nil},
				},
			})
		}
	}

	return rounds
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


