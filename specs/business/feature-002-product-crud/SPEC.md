# SPEC — CRUD de produtos

## Resumo Executivo

SPEC retrospectivo: o domínio `product` já está implementado. Esta spec documenta entidade `Product`, `Draft` com validação collect-all, validação de identificador UUID, `ProductFactory`, 5 use cases (create, find_all, find_by_id, update, delete), mock manual, repositório Postgres com soft delete, DTOs, handler HTTP com CRUD completo, testes unitários e teste integrado. A migration `2001_inventory_product.sql` registrada no `erp-backend-module-common` cria a tabela no schema do tenant.

---

## Impacto em Segurança e LGPD

- **Autenticação/Autorização por role:** operações de leitura exigem `inventory.read`; escrita exige `inventory.write`. Validação feita por middlewares antes de qualquer handler.
- **Autorização por recurso/tenant:** todas as queries de produto filtram pelo schema do tenant derivado do JWT. `tenantID` e `actorID` nunca vêm do body.
- **Validação de entrada no Draft/VO:** `product.NewDraft` valida todos os campos com collect-all (`[]error`) — título, SKU, unidade, preço, quantidade, EAN, UUID fiscal, NCM, origin, CEST. `NewIdentifier` valida UUID de path param.
- **Proteção contra mass assignment:** `id` (UUIDv7 gerado pela factory), `is_active` (sempre `true` na criação), `created_by`, `updated_by`, `created_at`, `updated_at`, `deleted_at`, `deleted_by` são definidos pelo servidor. Nenhum desses campos é aceito do body.
- **Minimização de dados em responses:** `created_by` e `updated_by` não são expostos no JSON público do produto (MVP). `description`, `ean`, `fiscal_profile_external_id`, `cest` usam `omitempty`.
- **SQL Injection e queries parametrizadas:** todas as queries usam `$N`. O schema do tenant é interpolado com `fmt.Sprintf` usando `shared.SchemaName(tenantID)` — `tenantID` vem do JWT e é validado como UUID antes de chegar ao repositório.
- **Isolamento de tenant no banco:** todas as queries operam no schema do tenant derivado do token. `UPDATE` e `DELETE` incluem `AND deleted_at IS NULL` no `WHERE`.
- **Concorrência e idempotência:** índice único parcial `inventory_product__sku_uk WHERE deleted_at IS NULL` garante unicidade de SKU no banco em race conditions.
- **Auditoria:** `created_by`, `updated_by`, `created_at`, `updated_at` em todas as operações. Soft delete registra `deleted_by` e `deleted_at`. Registros não são destruídos.
- **Logs e observabilidade:** erros de banco logados por categoria — sem PII ou payload sensível.
- **Segredos e credenciais:** nenhum segredo adicional nesta feature.
- **Rate limit e abuso:** não implementado no MVP. Operações protegidas por autenticação e role.
- **Dados pessoais (LGPD):** produtos são dados de negócio do tenant — sem PII de pessoas físicas. `created_by`/`updated_by` armazenam UUIDs técnicos de colaboradores.

---

## Decisões de Domínio e Clean Architecture

### Entidade `Product`

Struct central do domínio com todos os campos do produto, incluindo campos fiscais (`NCM`, `Origin`, `CEST`), auditoria (`CreatedBy`, `UpdatedBy`, `CreatedAt`, `UpdatedAt`) e soft delete (`DeletedAt`, `DeletedBy`).

Método `Update(draft Draft, actorID string) Product` na entidade encapsula a transição de estado de atualização — o use case não copia campos do Draft diretamente.

### Value Object `Draft`

`NewDraft` implementa variante collect-all (`[]error`): todos os campos são validados independentemente e todos os erros são retornados de uma vez. Normaliza SKU e unidade para uppercase. Valida EAN, UUID fiscal, NCM (regex 8 dígitos), origin (regex dígito 0-8), CEST (regex 7 dígitos, opcional).

### Value Object `Identifier` (`vo_identifier.go`)

`NewIdentifier(id string) (string, error)` valida que o path param é um UUID válido antes de chegar ao repositório, evitando queries desnecessárias com IDs malformados.

### Factory `ProductFactory`

Obrigatória porque a criação exige geração de ID (UUIDv7), preenchimento de `CreatedBy`, `UpdatedBy` e `IsActive: true`. O use case `CreateUseCase` não conhece a implementação do gerador de ID — recebe apenas a interface `IDGenerator`.

### Use Cases

- `CreateUseCase.Execute(tenantID, actorID, draft)` — chama factory, persiste, retorna produto criado
- `FindAllUseCase.Execute(tenantID, page, size, q)` — delega paginação e filtro ao repositório
- `FindByIDUseCase.Execute(tenantID, id)` — delega busca ao repositório
- `UpdateUseCase.Execute(tenantID, id, actorID, draft)` — busca entidade atual, chama `Product.Update()`, persiste
- `DeleteUseCase.Execute(tenantID, id, actorID)` — chama `SoftDelete` no repositório

Nenhum use case contém `if` com política de negócio — apenas orquestram fluxo.

### Repositório

Interface `Repository` declarada no domínio. `ProductPostgresRepository` (infraestrutura) implementa com queries SQL como constantes `const` no topo do arquivo. Erros Postgres mapeados para erros de domínio (`23505` → `ErrProductAlreadyExists`, `sql.ErrNoRows` → `ErrProductNotFound`).

### DTOs e Handler

`CreateProductRequest.ToDraft()` e `UpdateProductRequest.ToDraft()` convertem o body JSON em `Draft` — usando `errors.Join` para consolidar erros da variante collect-all. `productHttpHandler` tem construtor privado `newProductHttpHandler(db)`. Handler não valida regra de negócio — apenas parseia, chama use case e responde HTTP.

**Checklist de Qualidade Arquitetural:**
- [x] DDD: todas as regras de validação estão no `Draft` (VO); transição de atualização está na entidade `Product.Update()`
- [x] Modelo não anêmico: `Product.Update()` encapsula a mudança de estado; `NewAccessStatus` no domínio access
- [x] Use cases: apenas orquestram — sem IFs de política de negócio
- [x] Infraestrutura: repositório, DTO e handler sem regra de negócio
- [x] Clean Architecture: domínio não importa pacotes de infraestrutura
- [x] Contratos: response minimiza dados (`created_by`/`updated_by` não expostos no MVP)
- [x] Banco/modelagem: migrations `2001` e `2002` documentadas em MODELING.md
- [x] TDD: cobertura unitária de todas as variantes de validação e todos os use cases
- [x] Padrões CDStudio: um arquivo por use case, mock manual, SQL como const, construtor privado

---

## Débitos Técnicos da Feature

| Código | Origem | Débito técnico | Camada | Arquivos previstos | Verificação |
|--------|--------|----------------|--------|--------------------|-------------|
| DT-001 | RN-001 a RN-011 | Implementar `Draft` com validação collect-all de todos os campos (incluindo NCM, origin, CEST) | Domínio | `domain/product/vo_draft.go` | `go test ./src/internal/domain/product/...` |
| DT-002 | RN-002, RN-004 | Normalizar SKU para uppercase e unit para uppercase no Draft | Domínio | `domain/product/vo_draft.go` | Testes de normalização |
| DT-003 | RN-003 | Índice único parcial `sku WHERE deleted_at IS NULL` na migration | Banco | `2001_inventory_product.sql` | Tentativa de duplicação retorna 409 |
| DT-004 | RN-014 | Factory `ProductFactory` com `IDGenerator` para isolar geração de UUID e auditoria inicial | Domínio | `domain/product/factory_product.go` | `go test ./src/internal/domain/product/...` |
| DT-005 | RN-001 a RN-014 | Implementar repositório Postgres com soft delete, scan de campos nullable e mapeamento de erros | Infra | `infrastructure/postgres/product.go` | Testes integrados |
| DT-006 | RN-012, RN-013 | Soft delete: preencher `deleted_at`, `deleted_by`, `updated_at`, `updated_by` no `SoftDelete` do repositório | Infra | `infrastructure/postgres/product.go` | DELETE → 204; GET do deletado → 404; novo produto com mesmo SKU → 201 |
| DT-007 | RN-014 | Handler extrai `actor_id` do contexto JWT — nunca do body | HTTP | `infrastructure/rest/product.go` | Criar produto e verificar que `created_by` no banco é o UUID do colaborador |
| DT-008 | RN-015 | `CreateProductRequest.ToDraft()` usa `errors.Join` para consolidar erros collect-all | HTTP/DTO | `infrastructure/dto/product.go` | Request com múltiplos campos inválidos → 400 com todos os erros |
| DT-009 | RN-001 a RN-015 | Testes unitários de todos os use cases e variantes do Draft | Teste | `domain/product/*_test.go` | `go test ./...` |
| DT-010 | RN-001 a RN-015 | Script de teste integrado do CRUD completo via HTTP | Teste/Integração | `scripts/integration/product_crud.sh` | Execução do script sem falhas |
| DT-011 | RN-009, RN-010, RN-011 | Migration `2002_inventory_product_fiscal_fields.sql` adicionando `ncm`, `origin`, `cest` | Banco | `erp-backend-module-common/data/migrations/tenant/2002_inventory_product_fiscal_fields.sql` | Campos presentes na tabela após migration |

---

## Ordem de Implementação

1. `entity_product.go` — struct `Product`, `Page`, `Repository`, `IDGenerator`, erros de domínio, método `Update`
2. `vo_draft.go` — `Draft`, `NewDraft` com validação collect-all (`[]error`)
3. `vo_identifier.go` — validação de UUID para o identificador recebido em rotas
4. `factory_product.go` — `ProductFactory.Create(actorID, draft)` → `Product` com ID gerado e auditoria inicial
5. `usecase_create.go` — orquestra factory + repositório
6. `usecase_find_all.go` — delega paginação ao repositório
7. `usecase_find_by_id.go` — delega busca por ID ao repositório
8. `usecase_update.go` — busca entidade, chama `Product.Update()`, persiste
9. `usecase_delete.go` — chama `SoftDelete` no repositório
10. `mock_repository.go` — mock manual com campos `*Fn`
11. `infrastructure/postgres/product.go` — repositório concreto com queries SQL, helpers de scan e mapeamento de erros
12. `infrastructure/dto/wrapper.go` — `Paginate[T any]`
13. `infrastructure/dto/product.go` — `CreateProductRequest`, `UpdateProductRequest`, `ProductResponse`, `NewProductPaginated`
14. `infrastructure/rest/product.go` — `productHttpHandler` com 5 handlers
15. `scripts/integration/product_crud.sh` — teste integrado do CRUD HTTP real
16. (migration) `erp-backend-module-common/data/migrations/tenant/2001_inventory_product.sql`

---

## Arquivos Criados

---

### `src/internal/domain/product/entity_product.go`

**Responsabilidade:** Define a entidade `Product`, paginação, interface `Repository`, interface `IDGenerator`, erros de domínio e o método de atualização de estado.

```go
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
		FindAll(tenantID string, page, size int) (Page, error)
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
```

---

### `src/internal/domain/product/vo_draft.go`

**Responsabilidade:** Valida e normaliza o input de criação/atualização de produto. Variante collect-all — reporta todos os erros de validação de uma vez.

```go
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
```

---

### `src/internal/domain/product/vo_identifier.go`

**Responsabilidade:** Valida o identificador de produto vindo de path params antes de chegar ao repositório Postgres.

```go
package product

import (
	"strings"

	"github.com/google/uuid"
)

func NewIdentifier(id string) (string, error) {
	id = strings.TrimSpace(id)
	if _, err := uuid.Parse(id); err != nil {
		return "", ErrProductIDInvalid
	}
	return id, nil
}
```

---

### `src/internal/domain/product/factory_product.go`

**Responsabilidade:** Cria a entidade `Product` com ID gerado e auditoria inicial. Isola a geração de ID do use case.

```go
package product

type ProductFactory struct {
	idGenerator IDGenerator
}

func NewProductFactory(idGenerator IDGenerator) *ProductFactory {
	return &ProductFactory{idGenerator: idGenerator}
}

func (f *ProductFactory) Create(actorID string, draft Draft) Product {
	return Product{
		ID:                      f.idGenerator.Generate(),
		Title:                   draft.Title,
		Description:             draft.Description,
		SKU:                     draft.SKU,
		EAN:                     draft.EAN,
		Unit:                    draft.Unit,
		UnitPrice:               draft.UnitPrice,
		StockQuantity:           draft.StockQuantity,
		IsActive:                true,
		FiscalProfileExternalID: draft.FiscalProfileExternalID,
		CreatedBy:               actorID,
		UpdatedBy:               actorID,
	}
}
```

---

### `src/internal/domain/product/usecase_create.go`

```go
package product

import "fmt"

type CreateUseCase struct {
	repository Repository
	factory    *ProductFactory
}

func NewCreateUseCase(repository Repository, factory *ProductFactory) *CreateUseCase {
	return &CreateUseCase{repository: repository, factory: factory}
}

func (u *CreateUseCase) Execute(tenantID, actorID string, draft Draft) (Product, error) {
	p := u.factory.Create(actorID, draft)

	created, err := u.repository.Create(tenantID, p)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to create product: %w", err)
	}

	return created, nil
}
```

---

### `src/internal/domain/product/usecase_find_all.go`

```go
package product

import "fmt"

type FindAllUseCase struct {
	repository Repository
}

func NewFindAllUseCase(repository Repository) *FindAllUseCase {
	return &FindAllUseCase{repository: repository}
}

func (u *FindAllUseCase) Execute(tenantID string, page, size int) (Page, error) {
	result, err := u.repository.FindAll(tenantID, page, size)
	if err != nil {
		return Page{}, fmt.Errorf("error trying to list products: %w", err)
	}

	return result, nil
}
```

---

### `src/internal/domain/product/usecase_find_by_id.go`

```go
package product

import "fmt"

type FindByIDUseCase struct {
	repository Repository
}

func NewFindByIDUseCase(repository Repository) *FindByIDUseCase {
	return &FindByIDUseCase{repository: repository}
}

func (u *FindByIDUseCase) Execute(tenantID, id string) (Product, error) {
	p, err := u.repository.FindByID(tenantID, id)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to find product: %w", err)
	}

	return p, nil
}
```

---

### `src/internal/domain/product/usecase_update.go`

```go
package product

import "fmt"

type UpdateUseCase struct {
	repository Repository
}

func NewUpdateUseCase(repository Repository) *UpdateUseCase {
	return &UpdateUseCase{repository: repository}
}

func (u *UpdateUseCase) Execute(tenantID, id, actorID string, draft Draft) (Product, error) {
	current, err := u.repository.FindByID(tenantID, id)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to find product before update: %w", err)
	}

	updated := current.Update(draft, actorID)

	saved, err := u.repository.Update(tenantID, updated)
	if err != nil {
		return Product{}, fmt.Errorf("error trying to update product: %w", err)
	}

	return saved, nil
}
```

---

### `src/internal/domain/product/usecase_delete.go`

```go
package product

import "fmt"

type DeleteUseCase struct {
	repository Repository
}

func NewDeleteUseCase(repository Repository) *DeleteUseCase {
	return &DeleteUseCase{repository: repository}
}

func (u *DeleteUseCase) Execute(tenantID, id, actorID string) error {
	if err := u.repository.SoftDelete(tenantID, id, actorID); err != nil {
		return fmt.Errorf("error trying to delete product: %w", err)
	}

	return nil
}
```

---

### `src/internal/domain/product/mock_repository.go`

```go
package product

type MockProductRepository struct {
	CreateFn     func(tenantID string, p Product) (Product, error)
	FindAllFn    func(tenantID string, page, size int) (Page, error)
	FindByIDFn   func(tenantID, id string) (Product, error)
	UpdateFn     func(tenantID string, p Product) (Product, error)
	SoftDeleteFn func(tenantID, id, deletedBy string) error
}

func (m *MockProductRepository) Create(tenantID string, p Product) (Product, error) {
	return m.CreateFn(tenantID, p)
}

func (m *MockProductRepository) FindAll(tenantID string, page, size int) (Page, error) {
	return m.FindAllFn(tenantID, page, size)
}

func (m *MockProductRepository) FindByID(tenantID, id string) (Product, error) {
	return m.FindByIDFn(tenantID, id)
}

func (m *MockProductRepository) Update(tenantID string, p Product) (Product, error) {
	return m.UpdateFn(tenantID, p)
}

func (m *MockProductRepository) SoftDelete(tenantID, id, deletedBy string) error {
	return m.SoftDeleteFn(tenantID, id, deletedBy)
}
```

---

### `src/internal/infrastructure/postgres/product.go`

**Responsabilidade:** Repositório concreto Postgres. Queries como constantes `const` com `%s` para o schema do tenant (interpolado via `fmt.Sprintf`). Helpers privados para scan de linhas e mapeamento de erros.

```go
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
    created_by, updated_by
) VALUES (
    $1, $2, $3, $4, $5, $6,
    $7, $8, $9,
    $10,
    $11, $12
)
RETURNING id, title, description, sku, ean, unit,
          unit_price, stock_quantity, is_active,
          fiscal_profile_external_id,
          created_by, updated_by, created_at, updated_at, deleted_at, deleted_by`

	findAllProductsQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
       created_by, updated_by, created_at, updated_at, deleted_at, deleted_by
FROM %s.inventory_product
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

	countProductsQuery = `
SELECT COUNT(*) FROM %s.inventory_product WHERE deleted_at IS NULL`

	findProductByIDQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
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
    updated_by = $9,
    updated_at = now()
WHERE id = $10 AND deleted_at IS NULL
RETURNING id, title, description, sku, ean, unit,
          unit_price, stock_quantity, is_active,
          fiscal_profile_external_id,
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
		p.ID, p.Title, nullableString(p.Description),
		p.SKU, nullableString(p.EAN), p.Unit,
		p.UnitPrice, p.StockQuantity, p.IsActive,
		nullableString(p.FiscalProfileExternalID),
		p.CreatedBy, p.UpdatedBy,
	)

	created, err := scanProductRow(row)
	if err != nil {
		return product.Product{}, mapProductError(err)
	}

	return created, nil
}

func (r *ProductPostgresRepository) FindAll(tenantID string, page, size int) (product.Page, error) {
	schema := shared.SchemaName(tenantID)
	page, size = normalizePagination(page, size)
	offset := (page - 1) * size

	rows, err := r.db.Query(fmt.Sprintf(findAllProductsQuery, schema), size, offset)
	if err != nil {
		return product.Page{}, err
	}
	defer rows.Close()

	products, err := scanProductRows(rows)
	if err != nil {
		return product.Page{}, err
	}

	var total int
	if err := r.db.QueryRow(fmt.Sprintf(countProductsQuery, schema)).Scan(&total); err != nil {
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
		p.Title, nullableString(p.Description),
		p.SKU, nullableString(p.EAN), p.Unit,
		p.UnitPrice, p.StockQuantity,
		nullableString(p.FiscalProfileExternalID),
		p.UpdatedBy, p.ID,
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

	err := row.Scan(
		&p.ID, &p.Title, &description,
		&p.SKU, &ean, &p.Unit,
		&p.UnitPrice, &p.StockQuantity, &p.IsActive,
		&fiscalProfileExternalID,
		&p.CreatedBy, &p.UpdatedBy,
		&p.CreatedAt, &p.UpdatedAt, &deletedAt, &deletedBy,
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

		if err := rows.Scan(
			&p.ID, &p.Title, &description,
			&p.SKU, &ean, &p.Unit,
			&p.UnitPrice, &p.StockQuantity, &p.IsActive,
			&fiscalProfileExternalID,
			&p.CreatedBy, &p.UpdatedBy,
			&p.CreatedAt, &p.UpdatedAt, &deletedAt, &deletedBy,
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
```

---

### `src/internal/infrastructure/dto/wrapper.go`

```go
package dto

type Paginate[T any] struct {
	Data       []T `json:"data"`
	Page       int `json:"page"`
	Size       int `json:"size"`
	TotalPages int `json:"total_pages"`
	Total      int `json:"total"`
}
```

---

### `src/internal/infrastructure/dto/product.go`

**Responsabilidade:** Structs de request/response para o domínio `product`. `ToDraft()` converte a lista de erros em `errors.Join` para retorno único.

```go
package dto

import (
	"errors"
	"time"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/product"
)

type (
	CreateProductRequest struct {
		Title                   string  `json:"title"`
		Description             string  `json:"description"`
		SKU                     string  `json:"sku"`
		EAN                     string  `json:"ean"`
		Unit                    string  `json:"unit"`
		UnitPrice               *float64 `json:"unit_price"`
		StockQuantity           float64 `json:"stock_quantity"`
		FiscalProfileExternalID string  `json:"fiscal_profile_external_id"`
	}

	UpdateProductRequest struct {
		Title                   string  `json:"title"`
		Description             string  `json:"description"`
		SKU                     string  `json:"sku"`
		EAN                     string  `json:"ean"`
		Unit                    string  `json:"unit"`
		UnitPrice               *float64 `json:"unit_price"`
		StockQuantity           float64 `json:"stock_quantity"`
		FiscalProfileExternalID string  `json:"fiscal_profile_external_id"`
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
```

---

### `src/internal/infrastructure/rest/product.go`

**Responsabilidade:** Handler HTTP para o CRUD de produtos. Construtor privado instancia repositório, factory e todos os use cases.

```go
package rest

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/product"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/dto"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/postgres"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/shared"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/tenant"
	"github.com/gin-gonic/gin"
)

type productHttpHandler struct {
	createUseCase   *product.CreateUseCase
	findByIDUseCase *product.FindByIDUseCase
	findAllUseCase  *product.FindAllUseCase
	updateUseCase   *product.UpdateUseCase
	deleteUseCase   *product.DeleteUseCase
}

func newProductHttpHandler(db *sql.DB) *productHttpHandler {
	repository := postgres.NewProductPostgresRepository(db)
	factory := product.NewProductFactory(&shared.UUIDGenerator{})

	return &productHttpHandler{
		createUseCase:   product.NewCreateUseCase(repository, factory),
		findByIDUseCase: product.NewFindByIDUseCase(repository),
		findAllUseCase:  product.NewFindAllUseCase(repository),
		updateUseCase:   product.NewUpdateUseCase(repository),
		deleteUseCase:   product.NewDeleteUseCase(repository),
	}
}

func (h *productHttpHandler) HandleCreate(c *gin.Context) {
	var req dto.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		buildResponseError(c, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	draft, err := req.ToDraft()
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	p, err := h.createUseCase.Execute(tenant.GetTenantID(c), actorIDFromContext(c), draft)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusCreated, dto.NewProductResponse(p))
}

func (h *productHttpHandler) HandleList(c *gin.Context) {
	page := parseStringToInt(c.Query("page"), 1)
	size := parseStringToInt(c.Query("size"), 10)

	result, err := h.findAllUseCase.Execute(tenant.GetTenantID(c), page, size)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewProductPaginated(result))
}

func (h *productHttpHandler) HandleFindByID(c *gin.Context) {
	id, err := product.NewIdentifier(c.Param("id"))
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	p, err := h.findByIDUseCase.Execute(tenant.GetTenantID(c), id)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewProductResponse(p))
}

func (h *productHttpHandler) HandleUpdate(c *gin.Context) {
	id, err := product.NewIdentifier(c.Param("id"))
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	var req dto.UpdateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		buildResponseError(c, http.StatusBadRequest, errors.New("invalid request body"))
		return
	}

	draft, err := req.ToDraft()
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	p, err := h.updateUseCase.Execute(tenant.GetTenantID(c), id, actorIDFromContext(c), draft)
	if err != nil {
		handleProductError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewProductResponse(p))
}

func (h *productHttpHandler) HandleDelete(c *gin.Context) {
	id, err := product.NewIdentifier(c.Param("id"))
	if err != nil {
		buildResponseError(c, http.StatusBadRequest, err)
		return
	}

	if err := h.deleteUseCase.Execute(tenant.GetTenantID(c), id, actorIDFromContext(c)); err != nil {
		handleProductError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func handleProductError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, product.ErrProductNotFound):
		buildResponseError(c, http.StatusNotFound, err)
	case errors.Is(err, product.ErrProductAlreadyExists):
		buildResponseError(c, http.StatusConflict, err)
	default:
		log.Printf("product handler error: %v", err)
		buildResponseError(c, http.StatusInternalServerError, errors.New("internal server error"))
	}
}
```

---

### `erp-backend-module-common/data/migrations/tenant/2001_inventory_product.sql`

**Responsabilidade:** Cria a tabela `inventory_product` no schema do tenant. Executada pelo `migration.Runner` do módulo common no provisionamento de cada empresa.

```sql
CREATE TABLE IF NOT EXISTS {{schema}}.inventory_product (
    id                         uuid          NOT NULL DEFAULT gen_random_uuid(),
    title                      varchar(120)  NOT NULL,
    description                text,
    sku                        varchar(60)   NOT NULL,
    ean                        varchar(14),
    unit                       varchar(6)    NOT NULL,
    unit_price                 decimal(15,4) NOT NULL,
    stock_quantity             decimal(15,4) NOT NULL DEFAULT 0,
    is_active                  boolean       NOT NULL DEFAULT true,
    fiscal_profile_external_id uuid,
    created_by                 uuid          NOT NULL,
    updated_by                 uuid          NOT NULL,
    created_at                 timestamptz   NOT NULL DEFAULT now(),
    updated_at                 timestamptz   NOT NULL DEFAULT now(),
    deleted_at                 timestamptz,
    deleted_by                 uuid,
    CONSTRAINT inventory_product__pk PRIMARY KEY (id)
);

CREATE UNIQUE INDEX IF NOT EXISTS inventory_product__sku_uk
    ON {{schema}}.inventory_product (sku) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_inventory_product_fiscal_profile
    ON {{schema}}.inventory_product (fiscal_profile_external_id);

CREATE INDEX IF NOT EXISTS idx_inventory_product_is_active
    ON {{schema}}.inventory_product (is_active);
```

> **Nota:** `id` usa `DEFAULT gen_random_uuid()` como fallback no banco — o app sempre gera e envia o ID (UUIDv7 via `shared.GenerateID()`). O DEFAULT garante consistência mesmo em inserts diretos.

---

## Testes Unitários

### Arquivos a criar

| Arquivo | Cenários obrigatórios |
|---------|----------------------|
| `src/internal/domain/product/vo_draft_test.go` | Happy path completo; SKU normalizado para uppercase; `ErrTitleRequired`; `ErrSKURequired`; `ErrUnitRequired`; `ErrUnitPriceInvalid` (negativo); `ErrStockQuantityInvalid` (negativo); `ErrEANInvalid` (tamanho errado, caractere não-dígito); `ErrFiscalProfileExternalIDInvalid`; múltiplos erros retornados em uma única chamada |
| `src/internal/domain/product/vo_identifier_test.go` | UUID válido com trim; `ErrProductIDInvalid` para ID malformado |
| `src/internal/domain/product/usecase_create_test.go` | Happy path; `ErrProductAlreadyExists` do repositório |
| `src/internal/domain/product/usecase_find_all_test.go` | Happy path com paginação; lista vazia |
| `src/internal/domain/product/usecase_find_by_id_test.go` | Happy path; `ErrProductNotFound` |
| `src/internal/domain/product/usecase_update_test.go` | Happy path; `ErrProductNotFound` no FindByID; erro no Update |
| `src/internal/domain/product/usecase_delete_test.go` | Happy path; `ErrProductNotFound` |

### Naming obrigatório

```
TestProductDraft_NewDraft_Success
TestProductDraft_NewDraft_SKUNormalized
TestProductDraft_NewDraft_UnitNormalized
TestProductDraft_NewDraft_TitleRequired
TestProductDraft_NewDraft_SKURequired
TestProductDraft_NewDraft_UnitRequired
TestProductDraft_NewDraft_UnitPriceNegative
TestProductDraft_NewDraft_StockQuantityNegative
TestProductDraft_NewDraft_EANInvalidLength
TestProductDraft_NewDraft_EANInvalidCharacter
TestProductDraft_NewDraft_FiscalProfileExternalIDInvalid
TestProductDraft_NewDraft_MultipleErrors
TestProductIdentifier_NewIdentifier_Success
TestProductIdentifier_NewIdentifier_Invalid

TestCreateProductUseCase_Execute_Success
TestCreateProductUseCase_Execute_AlreadyExists

TestFindAllProductUseCase_Execute_Success
TestFindAllProductUseCase_Execute_EmptyList

TestFindByIDProductUseCase_Execute_Success
TestFindByIDProductUseCase_Execute_NotFound

TestUpdateProductUseCase_Execute_Success
TestUpdateProductUseCase_Execute_NotFound

TestDeleteProductUseCase_Execute_Success
TestDeleteProductUseCase_Execute_NotFound
```

---

## Testes Integrados

### Arquivo criado

| Arquivo | Cenários cobertos |
|---------|-------------------|
| `scripts/integration/product_crud.sh` | Login de collaborator com feature `inventory`; `unit_price` obrigatório; criação com normalização de SKU/unidade; resposta sem `created_by`/`updated_by`; bloqueio de SKU duplicado; listagem; ID malformado retornando 400; busca por ID; atualização; soft delete; busca de deletado retornando 404 |

### Exemplo de estrutura (`usecase_create_test.go`)

```go
package product

import (
    "errors"
    "testing"
)

func TestCreateProductUseCase_Execute_Success(t *testing.T) {
    expected := Product{ID: "uuid-1", Title: "Camiseta", SKU: "CAM-P", Unit: "UN", UnitPrice: 49.90, IsActive: true}
    repo := &MockProductRepository{
        CreateFn: func(tenantID string, p Product) (Product, error) { return expected, nil },
    }
    factory := NewProductFactory(&stubIDGenerator{id: "uuid-1"})
    uc := NewCreateUseCase(repo, factory)

    draft, _ := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "")
    got, err := uc.Execute("tenant-id", "actor-id", draft)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if got.ID != expected.ID {
        t.Errorf("expected ID %s, got %s", expected.ID, got.ID)
    }
}

func TestCreateProductUseCase_Execute_AlreadyExists(t *testing.T) {
    repo := &MockProductRepository{
        CreateFn: func(tenantID string, p Product) (Product, error) {
            return Product{}, ErrProductAlreadyExists
        },
    }
    factory := NewProductFactory(&stubIDGenerator{id: "uuid-1"})
    uc := NewCreateUseCase(repo, factory)

    draft, _ := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "")
    _, err := uc.Execute("tenant-id", "actor-id", draft)
    if !errors.Is(err, ErrProductAlreadyExists) {
        t.Errorf("expected ErrProductAlreadyExists, got %v", err)
    }
}

// stubIDGenerator — auxiliar para testes da factory
type stubIDGenerator struct{ id string }
func (g *stubIDGenerator) Generate() string { return g.id }
```

---

## MODELING.md

A tabela `tenant.inventory_product` já está documentada em [MODELING.md](../../../../MODELING.md) na seção "Schemas de Tenant — Módulo Inventário". Nenhuma atualização necessária.

DBML de referência:

```dbml
Table tenant.inventory_product {
  id                         uuid          [primary key]
  title                      varchar(120)  [not null]
  description                text
  sku                        varchar(60)   [not null, note: 'unique per tenant (partial index WHERE deleted_at IS NULL)']
  ean                        varchar(14)
  unit                       varchar(6)    [not null]
  unit_price                 decimal(15,4) [not null]
  stock_quantity             decimal(15,4) [not null, default: 0]
  is_active                  boolean       [not null, default: true]
  fiscal_profile_external_id uuid          [note: 'Opaque ref to tax.fiscal_profile; no FK cross-module; nullable']
  created_by                 uuid          [not null]
  updated_by                 uuid          [not null]
  created_at                 timestamptz   [not null]
  updated_at                 timestamptz   [not null]
  deleted_at                 timestamptz
  deleted_by                 uuid

  indexes {
    sku [unique, name: 'inventory_product__sku_uk', note: 'WHERE deleted_at IS NULL']
    fiscal_profile_external_id [name: 'idx_inventory_product_fiscal_profile']
    is_active [name: 'idx_inventory_product_is_active']
  }
}
```

---

## Arquivos Deletados

Nenhum.

---

## Checklist de Verificação

### Build

```bash
cd erp-backend-module-inventory
go build ./...
```

### Testes

```bash
go test ./...
```

### End-to-End

```bash
TOKEN="<JWT de collaborator com feature inventory habilitada e role inventory.write>"

# Criar produto
curl -s -X POST http://localhost:8082/api/inventories/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Camiseta Branca P","sku":"CAM-BRA-P","unit":"UN","unit_price":49.90,"stock_quantity":100}' | jq .
# Esperado: 201 com ProductResponse e id gerado

PRODUCT_ID="<id retornado acima>"

# Listar
curl -s "http://localhost:8082/api/inventories/products?page=1&size=10" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Esperado: 200 com data[], page, size, total_pages, total

# Buscar por ID
curl -s "http://localhost:8082/api/inventories/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Esperado: 200 com ProductResponse

# Atualizar
curl -s -X PUT "http://localhost:8082/api/inventories/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Camiseta Branca P (novo)","sku":"CAM-BRA-P","unit":"UN","unit_price":59.90,"stock_quantity":80}' | jq .
# Esperado: 200 com ProductResponse atualizado

# Deletar
curl -s -X DELETE "http://localhost:8082/api/inventories/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $TOKEN"
# Esperado: 204 No Content

# Buscar deletado → não encontrado
curl -s "http://localhost:8082/api/inventories/products/$PRODUCT_ID" \
  -H "Authorization: Bearer $TOKEN" | jq .
# Esperado: 404 {"message":"product not found"}

# Tentar criar SKU duplicado
curl -s -X POST http://localhost:8082/api/inventories/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Outra","sku":"CAM-BRA-P","unit":"UN","unit_price":10}' | jq .
# Esperado: 409 {"message":"product with this SKU already exists"}
```

## Nao Implementado

### Campos de auditoria de actor no response

Os campos `created_by` e `updated_by` existem na tabela `products` (conforme premissas de auditoria do `CLAUDE.md`) mas **nao sao expostos** no response JSON dos endpoints `GET /api/inventories/products/:id` e `GET /api/inventories/products`.

O DTO atual (`ProductResponse` em `dto/product.go`) omite esses campos intencionalmente para manter o contrato minimo do MVP.

**Impacto:** o frontend nao consegue exibir "Criado por" e "Atualizado por" na secao de Rastreabilidade da tela de detalhe do produto, que segue o padrao de auditoria definido em `COMPONETS.md` (linhas 215-216).

**O que e necessario para implementar:**
- Adicionar `CreatedBy string` e `UpdatedBy string` ao struct `ProductResponse` em `src/internal/infrastructure/dto/product.go`
- Preencher os campos no mapeador `ToProductResponse` lendo `product.CreatedBy` e `product.UpdatedBy`
- Garantir que o repositorio Postgres retorna esses campos nas queries `SELECT` de `FindByID` e `FindAll`

---

# Emenda v1 — Campos Fiscais Fixos no Produto

## Resumo Executivo

Emenda ao domínio `product`: 6 arquivos modificados, 1 migration criada. Adiciona `NCM string`, `Origin string`, `CEST *string` à entidade e ao Draft; propaga pela factory; atualiza queries SQL; expõe os campos no DTO. Nenhum arquivo deletado. Handler REST e router permanecem inalterados.

---

## Impacto em Segurança e LGPD

- `ncm` e `origin` obrigatórios; `cest` opcional — validados no `Draft` antes de qualquer persistência
- Queries parametrizadas: todos os `$N` mantidos; `fmt.Sprintf` apenas para interpolar schema derivado de `shared.SchemaName(tenantID)`
- Migration additive: `ALTER TABLE … ADD COLUMN IF NOT EXISTS` — sem remoção de dados

---

## Ordem de Implementação

1. `entity_product.go` — adiciona campos e erros novos; atualiza `Update()`
2. `vo_draft.go` — adiciona `NCM`, `Origin`, `CEST` ao Draft; atualiza `NewDraft` e `validateDraft` (inline)
3. `factory_product.go` — propaga novos campos do Draft para a entidade
4. `infrastructure/postgres/product.go` — atualiza todas as queries e funções de scan
5. `infrastructure/dto/product.go` — adiciona campos nos requests e na response
6. `domain/product/vo_draft_test.go` — novos cenários de validação + atualiza helper
7. `erp-backend-module-common/data/migrations/tenant/2002_inventory_product_fiscal_fields.sql` — nova migration

---

## Arquivos Modificados

### `erp-backend-module-inventory/src/internal/domain/product/entity_product.go`

**Antes:**

```go
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
```

**Depois:**

```go
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
	ErrNCMInvalid                     = errors.New("ncm must contain exactly 8 digits")
	ErrOriginInvalid                  = errors.New("origin must be a single digit between 0 and 8")
	ErrCESTInvalid                    = errors.New("cest must contain exactly 7 digits")
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
		NCM                     string
		Origin                  string
		CEST                    *string
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
	p.NCM = draft.NCM
	p.Origin = draft.Origin
	p.CEST = draft.CEST
	p.UpdatedBy = actorID
	return p
}
```

**O que mudou:** adicionados `NCM string`, `Origin string`, `CEST *string` à struct `Product`; adicionados `ErrNCMInvalid`, `ErrOriginInvalid`, `ErrCESTInvalid`; `Update()` propaga os três novos campos.

---

### `erp-backend-module-inventory/src/internal/domain/product/vo_draft.go`

**Antes:**

```go
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
```

**Depois:**

```go
package product

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	reNCM    = regexp.MustCompile(`^[0-9]{8}$`)
	reOrigin = regexp.MustCompile(`^[0-8]$`)
	reCEST   = regexp.MustCompile(`^[0-9]{7}$`)
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
	NCM                     string
	Origin                  string
	CEST                    *string
}

func NewDraft(
	title, description, sku, ean, unit string,
	unitPrice, stockQuantity float64,
	fiscalProfileExternalID string,
	ncm, origin string,
	cest *string,
) (Draft, []error) {
	var errs []error

	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	sku = strings.ToUpper(strings.TrimSpace(sku))
	ean = strings.TrimSpace(ean)
	unit = strings.ToUpper(strings.TrimSpace(unit))
	fiscalProfileExternalID = strings.TrimSpace(fiscalProfileExternalID)
	ncm = strings.TrimSpace(ncm)
	origin = strings.TrimSpace(origin)

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

	if !reNCM.MatchString(ncm) {
		errs = append(errs, ErrNCMInvalid)
	}

	if !reOrigin.MatchString(origin) {
		errs = append(errs, ErrOriginInvalid)
	}

	if cest != nil {
		trimmed := strings.TrimSpace(*cest)
		cest = &trimmed
		if !reCEST.MatchString(*cest) {
			errs = append(errs, ErrCESTInvalid)
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
		NCM:                     ncm,
		Origin:                  origin,
		CEST:                    cest,
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
```

**O que mudou:** `Draft` ganha `NCM string`, `Origin string`, `CEST *string`; `NewDraft` recebe três novos parâmetros (`ncm`, `origin`, `cest *string`) ao final da assinatura; variáveis package-level `reNCM`, `reOrigin`, `reCEST` compiladas com `regexp.MustCompile`; validação inline para os três campos; importa `regexp`.

---

### `erp-backend-module-inventory/src/internal/domain/product/factory_product.go`

**Antes:**

```go
package product

type ProductFactory struct {
	idGenerator IDGenerator
}

func NewProductFactory(idGenerator IDGenerator) *ProductFactory {
	return &ProductFactory{idGenerator: idGenerator}
}

func (f *ProductFactory) Create(actorID string, draft Draft) Product {
	return Product{
		ID:                      f.idGenerator.Generate(),
		Title:                   draft.Title,
		Description:             draft.Description,
		SKU:                     draft.SKU,
		EAN:                     draft.EAN,
		Unit:                    draft.Unit,
		UnitPrice:               draft.UnitPrice,
		StockQuantity:           draft.StockQuantity,
		IsActive:                true,
		FiscalProfileExternalID: draft.FiscalProfileExternalID,
		CreatedBy:               actorID,
		UpdatedBy:               actorID,
	}
}
```

**Depois:**

```go
package product

type ProductFactory struct {
	idGenerator IDGenerator
}

func NewProductFactory(idGenerator IDGenerator) *ProductFactory {
	return &ProductFactory{idGenerator: idGenerator}
}

func (f *ProductFactory) Create(actorID string, draft Draft) Product {
	return Product{
		ID:                      f.idGenerator.Generate(),
		Title:                   draft.Title,
		Description:             draft.Description,
		SKU:                     draft.SKU,
		EAN:                     draft.EAN,
		Unit:                    draft.Unit,
		UnitPrice:               draft.UnitPrice,
		StockQuantity:           draft.StockQuantity,
		IsActive:                true,
		FiscalProfileExternalID: draft.FiscalProfileExternalID,
		NCM:                     draft.NCM,
		Origin:                  draft.Origin,
		CEST:                    draft.CEST,
		CreatedBy:               actorID,
		UpdatedBy:               actorID,
	}
}
```

**O que mudou:** `Create()` propaga `NCM`, `Origin`, `CEST` do Draft para a entidade.

---

### `erp-backend-module-inventory/src/internal/infrastructure/postgres/product.go`

**Antes:**

```go
// (arquivo completo — ver seção "Arquivos Criados" da SPEC original)
// Queries sem ncm, origin, cest; scan sem esses campos.
```

Pontos-chave do estado anterior:
- `createProductQuery`: INSERT com 12 parâmetros (`$1`–`$12`); RETURNING sem `ncm`, `origin`, `cest`
- `findAllProductsQuery`, `findAllProductsWithSearchQuery`, `findProductByIDQuery`: SELECT sem `ncm`, `origin`, `cest`
- `updateProductQuery`: SET com `$1`–`$9`, WHERE `$10`; RETURNING sem os campos
- `scanProductRow` / `scanProductRows`: scan de 16 colunas sem `ncm`, `origin`, `cest`
- `Create()`: 12 argumentos para `QueryRow`
- `Update()`: 10 argumentos para `QueryRow`

**Depois:**

```go
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
  AND (title ILIKE $3 OR sku ILIKE $3)
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

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

// FindAll, FindByID, SoftDelete — inalterados na lógica; apenas as queries internas foram atualizadas acima.

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

// mapProductError, nullableString, calcTotalPages, normalizePagination — inalterados.
```

**O que mudou:** todas as queries SELECT/INSERT/UPDATE incluem `ncm`, `origin`, `cest`; `Create` passa 15 argumentos (3 novos); `Update` passa 13 argumentos (3 novos, deslocamento de `updated_by` e `id`); `scanProductRow` e `scanProductRows` adicionam scan para `&p.NCM`, `&p.Origin` (NOT NULL, scan direto) e `cest sql.NullString` para CEST nullable; `FindAll`, `FindByID`, `SoftDelete` inalterados em lógica.

---

### `erp-backend-module-inventory/src/internal/infrastructure/dto/product.go`

**Antes:**

```go
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
		CreatedAt:               p.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:               p.UpdatedAt.UTC().Format(time.RFC3339),
	}
}

// NewProductPaginated — inalterado.
```

**Depois:**

```go
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

// NewProductPaginated — inalterado.
```

**O que mudou:** `CreateProductRequest` e `UpdateProductRequest` ganham `NCM string`, `Origin string`, `CEST *string`; `ProductResponse` ganha os mesmos campos (`CEST` com `omitempty`); `ToDraft()` de ambos os requests passa `r.NCM, r.Origin, r.CEST` para `NewDraft`; `NewProductResponse` popula os três campos.

---

### `erp-backend-module-inventory/src/internal/domain/product/vo_draft_test.go`

**Antes:**

```go
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
```

**Depois:**

```go
package product

import (
	"errors"
	"testing"
)

// newTestDraft é um helper que preenche valores padrão válidos para os novos campos fiscais,
// permitindo que os testes existentes passem sem alterar suas assinaturas.
func newTestDraft(title, description, sku, ean, unit string, unitPrice, stockQuantity float64, fiscalProfileExternalID string) (Draft, []error) {
	return NewDraft(title, description, sku, ean, unit, unitPrice, stockQuantity, fiscalProfileExternalID, "12345678", "0", nil)
}

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
		"12345678",
		"0",
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
}

func TestProductDraft_NewDraft_SKUNormalized(t *testing.T) {
	draft, errs := newTestDraft("Camiseta", "", " cam-p ", "", "UN", 49.90, 0, "")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.SKU != "CAM-P" {
		t.Errorf("expected CAM-P, got %s", draft.SKU)
	}
}

func TestProductDraft_NewDraft_UnitNormalized(t *testing.T) {
	draft, errs := newTestDraft("Camiseta", "", "CAM-P", "", " un ", 49.90, 0, "")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %v", errs)
	}
	if draft.Unit != "UN" {
		t.Errorf("expected UN, got %s", draft.Unit)
	}
}

func TestProductDraft_NewDraft_TitleRequired(t *testing.T) {
	_, errs := newTestDraft("", "", "CAM-P", "", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrTitleRequired)
}

func TestProductDraft_NewDraft_SKURequired(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "", "", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrSKURequired)
}

func TestProductDraft_NewDraft_UnitRequired(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "CAM-P", "", "", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrUnitRequired)
}

func TestProductDraft_NewDraft_UnitPriceNegative(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "CAM-P", "", "UN", -1, 0, "")
	assertProductDraftError(t, errs, ErrUnitPriceInvalid)
}

func TestProductDraft_NewDraft_StockQuantityNegative(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, -1, "")
	assertProductDraftError(t, errs, ErrStockQuantityInvalid)
}

func TestProductDraft_NewDraft_EANInvalidLength(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "CAM-P", "123456789", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrEANInvalid)
}

func TestProductDraft_NewDraft_EANInvalidCharacter(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "CAM-P", "789123456789A", "UN", 49.90, 0, "")
	assertProductDraftError(t, errs, ErrEANInvalid)
}

func TestProductDraft_NewDraft_FiscalProfileExternalIDInvalid(t *testing.T) {
	_, errs := newTestDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "invalid")
	assertProductDraftError(t, errs, ErrFiscalProfileExternalIDInvalid)
}

func TestProductDraft_NewDraft_MultipleErrors(t *testing.T) {
	_, errs := NewDraft("", "", "", "123", "", -1, -1, "invalid", "INVALID", "9", nil)

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

func TestProductDraft_NewDraft_NCMInvalid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "1234567", "0", nil)
	assertProductDraftError(t, errs, ErrNCMInvalid)
}

func TestProductDraft_NewDraft_OriginInvalid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "12345678", "9", nil)
	assertProductDraftError(t, errs, ErrOriginInvalid)
}

func TestProductDraft_NewDraft_OriginRequired(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "12345678", "", nil)
	assertProductDraftError(t, errs, ErrOriginInvalid)
}

func TestProductDraft_NewDraft_CESTInvalid(t *testing.T) {
	cest := "123456" // 6 dígitos — inválido (precisa de 7)
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "12345678", "0", &cest)
	assertProductDraftError(t, errs, ErrCESTInvalid)
}

func TestProductDraft_NewDraft_CESTNil_Valid(t *testing.T) {
	_, errs := NewDraft("Camiseta", "", "CAM-P", "", "UN", 49.90, 0, "", "12345678", "0", nil)
	if len(errs) > 0 {
		t.Fatalf("expected no errors when cest is nil, got %v", errs)
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
```

**O que mudou:** introduzido helper `newTestDraft` que fixa `ncm="12345678"`, `origin="0"`, `cest=nil` para manter compatibilidade dos testes existentes; todas as chamadas diretas a `NewDraft` nos testes existentes atualizadas para a nova assinatura (11 parâmetros); `TestProductDraft_NewDraft_MultipleErrors` atualizado para incluir `ErrNCMInvalid` e `ErrOriginInvalid`; adicionados 5 novos cenários: `TestProductDraft_NewDraft_NCMInvalid`, `TestProductDraft_NewDraft_OriginInvalid`, `TestProductDraft_NewDraft_OriginRequired`, `TestProductDraft_NewDraft_CESTInvalid`, `TestProductDraft_NewDraft_CESTNil_Valid`.

---

## Arquivo Criado

### `erp-backend-module-common/data/migrations/tenant/2002_inventory_product_fiscal_fields.sql`

**Responsabilidade:** adiciona as colunas fiscais fixas à tabela `inventory_product` de cada tenant de forma idempotente.

```sql
ALTER TABLE {{schema}}.inventory_product
    ADD COLUMN IF NOT EXISTS ncm    varchar(8) NOT NULL DEFAULT '';

ALTER TABLE {{schema}}.inventory_product
    ADD COLUMN IF NOT EXISTS origin varchar(1) NOT NULL DEFAULT '0';

ALTER TABLE {{schema}}.inventory_product
    ADD CONSTRAINT inventory_product__origin_ck
        CHECK (origin ~ '^[0-8]$') NOT VALID;

ALTER TABLE {{schema}}.inventory_product
    ADD COLUMN IF NOT EXISTS cest   varchar(7);

ALTER TABLE {{schema}}.inventory_product
    ADD CONSTRAINT inventory_product__cest_ck
        CHECK (cest IS NULL OR cest ~ '^[0-9]{7}$') NOT VALID;

ALTER TABLE {{schema}}.inventory_product
    ADD CONSTRAINT inventory_product__ncm_ck
        CHECK (ncm = '' OR ncm ~ '^[0-9]{8}$') NOT VALID;
```

---

## Checklist de Verificação

### Build
```bash
cd erp-backend-module-inventory
go build ./...
```

### Testes
```bash
go test ./...
```
