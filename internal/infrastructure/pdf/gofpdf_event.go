package pdf

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"

	"github.com/jung-kurt/gofpdf"
)

type GoFpdfGenerator struct{}

func NewGoFpdfGenerator() *GoFpdfGenerator {
	return &GoFpdfGenerator{}
}

func (g *GoFpdfGenerator) GenerateTournamentReport(t *event.Event, divs []*division.Division) ([]byte, error) {
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

		text := tr("TORNEO TENIS DE MESA - " + strings.ToUpper(t.Name))
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

	// Build Content
	BuildTournamentPdfContent(pdf, t, divs, tr)

	var buf bytes.Buffer
	err := pdf.Output(&buf)
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
	Match   *event.Match
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

func buildPdfBracketRounds(t *event.Event, players []*player.Player) []pdfRoundView {
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

				if !v1 && !v2 {
					return nil
				}
				if v1 && !v2 {
					return m.P1
				}
				if !v1 && v2 {
					return m.P2
				}

				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Stage != stageNameCurrent {
						continue
					}
					if tm.Status == "finished" && len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
						if tm.TeamA[0].ID == m.P1.Player.ID && tm.TeamB[0].ID == m.P2.Player.ID {
							if tm.WinnerTeam == "A" {
								return m.P1
							} else {
								return m.P2
							}
						}
						if tm.TeamA[0].ID == m.P2.Player.ID && tm.TeamB[0].ID == m.P1.Player.ID {
							if tm.WinnerTeam == "A" {
								return m.P2
							} else {
								return m.P1
							}
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
			var foundMatch *event.Match
			if p1 != nil && p2 != nil && p1.Player != nil && p2.Player != nil {
				for k := range t.Matches {
					tm := t.Matches[k]
					if tm.TeamMatchID != nil {
						continue
					}
					if tm.Stage != stageNameCurrent {
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
				Match:   foundMatch,
				Stage:   stageNameCurrent,
				BestOf:  bestOfForStage(stageNameCurrent),
			})
		}

		name := fmt.Sprintf("Round of %d", len(current)*2)
		if len(current) == 4 {
			name = "Quarter-Finals"
		} else if len(current) == 2 {
			name = "Semi-Finals"
		} else if len(current) == 1 {
			name = "Final"
		}

		rounds = append(rounds, pdfRoundView{Name: name, Matches: rvMatches})

		current = next
	}

	if len(current) > 0 {
		var finalMatch *event.Match
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
				if tm.Stage != "final" {
					continue
				}
				if len(tm.TeamA) > 0 && len(tm.TeamB) > 0 {
					if (tm.TeamA[0].ID == p1.Player.ID && tm.TeamB[0].ID == p2.Player.ID) || (tm.TeamA[0].ID == p2.Player.ID && tm.TeamB[0].ID == p1.Player.ID) {
						finalMatch = &t.Matches[k]
						if tm.Status == "finished" {
							if tm.WinnerTeam == "A" {
								if tm.TeamA[0].ID == p1.Player.ID {
									champion = p1
								} else {
									champion = p2
								}
							} else {
								if tm.TeamB[0].ID == p1.Player.ID {
									champion = p1
								} else {
									champion = p2
								}
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
					Match:   finalMatch,
					Stage:   "final",
					BestOf:  bestOfForStage("final"),
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

func BuildTournamentPdfContent(pdf *gofpdf.Fpdf, t *event.Event, divs []*division.Division, tr func(string) string) {
	pdf.AddPage()

	// Event Title Block
	pdf.SetFont("Arial", "B", 16)
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(0, 12, strings.ToUpper(t.Name), "1", 1, "C", true, 0, "")
	pdf.Ln(4)

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

	loc, _ := time.LoadLocation("America/Managua")
	formatTime := func(tVal time.Time, isDate bool) string {
		if loc != nil {
			tVal = tVal.In(loc)
		}
		if isDate {
			return tVal.Format("02-Jan")
		}
		return tVal.Format("15:04")
	}

	type divisionToCheck struct {
		ID      string
		Name    string
		Players []*player.Player
	}
	var divsToCheck []divisionToCheck

	if t.SkipElo || len(divs) == 0 {
		var pList []*player.Player
		if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
			for _, team := range t.Teams {
				avgElo := team.AverageElo(t.Type)
				pList = append(pList, &player.Player{
					ID:         team.ID,
					FirstName:  team.Name,
					LastName:   " (Team)",
					SinglesElo: avgElo,
					DoublesElo: avgElo,
				})
			}
		} else {
			pList = t.Participants
		}
		divsToCheck = append(divsToCheck, divisionToCheck{
			ID:      "",
			Name:    "Open Bracket",
			Players: pList,
		})
	} else {
		// Find players per division. We do snake-style/elo-range mapping like seeding.go
		assigned := make(map[string]bool)
		var units []*player.Player
		if t.Type == "teams" || t.Type == "doubles" || t.Type == "mixed_doubles" {
			for _, team := range t.Teams {
				avgElo := team.AverageElo(t.Type)
				units = append(units, &player.Player{
					ID:         team.ID,
					FirstName:  team.Name,
					LastName:   " (Team)",
					SinglesElo: avgElo,
					DoublesElo: avgElo,
				})
			}
		} else {
			units = make([]*player.Player, len(t.Participants))
			copy(units, t.Participants)
		}

		// Sort by Elo
		sort.Slice(units, func(i, j int) bool {
			if t.Type == "doubles" || t.Type == "mixed_doubles" {
				return units[i].DoublesElo > units[j].DoublesElo
			}
			return units[i].SinglesElo > units[j].SinglesElo
		})

		for _, d := range divs {
			if d.MinElo == 0 && d.MaxElo == nil {
				continue // Skip 'No Division'
			}
			if d.Category != "both" && d.Category != t.Type {
				continue
			}
			var dPlayers []*player.Player
			for _, p := range units {
				if assigned[p.ID] {
					continue
				}
				elo := p.SinglesElo
				if t.Type == "doubles" || t.Type == "mixed_doubles" {
					elo = p.DoublesElo
				}
				if elo >= d.MinElo && (d.MaxElo == nil || elo <= *d.MaxElo) {
					dPlayers = append(dPlayers, p)
					assigned[p.ID] = true
				}
			}
			if len(dPlayers) > 0 {
				name := d.Name
				if strings.HasSuffix(strings.ToLower(name), " division") {
					name = name[:len(name)-9]
				}
				divsToCheck = append(divsToCheck, divisionToCheck{
					ID:      d.ID,
					Name:    name,
					Players: dPlayers,
				})
			}
		}

		// Unclassified
		var unassigned []*player.Player
		for _, p := range units {
			if !assigned[p.ID] {
				unassigned = append(unassigned, p)
			}
		}
		if len(unassigned) > 0 {
			divsToCheck = append(divsToCheck, divisionToCheck{
				ID:      "",
				Name:    "Unclassified",
				Players: unassigned,
			})
		}
	}

	// 1. FINAL STANDINGS / PLACINGS
	if t.Status == "finished" {
		hasPlaces := false
		for _, dt := range divsToCheck {
			f, s, td := GetDivisionPlaces(t, dt.ID, dt.Players)
			if f != "" || s != "" || td != "" {
				hasPlaces = true
				break
			}
		}

		if hasPlaces {
			writeHeader("POSICIONES FINALES")
			for _, dt := range divsToCheck {
				first, second, third := GetDivisionPlaces(t, dt.ID, dt.Players)
				if first != "" || second != "" || third != "" {
					pdf.SetFillColor(245, 247, 250) // clean light grey background
					pdf.SetFont("Arial", "B", 10)
					pdf.CellFormat(0, 8, tr("  "+strings.ToUpper(dt.Name)), "1", 1, "L", true, 0, "")

					pdf.SetFont("Arial", "", 9)
					if first != "" {
						pdf.CellFormat(45, 7, tr("  1er Lugar (Campeón):"), "1", 0, "L", false, 0, "")
						pdf.SetFont("Arial", "B", 9)
						pdf.CellFormat(0, 7, tr("  "+strings.ToUpper(first)), "1", 1, "L", false, 0, "")
						pdf.SetFont("Arial", "", 9)
					}
					if second != "" {
						pdf.CellFormat(45, 7, tr("  2do Lugar:"), "1", 0, "L", false, 0, "")
						pdf.CellFormat(0, 7, tr("  "+strings.ToUpper(second)), "1", 1, "L", false, 0, "")
					}
					if third != "" {
						pdf.CellFormat(45, 7, tr("  3er Lugar:"), "1", 0, "L", false, 0, "")
						pdf.CellFormat(0, 7, tr("  "+strings.ToUpper(third)), "1", 1, "L", false, 0, "")
					}
					pdf.Ln(4)
				}
			}
		}
	}

	// 2. PARTICIPANTS LIST (SEPARATED BY DIVISION)
	hasParticipants := false
	for _, dt := range divsToCheck {
		if len(dt.Players) > 0 {
			hasParticipants = true
			break
		}
	}

	if hasParticipants {
		for _, dt := range divsToCheck {
			if len(dt.Players) > 0 {
				pdf.Ln(4)
				writeHeader(fmt.Sprintf("LISTA DE INSCRITOS - %s (%d JUGADORES)", strings.ToUpper(dt.Name), len(dt.Players)))

				pdf.SetFont("Arial", "B", 10)
				pdf.CellFormat(30, 8, "Elo", "1", 0, "C", false, 0, "")
				pdf.CellFormat(150, 8, tr("NOMBRE"), "1", 1, "C", false, 0, "")

				// Sort division players by Elo descending
				sort.Slice(dt.Players, func(i, j int) bool {
					eloI := dt.Players[i].SinglesElo
					eloJ := dt.Players[j].SinglesElo
					if t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams" {
						eloI = dt.Players[i].DoublesElo
						eloJ = dt.Players[j].DoublesElo
					}
					return eloI > eloJ
				})

				pdf.SetFont("Arial", "", 10)
				for _, p := range dt.Players {
					elo := p.SinglesElo
					if t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams" {
						elo = p.DoublesElo
					}
					fullName := p.FirstNameWithSecond()
					if strings.TrimSpace(p.LastName) != "(Team)" && strings.TrimSpace(p.LastName) != "" {
						fullName += " " + p.LastNameWithSecond()
					}
					pdf.CellFormat(30, 8, fmt.Sprintf("%d", elo), "1", 0, "C", false, 0, "")
					pdf.CellFormat(150, 8, tr(fullName), "1", 1, "L", false, 0, "")
				}
			}
		}
	}

	// 2.5 GROUP STANDINGS / POOLS
	if t.Format == "round_robin" || t.Format == "groups_elimination" {
		type groupStandings struct {
			DivisionName string
			GroupName    string
			Players      []*player.Player
			Standings    []event.PlayerStanding
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
				var gMatches []event.Match
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
				st := event.BuildStandings(g.Players, gMatches)
				groupStages = append(groupStages, groupStandings{
					DivisionName: divName,
					GroupName:    grpName,
					Players:      g.Players,
					Standings:    st,
				})
			}
		}

		if len(groupStages) > 0 {
			pdf.Ln(8)
			writeHeader("TABLAS DE POSICIONES DE GRUPOS")

			for _, gs := range groupStages {
				// Check if there is enough space on the page for this group table
				_, pageHeight := pdf.GetPageSize()
				_, _, _, bottomMargin := pdf.GetMargins()
				// Title (8) + Ln(2) = 10
				// Header row (5) + n rows (n*5) = (n+1)*5
				// Bottom margin padding (10)
				reqHeight := 10.0 + float64(len(gs.Players)+1)*5.0 + 10.0
				if pdf.GetY()+reqHeight > pageHeight-bottomMargin {
					pdf.AddPage()
				}

				pdf.SetFont("Arial", "B", 10)
				pdf.CellFormat(0, 8, tr(strings.ToUpper(gs.DivisionName)+" - "+strings.ToUpper(gs.GroupName)), "", 1, "L", false, 0, "")
				pdf.Ln(2)

				// Find matches in this group
				var gMatches []event.Match
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
				pdf.CellFormat(12, 5, tr("Día"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(11, 5, tr("Hora"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(9, 5, tr("Mesa"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(10, 5, tr("Part."), "1", 1, "C", true, 0, "")

				pdf.SetFont("Arial", "", 7)
				for i := 0; i < len(gs.Players); i++ {
					// Draw up to 6 matches or len(gMatches) rows
					if i < len(gMatches) {
						m := gMatches[i]
						dStr := formatTime(t.StartDate, true)
						tStr := "10:00"
						if m.UpdatedAt != nil {
							dStr = formatTime(*m.UpdatedAt, true)
							tStr = formatTime(*m.UpdatedAt, false)
						}
						tblStr := "-"
						if m.TableNumber != nil {
							tblStr = fmt.Sprintf("%d", *m.TableNumber)
						}

						idxA, idxB := -1, -1
						for idx, gp := range gs.Players {
							if gp.ID == m.TeamA[0].ID {
								idxA = idx + 1
							}
							if gp.ID == m.TeamB[0].ID {
								idxB = idx + 1
							}
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

				// --- PART B: Cross-Table Matrix ---
				n := len(gs.Players)
				pdf.SetXY(15+42+3, startY)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212)
				pdf.CellFormat(48, 5, tr("   ")+tr(strings.ToUpper(gs.GroupName)), "1", 0, "L", true, 0, "")
				for col := 1; col <= n; col++ {
					pdf.CellFormat(8, 5, fmt.Sprintf("%d", col), "1", 0, "C", true, 0, "")
				}
				pdf.Ln(5)

				for rowIdx, p1 := range gs.Players {
					pdf.SetX(15 + 42 + 3)
					// Draw player/team info cell
					startX, currY := pdf.GetXY()
					pdf.CellFormat(48, 5, "", "1", 0, "L", false, 0, "")

					// Draw custom colored texts inside the cell
					pdf.SetXY(startX+2, currY+1)
					pdf.SetFont("Arial", "B", 7)
					pdf.SetTextColor(30, 80, 220) // blue index
					pdf.Text(pdf.GetX(), pdf.GetY()+2.5, fmt.Sprintf("%d", rowIdx+1))

					pdf.SetX(startX + 6)
					pdf.SetTextColor(0, 0, 0) // black name
					pdf.Text(pdf.GetX(), pdf.GetY()+2.5, tr(truncateStr(formatPlayerName(p1), 21)))

					pdf.SetTextColor(0, 0, 0)
					pdf.SetXY(startX+48, currY)

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

				// --- PART C: Points & Positions ---
				pdf.SetXY(15+42+3+48+float64(n)*8+3, startY)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212)
				pdf.CellFormat(8, 5, tr("Pts"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(14, 5, tr("Sets"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(16, 5, tr("Puntos"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(8, 5, "Pos.", "1", 1, "C", true, 0, "")

				for _, p := range gs.Players {
					pdf.SetX(15 + 42 + 3 + 48 + float64(n)*8 + 3)

					var wins, losses, setsW, setsL, ptsW, ptsL int
					for _, std := range gs.Standings {
						if std.Player.ID == p.ID {
							wins = std.Wins
							losses = std.Losses
							setsW = std.SetsWon
							setsL = std.SetsLost
							ptsW = std.PointsWon
							ptsL = std.PointsLost
							break
						}
					}
					pts := wins*2 + losses
					posVal := standingMap[p.ID]
					setsStr := fmt.Sprintf("%d/%d", setsW, setsL)
					ptsStr := fmt.Sprintf("%d/%d", ptsW, ptsL)

					pdf.SetFont("Arial", "", 7)
					pdf.CellFormat(8, 5, fmt.Sprintf("%d", pts), "1", 0, "C", false, 0, "")
					pdf.CellFormat(14, 5, setsStr, "1", 0, "C", false, 0, "")
					pdf.CellFormat(16, 5, ptsStr, "1", 0, "C", false, 0, "")
					pdf.SetFont("Arial", "B", 7)
					pdf.CellFormat(8, 5, fmt.Sprintf("%d", posVal), "1", 1, "C", false, 0, "")
				}

				pdf.SetXY(15, startY+float64(n+1)*5+3)
				pdf.Ln(6)
			}
		}
	}

	// 3. GROUP STAGE AND KNOCKOUT TABLES
	var groupMatches []event.Match
	var drawMatches []event.Match
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

	// 3.5 VISUAL BRACKET DRAW
	if t.Format == "elimination" || t.Format == "groups_elimination" {
		type divisionBracket struct {
			Name   string
			Group  *event.Group
			Rounds []pdfRoundView
		}
		var brackets []divisionBracket

		for _, dt := range divsToCheck {
			// 1. Look for saved group
			var savedGroup *event.Group
			for i := range t.Groups {
				g := &t.Groups[i]
				if g.Name == dt.Name+" - Bracket Draw" || g.Name == dt.Name+" - Knockout Seeds" {
					savedGroup = g
					break
				}
			}

			var bracketPlayers []*player.Player
			var ok = false

			if savedGroup != nil {
				bracketPlayers = savedGroup.Players
				ok = true
			} else if t.Format == "groups_elimination" {
				// Try to calculate it dynamically
				// Find round robin groups for this division
				var divRRGroups []*event.Group
				for i := range t.Groups {
					g := &t.Groups[i]
					if strings.Contains(g.Name, "- Knockout Seeds") || strings.Contains(g.Name, " - Bracket Draw") {
						continue
					}
					belongsToDiv := false
					prefix := dt.Name + " - "
					if strings.HasPrefix(g.Name, prefix) {
						belongsToDiv = true
					} else if dt.Name == "Open Bracket" && (strings.HasPrefix(g.Name, "Group ") || strings.HasPrefix(g.Name, "Open Bracket - Group ")) {
						belongsToDiv = true
					} else {
						for _, gp := range g.Players {
							for _, dp := range dt.Players {
								if gp.ID == dp.ID {
									belongsToDiv = true
									break
								}
							}
							if belongsToDiv {
								break
							}
						}
					}
					if belongsToDiv {
						divRRGroups = append(divRRGroups, g)
					}
				}

				// Sort groups by name to keep ordering stable
				sort.Slice(divRRGroups, func(a, b int) bool {
					return divRRGroups[a].Name < divRRGroups[b].Name
				})

				if len(divRRGroups) > 0 {
					// Check if all groups are finished
					allFinished := true
					for _, rg := range divRRGroups {
						if !isGroupFinished(t, rg) {
							allFinished = false
							break
						}
					}

					if allFinished {
						bracketPlayers = getITTFKnockoutSeeds(t, dt.ID, dt.Name, dt.Players, divRRGroups)
						ok = true
					}
				}
			}

			if ok && len(bracketPlayers) > 0 {
				rounds := buildPdfBracketRounds(t, bracketPlayers)
				if len(rounds) > 0 {
					brackets = append(brackets, divisionBracket{
						Name:   dt.Name,
						Group:  savedGroup, // could be nil if virtual, but that's fine
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
				pdf.Text(textX, marginT-3, tr(round.Name))
			}
			pdf.SetTextColor(0, 0, 0)

			getBracketPlayerText := func(sp *pdfMatchSlot) string {
				if sp == nil || sp.Player == nil {
					return "BYE"
				}
				return strings.ToUpper(formatPlayerName(sp.Player))
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

						pdf.SetFont("Arial", "B", 6)
						pdf.SetTextColor(0, 0, 0)
						champName := tr("TBD")
						if m.Player1 != nil && m.Player1.Player != nil {
							champName = strings.ToUpper(formatPlayerName(m.Player1.Player))
						}

						// Print champion text
						pdf.SetTextColor(0, 0, 0) // black name
						pdf.Text(x+2, y+boxH/2+1, tr("🏆 "+truncateStr(champName, 22)))

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

					p1Name := getBracketPlayerText(m.Player1)
					p2Name := getBracketPlayerText(m.Player2)

					// Print Player 1 text
					pdf.SetFont("Arial", p1Bold, 6)
					pdf.SetTextColor(0, 0, 0) // black
					pdf.Text(x+2, y+4, tr(truncateStr(p1Name, 22)))

					// Print Player 2 text
					pdf.SetFont("Arial", p2Bold, 6)
					pdf.SetTextColor(0, 0, 0) // black
					pdf.Text(x+2, y+10, tr(truncateStr(p2Name, 22)))
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
								dStr := formatTime(t.StartDate, true)
								tStr := "16:00"
								if mForDetails.Match.UpdatedAt != nil {
									dStr = formatTime(*mForDetails.Match.UpdatedAt, true)
									tStr = formatTime(*mForDetails.Match.UpdatedAt, false)
								}
								tblStr := ""
								if mForDetails.Match.TableNumber != nil {
									tblStr = fmt.Sprintf(" - Table %d", *mForDetails.Match.TableNumber)
								}
								matchDetails := fmt.Sprintf("%s - %sh%s", dStr, tStr, tblStr)

								pdf.SetFont("Arial", "", 5)
								pdf.SetTextColor(30, 80, 220) // blue
								pdf.Text(lineX1+1, currentMidY-1, tr(matchDetails))

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
	}

	// 5. EVENT METRICS
	if t.Status == "finished" && t.Metrics != nil {
		pdf.AddPageFormat("P", gofpdf.SizeType{Wd: 210, Ht: 297})
		pdf.SetMargins(15, 52, 15)

		writeHeader("ESTADÍSTICAS DEL TORNEO")

		pdf.SetFont("Arial", "", 10)
		pdf.SetFillColor(245, 247, 250)

		// Create a grid for metrics
		// Row 1
		pdf.CellFormat(60, 8, tr("Total Partidos: ")+fmt.Sprintf("%d", t.Metrics.TotalMatchesPlayed), "1", 0, "L", true, 0, "")
		pdf.CellFormat(60, 8, tr("Total Sets: ")+fmt.Sprintf("%d", t.Metrics.TotalSetsPlayed), "1", 0, "L", true, 0, "")
		pdf.CellFormat(60, 8, tr("Total Puntos: ")+fmt.Sprintf("%d", t.Metrics.TotalPointsScored), "1", 1, "L", true, 0, "")

		// Row 2
		pdf.CellFormat(60, 8, tr("Prom. Puntos/Partido: ")+fmt.Sprintf("%.1f", t.Metrics.AveragePointsPerMatch), "1", 0, "L", false, 0, "")
		pdf.CellFormat(60, 8, tr("Prom. Sets/Partido: ")+fmt.Sprintf("%.1f", t.Metrics.AverageSetsPerMatch), "1", 0, "L", false, 0, "")
		pdf.CellFormat(60, 8, tr("Barridas: ")+fmt.Sprintf("%d", t.Metrics.CleanSweeps), "1", 1, "L", false, 0, "")

		// Row 3
		pdf.CellFormat(90, 8, tr("Sets Decisivos: ")+fmt.Sprintf("%d", t.Metrics.DecidingSets), "1", 0, "L", true, 0, "")
		pdf.CellFormat(90, 8, tr("Prom. Elo Inicial: ")+fmt.Sprintf("%.1f", t.Metrics.AverageEloAtStart), "1", 1, "L", true, 0, "")

		// Division Metrics
		if len(t.Metrics.DivisionMetrics) > 0 {
			pdf.Ln(4)
			pdf.SetFont("Arial", "B", 9)
			pdf.CellFormat(0, 8, tr("Métricas por División"), "", 1, "L", false, 0, "")

			pdf.SetFont("Arial", "B", 8)
			pdf.SetFillColor(245, 247, 250)
			pdf.CellFormat(60, 6, tr("División"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(40, 6, tr("Partidos Jugados"), "1", 0, "C", true, 0, "")
			pdf.CellFormat(40, 6, tr("Prom. Puntos"), "1", 1, "C", true, 0, "")

			pdf.SetFont("Arial", "", 8)
			for divID, dm := range t.Metrics.DivisionMetrics {
				divName := divID
				if divID == "default" {
					divName = "Open"
				} else {
					for _, d := range divs {
						if d.ID == divID {
							divName = d.Name
							break
						}
					}
				}
				pdf.CellFormat(60, 6, tr(strings.ToUpper(divName)), "1", 0, "L", false, 0, "")
				pdf.CellFormat(40, 6, fmt.Sprintf("%d", dm.TotalMatchesPlayed), "1", 0, "C", false, 0, "")
				pdf.CellFormat(40, 6, fmt.Sprintf("%.1f", dm.AveragePointsPerMatch), "1", 1, "C", false, 0, "")
			}
		}
	}
}

type GroupStanding struct {
	Players   []*player.Player
	Standings []event.PlayerStanding
}

func isGroupFinished(t *event.Event, g *event.Group) bool {
	expectedMatches := len(g.Players) * (len(g.Players) - 1) / 2
	finished := 0
	for _, m := range t.Matches {
		if m.TeamMatchID != nil {
			continue
		}
		if m.Stage != "group" {
			continue
		}
		if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
			continue
		}
		p1InGroup, p2InGroup := false, false
		for _, p := range g.Players {
			if m.TeamA[0].ID == p.ID {
				p1InGroup = true
			}
			if m.TeamB[0].ID == p.ID {
				p2InGroup = true
			}
		}
		if p1InGroup && p2InGroup {
			if m.Status == "finished" {
				finished++
			}
		}
	}
	return expectedMatches > 0 && finished >= expectedMatches
}

func getITTFKnockoutSeeds(t *event.Event, divID, divName string, players []*player.Player, divRRGroups []*event.Group) []*player.Player {
	passCount := t.GetGroupPassCount(divID)
	if passCount == 0 {
		passCount = 2
	}

	var groupsStandings []GroupStanding
	for _, g := range divRRGroups {
		var rgMatches []event.Match
		for _, m := range t.Matches {
			if m.TeamMatchID != nil {
				continue
			}
			if m.Stage != "group" {
				continue
			}
			if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
				continue
			}
			p1InGroup, p2InGroup := false, false
			for _, gp := range g.Players {
				if gp.ID == m.TeamA[0].ID {
					p1InGroup = true
				}
				if gp.ID == m.TeamB[0].ID {
					p2InGroup = true
				}
			}
			if p1InGroup && p2InGroup {
				rgMatches = append(rgMatches, m)
			}
		}
		st := event.BuildStandings(g.Players, rgMatches)
		groupsStandings = append(groupsStandings, GroupStanding{
			Players:   g.Players,
			Standings: st,
		})
	}

	numGroups := len(groupsStandings)
	totalAdvancing := 0
	for _, g := range groupsStandings {
		take := int(passCount)
		if take > len(g.Standings) {
			take = len(g.Standings)
		}
		totalAdvancing += take
	}
	if totalAdvancing == 0 {
		return nil
	}

	bracketSize := nextPow2(totalAdvancing)
	arrangement := getSeedingArrangement(bracketSize)

	halfSize := len(arrangement) / 2
	topHalfSeeds := make(map[int]bool, halfSize)
	for _, s := range arrangement[:halfSize] {
		topHalfSeeds[s] = true
	}

	result := make([]*player.Player, totalAdvancing)

	winnerInTop := make([]bool, numGroups)
	for gi, g := range groupsStandings {
		if len(g.Standings) == 0 {
			continue
		}
		result[gi] = g.Standings[0].Player
		winnerInTop[gi] = topHalfSeeds[gi+1]
	}

	nextSlot := numGroups

	for layer := 1; layer < int(passCount); layer++ {
		layerSize := numGroups
		var topSlots, bottomSlots []int
		for i := nextSlot; i < nextSlot+layerSize && i < totalAdvancing; i++ {
			seedNum := i + 1
			if topHalfSeeds[seedNum] {
				topSlots = append(topSlots, i)
			} else {
				bottomSlots = append(bottomSlots, i)
			}
		}

		tsi, bsi := 0, 0
		for gi, g := range groupsStandings {
			if layer >= len(g.Standings) {
				continue
			}
			p := g.Standings[layer].Player
			if winnerInTop[gi] {
				if bsi < len(bottomSlots) {
					result[bottomSlots[bsi]] = p
					bsi++
				} else if tsi < len(topSlots) {
					result[topSlots[tsi]] = p
					tsi++
				}
			} else {
				if tsi < len(topSlots) {
					result[topSlots[tsi]] = p
					tsi++
				} else if bsi < len(bottomSlots) {
					result[bottomSlots[bsi]] = p
					bsi++
				}
			}
		}

		nextSlot += layerSize
	}

	var out []*player.Player
	for _, p := range result {
		if p != nil {
			out = append(out, p)
		}
	}
	return out
}

func GetDivisionPlaces(t *event.Event, divisionID string, divisionPlayers []*player.Player) (first, second, third string) {
	if t.Status != "finished" {
		return "", "", ""
	}

	if t.Format == "elimination" || t.Format == "groups_elimination" {
		// 1st and 2nd Place: Final Match for this division
		var finalMatch *event.Match
		for i := range t.Matches {
			m := &t.Matches[i]
			if m.Stage == "final" && m.Status == "finished" && m.TeamMatchID == nil && m.DivisionID == divisionID {
				finalMatch = m
				break
			}
		}
		if finalMatch != nil && finalMatch.WinnerTeam != "" {
			if finalMatch.WinnerTeam == "A" {
				first = event.GetTeamDisplayName(finalMatch.TeamA, t.Type)
				second = event.GetTeamDisplayName(finalMatch.TeamB, t.Type)
			} else {
				first = event.GetTeamDisplayName(finalMatch.TeamB, t.Type)
				second = event.GetTeamDisplayName(finalMatch.TeamA, t.Type)
			}
		}

		// 3rd Place: Semifinal losers for this division
		var semiLosers []string
		for i := range t.Matches {
			m := &t.Matches[i]
			if m.Stage == "semifinal" && m.Status == "finished" && m.TeamMatchID == nil && m.DivisionID == divisionID {
				if m.WinnerTeam == "A" {
					semiLosers = append(semiLosers, event.GetTeamDisplayName(m.TeamB, t.Type))
				} else if m.WinnerTeam == "B" {
					semiLosers = append(semiLosers, event.GetTeamDisplayName(m.TeamA, t.Type))
				}
			}
		}
		if len(semiLosers) > 0 {
			third = strings.Join(semiLosers, " & ")
		}

	} else if t.Format == "round_robin" {
		if len(divisionPlayers) > 0 {
			// Find matches for this division
			var divMatches []event.Match
			for _, m := range t.Matches {
				if m.DivisionID == divisionID && m.TeamMatchID == nil {
					divMatches = append(divMatches, m)
				}
			}
			standings := event.BuildStandings(divisionPlayers, divMatches)
			if len(standings) > 0 {
				first = event.GetTeamDisplayName([]*player.Player{standings[0].Player}, t.Type)
			}
			if len(standings) > 1 {
				second = event.GetTeamDisplayName([]*player.Player{standings[1].Player}, t.Type)
			}
			if len(standings) > 2 {
				third = event.GetTeamDisplayName([]*player.Player{standings[2].Player}, t.Type)
			}
		}
	}

	return first, second, third
}
