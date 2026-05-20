package product

import (
	"errors"
	"testing"
)

func TestFindByIDProductUseCase_Execute_Success(t *testing.T) {
	expected := Product{ID: "product-id", Title: "Camiseta"}
	repo := &MockProductRepository{
		FindByIDFn: func(tenantID, id string) (Product, error) {
			if tenantID != "tenant-id" {
				t.Errorf("expected tenant-id, got %s", tenantID)
			}
			if id != "product-id" {
				t.Errorf("expected product-id, got %s", id)
			}
			return expected, nil
		},
	}
	uc := NewFindByIDUseCase(repo)

	got, err := uc.Execute("tenant-id", "product-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.ID != expected.ID {
		t.Errorf("expected %s, got %s", expected.ID, got.ID)
	}
}

func TestFindByIDProductUseCase_Execute_NotFound(t *testing.T) {
	repo := &MockProductRepository{
		FindByIDFn: func(tenantID, id string) (Product, error) {
			return Product{}, ErrProductNotFound
		},
	}
	uc := NewFindByIDUseCase(repo)

	_, err := uc.Execute("tenant-id", "missing-id")
	if !errors.Is(err, ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}
