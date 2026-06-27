package tournament

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"table-tennis-backend/internal/domain/player"
	tournamentDomain "table-tennis-backend/internal/domain/tournament"
)

type mockTournamentRepository struct {
	t *tournamentDomain.Tournament
}

func (m *mockTournamentRepository) Save(ctx context.Context, t *tournamentDomain.Tournament) error {
	return nil
}
func (m *mockTournamentRepository) GetByID(ctx context.Context, id string) (*tournamentDomain.Tournament, error) {
	return m.t, nil
}
func (m *mockTournamentRepository) GetAll(ctx context.Context) ([]*tournamentDomain.Tournament, error) {
	return []*tournamentDomain.Tournament{m.t}, nil
}
func (m *mockTournamentRepository) Update(ctx context.Context, t *tournamentDomain.Tournament) error {
	return nil
}
func (m *mockTournamentRepository) UpdateGroups(ctx context.Context, t *tournamentDomain.Tournament) error {
	return nil
}
func (m *mockTournamentRepository) Delete(ctx context.Context, id string) error {
	return nil
}
func (m *mockTournamentRepository) UpdateParticipantElo(ctx context.Context, tournamentID string, playerID string, singlesElo, doublesElo int16) error {
	return nil
}
func (m *mockTournamentRepository) UpdateParticipantsElo(ctx context.Context, tournamentID string, players []*player.Player) error {
	return nil
}
func (m *mockTournamentRepository) SaveTeam(ctx context.Context, team *tournamentDomain.Team) error {
	return nil
}
func (m *mockTournamentRepository) DeleteTeam(ctx context.Context, id string) error {
	return nil
}
func (m *mockTournamentRepository) AddPlayerToTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}
func (m *mockTournamentRepository) RemovePlayerFromTeam(ctx context.Context, teamID string, playerID string) error {
	return nil
}
func (m *mockTournamentRepository) GetParticipantSnapshots(ctx context.Context, tournamentID string) ([]tournamentDomain.ParticipantSnapshot, error) {
	return nil, nil
}
func (m *mockTournamentRepository) GetParticipantOrOfficialByPIN(ctx context.Context, tournamentID string, pin string) (string, error) {
	return "", nil
}

func TestExportTournamentPdf_Execute(t *testing.T) {
	// Create mock tournament data with Spanish letters in names
	p1 := &player.Player{ID: uuid.New().String(), FirstName: "José", LastName: "Muñoz", Gender: "M", Country: "Nicaragua"}
	p2 := &player.Player{ID: uuid.New().String(), FirstName: "María", LastName: "Gómez", Gender: "F", Country: "Spain"}

	tourney := &tournamentDomain.Tournament{
		ID:           uuid.New().String(),
		Name:         "Torneo Relámpago Español",
		Type:         "singles",
		Format:       "groups_elimination",
		Status:       "finished",
		StartDate:    time.Now(),
		EndDate:      time.Now().Add(24 * time.Hour),
		Participants: []*player.Player{p1, p2},
		Matches: []tournamentDomain.Match{
			{
				ID:         uuid.New().String(),
				Stage:      "group",
				Status:     "finished",
				TeamA:      []*player.Player{p1},
				TeamB:      []*player.Player{p2},
				WinnerTeam: "A",
				Sets: []tournamentDomain.MatchSet{
					{Number: 1, ScoreA: 11, ScoreB: 9},
					{Number: 2, ScoreA: 11, ScoreB: 7},
					{Number: 3, ScoreA: 11, ScoreB: 5},
				},
			},
			{
				ID:         uuid.New().String(),
				Stage:      "final",
				Status:     "finished",
				TeamA:      []*player.Player{p1},
				TeamB:      []*player.Player{p2},
				WinnerTeam: "B",
				Sets: []tournamentDomain.MatchSet{
					{Number: 1, ScoreA: 8, ScoreB: 11},
					{Number: 2, ScoreA: 9, ScoreB: 11},
					{Number: 3, ScoreA: 11, ScoreB: 7},
					{Number: 4, ScoreA: 5, ScoreB: 11},
				},
			},
		},
	}

	repo := &mockTournamentRepository{t: tourney}
	useCase := NewExportTournamentPdfUseCase(repo)

	pdfBytes, err := useCase.Execute(context.Background(), tourney.ID)
	if err != nil {
		t.Fatalf("unexpected error exporting PDF: %v", err)
	}

	if len(pdfBytes) == 0 {
		t.Error("expected non-empty PDF byte slice")
	}
}
