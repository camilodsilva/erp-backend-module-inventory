# SPEC — Endpoint de busca para integração fiscal

## Resumo Executivo

Adiciona filtro textual opcional `?q=` ao endpoint existente `GET /api/inventories/products`. Quatro arquivos são modificados: interface `Repository` e `FindAllUseCase` no domínio, repositório Postgres (duas queries atualizadas), handler HTTP e mock. Nenhum arquivo novo é criado.

---

## ⚙️ Raciocínio Arquitetural

**Decisão:** opção (a) do PRD — `?q=` como filtro opcional no `GET /api/inventories/products`.

**Motivo:** manter contrato simples e retrocompatível. Requests sem `?q=` continuam funcionando identicamente ao comportamento atual.

**Abordagem SQL:** quando `q != ""`, usar `WHERE (title ILIKE $3 OR sku ILIKE $3)`. O placeholder `%q%` é montado em Go antes do bind para manter queries totalmente parametrizadas.

**Impacto na interface:** `Repository.FindAll` recebe `q string` como quarto argumento. O mock e todos os usos existentes precisam ser atualizados.

---

## Impacto em Segurança e LGPD

- **Autenticação/Autorização por role:** operação de leitura — exige `inventory.read`. Todos os middlewares da fundação se aplicam (JWT → feature gate → role check).
- **Autorização por recurso/tenant:** busca filtra apenas pelo schema do tenant derivado do JWT. Sem acesso cross-tenant.
- **Validação de entrada no Draft/VO:** `?q=` é extraído como string, trimado de espaços em Go (`strings.TrimSpace`), passado como parâmetro posicional. Sem validação adicional de domínio necessária.
- **Proteção contra mass assignment:** não aplicável — operação de leitura.
- **Minimização de dados em responses:** response idêntico ao `FindAll` — sem campos adicionais expostos.
- **SQL Injection:** `q` é passado como parâmetro posicional (`$3`). O padrão `%q%` é montado em Go com `fmt.Sprintf("%%%s%%", q)` antes do bind — `%` é literal Go, não SQL.
- **Isolamento de tenant:** queries incluem schema do tenant em `FROM %s.inventory_product`.
- **Concorrência e idempotência:** leitura pura — sem efeito colateral.
- **Auditoria:** sem operações de escrita.
- **Logs e observabilidade:** sem PII exposto.
- **Segredos e credenciais:** não aplicável.
- **Rate limit e abuso:** `ILIKE '%q%'` pode causar full table scan em catálogos grandes; mitigado pelo limite de `size` máximo de 100 no MVP.
- **Dados pessoais (LGPD):** produtos são dados de negócio — sem PII.

---

## Decisões de Domínio e Clean Architecture

**Decisão: `?q=` no endpoint existente (não endpoint separado):** opção retrocompatível que mantém contrato simples. Requests sem `?q=` funcionam identicamente ao comportamento anterior. A adição de `q string` à interface `Repository.FindAll` é a única mudança de contrato de domínio necessária.

**Impacto na interface de domínio:** `Repository.FindAll(tenantID string, page, size int, q string)` — quarto parâmetro adicionado. O mock e todos os usos existentes atualizam a assinatura. O value object `Draft` não é afetado — `q` não é um campo de criação/atualização, apenas um filtro de consulta.

**Repositório concreto:** quando `q == ""`, usa `findAllProductsQuery` e `countProductsQuery` (sem filtragem). Quando `q != ""`, usa `findAllProductsWithSearchQuery` e `countProductsWithSearchQuery` com `AND (title ILIKE $3 OR sku ILIKE $3)`. Duas constantes de query adicionadas — sem alteração de `scanProductRow`/`scanProductRows`.

**Handler:** extrai `q := strings.TrimSpace(c.Query("q"))` e repassa ao use case. Strings com apenas espaços são tratadas como `""` (sem filtro).

**Checklist de Qualidade Arquitetural:**
- [x] DDD: não há regra de negócio nova; `q` é apenas um filtro de consulta operacional
- [x] Modelo não anêmico: sem mudança em entidades ou VOs
- [x] Use cases: `FindAllUseCase.Execute` apenas repassa `q` ao repositório
- [x] Infraestrutura: lógica de `ILIKE` fica no repositório Postgres — correto
- [x] Clean Architecture: domínio recebe `q` como parâmetro simples sem conhecer SQL
- [x] Contratos: retrocompatibilidade garantida — sem `?q=` = comportamento anterior
- [x] Banco/modelagem: sem tabelas novas
- [x] TDD: 4 novos cenários de teste cobrindo busca com resultado, sem resultado e paginação com filtro
- [x] Padrões CDStudio: SQL como const, sem concatenação

---

## Débitos Técnicos da Feature

| Código | Origem | Débito técnico | Camada | Arquivos previstos | Verificação |
|--------|--------|----------------|--------|--------------------|-------------|
| DT-001 | RN-001 | Adicionar `q string` à interface `Repository.FindAll` e atualizar mock | Domínio | `domain/product/entity_product.go`, `domain/product/mock_repository.go` | Compilação sem erros |
| DT-002 | RN-001 | Atualizar `FindAllUseCase.Execute` para receber e repassar `q string` | Domínio | `domain/product/usecase_find_all.go` | Compilação sem erros |
| DT-003 | RN-002, RN-003, RN-004, RN-005 | Implementar queries `findAllProductsWithSearchQuery` e `countProductsWithSearchQuery` com `ILIKE`; atualizar método `FindAll` do repositório | Infra | `infrastructure/postgres/product.go` | Teste integrado `product_search.sh` |
| DT-004 | RN-001 | Atualizar `HandleList` para extrair `?q=` com `strings.TrimSpace` e repassar ao use case | HTTP | `infrastructure/rest/product.go` | Curl `?q=%20%20` → total igual a listagem sem filtro |
| DT-005 | RN-002, RN-003, RN-004, RN-005, RN-006 | Implementar testes unitários do `FindAllUseCase` com `q` não vazio | Teste | `domain/product/usecase_find_all_test.go` | `go test ./src/internal/domain/product/...` |
| DT-006 | RN-001 a RN-006 | Implementar script de teste integrado `product_search.sh` | Teste/Integração | `scripts/integration/product_search.sh` | `bash scripts/integration/product_search.sh` |

---

## Ordem de Implementação

1. `entity_product.go` — adicionar `q string` à assinatura de `Repository.FindAll`
2. `mock_repository.go` — atualizar assinatura de `FindAllFn` e `FindAll`
3. `usecase_find_all.go` — adicionar `q string` à assinatura de `Execute`
4. `postgres/product.go` — atualizar queries `findAllProductsQuery` e `countProductsQuery`, método `FindAll`
5. `rest/product.go` — extrair `q` do query param e passar ao use case

---

## Arquivos Modificados

---

### `src/internal/domain/product/entity_product.go`

**O que muda:** assinatura de `Repository.FindAll` recebe `q string`.

**Antes:**
```go
Repository interface {
    Create(tenantID string, p Product) (Product, error)
    FindAll(tenantID string, page, size int) (Page, error)
    FindByID(tenantID, id string) (Product, error)
    Update(tenantID string, p Product) (Product, error)
    SoftDelete(tenantID, id, deletedBy string) error
}
```

**Depois:**
```go
Repository interface {
    Create(tenantID string, p Product) (Product, error)
    FindAll(tenantID string, page, size int, q string) (Page, error)
    FindByID(tenantID, id string) (Product, error)
    Update(tenantID string, p Product) (Product, error)
    SoftDelete(tenantID, id, deletedBy string) error
}
```

---

### `src/internal/domain/product/mock_repository.go`

**O que muda:** assinatura de `FindAllFn` e do método `FindAll`.

**Antes:**
```go
type MockProductRepository struct {
    CreateFn     func(tenantID string, p Product) (Product, error)
    FindAllFn    func(tenantID string, page, size int) (Page, error)
    FindByIDFn   func(tenantID, id string) (Product, error)
    UpdateFn     func(tenantID string, p Product) (Product, error)
    SoftDeleteFn func(tenantID, id, deletedBy string) error
}

func (m *MockProductRepository) FindAll(tenantID string, page, size int) (Page, error) {
    return m.FindAllFn(tenantID, page, size)
}
```

**Depois:**
```go
type MockProductRepository struct {
    CreateFn     func(tenantID string, p Product) (Product, error)
    FindAllFn    func(tenantID string, page, size int, q string) (Page, error)
    FindByIDFn   func(tenantID, id string) (Product, error)
    UpdateFn     func(tenantID string, p Product) (Product, error)
    SoftDeleteFn func(tenantID, id, deletedBy string) error
}

func (m *MockProductRepository) FindAll(tenantID string, page, size int, q string) (Page, error) {
    return m.FindAllFn(tenantID, page, size, q)
}
```

---

### `src/internal/domain/product/usecase_find_all.go`

**O que muda:** `Execute` recebe `q string` e repassa ao repositório.

**Antes:**
```go
func (u *FindAllUseCase) Execute(tenantID string, page, size int) (Page, error) {
    result, err := u.repository.FindAll(tenantID, page, size)
    if err != nil {
        return Page{}, fmt.Errorf("error trying to list products: %w", err)
    }

    return result, nil
}
```

**Depois:**
```go
func (u *FindAllUseCase) Execute(tenantID string, page, size int, q string) (Page, error) {
    result, err := u.repository.FindAll(tenantID, page, size, q)
    if err != nil {
        return Page{}, fmt.Errorf("error trying to list products: %w", err)
    }

    return result, nil
}
```

---

### `src/internal/infrastructure/postgres/product.go`

**O que muda:** duas constantes de query e o método `FindAll`. Quando `q == ""`, o comportamento é idêntico ao atual. Quando `q != ""`, adiciona `AND (title ILIKE $3 OR sku ILIKE $3)` e ajusta a query de contagem.

**Antes (constantes):**
```go
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
```

**Depois (constantes):**
```go
findAllProductsQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
       created_by, updated_by, created_at, updated_at, deleted_at, deleted_by
FROM %s.inventory_product
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2`

findAllProductsWithSearchQuery = `
SELECT id, title, description, sku, ean, unit,
       unit_price, stock_quantity, is_active,
       fiscal_profile_external_id,
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
```

**Antes (método FindAll):**
```go
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
```

**Depois (método FindAll):**
```go
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
        defer rows.Close()

        if err := r.db.QueryRow(fmt.Sprintf(countProductsQuery, schema)).Scan(&total); err != nil {
            return product.Page{}, err
        }
    } else {
        pattern := fmt.Sprintf("%%%s%%", q)
        rows, err = r.db.Query(fmt.Sprintf(findAllProductsWithSearchQuery, schema), size, offset, pattern)
        if err != nil {
            return product.Page{}, err
        }
        defer rows.Close()

        if err := r.db.QueryRow(fmt.Sprintf(countProductsWithSearchQuery, schema), pattern).Scan(&total); err != nil {
            return product.Page{}, err
        }
    }

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
```

---

### `src/internal/infrastructure/rest/product.go`

**O que muda:** `HandleList` extrai `q` do query param e repassa ao use case.

**Antes:**
```go
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
```

**Depois:**
```go
func (h *productHttpHandler) HandleList(c *gin.Context) {
    page := parseStringToInt(c.Query("page"), 1)
    size := parseStringToInt(c.Query("size"), 10)
    q := strings.TrimSpace(c.Query("q"))

    result, err := h.findAllUseCase.Execute(tenant.GetTenantID(c), page, size, q)
    if err != nil {
        handleProductError(c, err)
        return
    }

    buildResponseSuccess(c, http.StatusOK, dto.NewProductPaginated(result))
}
```

> Adicionar `"strings"` aos imports do arquivo.

---

## Testes Unitários

### Arquivo implementado

`src/internal/domain/product/usecase_find_all_test.go`

| Teste | Comportamento verificado |
|-------|--------------------------|
| `TestFindAllProductUseCase_Execute_Success` | repositório chamado com `q=""`, retorna página com produtos |
| `TestFindAllProductUseCase_Execute_EmptyList` | repositório chamado com `q=""`, retorna página vazia |
| `TestFindAllProductUseCase_Execute_WithSearch` | repositório chamado com `q="camiseta"`, retorna produto filtrado |
| `TestFindAllProductUseCase_Execute_WithSearchEmptyResult` | repositório chamado com `q="xyz_inexistente"`, retorna `Total=0` e `Products=[]` |

---

## Testes de Integração

### Script implementado

`scripts/integration/product_search.sh` — segue o mesmo padrão dos scripts existentes (`set -euo pipefail`, helpers `assert_status`/`assert_body_contains`/`assert_body_absent`, `reset_database`, IDs fixos com prefixo `01980f04-*`).

| ID | Cenário |
|----|---------|
| `INT-SEARCH-01` | login do collaborator |
| `INT-SEARCH-02` | seed — criar Camiseta Branca P |
| `INT-SEARCH-03` | seed — criar Calca Jeans 38 |
| `INT-SEARCH-04` | listar sem `?q=` — retorna os dois produtos, `total:2` |
| `INT-SEARCH-05` | `?q=camiseta` — retorna só Camiseta, `total:1`, Calca ausente |
| `INT-SEARCH-06` | `?q=CAL` — retorna só Calca pelo SKU, `total:1`, Camiseta ausente |
| `INT-SEARCH-07` | `?q=xyz_inexistente` — retorna `total:0` |
| `INT-SEARCH-08` | `?q=a&page=1&size=1` — paginação com busca, `size:1`, total não zero |
| `INT-SEARCH-09` | `?q=%20%20` (espaços) — `strings.TrimSpace` trata como vazio, retorna `total:2` |

### Como executar

```bash
cd erp-backend-module-inventory
bash scripts/integration/product_search.sh
```

---

## MODELING.md

Nenhuma tabela criada ou alterada. Sem atualização necessária.

---

## Arquivos Criados

| Arquivo | Responsabilidade |
|---------|-----------------|
| `scripts/integration/product_search.sh` | Testes de integração do filtro `?q=` contra serviços reais |

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

### Testes unitários

```bash
go test ./...
```

### Testes de integração

```bash
# Requer: docker compose up -d, common e inventory rodando
cd erp-backend-module-inventory
bash scripts/integration/product_search.sh
```

### End-to-End manual

```bash
TOKEN="<JWT com feature inventory habilitada e role inventory.write>"

# Seed: criar dois produtos para testar busca
curl -s -X POST http://localhost:8082/api/inventories/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Camiseta Branca P","sku":"CAM-BRA-P","unit":"UN","unit_price":49.90}' | jq .id

curl -s -X POST http://localhost:8082/api/inventories/products \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Calça Jeans 38","sku":"CAL-JEA-38","unit":"UN","unit_price":129.90}' | jq .id

# Listar sem filtro (comportamento original preservado)
curl -s "http://localhost:8082/api/inventories/products" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, count: (.data | length)}'
# Esperado: total: 2, count: 2

# Busca por título
curl -s "http://localhost:8082/api/inventories/products?q=camiseta" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, titles: [.data[].title]}'
# Esperado: total: 1, titles: ["Camiseta Branca P"]

# Busca por SKU
curl -s "http://localhost:8082/api/inventories/products?q=CAL" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, skus: [.data[].sku]}'
# Esperado: total: 1, skus: ["CAL-JEA-38"]

# Busca sem resultado
curl -s "http://localhost:8082/api/inventories/products?q=xyz_inexistente" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, data: .data}'
# Esperado: total: 0, data: []

# Busca com paginação
curl -s "http://localhost:8082/api/inventories/products?q=a&page=1&size=1" \
  -H "Authorization: Bearer $TOKEN" | jq '{total: .total, size: .size, count: (.data | length)}'
# Esperado: total >= 1, size: 1, count: 1
```
