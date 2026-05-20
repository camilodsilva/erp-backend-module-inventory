package product

import (
	"errors"
	"testing"
)

func TestProductDraft_NewDraft_Success(t *testing.T) {
	draft, errs := NewDraft(
		" Camiseta Branca P ",
		" 100% algodao ",
		" cam-bra-p ",
		"7891234567890",
		" un ",
		49.90,
		100,
		"01960e4a-3f2b-7d1c-8e5f-9a0b1c2d3e4f",
	)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.Title != "Camiseta Branca P" {
		t.Errorf("expected trimmed title, got %q", draft.Title)
	}
	if draft.Description != "100% algodao" {
		t.Errorf("expected trimmed description, got %q", draft.Description)
	}
	if draft.SKU != "CAM-BRA-P" {
		t.Errorf("expected normalized SKU, got %q", draft.SKU)
	}
	if draft.Unit != "UN" {
		t.Errorf("expected normalized unit, got %q", draft.Unit)
	}
}

func TestProductDraft_NewDraft_SKUNormalized(t *testing.T) {
	draft, errs := NewDraft("Camiseta", "", " cam-p ", "", "UN", 49.90, 0, "")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.SKU != "CAM-P" {
		t.Errorf("expected CAM-P, got %s", draft.SKU)
	}
}

func TestProductDraft_NewDraft_UnitNormalized(t *testing.T) {
	draft, errs := NewDraft("Camiseta", "", "CAM-P", "", " un ", 49.90, 0, "")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.Unit != "UN" {
		t.Errorf("expected UN, got %s", draft.Unit)
	}
}

func TestProductDraft_NewDraft_TitleRequired(t *testing.T) {
	_, errs := NewDraft("", "", "CAM-P", "", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrTitleRequired)
}

func TestProductDraft_NewDraft_SKURequired(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "", "", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrSKURequired)
}

func TestProductDraft_NewDraft_UnitRequired(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrUnitRequired)
}

func TestProductDraft_NewDraft_UnitPriceNegative(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", -1, 0, "")
	assertProductDraftError(t, errs, ErrUnitPriceInvalid)
}

func TestProductDraft_NewDraft_StockQuantityNegative(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, -1, "")
	assertProductDraftError(t, errs, ErrStockQuantityInvalid)
}

func TestProductDraft_NewDraft_EANInvalidLength(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "123456789", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrEANInvalid)
}

func TestProductDraft_NewDraft_EANInvalidCharacter(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "789123456789A", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrEANInvalid)
}

func TestProductDraft_NewDraft_FiscalProfileExternalIDInvalid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "invalid")
	assertProductDraftError(t, errs, ErrFiscalProfileExternalIDInvalid)
}

func TestProductDraft_NewDraft_MultipleErrors(t *testing.T) {
	_, errs := NewDraft("", "", "", "123", "", -1, -1, "invalid")

	expected := []error{
		ErrTitleRequired,
		ErrSKURequired,
		ErrUnitRequired,
		ErrUnitPriceInvalid,
		ErrStockQuantityInvalid,
		ErrEANInvalid,
		ErrFiscalProfileExternalIDInvalid,
	}
	for _, expectedErr := range expected {
		assertProductDraftError(t, errs, expectedErr)
	}
}

func assertProductDraftError(t *testing.T, errs []error, expected error) {
	t.Helper()
	for _, err := range errs {
		if errors.Is(err, expected) {
			return
		}
	}
	t.Fatalf("expected %v in %v", expected, errs)
}
