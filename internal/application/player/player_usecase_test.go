package player_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"table-tennis-backend/internal/application/player"
	"table-tennis-backend/internal/domain/idgen"
	playerDomain "table-tennis-backend/internal/domain/player"
	"table-tennis-backend/internal/infrastructure/identity"

	"github.com/xuri/excelize/v2"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

type mockPlayerRepo struct {
	players map[string]*playerDomain.Player
}

func newMockPlayerRepo() *mockPlayerRepo {
	return &mockPlayerRepo{
		players: make(map[string]*playerDomain.Player),
	}
}

func (m *mockPlayerRepo) GetById(ctx context.Context, id string) (*playerDomain.Player, error) {
	p, ok := m.players[id]
	if !ok {
		return nil, errors.New("player not found")
	}
	return p, nil
}

func (m *mockPlayerRepo) GetByIDs(ctx context.Context, ids []string) ([]*playerDomain.Player, error) {
	var result []*playerDomain.Player
	for _, id := range ids {
		if p, ok := m.players[id]; ok {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepo) Save(ctx context.Context, p *playerDomain.Player) error {
	m.players[p.ID] = p
	return nil
}

func (m *mockPlayerRepo) SaveMultiple(ctx context.Context, players []*playerDomain.Player) error {
	for _, p := range players {
		m.players[p.ID] = p
	}
	return nil
}

func (m *mockPlayerRepo) Delete(ctx context.Context, id string) error {
	delete(m.players, id)
	return nil
}

func (m *mockPlayerRepo) Search(ctx context.Context, query string) ([]*playerDomain.Player, error) {
	var result []*playerDomain.Player
	for _, p := range m.players {
		if strings.Contains(strings.ToLower(p.FirstName), strings.ToLower(query)) ||
			strings.Contains(strings.ToLower(p.LastName), strings.ToLower(query)) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepo) SearchForSelection(ctx context.Context, query, gender string) ([]*playerDomain.Player, error) {
	var result []*playerDomain.Player
	for _, p := range m.players {
		if (gender == "" || p.Gender == gender) &&
			(strings.Contains(strings.ToLower(p.FirstName), strings.ToLower(query)) ||
				strings.Contains(strings.ToLower(p.LastName), strings.ToLower(query))) {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepo) GetAll(ctx context.Context) ([]*playerDomain.Player, error) {
	var result []*playerDomain.Player
	for _, p := range m.players {
		result = append(result, p)
	}
	return result, nil
}

func (m *mockPlayerRepo) GetAllSingles(ctx context.Context) ([]*playerDomain.Player, error) {
	return m.GetAll(ctx)
}

func (m *mockPlayerRepo) GetAllDoubles(ctx context.Context) ([]*playerDomain.Player, error) {
	return m.GetAll(ctx)
}

func (m *mockPlayerRepo) GetSinglesByGender(ctx context.Context, gender string) ([]*playerDomain.Player, error) {
	var result []*playerDomain.Player
	for _, p := range m.players {
		if p.Gender == gender {
			result = append(result, p)
		}
	}
	return result, nil
}

func (m *mockPlayerRepo) GetDoublesByGender(ctx context.Context, gender string) ([]*playerDomain.Player, error) {
	return m.GetSinglesByGender(ctx, gender)
}

// ── Tests ──────────────────────────────────────────────────────────────────

func TestRegisterPlayerUseCase(t *testing.T) {
	repo := newMockPlayerRepo()
	uc := player.NewRegisterPlayerUseCase(repo)
	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		p, err := uc.Execute(
			ctx,
			"John", "David", "Doe", "Smith",
			"1995-05-15", "M", "USA", "Dept1",
			"+1234567890", "ID123", 1250, 1150,
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.FirstName != "John" || p.LastName != "Doe" {
			t.Errorf("unexpected name: %s %s", p.FirstName, p.LastName)
		}
		if p.SinglesElo != 1250 || p.DoublesElo != 1150 {
			t.Errorf("unexpected elos: %d %d", p.SinglesElo, p.DoublesElo)
		}
		if p.WhatsAppNumber != "+1234567890" {
			t.Errorf("unexpected whatsapp: %s", p.WhatsAppNumber)
		}
	})

	t.Run("invalid birthdate", func(t *testing.T) {
		_, err := uc.Execute(
			ctx,
			"Jane", "", "Doe", "",
			"invalid-date", "F", "USA", "",
			"", "", 0, 0,
		)
		if err == nil {
			t.Fatal("expected error for invalid birthdate, got nil")
		}
	})

	t.Run("empty names", func(t *testing.T) {
		_, err := uc.Execute(
			ctx,
			"", "", "", "",
			"2000-01-01", "M", "USA", "",
			"", "", 0, 0,
		)
		if err == nil {
			t.Fatal("expected error for empty name, got nil")
		}
	})
}

func TestGetPlayerByIDUseCase(t *testing.T) {
	repo := newMockPlayerRepo()
	ctx := context.Background()

	p, _ := playerDomain.NewPlayer("p1", "Alice", "Smith", time.Now(), "F", "USA", "", "123")
	_ = repo.Save(ctx, p)

	uc := player.NewGetPlayerByIDUseCase(repo)

	t.Run("existing player", func(t *testing.T) {
		found, err := uc.Execute(ctx, "p1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if found.FirstName != "Alice" {
			t.Errorf("expected Alice, got %s", found.FirstName)
		}
	})

	t.Run("non-existing player", func(t *testing.T) {
		_, err := uc.Execute(ctx, "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent player")
		}
	})
}

func TestUpdatePlayerUseCase(t *testing.T) {
	repo := newMockPlayerRepo()
	ctx := context.Background()

	p, _ := playerDomain.NewPlayer("p1", "Bob", "Jones", time.Now(), "M", "USA", "", "123")
	_ = repo.Save(ctx, p)

	uc := player.NewUpdatePlayerUseCase(repo)

	updated, err := uc.Execute(
		ctx,
		"p1",
		"Robert", "Bobby", "Jones", "Jr",
		"1990-10-20", "M", "Canada", "Ontario",
		"+987654321", "ID999", 1400, 1300,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.FirstName != "Robert" || updated.SecondName != "Bobby" {
		t.Errorf("unexpected updated name: %s %s", updated.FirstName, updated.SecondName)
	}
	if updated.SinglesElo != 1400 || updated.DoublesElo != 1300 {
		t.Errorf("unexpected updated elo: %d %d", updated.SinglesElo, updated.DoublesElo)
	}
	if updated.Country != "Canada" || updated.Department != "Ontario" {
		t.Errorf("unexpected updated location: %s %s", updated.Country, updated.Department)
	}
}

func TestDeletePlayerUseCase(t *testing.T) {
	repo := newMockPlayerRepo()
	ctx := context.Background()

	p, _ := playerDomain.NewPlayer("p1", "Charlie", "Brown", time.Now(), "M", "USA", "", "123")
	_ = repo.Save(ctx, p)

	uc := player.NewDeletePlayerUseCase(repo)

	if err := uc.Execute(ctx, "p1"); err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}

	_, err := repo.GetById(ctx, "p1")
	if err == nil {
		t.Fatal("expected error getting deleted player")
	}
}

func TestSearchPlayersUseCases(t *testing.T) {
	repo := newMockPlayerRepo()
	ctx := context.Background()

	p1, _ := playerDomain.NewPlayer("p1", "Carlos", "Santana", time.Now(), "M", "MX", "", "1")
	p2, _ := playerDomain.NewPlayer("p2", "Carla", "Bruni", time.Now(), "F", "FR", "", "2")
	_ = repo.Save(ctx, p1)
	_ = repo.Save(ctx, p2)

	t.Run("search all", func(t *testing.T) {
		uc := player.NewSearchPlayersUseCase(repo)
		res, err := uc.Execute(ctx, "car")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 2 {
			t.Errorf("expected 2 matches, got %d", len(res))
		}
	})

	t.Run("search for selection filtered by gender", func(t *testing.T) {
		uc := player.NewSearchPlayersForSelectionUseCase(repo)
		res, err := uc.Execute(ctx, "car", "F")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].FirstName != "Carla" {
			t.Errorf("expected 1 female match, got %v", res)
		}
	})
}

func TestImportPlayersUseCase(t *testing.T) {
	repo := newMockPlayerRepo()
	uc := player.NewImportPlayersUseCase(repo)
	ctx := context.Background()

	t.Run("unsupported file extension", func(t *testing.T) {
		_, err := uc.Execute(ctx, "players.txt", strings.NewReader("some data"))
		if err == nil {
			t.Fatal("expected error for unsupported extension")
		}
	})

	t.Run("valid CSV import", func(t *testing.T) {
		csvContent := `first_name,last_name,birthdate,gender,country,singles_elo,doubles_elo,national_id
Alice,Smith,1995-01-01,F,USA,1300,1200,CED1
Bob,Johnson,02/05/1990,M,CAN,1400,1350,CED2
,InvalidRow,1990-01-01,M,USA,1000,1000,CED3`

		res, err := uc.Execute(ctx, "import.csv", bytes.NewBufferString(csvContent))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Imported != 2 {
			t.Errorf("expected 2 imported players, got %d", res.Imported)
		}
		if res.Skipped != 1 {
			t.Errorf("expected 1 skipped player, got %d", res.Skipped)
		}

		all, _ := repo.GetAll(ctx)
		if len(all) != 2 {
			t.Errorf("expected 2 players in repo, got %d", len(all))
		}
	})

	t.Run("valid Excel import", func(t *testing.T) {
		f := excelize.NewFile()
		sheet := "Sheet1"
		f.SetSheetRow(sheet, "A1", &[]interface{}{"first_name", "last_name", "birthdate", "gender", "country", "singles_elo", "doubles_elo", "national_id"})
		f.SetSheetRow(sheet, "A2", &[]interface{}{"Alice", "Smith", "1995-01-01", "F", "USA", 1300, 1200, "CED1"})
		f.SetSheetRow(sheet, "A3", &[]interface{}{"Bob", "Johnson", "02/05/1990", "M", "CAN", 1400, 1350, "CED2"})
		var buf bytes.Buffer
		if err := f.Write(&buf); err != nil {
			t.Fatalf("failed to write excel file: %v", err)
		}

		res, err := uc.Execute(ctx, "import.xlsx", &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Imported != 2 {
			t.Errorf("expected 2 imported players, got %d", res.Imported)
		}
		if res.Skipped != 0 {
			t.Errorf("expected 0 skipped player, got %d", res.Skipped)
		}

		all, _ := repo.GetAll(ctx)
		// Should be 4 total because 2 from CSV test and 2 from Excel test
		if len(all) != 4 {
			t.Errorf("expected 4 players in repo, got %d", len(all))
		}
	})

	t.Run("invalid Excel data", func(t *testing.T) {
		res, err := uc.Execute(ctx, "import.xlsx", strings.NewReader("not a valid excel file"))
		if err == nil {
			t.Fatal("expected error for invalid excel file")
		}
		if res != nil {
			t.Fatalf("expected nil result")
		}
	})

	t.Run("csv parse error", func(t *testing.T) {
		csvContent := `first_name,last_name,birthdate
Alice,Smith,1995-01-01
"Bob,Johnson,1990-01-01`
		res, err := uc.Execute(ctx, "import.csv", bytes.NewBufferString(csvContent))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res.Skipped != 1 {
			t.Errorf("expected 1 skipped player due to parse error, got %d", res.Skipped)
		}
		if len(res.Errors) == 0 {
			t.Error("expected parse error in Errors slice")
		}
	})
}
