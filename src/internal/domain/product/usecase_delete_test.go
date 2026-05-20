package product

import (
	"errors"
	"testing"
)

func TestDeleteProductUseCase_Execute_Success(t *testing.T) {
	repo := &MockProductRepository{
		SoftDeleteFn: func(tenantID, id, deletedBy string) error {
			if tenantID != "tenant-id" {
				t.Errorf("expected tenant-id, got %s", tenantID)
			}
			if id != "product-id" {
				t.Errorf("expected product-id, got %s", id)
			}
			if deletedBy != "actor-id" {
				t.Errorf("expected actor-id, got %s", deletedBy)
			}
			return nil
		},
	}
	uc := NewDeleteUseCase(repo)

	if err := uc.Execute("tenant-id", "product-id", "actor-id"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDeleteProductUseCase_Execute_NotFound(t *testing.T) {
	repo := &MockProductRepository{
		SoftDeleteFn: func(tenantID, id, deletedBy string) error {
			return ErrProductNotFound
		},
	}
	uc := NewDeleteUseCase(repo)

	err := uc.Execute("tenant-id", "missing-id", "actor-id")
	if !errors.Is(err, ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}
