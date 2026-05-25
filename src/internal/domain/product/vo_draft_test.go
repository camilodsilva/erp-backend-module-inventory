package product

import (
	"errors"
	"testing"
)

func validNCM() string   { return "01012100" }
func validOrigin() string { return "0" }

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
		validNCM(),
		validOrigin(),
		nil,
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
	if draft.NCM != validNCM() {
		t.Errorf("expected NCM %s, got %s", validNCM(), draft.NCM)
	}
	if draft.Origin != validOrigin() {
		t.Errorf("expected Origin %s, got %s", validOrigin(), draft.Origin)
	}
	if draft.CEST != nil {
		t.Errorf("expected nil CEST, got %v", draft.CEST)
	}
}

func TestProductDraft_NewDraft_SKUNormalized(t *testing.T) {
	draft, errs := NewDraft("Camiseta", "", " cam-p ", "", "UN", 49.90, 0, "", validNCM(), validOrigin(), nil)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.SKU != "CAM-P" {
		t.Errorf("expected CAM-P, got %s", draft.SKU)
	}
}

func TestProductDraft_NewDraft_UnitNormalized(t *testing.T) {
	draft, errs := NewDraft("Camiseta", "", "CAM-P", "", " un ", 49.90, 0, "", validNCM(), validOrigin(), nil)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.Unit != "UN" {
		t.Errorf("expected UN, got %s", draft.Unit)
	}
}

func TestProductDraft_NewDraft_TitleRequired(t *testing.T) {
	_, errs := NewDraft("", "", "CAM-P", "", "UN", 49.90, 0, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrTitleRequired)
}

func TestProductDraft_NewDraft_SKURequired(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "", "", "UN", 49.90, 0, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrSKURequired)
}

func TestProductDraft_NewDraft_UnitRequired(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "", 49.90, 0, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrUnitRequired)
}

func TestProductDraft_NewDraft_UnitPriceNegative(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", -1, 0, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrUnitPriceInvalid)
}

func TestProductDraft_NewDraft_StockQuantityNegative(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, -1, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrStockQuantityInvalid)
}

func TestProductDraft_NewDraft_EANInvalidLength(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "123456789", "UN", 49.90, 0, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrEANInvalid)
}

func TestProductDraft_NewDraft_EANInvalidCharacter(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "789123456789A", "UN", 49.90, 0, "", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrEANInvalid)
}

func TestProductDraft_NewDraft_FiscalProfileExternalIDInvalid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "invalid", validNCM(), validOrigin(), nil)
	assertProductDraftError(t, errs, ErrFiscalProfileExternalIDInvalid)
}

func TestProductDraft_NewDraft_NCMInvalid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "1234567", validOrigin(), nil)
	assertProductDraftError(t, errs, ErrNCMInvalid)
}

func TestProductDraft_NewDraft_NCMInvalidCharacter(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "0101A100", validOrigin(), nil)
	assertProductDraftError(t, errs, ErrNCMInvalid)
}

func TestProductDraft_NewDraft_OriginInvalid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", validNCM(), "9", nil)
	assertProductDraftError(t, errs, ErrOriginInvalid)
}

func TestProductDraft_NewDraft_CESTInvalid(t *testing.T) {
	cest := "123456"
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", validNCM(), validOrigin(), &cest)
	assertProductDraftError(t, errs, ErrCESTInvalid)
}

func TestProductDraft_NewDraft_CESTValid(t *testing.T) {
	cest := " 1234567 "
	draft, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", validNCM(), validOrigin(), &cest)
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.CEST == nil || *draft.CEST != "1234567" {
		t.Errorf("expected trimmed CEST '1234567', got %v", draft.CEST)
	}
}

func TestProductDraft_NewDraft_MultipleErrors(t *testing.T) {
	_, errs := NewDraft("", "", "", "123", "", -1, -1, "invalid", "bad", "9", nil)

	expected := []error{
		ErrTitleRequired,
		ErrSKURequired,
		ErrUnitRequired,
		ErrUnitPriceInvalid,
		ErrStockQuantityInvalid,
		ErrEANInvalid,
		ErrFiscalProfileExternalIDInvalid,
		ErrNCMInvalid,
		ErrOriginInvalid,
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
