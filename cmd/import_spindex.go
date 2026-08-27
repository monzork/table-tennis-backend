//go:build ignore

// Imports completed matches from a Spindex tournament data dump into this
// system. The dump (see -file) is not a clean REST response: it's a raw
// capture of Firestore's real-time Listen stream (concatenated JSON
// documents), captured from the browser network tab while viewing the
// Spindex tournament page. Re-run this against a fresher capture any time
// to pick up newly-completed matches -- it's idempotent via the
// spindex_*_id columns added in migration 061.
//
// Usage:
//
//	go run cmd/import_spindex.go -file stadiumtt.json -tournament <internal-tournament-uuid> [-commit] [-overrides spindex_player_overrides.json]
//
// Without -commit, it only prints what it would do (dry run).
//
// Player matching: an exact, accent/case-insensitive full-name match against
// existing players is applied automatically. Anything else (no match, or
// more than one match) is left unresolved and printed as a JSON skeleton to
// -overrides; edit that file to map each spindexPlayerId to either an
// existing internal player UUID or the literal string "CREATE" (to create a
// brand new player), then re-run.
//
// Elo: this script deliberately does NOT recalculate player Elo. The
// tournament is imported incrementally as Spindex data is captured, so
// running it mid-tournament must not push live rating changes to players
// based on a still-incomplete match set -- Elo stays whatever it already
// was (an estimate) until the tournament is fully finished. Once every
// match is imported, trigger the existing "Recalculate Elo" admin action
// for each touched event to commit final ratings.
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	matchApp "table-tennis-backend/internal/application/match"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/identity"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

// ---- Firestore Listen-stream parsing ----

type fsDoc struct {
	Name       string                 `json:"name"`
	Fields     map[string]fsValue     `json:"fields"`
	CreateTime string                 `json:"createTime"`
	UpdateTime string                 `json:"updateTime"`
}

type fsValue struct {
	StringValue  *string             `json:"stringValue"`
	IntegerValue *string             `json:"integerValue"`
	DoubleValue  *float64            `json:"doubleValue"`
	BooleanValue *bool               `json:"booleanValue"`
	NullValue    json.RawMessage     `json:"nullValue"`
	ArrayValue   *struct {
		Values []fsValue `json:"values"`
	} `json:"arrayValue"`
	MapValue *struct {
		Fields map[string]fsValue `json:"fields"`
	} `json:"mapValue"`
}

func (v fsValue) Str() string {
	if v.StringValue != nil {
		return *v.StringValue
	}
	return ""
}

func (v fsValue) Int() int {
	if v.IntegerValue != nil {
		n, _ := strconv.Atoi(*v.IntegerValue)
		return n
	}
	return 0
}

func (v fsValue) Bool() bool {
	return v.BooleanValue != nil && *v.BooleanValue
}

func (v fsValue) IsNull() bool {
	return v.NullValue != nil
}

// parseFirestoreDump walks every concatenated JSON document in the file and
// extracts documentChange payloads, keeping only the LAST (most recent)
// snapshot per document name.
func parseFirestoreDump(path string) (map[string]fsDoc, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	content := string(raw)
	dec := json.NewDecoder(strings.NewReader(content))

	docs := make(map[string]fsDoc)
	var walk func(v interface{})
	walk = func(v interface{}) {
		switch t := v.(type) {
		case map[string]interface{}:
			if dc, ok := t["documentChange"]; ok {
				if dcMap, ok := dc.(map[string]interface{}); ok {
					if docRaw, ok := dcMap["document"]; ok {
						b, _ := json.Marshal(docRaw)
						var d fsDoc
						if json.Unmarshal(b, &d) == nil && d.Name != "" {
							docs[d.Name] = d
						}
					}
				}
			}
			for _, vv := range t {
				walk(vv)
			}
		case []interface{}:
			for _, vv := range t {
				walk(vv)
			}
		}
	}

	for {
		var v interface{}
		err := dec.Decode(&v)
		if err != nil {
			break
		}
		walk(v)
	}
	return docs, nil
}

var collectionRe = regexp.MustCompile(`/documents/([^/]+)/(.+)$`)

func collectionOf(name string) (collection, id string) {
	m := collectionRe.FindStringSubmatch(name)
	if m == nil {
		return "", ""
	}
	return m[1], m[2]
}

// ---- Domain-shaped extracts from the dump ----

type spindexPlayer struct {
	SpindexID string
	FirstName string
	LastName  string
}

type spindexMatch struct {
	SpindexID        string
	SpindexEventID   string
	PlayerA          string
	PlayerB          string
	FirstCompletedAt int
	GameScores       []gameScore // ordered by game index
}

type gameScore struct {
	ScoreA int
	ScoreB int
}

func extractGameScores(v fsValue) []gameScore {
	if v.MapValue == nil {
		return nil
	}
	// keys are "0","1","2",... game indices
	keys := make([]int, 0, len(v.MapValue.Fields))
	idx := make(map[int]fsValue)
	for k, gv := range v.MapValue.Fields {
		n, err := strconv.Atoi(k)
		if err != nil {
			continue
		}
		keys = append(keys, n)
		idx[n] = gv
	}
	sort.Ints(keys)
	var out []gameScore
	for _, k := range keys {
		gv := idx[k]
		if gv.MapValue == nil {
			continue
		}
		a := gv.MapValue.Fields["0"].Int()
		b := gv.MapValue.Fields["1"].Int()
		out = append(out, gameScore{ScoreA: a, ScoreB: b})
	}
	return out
}

// ---- Name normalization for exact-match player resolution ----

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	replacer := strings.NewReplacer(
		"á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ñ", "n", "ü", "u",
	)
	return replacer.Replace(s)
}

func main() {
	filePath := flag.String("file", "", "path to the Spindex Firestore dump JSON file")
	tournamentID := flag.String("tournament", "", "internal tournament UUID to import into")
	commit := flag.Bool("commit", false, "actually write changes (default: dry run)")
	repair := flag.Bool("repair", false, "re-apply correctly-oriented scores to matches already tagged with spindex_match_id (fixes team-order bugs from earlier runs); implies -commit")
	overridesPath := flag.String("overrides", "spindex_player_overrides.json", "path to player-mapping overrides file")
	flag.Parse()

	if *filePath == "" || *tournamentID == "" {
		log.Fatal("-file and -tournament are required")
	}

	godotenv.Load()
	bun.Connect()
	idgen.Register(identity.NewUUIDGenerator())
	ctx := context.Background()

	docs, err := parseFirestoreDump(*filePath)
	if err != nil {
		log.Fatalf("failed to parse dump: %v", err)
	}
	log.Printf("parsed %d distinct Firestore documents", len(docs))

	// ---- locate the tournament doc ----
	var spindexTournamentID string
	for name, d := range docs {
		coll, _ := collectionOf(name)
		if coll == "tournaments" {
			spindexTournamentID = d.Fields["id"].Str()
			break
		}
	}
	if spindexTournamentID == "" {
		log.Fatal("no tournaments document found in the dump")
	}
	log.Printf("Spindex tournament ID: %s", spindexTournamentID)

	// ---- events ----
	type spindexEvent struct {
		SpindexID string
		Name      string
	}
	var spindexEvents []spindexEvent
	for name, d := range docs {
		coll, _ := collectionOf(name)
		if coll != "events" {
			continue
		}
		if d.Fields["tournamentId"].Str() != spindexTournamentID {
			continue
		}
		spindexEvents = append(spindexEvents, spindexEvent{
			SpindexID: d.Fields["id"].Str(),
			Name:      d.Fields["name"].Str(),
		})
	}
	log.Printf("found %d Spindex events for this tournament", len(spindexEvents))

	// ---- players (scoped to this tournament) ----
	players := make(map[string]spindexPlayer) // spindexPlayerID -> player
	for name, d := range docs {
		coll, _ := collectionOf(name)
		if coll != "public-players" {
			continue
		}
		if d.Fields["tournamentId"].Str() != spindexTournamentID {
			continue
		}
		id := d.Fields["id"].Str()
		players[id] = spindexPlayer{
			SpindexID: id,
			FirstName: strings.TrimSpace(d.Fields["firstName"].Str()),
			LastName:  strings.TrimSpace(d.Fields["lastName"].Str()),
		}
	}
	log.Printf("found %d Spindex players for this tournament", len(players))

	// ---- matches: only completed ones ----
	var matches []spindexMatch
	for name, d := range docs {
		coll, _ := collectionOf(name)
		if coll != "matches" {
			continue
		}
		if d.Fields["tournamentId"].Str() != spindexTournamentID {
			continue
		}
		fc := d.Fields["firstCompletedAt"]
		if fc.IsNull() || fc.IntegerValue == nil {
			continue // not yet played
		}
		matches = append(matches, spindexMatch{
			SpindexID:        d.Fields["id"].Str(),
			SpindexEventID:   d.Fields["eventId"].Str(),
			PlayerA:          d.Fields["playerA"].Str(),
			PlayerB:          d.Fields["playerB"].Str(),
			FirstCompletedAt: fc.Int(),
			GameScores:       extractGameScores(d.Fields["gameScores"]),
		})
	}
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].FirstCompletedAt < matches[j].FirstCompletedAt
	})
	log.Printf("found %d completed Spindex matches", len(matches))

	// ---- wire up repos / use cases exactly like cmd/server/container.go ----
	playerRepo := bun.NewPlayerRepository(bun.DB)
	eventRepo := bun.NewEventRepository(bun.DB)
	divisionRepo := bun.NewDivisionRepository(bun.DB)
	matchRepo := bun.NewMatchRepository(bun.DB, playerRepo)

	createMatchUC := matchApp.NewCreateMatchUseCase(matchRepo, playerRepo, eventRepo, divisionRepo)
	updateScoreUC := matchApp.NewUpdateMatchScoreUseCase(matchRepo, eventRepo)

	// ---- resolve internal events for this tournament ----
	rows, err := bun.DB.QueryContext(ctx, `SELECT id, name, spindex_event_id FROM events WHERE tournament_id = ?`, *tournamentID)
	if err != nil {
		log.Fatalf("failed to load internal events: %v", err)
	}
	type internalEvent struct {
		ID              string
		Name            string
		SpindexEventID  sql.NullString
	}
	var internalEvents []internalEvent
	for rows.Next() {
		var ie internalEvent
		if err := rows.Scan(&ie.ID, &ie.Name, &ie.SpindexEventID); err != nil {
			log.Fatal(err)
		}
		internalEvents = append(internalEvents, ie)
	}
	rows.Close()

	// keyword-based category matching: Spindex "Primera/Segunda/Tercera
	// Categoría" <-> our "PRIMERA/SEGUNDA/TERCERA DIVISION"
	keywords := []string{"primera", "segunda", "tercera"}
	eventMapping := make(map[string]string) // spindexEventID -> internal event ID
	for _, se := range spindexEvents {
		seName := normalizeName(se.Name)
		var matchedInternal *internalEvent
		for _, kw := range keywords {
			if !strings.Contains(seName, kw) {
				continue
			}
			for i := range internalEvents {
				if strings.Contains(normalizeName(internalEvents[i].Name), kw) {
					matchedInternal = &internalEvents[i]
					break
				}
			}
			break
		}
		if matchedInternal == nil {
			log.Printf("WARNING: could not map Spindex event %q (%s) to an internal event -- its matches will be skipped", se.Name, se.SpindexID)
			continue
		}
		eventMapping[se.SpindexID] = matchedInternal.ID
		log.Printf("event mapping: %-30s -> %s (%s)", se.Name, matchedInternal.Name, matchedInternal.ID)
	}

	// ---- resolve players: exact normalized-name match, else overrides file ----
	overrides := make(map[string]string)
	if b, err := os.ReadFile(*overridesPath); err == nil {
		if err := json.Unmarshal(b, &overrides); err != nil {
			log.Fatalf("failed to parse overrides file: %v", err)
		}
	}

	playerRows, err := bun.DB.QueryContext(ctx, `SELECT id, first_name, last_name FROM players`)
	if err != nil {
		log.Fatal(err)
	}
	type dbPlayer struct{ ID, First, Last string }
	nameIndex := make(map[string][]dbPlayer)
	for playerRows.Next() {
		var p dbPlayer
		if err := playerRows.Scan(&p.ID, &p.First, &p.Last); err != nil {
			log.Fatal(err)
		}
		key := normalizeName(p.First + " " + p.Last)
		nameIndex[key] = append(nameIndex[key], p)
	}
	playerRows.Close()

	uuidRe := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	playerMapping := make(map[string]string) // spindexPlayerID -> internal player ID
	toCreate := make(map[string]spindexPlayer)
	unresolved := make(map[string]spindexPlayer)

	for id, sp := range players {
		if ov, ok := overrides[id]; ok {
			switch {
			case ov == "CREATE":
				toCreate[id] = sp
				continue
			case uuidRe.MatchString(ov):
				playerMapping[id] = ov
				continue
			default:
				// Still a placeholder (e.g. "UNRESOLVED: ..."); fall through
				// and treat like it was never overridden.
			}
		}
		key := normalizeName(sp.FirstName + " " + sp.LastName)
		candidates := nameIndex[key]
		if len(candidates) == 1 {
			playerMapping[id] = candidates[0].ID
		} else {
			unresolved[id] = sp
		}
	}

	if len(unresolved) > 0 {
		skeleton := make(map[string]string)
		if b, err := os.ReadFile(*overridesPath); err == nil {
			json.Unmarshal(b, &skeleton)
		}
		for id, sp := range unresolved {
			if _, exists := skeleton[id]; !exists {
				skeleton[id] = fmt.Sprintf("UNRESOLVED: %s %s -- put an internal player UUID here, or \"CREATE\"", sp.FirstName, sp.LastName)
			}
		}
		out, _ := json.MarshalIndent(skeleton, "", "  ")
		if err := os.WriteFile(*overridesPath, out, 0644); err != nil {
			log.Fatalf("failed to write overrides file: %v", err)
		}
		log.Printf("%d Spindex players need manual resolution -- see %s, then re-run", len(unresolved), *overridesPath)
	}

	if !*commit && !*repair {
		log.Printf("DRY RUN: %d players auto-matched, %d to create, %d unresolved, %d completed matches would be considered. Pass -commit to apply.",
			len(playerMapping), len(toCreate), len(unresolved), len(matches))
		return
	}

	// ---- create brand-new players & set spindex_tournament_id/spindex_player_id ----
	if _, err := bun.DB.ExecContext(ctx, `UPDATE tournaments SET spindex_tournament_id = ? WHERE id = ?`, spindexTournamentID, *tournamentID); err != nil {
		log.Fatalf("failed to set spindex_tournament_id: %v", err)
	}
	for spindexEventID, internalID := range eventMapping {
		if _, err := bun.DB.ExecContext(ctx, `UPDATE events SET spindex_event_id = ? WHERE id = ?`, spindexEventID, internalID); err != nil {
			log.Fatalf("failed to set spindex_event_id: %v", err)
		}
	}

	for id, sp := range toCreate {
		newID := idgen.Generate()
		p, err := player.NewPlayer(newID, sp.FirstName, sp.LastName, time.Time{}, "M", "NI", "", "")
		if err != nil {
			log.Printf("skipping unresolvable player %s %s: %v", sp.FirstName, sp.LastName, err)
			continue
		}
		if err := playerRepo.Save(ctx, p); err != nil {
			log.Fatalf("failed to save new player: %v", err)
		}
		playerMapping[id] = newID
		log.Printf("created new player %s %s (%s)", sp.FirstName, sp.LastName, newID)
	}
	for spindexPlayerID, internalID := range playerMapping {
		if _, err := bun.DB.ExecContext(ctx, `UPDATE players SET spindex_player_id = ? WHERE id = ?`, spindexPlayerID, internalID); err != nil {
			log.Fatalf("failed to set spindex_player_id: %v", err)
		}
	}

	// ---- snapshot who's already enrolled per event; matches for anyone
	// not already a participant are skipped rather than auto-enrolling
	// them (e.g. players intentionally removed from the event) ----
	enrolled := make(map[string]map[string]bool) // eventID -> playerID -> true
	for spindexEventID, internalEventID := range eventMapping {
		existing, err := eventRepo.GetByID(ctx, internalEventID)
		if err != nil {
			log.Fatalf("failed to load event %s: %v", internalEventID, err)
		}
		set := make(map[string]bool)
		for _, p := range existing.Participants {
			set[p.ID] = true
		}
		enrolled[spindexEventID] = set
	}

	created, skippedDup, skippedNoMapping := 0, 0, 0
	touchedEvents := make(map[string]bool)

	for _, sm := range matches {
		internalEventID, ok := eventMapping[sm.SpindexEventID]
		if !ok {
			skippedNoMapping++
			continue
		}
		pAID, okA := playerMapping[sm.PlayerA]
		pBID, okB := playerMapping[sm.PlayerB]
		if !okA || !okB {
			skippedNoMapping++
			continue
		}

		// Players intentionally removed from the event (e.g. dropped out)
		// must not be re-added -- skip matches involving anyone who isn't
		// already an enrolled participant.
		if !enrolled[sm.SpindexEventID][pAID] || !enrolled[sm.SpindexEventID][pBID] {
			skippedNoMapping++
			continue
		}

		// idempotency: already imported?
		var existingMatchID string
		hasExisting := false
		if err := bun.DB.QueryRowContext(ctx, `SELECT id FROM matches WHERE spindex_match_id = ?`, sm.SpindexID).Scan(&existingMatchID); err == nil {
			hasExisting = true
		}
		if hasExisting && !*repair {
			skippedDup++
			continue
		}

		// The match's team_a/team_b order may be the reverse of Spindex's
		// playerA/playerB -- swap each set's scores if so, or the winner
		// gets attributed to the wrong side.
		var matchID string
		var swapped bool
		if hasExisting {
			// -repair: re-apply the correctly-oriented score to a match
			// that was already tagged (possibly with the bug above).
			existing, err := matchRepo.GetByID(ctx, existingMatchID)
			if err != nil {
				log.Printf("failed to load existing match %s: %v", existingMatchID, err)
				continue
			}
			matchID = existing.ID
			swapped = len(existing.TeamA) > 0 && existing.TeamA[0].ID == pBID
		} else {
			// round-robin dedup: at most one match per pair per event/stage
			existingMatch, _ := matchRepo.GetMatchByParticipants(ctx, internalEventID, pAID, pBID, "group")
			swapped = existingMatch != nil && len(existingMatch.TeamA) > 0 && existingMatch.TeamA[0].ID == pBID
			if existingMatch != nil {
				matchID = existingMatch.ID
			} else {
				m, err := createMatchUC.Execute(ctx, internalEventID, "singles", []string{pAID}, []string{pBID}, "group")
				if err != nil {
					log.Printf("failed to create match %s vs %s: %v", pAID, pBID, err)
					continue
				}
				matchID = m.ID
			}
		}

		var sets []string
		for _, g := range sm.GameScores {
			if swapped {
				sets = append(sets, fmt.Sprintf("%d-%d", g.ScoreB, g.ScoreA))
			} else {
				sets = append(sets, fmt.Sprintf("%d-%d", g.ScoreA, g.ScoreB))
			}
		}

		if err := updateScoreUC.Execute(ctx, matchID, sets, internalEventID, "group"); err != nil {
			log.Printf("failed to score match %s: %v", matchID, err)
			continue
		}
		if !hasExisting {
			if _, err := bun.DB.ExecContext(ctx, `UPDATE matches SET spindex_match_id = ? WHERE id = ?`, sm.SpindexID, matchID); err != nil {
				log.Fatalf("failed to set spindex_match_id: %v", err)
			}
		}
		created++
		touchedEvents[internalEventID] = true
	}

	log.Printf("imported %d matches, skipped %d already-imported, skipped %d unmappable", created, skippedDup, skippedNoMapping)

	if len(touchedEvents) > 0 {
		var names []string
		for eventID := range touchedEvents {
			names = append(names, eventID)
		}
		log.Printf("Elo left as-is (estimate) for %d touched event(s): %v -- once the tournament is fully imported, use the admin \"Recalculate Elo\" action for each to commit final ratings", len(names), names)
	}
}
