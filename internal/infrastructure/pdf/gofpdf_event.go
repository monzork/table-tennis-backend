package pdf

import (
	"bytes"
	"context"
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

	"github.com/go-pdf/fpdf"
)

// PhotoDownloader fetches an uploaded file's raw bytes by its storage object
// path. Satisfied by internal/infrastructure/storage.SupabaseStorage.
type PhotoDownloader interface {
	Download(ctx context.Context, path string) ([]byte, error)
}

type GoFpdfGenerator struct {
	photoDownloader PhotoDownloader
}

func NewGoFpdfGenerator() *GoFpdfGenerator {
	return &GoFpdfGenerator{}
}

// WithPhotoDownloader enables appending each player's cédula de identidad
// photos to the tournament report. Without it (e.g. Supabase isn't
// configured), reports generate exactly as before, just without that
// section — same optional-wiring pattern as PlayerHandler.WithUploader.
func (g *GoFpdfGenerator) WithPhotoDownloader(d PhotoDownloader) *GoFpdfGenerator {
	g.photoDownloader = d
	return g
}

func (g *GoFpdfGenerator) GenerateTournamentReport(t *event.Event, divs []*division.Division) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
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
	// BracketPos is the slot's 0-based leaf position in the canonical
	// seeding arrangement (getSeedingArrangement) at the bracket's initial
	// size -- see domain/bracket.MatchSlot.BracketPos. Sorting a round's
	// pairs by ascending BracketPos recovers true bracket-tree adjacency
	// independent of the arbitrary order real match rows come back from the
	// DB in.
	BracketPos int
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
	// NextIndex[j] is the index into the NEXT round's Matches that this
	// round's match j structurally feeds into. It is NOT always j/2:
	// groupPdfSlotsByRealMatches pairs winners by which real next-stage
	// match they actually played, which can pair non-adjacent slots (e.g.
	// the winner of match 0 and the winner of match 3), so the geometry
	// that draws connector lines and centers boxes must follow this
	// mapping rather than assume positional adjacency.
	NextIndex []int
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

type pdfBracketPair struct {
	P1 *pdfMatchSlot
	P2 *pdfMatchSlot
}

// pdfEliminationStageOrder lists knockout stage names from the earliest
// round to the final. The PDF report never renders tiered/losers brackets,
// so unlike its domain/bracket counterpart this has no tier prefix.
func pdfEliminationStageOrder() []string {
	return []string{"r32", "r16", "quarterfinal", "semifinal", "final"}
}

// matchesAtPdfStage returns every real (non-team-sub-match) match at the
// given stage, in t.Matches order, regardless of DivisionID -- see
// domain/bracket.matchesAtStage for why DivisionID can't be trusted here.
// Callers that need to scope to one division's real roster should filter
// by player identity via matchesAtPdfStageForRoster instead.
func matchesAtPdfStage(t *event.Event, stage string) []*event.Match {
	var out []*event.Match
	for i := range t.Matches {
		m := &t.Matches[i]
		if m.TeamMatchID != nil || m.Stage != stage {
			continue
		}
		if len(m.TeamA) == 0 || len(m.TeamB) == 0 {
			continue
		}
		out = append(out, m)
	}
	return out
}

// matchesAtPdfStageForRoster is matchesAtPdfStage further restricted to
// matches whose both sides are members of roster -- see
// domain/bracket.matchesAtStageForRoster.
func matchesAtPdfStageForRoster(t *event.Event, stage string, roster map[string]bool) []*event.Match {
	var out []*event.Match
	for _, m := range matchesAtPdfStage(t, stage) {
		if roster[m.TeamA[0].ID] && roster[m.TeamB[0].ID] {
			out = append(out, m)
		}
	}
	return out
}

// pdfStageNameForCount maps a round's pair count to its stage name, the same
// convention buildPdfBracketRounds's stageNameCurrent uses.
func pdfStageNameForCount(pairCount int) string {
	switch pairCount {
	case 8:
		return "r16"
	case 4:
		return "quarterfinal"
	case 2:
		return "semifinal"
	case 1:
		return "final"
	default:
		return "r32"
	}
}

// groupPdfSlotsByRealMatches groups a round's resolved winner slots two-at-
// a-time using the next round's actual recorded matches, instead of pairing
// them by position -- see domain/bracket.groupSlotsByRealMatches for why.
// The returned parentIndex is parallel to slots: parentIndex[i] is the index
// into the returned pairs slice that slots[i] ended up in. Because pairing
// follows real results rather than position, a pair can combine two
// non-adjacent slots (e.g. slots[0] actually played slots[3]) -- callers
// that draw bracket geometry must use parentIndex rather than assume
// slots[2*j]/slots[2*j+1] feed pairs[j].
func groupPdfSlotsByRealMatches(slots []*pdfMatchSlot, nextMatches []*event.Match) (pairs []pdfBracketPair, parentIndex []int) {
	idxByPlayer := make(map[string]int, len(slots))
	for i, s := range slots {
		if s != nil && s.Player != nil {
			if _, exists := idxByPlayer[s.Player.ID]; !exists {
				idxByPlayer[s.Player.ID] = i
			}
		}
	}

	// Each pair is tracked alongside the lower of its two slot indices (as a
	// tie-break for the sort below) and which slot indices it consumed (so
	// parentIndex can be rebuilt from the pair's post-sort position).
	type pdfIndexedPair struct {
		pair     pdfBracketPair
		minIdx   int
		slotIdxs []int
	}
	used := make([]bool, len(slots))
	var indexed []pdfIndexedPair
	for _, m := range nextMatches {
		idxA, okA := idxByPlayer[m.TeamA[0].ID]
		idxB, okB := idxByPlayer[m.TeamB[0].ID]
		if !okA || !okB || idxA == idxB || used[idxA] || used[idxB] {
			continue
		}
		used[idxA], used[idxB] = true, true
		minIdx := idxA
		if idxB < minIdx {
			minIdx = idxB
		}
		indexed = append(indexed, pdfIndexedPair{
			pair:     pdfBracketPair{P1: slots[idxA], P2: slots[idxB]},
			minIdx:   minIdx,
			slotIdxs: []int{idxA, idxB},
		})
	}

	var leftover []int
	for i := range slots {
		if !used[i] {
			leftover = append(leftover, i)
		}
	}
	for i := 0; i < len(leftover); i += 2 {
		idxA := leftover[i]
		var p2 *pdfMatchSlot
		slotIdxs := []int{idxA}
		if i+1 < len(leftover) {
			idxB := leftover[i+1]
			p2 = slots[idxB]
			slotIdxs = append(slotIdxs, idxB)
		}
		indexed = append(indexed, pdfIndexedPair{
			pair:     pdfBracketPair{P1: slots[idxA], P2: p2},
			minIdx:   idxA,
			slotIdxs: slotIdxs,
		})
	}

	// Primary key: BracketPos, which restores true bracket-tree adjacency
	// regardless of arrival order -- this is what fixes a pair actually
	// combining non-adjacent quarters/halves (see pdfMatchSlot.BracketPos).
	// Secondary key: minIdx, a fallback for slots with no (or tied)
	// BracketPos that keeps the prior top-to-bottom rendering order.
	sort.SliceStable(indexed, func(i, j int) bool {
		bi, bj := pdfPairBracketPos(indexed[i].pair), pdfPairBracketPos(indexed[j].pair)
		if bi != bj {
			return bi < bj
		}
		return indexed[i].minIdx < indexed[j].minIdx
	})

	pairs = make([]pdfBracketPair, len(indexed))
	parentIndex = make([]int, len(slots))
	for i := range parentIndex {
		parentIndex[i] = -1
	}
	for newPos, ip := range indexed {
		pairs[newPos] = ip.pair
		for _, si := range ip.slotIdxs {
			parentIndex[si] = newPos
		}
	}
	return pairs, parentIndex
}

// pdfPairBracketPos returns pr's lower BracketPos, or a large sentinel when
// neither slot has a resolved player, so ties fall back to minIdx instead of
// being reshuffled -- see domain/bracket.pairBracketPos.
func pdfPairBracketPos(pr pdfBracketPair) int {
	const unresolved = 1 << 30
	a, b := unresolved, unresolved
	if pr.P1 != nil && pr.P1.Player != nil {
		a = pr.P1.BracketPos
	}
	if pr.P2 != nil && pr.P2.Player != nil {
		b = pr.P2.BracketPos
	}
	if a < b {
		return a
	}
	return b
}

// sortPdfPairsByBracketPos restores true bracket-tree adjacency order
// across pairs gathered from real match data in arbitrary (DB row) order --
// see domain/bracket.sortPairsByBracketPos.
func sortPdfPairsByBracketPos(pairs []pdfBracketPair) {
	sort.SliceStable(pairs, func(i, j int) bool {
		return pdfPairBracketPos(pairs[i]) < pdfPairBracketPos(pairs[j])
	})
}

// firstRoundPdfPairsFromRealMatches reconstructs round 1's pairing directly
// from recorded matches instead of a hypothetical seeded arrangement, for a
// division whose knockout bracket has actually started/finished but whose
// seed order (from a saved "Bracket Draw"/"Knockout Seeds" group, or
// recomputed via getITTFKnockoutSeeds when there's no saved group) doesn't
// reproduce the pairing actually played -- recomputing from *current* group
// standings/GroupPassCount can silently diverge from whatever seeding was
// used at draw time. A seeded-arrangement pairing that doesn't match reality
// makes every later round's "does a real match exist between these two
// players" lookup fail, so the whole bracket beyond round 1 renders as
// unresolved/BYE even though it's fully played with a real champion. See the
// identical fix in domain/bracket.firstRoundPairsFromRealMatches.
//
// Returns nil (caller falls back to the projected/preview pairing) when
// there's no real match yet for this division at any elimination stage --
// nothing has actually been drawn/played, so a preview is exactly what
// should be shown.
func firstRoundPdfPairsFromRealMatches(t *event.Event, divID string, players []*player.Player) []pdfBracketPair {
	stages := pdfEliminationStageOrder()

	roster := make(map[string]bool, len(players))
	for _, p := range players {
		if p != nil {
			roster[p.ID] = true
		}
	}

	firstIdx := -1
	var firstStageMatches []*event.Match
	for i, stage := range stages {
		if ms := matchesAtPdfStageForRoster(t, stage, roster); len(ms) > 0 {
			firstIdx, firstStageMatches = i, ms
			break
		}
	}
	if firstStageMatches == nil {
		return nil
	}

	seedOf := make(map[string]int, len(players))
	for i, p := range players {
		if p != nil {
			seedOf[p.ID] = i + 1
		}
	}

	// posOfSeed maps a seed to its 0-based leaf position in the canonical
	// arrangement, so real-match providers can be given a true BracketPos
	// even though they're discovered in arbitrary DB order -- see
	// domain/bracket.firstRoundPairsFromRealMatches's identical helper.
	arrangementSize := nextPow2(len(players))
	if arrangementSize < 2 {
		arrangementSize = 2
	}
	posOfSeed := make(map[int]int, arrangementSize)
	for i, seed := range getSeedingArrangement(arrangementSize) {
		posOfSeed[seed] = i
	}
	bracketPosFor := func(playerID string, fallbackIdx int) int {
		if seed, ok := seedOf[playerID]; ok {
			if pos, ok2 := posOfSeed[seed]; ok2 {
				return pos
			}
		}
		return arrangementSize + fallbackIdx
	}

	minSeed := func(m *event.Match) int {
		a, b := seedOf[m.TeamA[0].ID], seedOf[m.TeamB[0].ID]
		if a == 0 {
			a = len(players) + 1
		}
		if b == 0 {
			b = len(players) + 1
		}
		if a < b {
			return a
		}
		return b
	}
	sort.SliceStable(firstStageMatches, func(i, j int) bool {
		si, sj := minSeed(firstStageMatches[i]), minSeed(firstStageMatches[j])
		if si != sj {
			return si < sj
		}
		return firstStageMatches[i].ID < firstStageMatches[j].ID
	})

	// Each "provider" is one round-1 slot that will resolve to a single
	// winner feeding round 2: either a real round-1 pair, or a solo bye
	// (a player who skipped round 1 entirely -- e.g. an odd round-1 match
	// count for this division). idxByPlayer maps every player who has a
	// provider to that provider's index, for the round-2-based grouping
	// below.
	providers := make([]pdfBracketPair, 0, len(firstStageMatches))
	idxByPlayer := make(map[string]int, len(firstStageMatches)*2)
	for _, m := range firstStageMatches {
		// A stale/duplicate match row (bad DivisionID tagging, forfeit
		// correction leftovers, etc.) can feature a player who already has a
		// provider from an earlier row in this loop -- see
		// domain/bracket.firstRoundPairsFromRealMatches's identical guard.
		if _, ok := idxByPlayer[m.TeamA[0].ID]; ok {
			continue
		}
		if _, ok := idxByPlayer[m.TeamB[0].ID]; ok {
			continue
		}
		idx := len(providers)
		providers = append(providers, pdfBracketPair{
			P1: &pdfMatchSlot{Seed: seedOf[m.TeamA[0].ID], Player: m.TeamA[0], BracketPos: bracketPosFor(m.TeamA[0].ID, idx*2)},
			P2: &pdfMatchSlot{Seed: seedOf[m.TeamB[0].ID], Player: m.TeamB[0], BracketPos: bracketPosFor(m.TeamB[0].ID, idx*2+1)},
		})
		idxByPlayer[m.TeamA[0].ID] = idx
		idxByPlayer[m.TeamB[0].ID] = idx
	}
	var nextStageMatches []*event.Match
	if firstIdx+1 < len(stages) {
		nextStageMatches = matchesAtPdfStageForRoster(t, stages[firstIdx+1], roster)
		for _, m := range nextStageMatches {
			for _, side := range [2]*player.Player{m.TeamA[0], m.TeamB[0]} {
				if _, ok := idxByPlayer[side.ID]; !ok {
					idxByPlayer[side.ID] = len(providers)
					providers = append(providers, pdfBracketPair{P1: &pdfMatchSlot{Seed: seedOf[side.ID], Player: side, BracketPos: bracketPosFor(side.ID, len(providers))}})
				}
			}
		}
	}

	// Group providers two-at-a-time by round 2's real matches -- whichever
	// two providers actually played each other next, per the recorded
	// data, rather than by position -- since a bye's actual round-2
	// opponent isn't necessarily adjacent in providers/minSeed order.
	used := make([]bool, len(providers))
	pairs := make([]pdfBracketPair, 0, len(providers))
	for _, m := range nextStageMatches {
		idxA, okA := idxByPlayer[m.TeamA[0].ID]
		idxB, okB := idxByPlayer[m.TeamB[0].ID]
		if !okA || !okB || idxA == idxB || used[idxA] || used[idxB] {
			continue
		}
		used[idxA], used[idxB] = true, true
		pairs = append(pairs, providers[idxA], providers[idxB])
	}
	// Providers with no (yet-played) round-2 match -- round 2 hasn't
	// happened yet for them -- fall back to sequential minSeed-order
	// pairing among themselves, same as when there's no round-2 data at
	// all.
	for i, pr := range providers {
		if !used[i] {
			pairs = append(pairs, pr)
		}
	}
	if len(pairs) > 1 && len(pairs)%2 != 0 {
		// Defensive: should be unreachable (an odd leftover count would
		// mean a player has no recorded opponent anywhere), but pad
		// rather than let the adjacent-pair indexing in the caller panic.
		pairs = append(pairs, pdfBracketPair{})
	}

	sortPdfPairsByBracketPos(pairs)
	return pairs
}

func buildPdfBracketRounds(t *event.Event, divID string, players []*player.Player) []pdfRoundView {
	if len(players) == 0 {
		return nil
	}
	unresolvedSlot := &pdfMatchSlot{Seed: 0, Player: nil}

	roster := make(map[string]bool, len(players))
	for _, p := range players {
		if p != nil {
			roster[p.ID] = true
		}
	}

	current := firstRoundPdfPairsFromRealMatches(t, divID, players)
	if current == nil {
		size := nextPow2(len(players))
		if size < 2 {
			size = 2
		}
		arrangement := getSeedingArrangement(size)

		for i := 0; i < len(arrangement); i += 2 {
			s1 := arrangement[i] - 1
			s2 := -1
			if i+1 < len(arrangement) {
				s2 = arrangement[i+1] - 1
			}

			var p1, p2 *pdfMatchSlot
			if s1 >= 0 && s1 < len(players) {
				p1 = &pdfMatchSlot{Seed: s1 + 1, Player: players[s1], BracketPos: i}
			}
			if s2 >= 0 && s2 < len(players) {
				p2 = &pdfMatchSlot{Seed: s2 + 1, Player: players[s2], BracketPos: i + 1}
			}
			current = append(current, pdfBracketPair{P1: p1, P2: p2})
		}
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
		var rvMatches []pdfBracketMatchView

		rem := len(current)
		stageNameCurrent := pdfStageNameForCount(rem)

		getWinner := func(m pdfBracketPair) *pdfMatchSlot {
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

		winners := make([]*pdfMatchSlot, len(current))
		for i, pr := range current {
			winners[i] = getWinner(pr)
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

		// Group this round's winners into the next round's pairs using the
		// next stage's actual recorded matches -- see
		// groupPdfSlotsByRealMatches / firstRoundPdfPairsFromRealMatches.
		// nextIndex is captured before the round is appended so the round's
		// own connector-line geometry can follow the real pairing instead of
		// assuming winner 2*j/2*j+1 feeds pairs[j].
		nextStageName := pdfStageNameForCount(len(current) / 2)
		var nextIndex []int
		current, nextIndex = groupPdfSlotsByRealMatches(winners, matchesAtPdfStageForRoster(t, nextStageName, roster))

		rounds = append(rounds, pdfRoundView{Name: name, Matches: rvMatches, NextIndex: nextIndex})
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
			// The Final always feeds the single Champion box (index 0) when
			// one is appended below.
			NextIndex: []int{0},
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

func BuildTournamentPdfContent(pdf *fpdf.Fpdf, t *event.Event, divs []*division.Division, tr func(string) string) {
	pdf.AddPage()

	// Event Title Block -- shrink the font to fit long event names instead of
	// letting them overflow the bordered cell/page width at a fixed 16pt,
	// same auto-shrink pattern the page header uses.
	titleText := strings.ToUpper(t.Name)
	titleFontSize := 16.0
	pdf.SetFont("Arial", "B", titleFontSize)
	w, _ := pdf.GetPageSize()
	maxTitleWidth := w - 15 - 15 - 4 // page width minus L/R margins, minus cell padding
	for pdf.GetStringWidth(titleText) > maxTitleWidth && titleFontSize > 8.0 {
		titleFontSize -= 0.5
		pdf.SetFont("Arial", "B", titleFontSize)
	}
	pdf.SetFillColor(240, 240, 240)
	pdf.CellFormat(0, 12, titleText, "1", 1, "C", true, 0, "")
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

			// Large round-robin groups (many teams -> many cross-table columns) don't
			// fit the schedule + matrix + points layout within an A4 portrait page's
			// printable width (~180mm: 210 - 15 left margin - 15 right margin). Switch
			// those specific groups to a landscape page (~267mm printable, fits up to
			// ~15 teams at the normal column width) and, for groups too large even for
			// that, shrink the matrix column width (and its font) so the whole table -
			// including the trailing Sets/Points/Pos columns - still lands on one page
			// instead of running off the right edge.
			curOrientation := "P"
			const (
				portraitPrintableWidth  = 180.0
				landscapePrintableWidth = 267.0
				// Schedule (42) + gap (3) + matrix name col (48) + gap (3) + points cols (8+14+16+8=46)
				fixedTableWidth = 142.0
				defaultColWidth = 8.0
				minColWidth     = 5.0
			)

			for _, gs := range groupStages {
				n := len(gs.Players)
				colW := defaultColWidth
				wantOrientation := "P"
				if fixedTableWidth+float64(n)*colW > portraitPrintableWidth {
					wantOrientation = "L"
				}
				availWidth := portraitPrintableWidth
				if wantOrientation == "L" {
					availWidth = landscapePrintableWidth
				}
				if fixedTableWidth+float64(n)*colW > availWidth {
					colW = (availWidth - fixedTableWidth) / float64(n)
					if colW < minColWidth {
						colW = minColWidth
					}
				}
				matrixFontSize := 7.0
				if colW < 6.5 {
					matrixFontSize = 5.5
				}

				// Title (8) + Ln(2) = 10
				// Header row (5) + n rows (n*5) = (n+1)*5
				// Bottom margin padding (10)
				reqHeight := 10.0 + float64(len(gs.Players)+1)*5.0 + 10.0
				_, pageHeight := pdf.GetPageSize()
				_, _, _, bottomMargin := pdf.GetMargins()

				if wantOrientation != curOrientation {
					pdf.AddPageFormat(wantOrientation, fpdf.SizeType{Wd: 210, Ht: 297})
					pdf.SetMargins(15, 52, 15)
					curOrientation = wantOrientation
				} else if pdf.GetY()+reqHeight > pageHeight-bottomMargin {
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
				pdf.SetXY(15+42+3, startY)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212)
				pdf.CellFormat(48, 5, tr("   ")+tr(strings.ToUpper(gs.GroupName)), "1", 0, "L", true, 0, "")
				pdf.SetFont("Arial", "B", matrixFontSize)
				for col := 1; col <= n; col++ {
					pdf.CellFormat(colW, 5, fmt.Sprintf("%d", col), "1", 0, "C", true, 0, "")
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
					pdf.SetFont("Arial", "", matrixFontSize)
					for colIdx, p2 := range gs.Players {
						if rowIdx == colIdx {
							pdf.SetFillColor(220, 220, 220) // gray diagonal
							pdf.CellFormat(colW, 5, "", "1", 0, "C", true, 0, "")
						} else {
							// Find match between p1 and p2
							var mVal = "-"
							for _, m := range gMatches {
								if (m.TeamA[0].ID == p1.ID && m.TeamB[0].ID == p2.ID) || (m.TeamA[0].ID == p2.ID && m.TeamB[0].ID == p1.ID) {
									if m.Status == "finished" {
										if m.IsForfeit {
											mVal = "NSP"
										} else if m.TeamA[0].ID == p1.ID {
											mVal = fmt.Sprintf("%d-%d", m.ScoreA(), m.ScoreB())
										} else {
											mVal = fmt.Sprintf("%d-%d", m.ScoreB(), m.ScoreA())
										}
									}
									break
								}
							}
							pdf.CellFormat(colW, 5, mVal, "1", 0, "C", false, 0, "")
						}
					}
					pdf.Ln(5)
				}

				// --- PART C: Points & Positions ---
				pdf.SetXY(15+42+3+48+float64(n)*colW+3, startY)
				pdf.SetFont("Arial", "B", 7)
				pdf.SetFillColor(254, 254, 212)
				pdf.CellFormat(8, 5, tr("Pts"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(14, 5, tr("Sets"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(16, 5, tr("Puntos"), "1", 0, "C", true, 0, "")
				pdf.CellFormat(8, 5, "Pos.", "1", 1, "C", true, 0, "")

				for _, p := range gs.Players {
					pdf.SetX(15 + 42 + 3 + 48 + float64(n)*colW + 3)

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

			// Later sections assume a portrait page; switch back if the last
			// group table forced a landscape page.
			if curOrientation != "P" {
				pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
				pdf.SetMargins(15, 52, 15)
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
				rounds := buildPdfBracketRounds(t, dt.ID, bracketPlayers)
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
			pdf.AddPageFormat("L", fpdf.SizeType{Wd: 210, Ht: 297})
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

			// Subsequent rounds are calculated as midpoints of their children,
			// using each round's NextIndex to find which previous-round
			// matches actually feed each match here -- not 2*j/2*j+1, since
			// real-match-based pairing can combine non-adjacent slots.
			for r := 1; r < numRounds; r++ {
				feeders := make([][]int, len(rounds[r].Matches))
				for i, nj := range rounds[r-1].NextIndex {
					if nj >= 0 && nj < len(feeders) {
						feeders[nj] = append(feeders[nj], i)
					}
				}
				for j := range rounds[r].Matches {
					if rounds[r].Name == "Champion" {
						centers[r][0] = centers[r-1][0]
						continue
					}
					switch fs := feeders[j]; len(fs) {
					case 2:
						centers[r][j] = (centers[r-1][fs[0]] + centers[r-1][fs[1]]) / 2
					case 1:
						centers[r][j] = centers[r-1][fs[0]]
					default:
						centers[r][j] = marginT + printableH/2
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
					if nextNumMatches > 0 {
						// Group this round's match indices by which next-round
						// match they actually feed (round.NextIndex), not
						// positional adjacency -- see groupPdfSlotsByRealMatches:
						// real-match-based pairing can combine non-adjacent
						// slots (e.g. match 0's winner actually played match
						// 3's winner next), so 2*j/2*j+1 would draw the wrong
						// connector lines and mislabel which box a given pair
						// of winners feeds into.
						feederGroups := make(map[int][]int, numMatches)
						for j := 0; j < numMatches; j++ {
							nj := j / 2
							if j < len(round.NextIndex) && round.NextIndex[j] >= 0 {
								nj = round.NextIndex[j]
							}
							feederGroups[nj] = append(feederGroups[nj], j)
						}

						for j := 0; j < numMatches; j++ {
							currentMidY := centers[r][j]
							nextJ := j / 2
							if j < len(round.NextIndex) && round.NextIndex[j] >= 0 {
								nextJ = round.NextIndex[j]
							}
							nextMidY := centers[r+1][nextJ]

							lineX1 := x + boxW
							lineX2 := x + boxW + (colW-boxW)/2

							pdf.SetDrawColor(180, 180, 180)
							pdf.Line(lineX1, currentMidY, lineX2, currentMidY)

							// Print match details above and score below the
							// line -- always, including the round right
							// before the Champion box (the Final itself),
							// which used to have this whole block skipped
							// because the old code nested it inside a check
							// that excluded a next round named "Champion".
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

								if mForDetails.Match.Status == "finished" {
									scoreStr := fmt.Sprintf("(%d-%d)", mForDetails.Match.ScoreA(), mForDetails.Match.ScoreB())
									if mForDetails.Match.IsForfeit {
										scoreStr = "(NSP)"
									}
									pdf.SetFont("Arial", "B", 6)
									pdf.SetTextColor(0, 0, 0)
									pdf.Text(lineX1+1, currentMidY+3, tr(scoreStr))
								}

								pdf.SetTextColor(0, 0, 0)

								// Draw the merge/next-box lines once per
								// feeder group (its lowest match index),
								// instead of once per position-adjacent pair.
								group := feederGroups[nextJ]
								if len(group) > 0 && group[0] == j {
									if len(group) == 2 {
										siblingMidY := centers[r][group[1]]
										pdf.Line(lineX2, currentMidY, lineX2, siblingMidY)
									}
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
		pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
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

	// 6. PLAYER STATISTICS (per participant, all stages including knockout)
	if len(t.Participants) > 0 {
		statsByPlayer := event.BuildAllPlayerEventStats(t.Matches)
		snapshotByPlayer := make(map[string]event.ParticipantSnapshot, len(t.ParticipantSnapshots))
		for _, snap := range t.ParticipantSnapshots {
			snapshotByPlayer[snap.PlayerID] = snap
		}
		isDoublesType := t.Type == "doubles" || t.Type == "mixed_doubles" || t.Type == "teams"

		// The bracket section above draws with absolute coordinates and never
		// advances fpdf's page/cursor state, so this section must always start
		// on its own fresh portrait page rather than continuing wherever the
		// cursor was left (which can still be mid-way through a landscape
		// bracket page).
		pdf.AddPageFormat("P", fpdf.SizeType{Wd: 210, Ht: 297})
		pdf.SetMargins(15, 52, 15)

		writeHeader("ESTADÍSTICAS DE JUGADORES")

		pdf.SetFont("Arial", "B", 8)
		pdf.SetFillColor(245, 247, 250)
		pdf.CellFormat(55, 7, tr("Jugador"), "1", 0, "L", true, 0, "")
		pdf.CellFormat(20, 7, tr("Jug."), "1", 0, "C", true, 0, "")
		pdf.CellFormat(20, 7, tr("G-P"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, tr("Sets"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(25, 7, tr("Puntos"), "1", 0, "C", true, 0, "")
		pdf.CellFormat(35, 7, tr("Elo"), "1", 1, "C", true, 0, "")

		pdf.SetFont("Arial", "", 8)
		for _, p := range t.Participants {
			stats := statsByPlayer[p.ID]
			eloStr := "-"
			if snap, ok := snapshotByPlayer[p.ID]; ok {
				before, after := snap.EloBeforeSingles, snap.EloAfterSingles
				if isDoublesType {
					before, after = snap.EloBeforeDoubles, snap.EloAfterDoubles
				}
				if before != nil {
					if after != nil {
						eloStr = fmt.Sprintf("%d -> %d", *before, *after)
					} else {
						eloStr = fmt.Sprintf("%d", *before)
					}
				}
			}

			pdf.CellFormat(55, 6, tr(truncateStr(formatPlayerName(p), 32)), "1", 0, "L", false, 0, "")
			pdf.CellFormat(20, 6, fmt.Sprintf("%d", stats.Played), "1", 0, "C", false, 0, "")
			pdf.CellFormat(20, 6, fmt.Sprintf("%d-%d", stats.Wins, stats.Losses), "1", 0, "C", false, 0, "")
			pdf.CellFormat(25, 6, fmt.Sprintf("%d-%d", stats.SetsWon, stats.SetsLost), "1", 0, "C", false, 0, "")
			pdf.CellFormat(25, 6, fmt.Sprintf("%d-%d", stats.PointsWon, stats.PointsLost), "1", 0, "C", false, 0, "")
			pdf.CellFormat(35, 6, eloStr, "1", 1, "C", false, 0, "")
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
	passCount := t.GroupPassCount
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
