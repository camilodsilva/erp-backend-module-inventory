# erp-backend-module-inventory

Módulo de inventário do ERP CDStudio. Gerencia o catálogo de produtos que o cliente comercializa.

## Porta

`8082`

## Endpoints

| Método | Rota | Descrição |
|---|---|---|
| `GET` | `/api/inventories/health` | Health check |
| `GET` | `/api/inventories/access` | Status de acesso ao módulo |
| `POST` | `/api/inventories/products` | Criar produto |
| `GET` | `/api/inventories/products` | Listar produtos (paginado) |
| `GET` | `/api/inventories/products/:id` | Buscar produto por ID |
| `PUT` | `/api/inventories/products/:id` | Atualizar produto |
| `DELETE` | `/api/inventories/products/:id` | Remover produto (soft delete) |

## Variáveis de Ambiente

| Variável | Descrição |
|---|---|
| `POSTGRES_HOST` | Host do banco de dados |
| `POSTGRES_PORT` | Porta do banco de dados |
| `POSTGRES_USER` | Usuário do banco |
| `POSTGRES_PASSWORD` | Senha do banco |
| `POSTGRES_DB` | Nome do banco (`erp_common`) |
| `JWT_SECRET` | Secret JWT compartilhado com `erp-backend-module-common` |

## Como rodar localmente

```bash
cp .env.example .env
# preencher .env

go run src/cmd/main.go
```

## Testes

```bash
go test ./...
BASE_URL=http://localhost:8082 COMMON_URL=http://localhost:8080 ./scripts/integration/product_crud.sh
```

## Migrations

As migrations ficam em `erp-backend-module-common/data/migrations/tenant/`:

- `2001_inventory_product.sql` — tabela `inventory_product`

## Integração com outros módulos

Produtos podem ser referenciados de forma opaca por outros módulos via seu `id` (UUID). O módulo fiscal (`erp-backend-module-tax`) usa `product_external_id` nos perfis fiscais para vincular um produto a sua classificação tributária.

O produto também pode apontar para um perfil fiscal via `fiscal_profile_external_id` (UUID opaco, sem FK cross-module).
