# SPEC — Feature Gate do Inventário

## Resumo Executivo

Toda a implementação descrita nesta feature foi entregue como parte de `inventory.business.feature.001` (Fundação). Nenhum arquivo novo precisa ser criado ou modificado. Esta SPEC documenta os artefatos existentes que realizam o feature gate e o endpoint `/access`.

---

## Impacto em Segurança e LGPD

- **Autenticação/Autorização por role:** feature gate é executado após validação JWT e antes de qualquer handler. Endpoint `/access` exige adicionalmente role `inventory.read`. Sequência de middlewares: `RequireCollaboratorAuth` → `RequireInventoryFeature` → `RequireFeatureRead("inventory")`.
- **Autorização por recurso/tenant:** `company_id` para consulta de feature vem exclusivamente do JWT. Sem possibilidade de spoofing pelo cliente.
- **Validação de entrada no Draft/VO:** `access.NewDraft` valida que `tenantID` não está vazio. Endpoint `/access` não aceita body.
- **Proteção contra mass assignment:** não aplicável — sem body de entrada.
- **Minimização de dados em responses:** response de `/access` expõe apenas metadados de permissão — sem PII ou dados do catálogo.
- **SQL Injection:** query do feature gate usa `$1` e `$2` — sem concatenação.
- **Isolamento de tenant:** `company_id` do token garante que a verificação é sempre para a empresa correta.
- **Concorrência e idempotência:** verificação é leitura pura — sem efeito colateral.
- **Auditoria:** sem operações de escrita.
- **Logs e observabilidade:** erros de banco no feature gate retornam 500 — sem exposição de payload sensível.
- **Segredos e credenciais:** nenhum segredo adicional além do `JWT_SECRET`.
- **Rate limit e abuso:** não implementado no MVP.
- **Dados pessoais (LGPD):** nenhum dado pessoal coletado ou processado.

---

## Decisões de Domínio e Clean Architecture

**Feature gate como middleware de infraestrutura:** `RequireInventoryFeature` é middleware de infraestrutura pura — não contém regra de negócio além da decisão binária de bloquear ou não baseada no resultado do banco. A interface interna `featureGateRepository` permite testar o comportamento sem banco real.

**Domínio `access` sem repositório:** `CheckUseCase` não persiste — orquestra apenas a construção de `AccessStatus` a partir do draft validado. Esse padrão é válido quando o resultado deriva exclusivamente de dados do token e não exige persistência.

**Sequência de middlewares:** a ordem `autenticação → feature gate → role check` é intencional. O feature gate só executa depois que o colaborador está autenticado — sem autenticação, não há `company_id` disponível. O role check vem por último porque depende do feature gate já ter liberado o acesso.

**Checklist de Qualidade Arquitetural:**
- [x] DDD: regra de validação de `tenantID` está no VO `access.NewDraft`
- [x] Modelo não anêmico: `NewAccessStatus` constrói estado completo
- [x] Use cases: `CheckUseCase` apenas orquestra — sem política de negócio
- [x] Infraestrutura: `RequireInventoryFeature` e `rolesFromContext` são infraestrutura pura
- [x] Clean Architecture: domínio `access` não importa pacotes de infraestrutura
- [x] Contratos: `/access` response minimiza dados
- [x] Banco/modelagem: sem tabelas novas
- [x] TDD: testes cobrem `vo_draft` e `usecase_check`; cenários de feature gate adequados para integração
- [x] Padrões CDStudio: construtor privado, mock manual (stub vazio), SQL como const

---

## Débitos Técnicos da Feature

Toda a implementação foi entregue na feature-001. Os DTs abaixo são de verificação.

| Código | Origem | Débito técnico | Camada | Arquivos previstos | Verificação |
|--------|--------|----------------|--------|--------------------|-------------|
| DT-001 | RN-001 | Verificar que `RequireInventoryFeature` bloqueia com 403 quando feature não habilitada | Infra | `featuregate/middleware.go` | `bash scripts/integration/inventory_foundation.sh` |
| DT-002 | RN-002 | Verificar que `/access` retorna `can_read`, `can_write`, `ready`, `pending_requirements` corretos | HTTP | `rest/access.go` | Curl com token de read-only → `can_write: false` |
| DT-003 | RN-003 | Verificar que `/access` retorna 401 sem token | HTTP | `rest/router.go` (middleware) | `curl /access` sem Authorization → 401 |
| DT-004 | RN-004 | Verificar que `pending_requirements: []` e `ready: true` para todo acesso concedido | Domínio | `domain/access/entity_access.go` | `go test ./src/internal/domain/access/...` |
| DT-005 | RN-005 | Verificar que `company_id` do feature gate vem exclusivamente do JWT | Infra | `featuregate/middleware.go` | Code review — `c.GetString("company_id")` usa valor do contexto JWT |

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
