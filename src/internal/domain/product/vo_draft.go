package product

import (
	"strings"

	"github.com/google/uuid"
)

type Draft struct {
	Title                   string
	Description             string
	SKU                     string
	EAN                     string
	Unit                    string
	UnitPrice               float64
	StockQuantity           float64
	FiscalProfileExternalID string
}

func NewDraft(
	title, description, sku, ean, unit string,
	unitPrice, stockQuantity float64,
	fiscalProfileExternalID string,
) (Draft, []error) {
	var errs []error

	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	sku = strings.ToUpper(strings.TrimSpace(sku))
	ean = strings.TrimSpace(ean)
	unit = strings.ToUpper(strings.TrimSpace(unit))
	fiscalProfileExternalID = strings.TrimSpace(fiscalProfileExternalID)

	if title == "" {
		errs = append(errs, ErrTitleRequired)
	} else if len(title) > 120 {
		errs = append(errs, ErrTitleTooLong)
	}

	if sku == "" {
		errs = append(errs, ErrSKURequired)
	} else if len(sku) > 60 {
		errs = append(errs, ErrSKUTooLong)
	}

	if unit == "" {
		errs = append(errs, ErrUnitRequired)
	} else if len(unit) > 6 {
		errs = append(errs, ErrUnitTooLong)
	}

	if unitPrice < 0 {
		errs = append(errs, ErrUnitPriceInvalid)
	}

	if stockQuantity < 0 {
		errs = append(errs, ErrStockQuantityInvalid)
	}

	if ean != "" && !isValidEAN(ean) {
		errs = append(errs, ErrEANInvalid)
	}

	if fiscalProfileExternalID != "" {
		if _, err := uuid.Parse(fiscalProfileExternalID); err != nil {
			errs = append(errs, ErrFiscalProfileExternalIDInvalid)
		}
	}

	if len(errs) > 0 {
		return Draft{}, errs
	}

	return Draft{
		Title:                   title,
		Description:             description,
		SKU:                     sku,
		EAN:                     ean,
		Unit:                    unit,
		UnitPrice:               unitPrice,
		StockQuantity:           stockQuantity,
		FiscalProfileExternalID: fiscalProfileExternalID,
	}, nil
}

func isValidEAN(ean string) bool {
	if len(ean) != 8 && len(ean) != 13 && len(ean) != 14 {
		return false
	}
	for _, ch := range ean {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
