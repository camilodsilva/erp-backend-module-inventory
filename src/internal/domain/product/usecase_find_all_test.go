package product

import "testing"

func TestFindAllProductUseCase_Execute_Success(t *testing.T) {
	expected := Page{
		Products:   []Product{{ID: "product-id", Title: "Camiseta"}},
		Page:       2,
		Size:       5,
		TotalPages: 3,
		Total:      11,
	}
	repo := &MockProductRepository{
		FindAllFn: func(tenantID string, page, size int) (Page, error) {
			if tenantID != "tenant-id" {
				t.Errorf("expected tenant-id, got %s", tenantID)
			}
			if page != 2 || size != 5 {
				t.Errorf("expected page=2 size=5, got page=%d size=%d", page, size)
			}
			return expected, nil
		},
	}
	uc := NewFindAllUseCase(repo)

	got, err := uc.Execute("tenant-id", 2, 5)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Total != expected.Total || len(got.Products) != 1 {
		t.Errorf("unexpected page: %+v", got)
	}
}

func TestFindAllProductUseCase_Execute_EmptyList(t *testing.T) {
	expected := Page{Products: []Product{}, Page: 1, Size: 10, TotalPages: 1, Total: 0}
	repo := &MockProductRepository{
		FindAllFn: func(tenantID string, page, size int) (Page, error) {
			return expected, nil
		},
	}
	uc := NewFindAllUseCase(repo)

	got, err := uc.Execute("tenant-id", 1, 10)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got.Total != 0 || len(got.Products) != 0 {
		t.Errorf("expected empty page, got %+v", got)
	}
}
