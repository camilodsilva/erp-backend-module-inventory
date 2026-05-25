package dto

import (
	"errors"
	"time"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/product"
)

type (
	CreateProductRequest struct {
		Title                   string   `json:"title"`
		Description             string   `json:"description"`
		SKU                     string   `json:"sku"`
		EAN                     string   `json:"ean"`
		Unit                    string   `json:"unit"`
		UnitPrice               *float64 `json:"unit_price"`
		StockQuantity           float64  `json:"stock_quantity"`
		FiscalProfileExternalID string   `json:"fiscal_profile_external_id"`
		NCM                     string   `json:"ncm"`
		Origin                  string   `json:"origin"`
		CEST                    *string  `json:"cest"`
	}

	UpdateProductRequest struct {
		Title                   string   `json:"title"`
		Description             string   `json:"description"`
		SKU                     string   `json:"sku"`
		EAN                     string   `json:"ean"`
		Unit                    string   `json:"unit"`
		UnitPrice               *float64 `json:"unit_price"`
		StockQuantity           float64  `json:"stock_quantity"`
		FiscalProfileExternalID string   `json:"fiscal_profile_external_id"`
		NCM                     string   `json:"ncm"`
		Origin                  string   `json:"origin"`
		CEST                    *string  `json:"cest"`
	}

	ProductResponse struct {
		ID                      string  `json:"id"`
		Title                   string  `json:"title"`
		Description             string  `json:"description,omitempty"`
		SKU                     string  `json:"sku"`
		EAN                     string  `json:"ean,omitempty"`
		Unit                    string  `json:"unit"`
		UnitPrice               float64 `json:"unit_price"`
		StockQuantity           float64 `json:"stock_quantity"`
		IsActive                bool    `json:"is_active"`
		FiscalProfileExternalID string  `json:"fiscal_profile_external_id,omitempty"`
		NCM                     string  `json:"ncm"`
		Origin                  string  `json:"origin"`
		CEST                    *string `json:"cest,omitempty"`
		CreatedAt               string  `json:"created_at"`
		UpdatedAt               string  `json:"updated_at"`
	}
)

func (r CreateProductRequest) ToDraft() (product.Draft, error) {
	if r.UnitPrice == nil {
		return product.Draft{}, product.ErrUnitPriceRequired
	}

	draft, errs := product.NewDraft(
		r.Title, r.Description, r.SKU, r.EAN, r.Unit,
		*r.UnitPrice, r.StockQuantity,
		r.FiscalProfileExternalID,
		r.NCM, r.Origin, r.CEST,
	)
	if len(errs) > 0 {
		return product.Draft{}, errors.Join(errs...)
	}
	return draft, nil
}

func (r UpdateProductRequest) ToDraft() (product.Draft, error) {
	if r.UnitPrice == nil {
		return product.Draft{}, product.ErrUnitPriceRequired
	}

	draft, errs := product.NewDraft(
		r.Title, r.Description, r.SKU, r.EAN, r.Unit,
		*r.UnitPrice, r.StockQuantity,
		r.FiscalProfileExternalID,
		r.NCM, r.Origin, r.CEST,
	)
	if len(errs) > 0 {
		return product.Draft{}, errors.Join(errs...)
	}
	return draft, nil
}

func NewProductResponse(p product.Product) ProductResponse {
	return ProductResponse{
		ID:                      p.ID,
		Title:                   p.Title,
		Description:             p.Description,
		SKU:                     p.SKU,
		EAN:                     p.EAN,
		Unit:                    p.Unit,
		UnitPrice:               p.UnitPrice,
		StockQuantity:           p.StockQuantity,
		IsActive:                p.IsActive,
		FiscalProfileExternalID: p.FiscalProfileExternalID,
		NCM:                     p.NCM,
		Origin:                  p.Origin,
		CEST:                    p.CEST,
		CreatedAt:               p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:               p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

func NewProductPaginated(page product.Page) Paginate[ProductResponse] {
	data := make([]ProductResponse, len(page.Products))
	for i, p := range page.Products {
		data[i] = NewProductResponse(p)
	}
	return Paginate[ProductResponse]{
		Data:       data,
		Page:       page.Page,
		Size:       page.Size,
		TotalPages: page.TotalPages,
		Total:      page.Total,
	}
}
