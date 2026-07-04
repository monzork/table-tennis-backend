package division_test

import (
	"context"
	"errors"
	"testing"

	"table-tennis-backend/internal/application/division"
	domain "table-tennis-backend/internal/domain/division"
	"table-tennis-backend/internal/domain/idgen"
	"table-tennis-backend/internal/infrastructure/identity"
)

func init() {
	idgen.Register(identity.NewUUIDGenerator())
}

type mockDivisionRepo struct {
	divisions map[string]*domain.Division
}

func newMockRepo() *mockDivisionRepo {
	return &mockDivisionRepo{
		divisions: make(map[string]*domain.Division),
	}
}

func (m *mockDivisionRepo) Save(ctx context.Context, d *domain.Division) error {
	m.divisions[d.ID] = d
	return nil
}

func (m *mockDivisionRepo) GetById(ctx context.Context, id string) (*domain.Division, error) {
	d, ok := m.divisions[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return d, nil
}

func (m *mockDivisionRepo) GetAll(ctx context.Context) ([]*domain.Division, error) {
	res := make([]*domain.Division, 0, len(m.divisions))
	for _, d := range m.divisions {
		res = append(res, d)
	}
	return res, nil
}

func (m *mockDivisionRepo) Delete(ctx context.Context, id string) error {
	delete(m.divisions, id)
	return nil
}

func TestDivisionUseCase_Save(t *testing.T) {
	repo := newMockRepo()
	uc := division.NewDivisionUseCase(repo)
	ctx := context.Background()

	err := uc.Save(ctx, "", "Div 1", 1, 1000, nil, "singles", "#ff0000")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	divs, _ := repo.GetAll(ctx)
	if len(divs) != 1 {
		t.Fatalf("expected 1 division, got %d", len(divs))
	}
	if divs[0].Name != "Div 1" {
		t.Errorf("expected Div 1, got %s", divs[0].Name)
	}
}

func TestDivisionUseCase_GetAll(t *testing.T) {
	repo := newMockRepo()
	uc := division.NewDivisionUseCase(repo)
	ctx := context.Background()

	_ = uc.Save(ctx, "", "Div 1", 1, 1000, nil, "", "")
	_ = uc.Save(ctx, "", "Div 2", 2, 800, nil, "", "")

	res, err := uc.GetAll(ctx)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res) != 2 {
		t.Fatalf("expected 2 divisions, got %d", len(res))
	}
}

func TestDivisionUseCase_Delete(t *testing.T) {
	repo := newMockRepo()
	uc := division.NewDivisionUseCase(repo)
	ctx := context.Background()

	_ = uc.Save(ctx, "id1", "Div 1", 1, 1000, nil, "", "")

	err := uc.Delete(ctx, "id1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	divs, _ := repo.GetAll(ctx)
	if len(divs) != 0 {
		t.Fatalf("expected 0 divisions, got %d", len(divs))
	}
}
