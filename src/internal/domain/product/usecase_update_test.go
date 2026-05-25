package product

import (
	"errors"
	"testing"
)

func TestUpdateProductUseCase_Execute_Success(t *testing.T) {
	repo := &MockProductRepository{
		FindByIDFn: func(tenantID, id string) (Product, error) {
			return Product{ID: id, Title: "Antiga", SKU: "OLD", Unit: "UN", UnitPrice: 10, CreatedBy: "creator"}, nil
		},
		UpdateFn: func(tenantID string, p Product) (Product, error) {
			if p.Title != "Nova" || p.SKU != "NEW" || p.UpdatedBy != "actor-id" {
				t.Errorf("expected updated domain state, got %+v", p)
			}
			return p, nil
		},
	}
	uc := NewUpdateUseCase(repo)
	draft, _ := NewDraft("Nova", "", "NEW", "", "UN", 20, 5, "", "01012100", "0", nil)

	got, err := uc.Execute("tenant-id", "product-id", "actor-id", draft)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Title != "Nova" {
		t.Errorf("expected Nova, got %s", got.Title)
	}
}

func TestUpdateProductUseCase_Execute_NotFound(t *testing.T) {
	repo := &MockProductRepository{
		FindByIDFn: func(tenantID, id string) (Product, error) {
			return Product{}, ErrProductNotFound
		},
	}
	uc := NewUpdateUseCase(repo)
	draft, _ := NewDraft("Nova", "", "NEW", "", "UN", 20, 5, "", "01012100", "0", nil)

	_, err := uc.Execute("tenant-id", "missing-id", "actor-id", draft)
	if !errors.Is(err, ErrProductNotFound) {
		t.Errorf("expected ErrProductNotFound, got %v", err)
	}
}

func TestUpdateProductUseCase_Execute_UpdateError(t *testing.T) {
	updateErr := errors.New("update failed")
	repo := &MockProductRepository{
		FindByIDFn: func(tenantID, id string) (Product, error) {
			return Product{ID: id, Title: "Antiga", SKU: "OLD", Unit: "UN", UnitPrice: 10}, nil
		},
		UpdateFn: func(tenantID string, p Product) (Product, error) {
			return Product{}, updateErr
		},
	}
	uc := NewUpdateUseCase(repo)
	draft, _ := NewDraft("Nova", "", "NEW", "", "UN", 20, 5, "", "01012100", "0", nil)

	_, err := uc.Execute("tenant-id", "product-id", "actor-id", draft)
	if !errors.Is(err, updateErr) {
		t.Errorf("expected update error, got %v", err)
	}
}
