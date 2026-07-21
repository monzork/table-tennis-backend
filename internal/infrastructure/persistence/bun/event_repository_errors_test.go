package bun_test

import (
	"context"
	"testing"

	"table-tennis-backend/internal/domain/event"
	"table-tennis-backend/internal/domain/player"
	bunRepo "table-tennis-backend/internal/infrastructure/persistence/bun"

	"github.com/google/uuid"
)

// Cheap coverage sweep of the uuid.Parse error branches that guard nearly
// every EventRepository method taking string IDs.
func TestEventRepository_InvalidIDErrorPaths(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	cases := []struct {
		name string
		fn   func() error
	}{
		{"Save invalid event id", func() error {
			return eventRepo.Save(ctx, &event.Event{ID: "bad-id"})
		}},
		{"Save invalid nested event-of-tournament id", func() error {
			bad := "bad-id"
			return eventRepo.Save(ctx, &event.Event{ID: uuid.NewString(), EventID: &bad})
		}},
		{"SaveTeam invalid tournament id", func() error {
			return eventRepo.SaveTeam(ctx, &event.Team{ID: uuid.NewString(), TournamentID: "bad-id"})
		}},
		{"SaveTeam invalid team id", func() error {
			return eventRepo.SaveTeam(ctx, &event.Team{ID: "bad-id", TournamentID: uuid.NewString()})
		}},
		{"DeleteTeam invalid id", func() error {
			return eventRepo.DeleteTeam(ctx, "bad-id")
		}},
		{"AddPlayerToTeam invalid team id", func() error {
			return eventRepo.AddPlayerToTeam(ctx, "bad-id", uuid.NewString())
		}},
		{"AddPlayerToTeam invalid player id", func() error {
			return eventRepo.AddPlayerToTeam(ctx, uuid.NewString(), "bad-id")
		}},
		{"RemovePlayerFromTeam invalid team id", func() error {
			return eventRepo.RemovePlayerFromTeam(ctx, "bad-id", uuid.NewString())
		}},
		{"RemovePlayerFromTeam invalid player id", func() error {
			return eventRepo.RemovePlayerFromTeam(ctx, uuid.NewString(), "bad-id")
		}},
		{"UpdateParticipantElo invalid tournament id", func() error {
			return eventRepo.UpdateParticipantElo(ctx, "bad-id", uuid.NewString(), 1, 1)
		}},
		{"UpdateParticipantElo invalid player id", func() error {
			return eventRepo.UpdateParticipantElo(ctx, uuid.NewString(), "bad-id", 1, 1)
		}},
		{"UpdateParticipantEloBefore invalid tournament id", func() error {
			return eventRepo.UpdateParticipantEloBefore(ctx, "bad-id", uuid.NewString(), 1, 1)
		}},
		{"UpdateParticipantEloBefore invalid player id", func() error {
			return eventRepo.UpdateParticipantEloBefore(ctx, uuid.NewString(), "bad-id", 1, 1)
		}},
		{"UpdateParticipantsElo invalid player id", func() error {
			return eventRepo.UpdateParticipantsElo(ctx, uuid.NewString(), []*player.Player{{ID: "bad-id"}})
		}},
		{"AddParticipant invalid tournament id", func() error {
			return eventRepo.AddParticipant(ctx, "bad-id", uuid.NewString(), 1, 1)
		}},
		{"AddParticipant invalid player id", func() error {
			return eventRepo.AddParticipant(ctx, uuid.NewString(), "bad-id", 1, 1)
		}},
		{"RemoveParticipant invalid tournament id", func() error {
			return eventRepo.RemoveParticipant(ctx, "bad-id", uuid.NewString())
		}},
		{"RemoveParticipant invalid player id", func() error {
			return eventRepo.RemoveParticipant(ctx, uuid.NewString(), "bad-id")
		}},
		{"GetParticipantSnapshots invalid id", func() error {
			_, err := eventRepo.GetParticipantSnapshots(ctx, "bad-id")
			return err
		}},
		{"GetParticipantPIN invalid tournament id", func() error {
			_, err := eventRepo.GetParticipantPIN(ctx, "bad-id", uuid.NewString())
			return err
		}},
		{"GetParticipantPIN invalid player id", func() error {
			_, err := eventRepo.GetParticipantPIN(ctx, uuid.NewString(), "bad-id")
			return err
		}},
		{"GetParticipantPIN not found", func() error {
			_, err := eventRepo.GetParticipantPIN(ctx, uuid.NewString(), uuid.NewString())
			return err
		}},
		{"GetParticipantPINsByTournament invalid id", func() error {
			_, err := eventRepo.GetParticipantPINsByTournament(ctx, "bad-id")
			return err
		}},
		{"AddOfficial invalid tournament id", func() error {
			return eventRepo.AddOfficial(ctx, "bad-id", uuid.NewString(), "1234")
		}},
		{"AddOfficial invalid player id", func() error {
			return eventRepo.AddOfficial(ctx, uuid.NewString(), "bad-id", "1234")
		}},
		{"RemoveOfficial invalid tournament id", func() error {
			return eventRepo.RemoveOfficial(ctx, "bad-id", uuid.NewString())
		}},
		{"RemoveOfficial invalid player id", func() error {
			return eventRepo.RemoveOfficial(ctx, uuid.NewString(), "bad-id")
		}},
		{"GetOfficials invalid id", func() error {
			_, err := eventRepo.GetOfficials(ctx, "bad-id")
			return err
		}},
		{"AddPlayerToTeam missing team", func() error {
			return eventRepo.AddPlayerToTeam(ctx, uuid.NewString(), uuid.NewString())
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.fn(); err == nil {
				t.Fatalf("expected error for %q, got nil", tc.name)
			}
		})
	}
}

func TestEventRepository_GetAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	eventRepo := bunRepo.NewEventRepository(db)
	ctx := context.Background()

	all, err := eventRepo.GetAll(ctx)
	if err != nil {
		t.Fatalf("GetAll: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("expected 0 events, got %d", len(all))
	}
}
