package leaderboard_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/leaderboard"
	"table-tennis-backend/internal/domain/player"
)

type mockPlayerRepo struct {
	singles        []*player.Player
	doubles        []*player.Player
	singlesGender  map[string][]*player.Player
	doublesGender  map[string][]*player.Player
	errToReturn    error
}

func newMockPlayerRepo() *mockPlayerRepo {
	return &mockPlayerRepo{
		singlesGender: make(map[string][]*player.Player),
		doublesGender: make(map[string][]*player.Player),
	}
}

func (m *mockPlayerRepo) GetById(ctx context.Context, id string) (*player.Player, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPlayerRepo) GetByIDs(ctx context.Context, ids []string) ([]*player.Player, error) {
	return nil, errors.New("not implemented")
}

func (m *mockPlayerRepo) Save(ctx context.Context, p *player.Player) error {
	return nil
}

func (m *mockPlayerRepo) SaveMultiple(ctx context.Context, players []*player.Player) error {
	return nil
}

func (m *mockPlayerRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func (m *mockPlayerRepo) Search(ctx context.Context, query string) ([]*player.Player, error) {
	return nil, nil
}

func (m *mockPlayerRepo) SearchForSelection(ctx context.Context, query, gender string) ([]*player.Player, error) {
	return nil, nil
}

func (m *mockPlayerRepo) GetAll(ctx context.Context) ([]*player.Player, error) {
	return m.singles, m.errToReturn
}

func (m *mockPlayerRepo) GetAllSingles(ctx context.Context) ([]*player.Player, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.singles, nil
}

func (m *mockPlayerRepo) GetAllDoubles(ctx context.Context) ([]*player.Player, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.doubles, nil
}

func (m *mockPlayerRepo) GetSinglesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.singlesGender[gender], nil
}

func (m *mockPlayerRepo) GetDoublesByGender(ctx context.Context, gender string) ([]*player.Player, error) {
	if m.errToReturn != nil {
		return nil, m.errToReturn
	}
	return m.doublesGender[gender], nil
}

func TestGetLeaderboardUseCase_Execute(t *testing.T) {
	repo := newMockPlayerRepo()
	p1 := &player.Player{ID: "1", FirstName: "Alice", SinglesElo: 1200}
	p2 := &player.Player{ID: "2", FirstName: "Bob", DoublesElo: 1100}
	repo.singles = []*player.Player{p1}
	repo.doubles = []*player.Player{p2}

	uc := leaderboard.NewGetLeaderboardUseCase(repo)
	ctx := context.Background()

	t.Run("singles ranking", func(t *testing.T) {
		res, err := uc.Execute(ctx, "singles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].ID != "1" {
			t.Errorf("expected player 1, got %v", res)
		}
	})

	t.Run("doubles ranking", func(t *testing.T) {
		res, err := uc.Execute(ctx, "doubles")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].ID != "2" {
			t.Errorf("expected player 2, got %v", res)
		}
	})
}

func TestGetLeaderboardUseCase_ExecuteByGender(t *testing.T) {
	repo := newMockPlayerRepo()
	pMale := &player.Player{ID: "1", FirstName: "John", Gender: "M"}
	pFemale := &player.Player{ID: "2", FirstName: "Jane", Gender: "F"}

	repo.singlesGender["M"] = []*player.Player{pMale}
	repo.doublesGender["F"] = []*player.Player{pFemale}

	uc := leaderboard.NewGetLeaderboardUseCase(repo)
	ctx := context.Background()

	t.Run("singles male", func(t *testing.T) {
		res, err := uc.ExecuteByGender(ctx, "singles", "M")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].Gender != "M" {
			t.Errorf("expected male player, got %v", res)
		}
	})

	t.Run("doubles female", func(t *testing.T) {
		res, err := uc.ExecuteByGender(ctx, "doubles", "F")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(res) != 1 || res[0].Gender != "F" {
			t.Errorf("expected female player, got %v", res)
		}
	})
}

func TestGetLeaderboardUseCase_ExecuteSingles(t *testing.T) {
	repo := newMockPlayerRepo()
	p1 := &player.Player{ID: "1", FirstName: "Alice"}
	repo.singles = []*player.Player{p1}

	uc := leaderboard.NewGetLeaderboardUseCase(repo)
	ctx := context.Background()

	res, err := uc.ExecuteSingles(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) != 1 || res[0].ID != "1" {
		t.Errorf("expected player 1, got %v", res)
	}
}
