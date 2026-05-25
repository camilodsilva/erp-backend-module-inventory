package postgres

import (
	"database/sql"
	"fmt"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/product"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/shared"
	"github.com/lib/pq"
)

const (
	createProductQuery = `
INSERT INTO %s.inventory_product (
    id, title, description, sku, ean, unit,
    unit_price, stock_quantity, is_active,
    fiscal_profile_external_id,
    ncm, origin, cest,
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10,
    $11, $12, $13,
    $14, $15
)
RETURNING id, title, description, sku, ean, unit,
          unit_price, stock_quantity, is_active,
          fiscal_profile_external_id,
          ncm, origin, cest,
          created_by, updated_by, created_at, updated_at, deleted_at, deleted_by`

	findAllProductsQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
       ncm, origin, cest,
       created_by, updated_by, created_at, updated_at, deleted_at, deleted_by
FROM %s.inventory_product
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

	findAllProductsWithSearchQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
       ncm, origin, cest,
       created_by, updated_by, created_at, updated_at, deleted_at, deleted_by
FROM %s.inventory_product
WHERE deleted_at IS NULL
  AND (title ILIKE $1 OR sku ILIKE $1)
ORDER BY created_at DESC
LIMIT $2 OFFSET $3`

	countProductsQuery = `
SELECT COUNT(*) FROM %s.inventory_product WHERE deleted_at IS NULL`

	countProductsWithSearchQuery = `
SELECT COUNT(*) FROM %s.inventory_product
WHERE deleted_at IS NULL
  AND (title ILIKE $1 OR sku ILIKE $1)`

	findProductByIDQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
       ncm, origin, cest,
       created_by, updated_by, created_at, updated_at, deleted_at, deleted_by
FROM %s.inventory_product
WHERE id = $1 AND deleted_at IS NULL`

	updateProductQuery = `
UPDATE %s.inventory_product
SET title = $1,
    description = $2,
    sku = $3,
    ean = $4,
    unit = $5,
    unit_price = $6,
    stock_quantity = $7,
    fiscal_profile_external_id = $8,
    ncm = $9,
    origin = $10,
    cest = $11,
    updated_by = $12,
    updated_at = now()
WHERE id = $13 AND deleted_at IS NULL
RETURNING id, title, description, sku, ean, unit,
          unit_price, stock_quantity, is_active,
          fiscal_profile_external_id,
          ncm, origin, cest,
          created_by, updated_by, created_at, updated_at, deleted_at, deleted_by`

	softDeleteProductQuery = `
UPDATE %s.inventory_product
SET deleted_at = now(),
    updated_at = now(),
    deleted_by = $1,
    updated_by = $2
WHERE id = $3 AND deleted_at IS NULL
RETURNING id`
)

type ProductPostgresRepository struct {
	db *sql.DB
}

func NewProductPostgresRepository(db *sql.DB) *ProductPostgresRepository {
	return &ProductPostgresRepository{db: db}
}

func (r *ProductPostgresRepository) Create(tenantID string, p product.Product) (product.Product, error) {
	schema := shared.SchemaName(tenantID)
	row := r.db.QueryRow(
		fmt.Sprintf(createProductQuery, schema),
		p.ID,
		p.Title,
		nullableString(p.Description),
		p.SKU,
		nullableString(p.EAN),
		p.Unit,
		p.UnitPrice,
		p.StockQuantity,
		p.IsActive,
		nullableString(p.FiscalProfileExternalID),
		p.NCM,
		p.Origin,
		p.CEST,
		p.CreatedBy,
		p.UpdatedBy,
	)

	created, err := scanProductRow(row)
	if err != nil {
		return product.Product{}, mapProductError(err)
	}

	return created, nil
}

func (r *ProductPostgresRepository) FindAll(tenantID string, page, size int, q string) (product.Page, error) {
	schema := shared.SchemaName(tenantID)
	page, size = normalizePagination(page, size)
	offset := (page - 1) * size

	var rows *sql.Rows
	var err error
	var total int

	if q == "" {
		rows, err = r.db.Query(fmt.Sprintf(findAllProductsQuery, schema), size, offset)
		if err != nil {
			return product.Page{}, err
		}
		if err := r.db.QueryRow(fmt.Sprintf(countProductsQuery, schema)).Scan(&total); err != nil {
			rows.Close()
			return product.Page{}, err
		}
	} else {
		pattern := fmt.Sprintf("%%%s%%", q)
		rows, err = r.db.Query(fmt.Sprintf(findAllProductsWithSearchQuery, schema), pattern, size, offset)
		if err != nil {
			return product.Page{}, err
		}
		if err := r.db.QueryRow(fmt.Sprintf(countProductsWithSearchQuery, schema), pattern).Scan(&total); err != nil {
			rows.Close()
			return product.Page{}, err
		}
	}
	defer rows.Close()

	products, err := scanProductRows(rows)
	if err != nil {
		return product.Page{}, err
	}

	return product.Page{
		Products:   products,
		Page:       page,
		Size:       size,
		TotalPages: calcTotalPages(total, size),
		Total:      total,
	}, nil
}

func (r *ProductPostgresRepository) FindByID(tenantID, id string) (product.Product, error) {
	schema := shared.SchemaName(tenantID)
	row := r.db.QueryRow(fmt.Sprintf(findProductByIDQuery, schema), id)

	p, err := scanProductRow(row)
	if err != nil {
		return product.Product{}, mapProductError(err)
	}

	return p, nil
}

func (r *ProductPostgresRepository) Update(tenantID string, p product.Product) (product.Product, error) {
	schema := shared.SchemaName(tenantID)
	row := r.db.QueryRow(
		fmt.Sprintf(updateProductQuery, schema),
		p.Title,
		nullableString(p.Description),
		p.SKU,
		nullableString(p.EAN),
		p.Unit,
		p.UnitPrice,
		p.StockQuantity,
		nullableString(p.FiscalProfileExternalID),
		p.NCM,
		p.Origin,
		p.CEST,
		p.UpdatedBy,
		p.ID,
	)

	updated, err := scanProductRow(row)
	if err != nil {
		return product.Product{}, mapProductError(err)
	}

	return updated, nil
}

func (r *ProductPostgresRepository) SoftDelete(tenantID, id, deletedBy string) error {
	schema := shared.SchemaName(tenantID)
	var deletedID string
	err := r.db.QueryRow(fmt.Sprintf(softDeleteProductQuery, schema), deletedBy, deletedBy, id).Scan(&deletedID)
	return mapProductError(err)
}

func scanProductRow(row *sql.Row) (product.Product, error) {
	var p product.Product
	var deletedAt sql.NullTime
	var description, ean, fiscalProfileExternalID, deletedBy sql.NullString
	var cest sql.NullString

	err := row.Scan(
		&p.ID,
		&p.Title,
		&description,
		&p.SKU,
		&ean,
		&p.Unit,
		&p.UnitPrice,
		&p.StockQuantity,
		&p.IsActive,
		&fiscalProfileExternalID,
		&p.NCM,
		&p.Origin,
		&cest,
		&p.CreatedBy,
		&p.UpdatedBy,
		&p.CreatedAt,
		&p.UpdatedAt,
		&deletedAt,
		&deletedBy,
	)
	if err != nil {
		return product.Product{}, err
	}

	if description.Valid {
		p.Description = description.String
	}
	if ean.Valid {
		p.EAN = ean.String
	}
	if fiscalProfileExternalID.Valid {
		p.FiscalProfileExternalID = fiscalProfileExternalID.String
	}
	if cest.Valid {
		p.CEST = &cest.String
	}
	if deletedAt.Valid {
		p.DeletedAt = &deletedAt.Time
	}
	if deletedBy.Valid {
		p.DeletedBy = deletedBy.String
	}

	return p, nil
}

func scanProductRows(rows *sql.Rows) ([]product.Product, error) {
	products := make([]product.Product, 0)

	for rows.Next() {
		var p product.Product
		var deletedAt sql.NullTime
		var description, ean, fiscalProfileExternalID, deletedBy sql.NullString
		var cest sql.NullString

		if err := rows.Scan(
			&p.ID,
			&p.Title,
			&description,
			&p.SKU,
			&ean,
			&p.Unit,
			&p.UnitPrice,
			&p.StockQuantity,
			&p.IsActive,
			&fiscalProfileExternalID,
			&p.NCM,
			&p.Origin,
			&cest,
			&p.CreatedBy,
			&p.UpdatedBy,
			&p.CreatedAt,
			&p.UpdatedAt,
			&deletedAt,
			&deletedBy,
		); err != nil {
			return nil, err
		}

		if description.Valid {
			p.Description = description.String
		}
		if ean.Valid {
			p.EAN = ean.String
		}
		if fiscalProfileExternalID.Valid {
			p.FiscalProfileExternalID = fiscalProfileExternalID.String
		}
		if cest.Valid {
			p.CEST = &cest.String
		}
		if deletedAt.Valid {
			p.DeletedAt = &deletedAt.Time
		}
		if deletedBy.Valid {
			p.DeletedBy = deletedBy.String
		}

		products = append(products, p)
	}

	return products, rows.Err()
}

func mapProductError(err error) error {
	if err == nil {
		return nil
	}
	if err == sql.ErrNoRows {
		return product.ErrProductNotFound
	}
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
		return product.ErrProductAlreadyExists
	}
	return err
}

func nullableString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

func calcTotalPages(total, size int) int {
	if total == 0 {
		return 1
	}
	return (total + size - 1) / size
}

func normalizePagination(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
