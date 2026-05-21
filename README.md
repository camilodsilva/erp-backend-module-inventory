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

---

## Testes Integrados com Postgres Transitório

Os scripts de inventário aceitam variáveis de ambiente para rodar contra o banco efêmero `postgres-erp-it`.

### Fluxo completo (Common + Inventory)

```bash
# 1. Subir banco transitório
cd erp-infrastructure
./scripts/integration-postgres.sh up

# 2. Iniciar Common em :8080 apontando para o banco transitório
cd erp-backend-module-common
POSTGRES_HOST=localhost POSTGRES_PORT=55432 POSTGRES_USER=postgres \
  POSTGRES_PASSWORD=postgres POSTGRES_DB=erp_common \
  JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  go run src/cmd/main.go &

# 3. Iniciar Inventory em :8082 com o mesmo JWT_SECRET
cd erp-backend-module-inventory
POSTGRES_HOST=localhost POSTGRES_PORT=55432 POSTGRES_USER=postgres \
  POSTGRES_PASSWORD=postgres POSTGRES_DB=erp_common \
  JWT_SECRET=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef \
  go run src/cmd/main.go &

# 4. Rodar baterias de inventário
POSTGRES_CONTAINER=postgres-erp-it \
  COMMON_URL=http://localhost:8080 \
  BASE_URL=http://localhost:8082 \
  ./scripts/integration/inventory_foundation.sh
```

### Variáveis disponíveis

| Variável | Default | Descrição |
| -------- | ------- | --------- |
| `POSTGRES_CONTAINER` | `postgres` | Container alvo dos scripts psql |
| `POSTGRES_DB` | `erp_common` | Banco usado nas queries |
| `POSTGRES_USER` | `postgres` | Usuário psql |
| `COMMON_URL` | `http://localhost:8080` | Base URL do Common |
| `BASE_URL` | `http://localhost:8082` | Base URL do Inventory |
