# PRD — Feature gate do inventário

**ID:** `inventory.business.feature.003`
**Status:** `prd`

## Contexto

O acesso ao módulo de inventário é controlado pela feature `"inventory"` cadastrada em `public.company_features`. O middleware `RequireInventoryFeature` bloqueia requests de tenants sem a feature habilitada antes mesmo de chegar nos handlers de produto.

O endpoint `GET /api/inventories/access` informa ao frontend o status de acesso do usuário atual, incluindo se pode ler/escrever e se o módulo está pronto para uso (sem requisitos pendentes).

## Comportamento atual (já implementado na feature-001)

- `RequireInventoryFeature(db)` — middleware que consulta `public.company_features` e retorna 403 se feature `"inventory"` não está habilitada.
- `GET /api/inventories/access` — retorna `AccessStatus` com `module`, `enabled`, `can_read`, `can_write`, `ready`, `pending_requirements`.

## Para inventory MVP

Não há pré-requisitos além da feature estar habilitada. `pending_requirements` é sempre `[]` e `ready` é `true` quando a feature está ativa.

## Response do /access

```json
{
  "module": "inventory",
  "enabled": true,
  "can_read": true,
  "can_write": true,
  "ready": true,
  "pending_requirements": []
}
```
