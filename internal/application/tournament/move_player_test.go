package tournament

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type mockMovePlayerRepository struct {
	t        *tournamentDomain.Tournament
	getErr   error
	updErr   error
	updCalls int
}

func (m *mockMovePlayerRepository) Save(ctx context.Context, t *tournamentDomain.Tournament) error {
	return nil
}

func (m *mockMovePlayerRepository) GetByID(ctx context.Context, id string) (*tournamentDomain.Tournament, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.t, nil
}

func (m *mockMovePlayerRepository) GetAll(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	return nil, nil
}

func (m *mockMovePlayerRepository) Update(ctx context.Context, t *tournamentDomain.Tournament) error {
	return nil
}

func (m *mockMovePlayerRepository) UpdateGroups(ctx context.Context, t *tournamentDomain.Tournament) error {
	m.updCalls++
	return m.updErr
}

func (m *mockMovePlayerRepository) UpdateEventIDBulk(ctx context.Context, tournamentIDs []string, eventID string) error {
	return nil
}

func (m *mockMovePlayerRepository) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockMovePlayerRepository) UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}

func (m *mockMovePlayerRepository) UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*player.Player) error {
	return nil
}

func (m *mockMovePlayerRepository) UpdateParticipantEloBefore(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}

func (m *mockMovePlayerRepository) SaveTeam(ctx context.Context, team *tournamentDomain.Team) error {
	return nil
}

func (m *mockMovePlayerRepository) DeleteTeam(ctx context.Context, id string) error {
	return nil
}

func (m *mockMovePlayerRepository) AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}

func (m *mockMovePlayerRepository) RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}

func (m *mockMovePlayerRepository) GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]tournamentDomain.ParticipantSnapshot, error) {
	return nil, nil
}

func (m *mockMovePlayerRepository) GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error) {
	return "", nil
}

func (m *mockMovePlayerRepository) AddOfficial(ctx context.Context, tournamentID string, playerID string, pin string) error {
	return nil
}

func (m *mockMovePlayerRepository) RemoveOfficial(ctx context.Context, tournamentID string, playerID string) error {
	return nil
}

func (m *mockMovePlayerRepository) GetOfficials(ctx context.Context, tournamentID string) ([]tournamentDomain.ParticipantSnapshot, error) {
	return nil, nil
}

func TestMovePlayerUseCase_Execute(t *testing.T) {
	pID := uuid.New().String()
	p := &player.Player{ID: pID, FirstName: "Kevin", LastName: "Muñoz", SinglesElo: 2400}

	g1ID := uuid.New().String()
	g1 := tournamentDomain.Group{
		ID:      g1ID,
		Name:    "First Division - Group A",
		Players: []*player.Player{p},
	}

	g2ID := uuid.New().String()
	g2 := tournamentDomain.Group{
		ID:      g2ID,
		Name:    "First Division - Group B",
		Players: []*player.Player{},
	}

	tourney := &tournamentDomain.Tournament{
		ID:           uuid.New().String(),
		Name:         "Orlando Jose in Memoriam",
		Status:       "scheduled",
		Format:       "groups_elimination",
		Participants: []*player.Player{p},
		Groups:       []tournamentDomain.Group{g1, g2},
	}

	t.Run("successful move", func(t *testing.T) {
		repo := &mockMovePlayerRepository{t: tourney}
		uc := NewMovePlayerUseCase(repo)

		err := uc.Execute(context.Background(), tourney.ID, pID, g2ID, 0)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if repo.updCalls != 1 {
			t.Errorf("expected repo.UpdateGroups to be called 1 time, got %d", repo.updCalls)
		}

		// Verify target group has player
		var targetGroup *tournamentDomain.Group
		for i := range repo.t.Groups {
			if repo.t.Groups[i].ID == g2ID {
				targetGroup = &repo.t.Groups[i]
			}
		}
		if targetGroup == nil || len(targetGroup.Players) != 1 || targetGroup.Players[0].ID != pID {
			t.Errorf("expected player to be moved to target group")
		}
	})

	t.Run("loading error", func(t *testing.T) {
		repo := &mockMovePlayerRepository{getErr: errors.New("database connection failed")}
		uc := NewMovePlayerUseCase(repo)

		err := uc.Execute(context.Background(), tourney.ID, pID, g2ID, 0)
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "database connection failed" {
			t.Errorf("expected database connection failed error, got %v", err)
		}
	})

	t.Run("invalid move (match already started)", func(t *testing.T) {
		tourneyStarted := &tournamentDomain.Tournament{
			ID:           uuid.New().String(),
			Name:         "Orlando Jose in Memoriam",
			Status:       "scheduled",
			Format:       "groups_elimination",
			Participants: []*player.Player{p},
			Groups:       []tournamentDomain.Group{g1, g2},
			Matches: []tournamentDomain.Match{
				{
					ID:     uuid.New().String(),
					Status: "in_progress",
					TeamA:  []*player.Player{p},
				},
			},
		}

		repo := &mockMovePlayerRepository{t: tourneyStarted}
		uc := NewMovePlayerUseCase(repo)

		err := uc.Execute(context.Background(), tourneyStarted.ID, pID, g2ID, 0)
		if err == nil {
			t.Fatal("expected error because tournament match has started, got nil")
		}
	})
}
