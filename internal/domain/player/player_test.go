package player_test

import (
	"testing"
	"time"

	"table-tennis-backend/internal/domain/player"
)

func TestNewPlayer_Success(t *testing.T) {
	bdate := time.Date(1995, time.May, 15, 0, 0, 0, 0, time.UTC)
	p, err := player.NewPlayer("p-1", "John", "Doe", bdate, "M", "USA", "Dept1", "NID123")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p.ID != "p-1" {
		t.Errorf("expected ID 'p-1', got '%s'", p.ID)
	}
	if p.FirstName != "John" {
		t.Errorf("expected FirstName 'John', got '%s'", p.FirstName)
	}
	if p.LastName != "Doe" {
		t.Errorf("expected LastName 'Doe', got '%s'", p.LastName)
	}
	if p.Gender != "M" {
		t.Errorf("expected Gender 'M', got '%s'", p.Gender)
	}
	if p.SinglesElo != 1000 {
		t.Errorf("expected SinglesElo 1000, got %d", p.SinglesElo)
	}
	if p.DoublesElo != 1000 {
		t.Errorf("expected DoublesElo 1000, got %d", p.DoublesElo)
	}
	if p.Country != "USA" {
		t.Errorf("expected Country 'USA', got '%s'", p.Country)
	}
	if p.Department != "Dept1" {
		t.Errorf("expected Department 'Dept1', got '%s'", p.Department)
	}
	if p.NationalID != "NID123" {
		t.Errorf("expected NationalID 'NID123', got '%s'", p.NationalID)
	}
}

func TestNewPlayer_DefaultGender(t *testing.T) {
	p, err := player.NewPlayer("p-2", "Jane", "Smith", time.Now(), "", "CAN", "Dept2", "NID456")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if p.Gender != "M" {
		t.Errorf("expected default gender 'M', got '%s'", p.Gender)
	}
}

func TestNewPlayer_InvalidName(t *testing.T) {
	tests := []struct {
		firstName string
		lastName  string
		label     string
	}{
		{"", "Doe", "empty first name"},
		{"John", "", "empty last name"},
		{"", "", "empty first and last name"},
	}

	for _, tt := range tests {
		t.Run(tt.label, func(t *testing.T) {
			_, err := player.NewPlayer("p-1", tt.firstName, tt.lastName, time.Now(), "M", "USA", "Dept1", "NID123")
			if err != player.ErrInvalidName {
				t.Errorf("expected ErrInvalidName, got %v", err)
			}
		})
	}
}

func TestPlayer_UpdateSinglesElo(t *testing.T) {
	p, _ := player.NewPlayer("p-1", "John", "Doe", time.Now(), "M", "USA", "Dept1", "NID123")

	p.UpdateSinglesElo(1200)
	if p.SinglesElo != 1200 {
		t.Errorf("expected SinglesElo 1200, got %d", p.SinglesElo)
	}

	p.UpdateSinglesElo(0)
	if p.SinglesElo != 0 {
		t.Errorf("expected SinglesElo 0, got %d", p.SinglesElo)
	}

	// Negative values should be ignored
	p.UpdateSinglesElo(-50)
	if p.SinglesElo != 0 {
		t.Errorf("expected SinglesElo to remain 0, got %d", p.SinglesElo)
	}
}

func TestPlayer_UpdateDoublesElo(t *testing.T) {
	p, _ := player.NewPlayer("p-1", "John", "Doe", time.Now(), "M", "USA", "Dept1", "NID123")

	p.UpdateDoublesElo(1150)
	if p.DoublesElo != 1150 {
		t.Errorf("expected DoublesElo 1150, got %d", p.DoublesElo)
	}

	p.UpdateDoublesElo(0)
	if p.DoublesElo != 0 {
		t.Errorf("expected DoublesElo 0, got %d", p.DoublesElo)
	}

	// Negative values should be ignored
	p.UpdateDoublesElo(-100)
	if p.DoublesElo != 0 {
		t.Errorf("expected DoublesElo to remain 0, got %d", p.DoublesElo)
	}
}

func TestPlayer_NameHelpers(t *testing.T) {
	p := &player.Player{
		FirstName:      "John",
		SecondName:     "Robert",
		LastName:       "Doe",
		SecondLastName: "Smith",
	}

	if p.FullName() != "John Doe" {
		t.Errorf("expected FullName 'John Doe', got '%s'", p.FullName())
	}

	if p.FirstNameWithSecond() != "John Robert" {
		t.Errorf("expected FirstNameWithSecond 'John Robert', got '%s'", p.FirstNameWithSecond())
	}

	if p.LastNameWithSecond() != "Doe Smith" {
		t.Errorf("expected LastNameWithSecond 'Doe Smith', got '%s'", p.LastNameWithSecond())
	}

	// Test without second names
	pNoSecond := &player.Player{
		FirstName: "Jane",
		LastName:  "Austin",
	}

	if pNoSecond.FirstNameWithSecond() != "Jane" {
		t.Errorf("expected FirstNameWithSecond 'Jane', got '%s'", pNoSecond.FirstNameWithSecond())
	}

	if pNoSecond.LastNameWithSecond() != "Austin" {
		t.Errorf("expected LastNameWithSecond 'Austin', got '%s'", pNoSecond.LastNameWithSecond())
	}
}

func TestNewGuardianChildPlayer(t *testing.T) {
	bdate := time.Date(2012, time.March, 1, 0, 0, 0, 0, time.UTC)

	t.Run("happy path sets GuardianAccountID", func(t *testing.T) {
		p, err := player.NewGuardianChildPlayer("p-2", "acc-1", "Kid", "Smith", bdate, "F", "USA", "Dept1")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.GuardianAccountID == nil || *p.GuardianAccountID != "acc-1" {
			t.Fatalf("expected GuardianAccountID 'acc-1', got %v", p.GuardianAccountID)
		}
		if p.SinglesElo != 1000 || p.DoublesElo != 1000 {
			t.Errorf("expected starting Elo 1000/1000, got %d/%d", p.SinglesElo, p.DoublesElo)
		}
		if p.Gender != "F" {
			t.Errorf("expected Gender 'F', got %q", p.Gender)
		}
	})

	t.Run("defaults gender to M when empty", func(t *testing.T) {
		p, err := player.NewGuardianChildPlayer("p-3", "acc-1", "Kid", "Smith", bdate, "", "USA", "")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if p.Gender != "M" {
			t.Errorf("expected default Gender 'M', got %q", p.Gender)
		}
	})

	t.Run("missing names error", func(t *testing.T) {
		if _, err := player.NewGuardianChildPlayer("p-4", "acc-1", "", "Smith", bdate, "M", "USA", ""); err != player.ErrInvalidName {
			t.Fatalf("expected ErrInvalidName, got %v", err)
		}
		if _, err := player.NewGuardianChildPlayer("p-5", "acc-1", "Kid", "", bdate, "M", "USA", ""); err != player.ErrInvalidName {
			t.Fatalf("expected ErrInvalidName, got %v", err)
		}
	})

	t.Run("NewPlayer keeps GuardianAccountID nil", func(t *testing.T) {
		p, err := player.NewPlayer("p-6", "John", "Doe", bdate, "M", "USA", "Dept1", "NID123")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.GuardianAccountID != nil {
			t.Errorf("expected nil GuardianAccountID for NewPlayer, got %v", p.GuardianAccountID)
		}
	})
}

func TestPlayer_EloFor_And_UpdateEloFor(t *testing.T) {
	newTestPlayer := func() *player.Player {
		p, err := player.NewPlayer("p-elo", "Age", "Category", time.Now(), "M", "USA", "", "")
		if err != nil {
			t.Fatalf("NewPlayer: %v", err)
		}
		return p
	}

	t.Run("every bracket defaults to 1000 for both rank types", func(t *testing.T) {
		p := newTestPlayer()
		for _, ageCategory := range []string{"", "open", "u11", "u13", "u15", "u19", "unknown"} {
			for _, rankType := range []string{"singles", "doubles"} {
				if got := p.EloFor(ageCategory, rankType); got != 1000 {
					t.Errorf("EloFor(%q, %q) = %d, want 1000", ageCategory, rankType, got)
				}
			}
		}
	})

	t.Run("UpdateEloFor targets only the given bracket and rank type", func(t *testing.T) {
		p := newTestPlayer()
		p.UpdateEloFor("u13", "singles", 1200)

		if got := p.EloFor("u13", "singles"); got != 1200 {
			t.Errorf("expected u13 singles 1200, got %d", got)
		}
		if p.SinglesEloU13 != 1200 {
			t.Errorf("expected SinglesEloU13 field to be 1200, got %d", p.SinglesEloU13)
		}
		// Every other bracket/rank-type combo, including Open, is untouched.
		if got := p.EloFor("u13", "doubles"); got != 1000 {
			t.Errorf("expected u13 doubles untouched at 1000, got %d", got)
		}
		if got := p.EloFor("open", "singles"); got != 1000 {
			t.Errorf("expected Open singles untouched at 1000, got %d", got)
		}
		if got := p.EloFor("u11", "singles"); got != 1000 {
			t.Errorf("expected u11 singles untouched at 1000, got %d", got)
		}
	})

	t.Run("unknown or empty age category routes to Open SinglesElo/DoublesElo", func(t *testing.T) {
		p := newTestPlayer()
		p.UpdateEloFor("", "singles", 1111)
		p.UpdateEloFor("open", "doubles", 2222)
		if p.SinglesElo != 1111 {
			t.Errorf("expected SinglesElo 1111, got %d", p.SinglesElo)
		}
		if p.DoublesElo != 2222 {
			t.Errorf("expected DoublesElo 2222, got %d", p.DoublesElo)
		}
	})

	t.Run("all four brackets are independently addressable", func(t *testing.T) {
		p := newTestPlayer()
		p.UpdateEloFor("u11", "singles", 900)
		p.UpdateEloFor("u13", "singles", 1000)
		p.UpdateEloFor("u15", "singles", 1100)
		p.UpdateEloFor("u19", "singles", 1200)

		if p.EloFor("u11", "singles") != 900 || p.EloFor("u13", "singles") != 1000 ||
			p.EloFor("u15", "singles") != 1100 || p.EloFor("u19", "singles") != 1200 {
			t.Errorf("expected each bracket to hold its own value, got u11=%d u13=%d u15=%d u19=%d",
				p.EloFor("u11", "singles"), p.EloFor("u13", "singles"), p.EloFor("u15", "singles"), p.EloFor("u19", "singles"))
		}
	})

	t.Run("doubles is independently addressable for every bracket", func(t *testing.T) {
		for _, ac := range []string{"u11", "u13", "u15", "u19"} {
			p := newTestPlayer()
			p.UpdateEloFor(ac, "doubles", 1234)
			if got := p.EloFor(ac, "doubles"); got != 1234 {
				t.Errorf("EloFor(%q, doubles) = %d, want 1234", ac, got)
			}
			if got := p.EloFor(ac, "singles"); got != 1000 {
				t.Errorf("expected %s singles untouched by a doubles update, got %d", ac, got)
			}
		}
	})

	t.Run("negative Elo is rejected, mirroring UpdateSinglesElo/UpdateDoublesElo", func(t *testing.T) {
		p := newTestPlayer()
		p.UpdateEloFor("u13", "singles", -5)
		if p.EloFor("u13", "singles") != 1000 {
			t.Errorf("expected negative update to be ignored, got %d", p.EloFor("u13", "singles"))
		}
	})
}
