# PRD — CRUD de produtos

**ID:** `inventory.business.feature.002`
**Status:** `verified`

## Contexto

Permite ao cliente cadastrar e gerenciar o catálogo de produtos que comercializa. Cada produto contém dados descritivos e operacionais (SKU, unidade, preço, quantidade em estoque) e pode referenciar de forma opaca um perfil fiscal no módulo tax.

## Contrato HTTP

Todos os endpoints exigem `Authorization: Bearer <token>` de collaborator com a feature `inventory` habilitada.

| Método | Rota | Auth | Status de sucesso |
|---|---|---|---|
| `POST` | `/api/inventories/products` | write | `201` |
| `GET` | `/api/inventories/products` | read | `200` |
| `GET` | `/api/inventories/products/:id` | read | `200` |
| `PUT` | `/api/inventories/products/:id` | write | `200` |
| `DELETE` | `/api/inventories/products/:id` | write | `204` |

## Request — Criar produto

```json
{
  "title": "Camiseta Branca P",
  "description": "Camiseta 100% algodão tamanho P",
  "sku": "CAM-BRA-P",
  "ean": "7891234567890",
  "unit": "UN",
  "unit_price": 49.90,
  "stock_quantity": 100,
  "fiscal_profile_external_id": "01960e4a-3f2b-7d1c-8e5f-9a0b1c2d3e4f"
}
```

Campos obrigatórios: `title`, `sku`, `unit`, `unit_price`.
`ean`, `description`, `fiscal_profile_external_id` são opcionais.

## Response — Produto

```json
{
  "id": "01960e4a-3f2b-7d1c-8e5f-9a0b1c2d3e4f",
  "title": "Camiseta Branca P",
  "description": "Camiseta 100% algodão tamanho P",
  "sku": "CAM-BRA-P",
  "ean": "7891234567890",
  "unit": "UN",
  "unit_price": 49.90,
  "stock_quantity": 100.0,
  "is_active": true,
  "fiscal_profile_external_id": "01960e4a-3f2b-7d1c-8e5f-9a0b1c2d3e4f",
  "created_at": "2026-05-20T10:00:00Z",
  "updated_at": "2026-05-20T10:00:00Z"
}
```

## Response — Lista paginada

```json
{
  "data": [ ... ],
  "page": 1,
  "size": 10,
  "total_pages": 3,
  "total": 28
}
```

## Validações

| Campo | Regra |
|---|---|
| `title` | Obrigatório, máx 120 caracteres |
| `sku` | Obrigatório, máx 60 caracteres; único por tenant (case-insensitive, uppercase normalizado) |
| `unit` | Obrigatório, máx 6 caracteres |
| `unit_price` | Obrigatório, >= 0 |
| `stock_quantity` | Opcional (default 0), >= 0 |
| `ean` | Opcional; se presente: 8, 13 ou 14 dígitos numéricos |
| `fiscal_profile_external_id` | Opcional; se presente: UUID válido |

## Mapeamento de erros

| Erro de domínio | HTTP |
|---|---|
| `ErrProductNotFound` | 404 |
| `ErrProductAlreadyExists` | 409 |
| Erro de validação (Draft) | 400 |
| Outros | 500 |

## Schema de banco

Tabela `inventory_product` no schema do tenant. Ver MODELING.md.

## Soft delete

DELETE faz soft delete: preenche `deleted_at`, `deleted_by`, `updated_at`, `updated_by`. Produto deletado não aparece em listagens nem em buscas por ID.

## Integração com módulo fiscal

`fiscal_profile_external_id` é uma referência opaca ao `id` de um `fiscal_profile` no `erp-backend-module-tax`. Não há FK cross-module. O módulo de inventário não valida se o UUID existe no módulo fiscal.
