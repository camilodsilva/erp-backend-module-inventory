package product

import (
	"errors"
	"testing"
)

func TestCreateProductUseCase_Execute_Success(t *testing.T) {
	expected := Product{ID: "uuid-1", Title: "Camiseta", SKU: "CAM-P", Unit: "UN", UnitPrice: 49.90, IsActive: true}
	repo := &MockProductRepository{
		CreateFn: func(tenantID string, p Product) (Product, error) {
			if tenantID != "tenant-id" {
				t.Errorf("expected tenant-id, got %s", tenantID)
			}
			if p.ID != "uuid-1" || p.CreatedBy != "actor-id" || p.UpdatedBy != "actor-id" {
				t.Errorf("expected generated id and audit fields, got %+v", p)
			}
			return expected, nil
		},
	}
	factory := NewProductFactory(&stubIDGenerator{id: "uuid-1"})
	uc := NewCreateUseCase(repo, factory)

	draft, _ := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "01012100", "0", nil)
	got, err := uc.Execute("tenant-id", "actor-id", draft)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected ID %s, got %s", expected.ID, got.ID)
	}
}

func TestCreateProductUseCase_Execute_AlreadyExists(t *testing.T) {
	repo := &MockProductRepository{
		CreateFn: func(tenantID string, p Product) (Product, error) {
			return Product{}, ErrProductAlreadyExists
		},
	}
	factory := NewProductFactory(&stubIDGenerator{id: "uuid-1"})
	uc := NewCreateUseCase(repo, factory)

	draft, _ := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "01012100", "0", nil)
	_, err := uc.Execute("tenant-id", "actor-id", draft)
	if !errors.Is(err, ErrProductAlreadyExists) {
		t.Errorf("expected ErrProductAlreadyExists, got %v", err)
	}
}

type stubIDGenerator struct{ id string }

func (g *stubIDGenerator) Generate() string { return g.id }
