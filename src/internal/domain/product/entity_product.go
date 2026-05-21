package product

import (
	"errors"
	"time"
)

var (
	ErrProductNotFound                = errors.New("product not found")
	ErrProductAlreadyExists           = errors.New("product with this SKU already exists")
	ErrTitleRequired                  = errors.New("title is required")
	ErrTitleTooLong                   = errors.New("title must have at most 120 characters")
	ErrSKURequired                    = errors.New("sku is required")
	ErrSKUTooLong                     = errors.New("sku must have at most 60 characters")
	ErrUnitRequired                   = errors.New("unit is required")
	ErrUnitTooLong                    = errors.New("unit must have at most 6 characters")
	ErrUnitPriceRequired              = errors.New("unit_price is required")
	ErrUnitPriceInvalid               = errors.New("unit_price must be greater than or equal to 0")
	ErrStockQuantityInvalid           = errors.New("stock_quantity must be greater than or equal to 0")
	ErrEANInvalid                     = errors.New("ean must contain 8, 13 or 14 digits")
	ErrFiscalProfileExternalIDInvalid = errors.New("fiscal_profile_external_id is not a valid UUID")
	ErrProductIDInvalid               = errors.New("product id is not a valid UUID")
)

type (
	Product struct {
		ID                      string
		Title                   string
		Description             string
		SKU                     string
		EAN                     string
		Unit                    string
		UnitPrice               float64
		StockQuantity           float64
		IsActive                bool
		FiscalProfileExternalID string
		CreatedBy               string
		UpdatedBy               string
		CreatedAt               time.Time
		UpdatedAt               time.Time
		DeletedAt               *time.Time
		DeletedBy               string
	}

	Page struct {
		Products   []Product
		Page       int
		Size       int
		TotalPages int
		Total      int
	}

	Repository interface {
		Create(tenantID string, p Product) (Product, error)
		FindAll(tenantID string, page, size int, q string) (Page, error)
		FindByID(tenantID, id string) (Product, error)
		Update(tenantID string, p Product) (Product, error)
		SoftDelete(tenantID, id, deletedBy string) error
	}

	IDGenerator interface {
		Generate() string
	}
)

func (p Product) Update(draft Draft, actorID string) Product {
	p.Title = draft.Title
	p.Description = draft.Description
	p.SKU = draft.SKU
	p.EAN = draft.EAN
	p.Unit = draft.Unit
	p.UnitPrice = draft.UnitPrice
	p.StockQuantity = draft.StockQuantity
	p.FiscalProfileExternalID = draft.FiscalProfileExternalID
	p.UpdatedBy = actorID
	return p
}
