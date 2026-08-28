//go:build ignore

// One-off fixup for tournaments imported via import_spindex.go before that
// script stopped auto-committing Elo. It undoes a premature Elo commit by
// resetting each participant's LIVE player.singles_elo/doubles_elo back to
// the event_participants.elo_before_singles/doubles snapshot -- the value
// captured once at enrollment, before any tournament match was played, and
// never touched by RecalculateTournamentElo. That snapshot is exactly the
// "estimate" the player's rating should still show until the tournament is
// actually finished.
//
// Scope: only events under -tournament that have spindex_event_id set (i.e.
// events actually touched by the spindex importer).
//
// Usage:
//
//	go run cmd/revert_spindex_elo.go -tournament <internal-tournament-uuid> [-commit]
//
// Without -commit, it only prints what it would change (dry run).
package main

import (
	"context"
	"flag"
	"log"

	"github.com/joho/godotenv"

	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/identity"
	"table-tennis-backend/internal/infrastructure/persistence/bun"
)

func main() {
	tournamentID := flag.String("tournament", "", "internal tournament UUID whose spindex-touched events should have Elo reverted")
	commit := flag.Bool("commit", false, "actually write changes (default: dry run)")
	flag.Parse()

	if *tournamentID == "" {
		log.Fatal("-tournament is required")
	}

	godotenv.Load()
	bun.Connect()
	idgen.Register(identity.NewUUIDGenerator())
	ctx := context.Background()

	playerRepo := bun.NewPlayerRepository(bun.DB)
	eventRepo := bun.NewEventRepository(bun.DB)

	rows, err := bun.DB.QueryContext(ctx, `SELECT id, name FROM events WHERE tournament_id = ? AND spindex_event_id IS NOT NULL`, *tournamentID)
	if err != nil {
		log.Fatalf("failed to load spindex-touched events: %v", err)
	}
	type touchedEvent struct{ ID, Name string }
	var events []touchedEvent
	for rows.Next() {
		var e touchedEvent
		if err := rows.Scan(&e.ID, &e.Name); err != nil {
			log.Fatal(err)
		}
		events = append(events, e)
	}
	rows.Close()

	if len(events) == 0 {
		log.Println("no spindex-touched events found for this tournament -- nothing to revert")
		return
	}

	changed := 0
	for _, ev := range events {
		snapshots, err := eventRepo.GetParticipantSnapshots(ctx, ev.ID)
		if err != nil {
			log.Fatalf("failed to load participant snapshots for event %s (%s): %v", ev.Name, ev.ID, err)
		}

		var pids []string
		for _, s := range snapshots {
			if s.EloBeforeSingles != nil || s.EloBeforeDoubles != nil {
				pids = append(pids, s.PlayerID)
			}
		}
		if len(pids) == 0 {
			continue
		}

		players, err := playerRepo.GetByIDs(ctx, pids)
		if err != nil {
			log.Fatalf("failed to load players for event %s: %v", ev.Name, err)
		}
		playerByID := make(map[string]int) // playerID -> index in players
		for i, p := range players {
			playerByID[p.ID] = i
		}

		var toSave []*player.Player
		for _, s := range snapshots {
			idx, ok := playerByID[s.PlayerID]
			if !ok {
				continue
			}
			p := players[idx]
			before := *p
			if s.EloBeforeSingles != nil {
				p.UpdateSinglesElo(*s.EloBeforeSingles)
			}
			if s.EloBeforeDoubles != nil {
				p.UpdateDoublesElo(*s.EloBeforeDoubles)
			}
			if p.SinglesElo != before.SinglesElo || p.DoublesElo != before.DoublesElo {
				log.Printf("event %-30s player %s %s: singles %d -> %d, doubles %d -> %d",
					ev.Name, p.FirstName, p.LastName, before.SinglesElo, p.SinglesElo, before.DoublesElo, p.DoublesElo)
				toSave = append(toSave, p)
				changed++
			}
		}

		if *commit && len(toSave) > 0 {
			if err := playerRepo.UpdateElo(ctx, toSave); err != nil {
				log.Fatalf("failed to save reverted players for event %s: %v", ev.Name, err)
			}
			// Clear the after-snapshot too, since it no longer reflects live Elo.
			if err := eventRepo.UpdateParticipantsElo(ctx, ev.ID, toSave); err != nil {
				log.Printf("WARNING: failed to clear elo_after snapshot for event %s: %v", ev.Name, err)
			}
		}
	}

	if !*commit {
		log.Printf("DRY RUN: %d player Elo value(s) would be reverted. Pass -commit to apply.", changed)
	} else {
		log.Printf("reverted %d player Elo value(s)", changed)
	}
}
