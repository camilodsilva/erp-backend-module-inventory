# PRD — Fundação do módulo de inventário

**ID:** `inventory.business.feature.001`
**Status:** `in_progress`

## Contexto

Criação do módulo backend `erp-backend-module-inventory`. Espelha a estrutura do `erp-backend-module-tax` com Clean Architecture + DDD, multi-tenant via schema Postgres por cliente.

## O que entrega

- Projeto Go funcional com `go.mod`, `Dockerfile`, variáveis de ambiente documentadas.
- `GET /api/inventories/health` respondendo `{"status": "ok", "module": "inventory"}`.
- Router base com autenticação JWT de collaborator e feature gate `"inventory"`.
- `GET /api/inventories/access` retornando status de acesso ao módulo.
- Migration SQL `2001_inventory_product.sql` registrada em `erp-backend-module-common`.

## Variáveis de ambiente

```
POSTGRES_HOST, POSTGRES_PORT, POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB
JWT_SECRET  — compartilhado com erp-backend-module-common
```

## Schema (migration 2001)

Tabela `inventory_product` no schema do tenant. Ver MODELING.md para DBML completo.

## Porta

`8082`

## Dependências

- `erp-backend-module-common` deve estar em execução (JWT_SECRET compartilhado, tabela `company_features` acessível).
