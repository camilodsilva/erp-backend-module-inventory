# SPEC — Feature gate do inventário

## Resumo Executivo

Toda a implementação descrita nesta feature foi entregue como parte de `inventory.business.feature.001` (Fundação). Nenhum arquivo novo precisa ser criado ou modificado. Esta SPEC documenta os artefatos existentes que realizam o feature gate e o endpoint `/access`.

---

## Implementação — Referência Cruzada com feature-001

### Middleware de feature gate

**Arquivo:** [src/internal/infrastructure/featuregate/middleware.go](../../../src/internal/infrastructure/featuregate/middleware.go)

Consulta `public.company_features` via query parametrizada. Retorna `403` se a feature `"inventory"` não estiver habilitada para a `company_id` do token JWT.

Registrado no router como segundo middleware do grupo `/api/inventories`:

```go
inventory.Use(auth.RequireCollaboratorAuth(jwtSecret))
inventory.Use(featuregate.RequireInventoryFeature(db))   // ← aqui
inventory.Use(auth.RequireFeatureRead("inventory"))
```

### Domínio access

**Arquivos:**
- [src/internal/domain/access/entity_access.go](../../../src/internal/domain/access/entity_access.go) — `AccessStatus`, `NewAccessStatus()`
- [src/internal/domain/access/vo_draft.go](../../../src/internal/domain/access/vo_draft.go) — `Draft`, `NewDraft()` (valida `tenantID` não vazio)
- [src/internal/domain/access/usecase_check.go](../../../src/internal/domain/access/usecase_check.go) — `CheckUseCase.Execute()` constrói `AccessStatus` a partir do draft

### Endpoint `/access`

**Arquivos:**
- [src/internal/infrastructure/dto/access.go](../../../src/internal/infrastructure/dto/access.go) — `AccessResponse`, `NewAccessResponse()`
- [src/internal/infrastructure/rest/access.go](../../../src/internal/infrastructure/rest/access.go) — `accessHttpHandler.HandleCheck`

**Rota registrada em** [src/internal/infrastructure/rest/router.go](../../../src/internal/infrastructure/rest/router.go):

```go
inventory.GET("/access", accessHandler.HandleCheck)
```

---

## Comportamento documentado

### `GET /api/inventories/access`

**Pré-condições de chegada:** JWT válido de collaborator + feature `"inventory"` habilitada para a company (ambos verificados pelos middlewares anteriores).

**Response 200:**
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

**Erros mapeados no handler:**

| Condição | Status | Body |
|----------|--------|------|
| `tenantID` vazio no contexto | 403 | `{"message":"tenant id is required"}` |
| JWT ausente ou inválido | 401 | `{"message":"unauthorized"}` (middleware) |
| Feature `inventory` desabilitada | 403 | `{"message":"inventory module not enabled for this company"}` (middleware) |
| Role `inventory.read` ausente | 403 | `{"message":"forbidden"}` (middleware) |

---

## Testes Unitários

Os testes desta feature residem nos arquivos da feature-001, que é onde a implementação vive.

### Arquivos a criar (cobertos pela feature-001)

| Arquivo | Cenários obrigatórios |
|---------|----------------------|
| `src/internal/domain/access/vo_draft_test.go` | Já especificado em feature-001 SPEC |
| `src/internal/domain/access/usecase_check_test.go` | Já especificado em feature-001 SPEC |

### Testes adicionais específicos do feature gate

O `RequireInventoryFeature` usa uma interface interna `featureGateRepository` que permite testar o middleware sem banco:

| Cenário | Comportamento esperado |
|---------|----------------------|
| `company_id` vazio no contexto | `403 Forbidden` com `"inventory module not enabled for this company"` |
| `HasFeature` retorna `false` | `403 Forbidden` |
| `HasFeature` retorna erro de banco | `500 Internal Server Error` |
| `HasFeature` retorna `true` | `c.Next()` chamado — request prossegue |

Esses cenários são mais adequados como testes de integração (contra banco real) no script `scripts/integration/`.

---

## MODELING.md

Nenhuma tabela nova. Sem atualização necessária.

---

## Arquivos Criados

Nenhum.

## Arquivos Modificados

Nenhum.

---

## Checklist de Verificação

### Build

```bash
cd erp-backend-module-inventory
go build ./...
```

### End-to-End

```bash
TOKEN_READ="<JWT com roles: [inventory.read]>"
TOKEN_WRITE="<JWT com roles: [inventory.read, inventory.write]>"
TOKEN_NO_FEATURE="<JWT de company sem feature inventory>"

# Access — read only
curl -s http://localhost:8082/api/inventories/access \
  -H "Authorization: Bearer $TOKEN_READ" | jq .
# Esperado: 200, can_read: true, can_write: false

# Access — read+write
curl -s http://localhost:8082/api/inventories/access \
  -H "Authorization: Bearer $TOKEN_WRITE" | jq .
# Esperado: 200, can_read: true, can_write: true

# Access — feature não habilitada
curl -s http://localhost:8082/api/inventories/access \
  -H "Authorization: Bearer $TOKEN_NO_FEATURE" | jq .
# Esperado: 403 {"message":"inventory module not enabled for this company"}

# Access — sem token
curl -s http://localhost:8082/api/inventories/access
# Esperado: 401 {"message":"unauthorized"}
```
