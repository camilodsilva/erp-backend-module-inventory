# SPEC — Fundação do módulo de inventário

## Resumo Executivo

SPEC retrospectivo: o módulo já está implementado. Esta spec documenta os 17 arquivos criados que formam a fundação do `erp-backend-module-inventory`: projeto Go com `go.mod`, `Dockerfile`, configuração Postgres, autenticação JWT de collaborator, feature gate por company, domínio `access`, endpoint de health e endpoint `/access`. Nenhum arquivo foi modificado em outro módulo.

---

## Impacto em Segurança e LGPD

- JWT validado no middleware `RequireCollaboratorAuth` — header `Authorization: Bearer <token>`, assinatura HMAC HS256 verificada antes de qualquer handler
- `company_id` e `tenant_id` extraídos do token, nunca do body — prevenção de spoofing
- Feature gate consulta `public.company_features` via query parametrizada (`$1`, `$2`) — sem concatenação
- Roles `inventory.read` e `inventory.write` controlam acesso granular por operação
- Nenhum dado pessoal coletado nesta feature — apenas metadados de permissão

---

## Ordem de Implementação

1. `go.mod` + `go.sum` — módulo Go com dependências
2. `Dockerfile` — imagem multi-stage com `golang:1.24-alpine`
3. `src/internal/infrastructure/config/postgres.go` — `InitPostgres()`
4. `src/internal/infrastructure/shared/id.go` — `GenerateID()`, `SchemaName()`
5. `src/internal/infrastructure/tenant/context.go` — `SetTenantID()`, `GetTenantID()`
6. `src/internal/infrastructure/auth/jwt.go` — `ValidateCollaboratorToken()`
7. `src/internal/infrastructure/auth/middleware.go` — `RequireCollaboratorAuth()`
8. `src/internal/infrastructure/auth/roles.go` — `CanReadFeature()`, `CanWriteFeature()`, `RequireFeatureRead()`, `RequireFeatureWrite()`
9. `src/internal/infrastructure/featuregate/middleware.go` — `RequireInventoryFeature()`
10. `src/internal/domain/access/entity_access.go` — `AccessStatus`, `NewAccessStatus()`
11. `src/internal/domain/access/vo_draft.go` — `Draft`, `NewDraft()`
12. `src/internal/domain/access/usecase_check.go` — `CheckUseCase`
13. `src/internal/domain/access/mock_repository.go` — stub vazio (domínio access não tem repositório)
14. `src/internal/infrastructure/dto/access.go` — `AccessResponse`, `NewAccessResponse()`
15. `src/internal/infrastructure/rest/access.go` — `accessHttpHandler`, `HandleCheck`
16. `src/internal/infrastructure/rest/router.go` — `NewRouter()`, helpers, rotas
17. `src/cmd/main.go` — wiring e startup

---

## Arquivos Criados

---

### `go.mod`

**Responsabilidade:** Declara o módulo e as dependências diretas.

```go
module github.com/camilodsilva/erp-erp-backend-module-inventory

go 1.24.2

require (
	github.com/gin-gonic/gin v1.11.0
	github.com/golang-jwt/jwt/v5 v5.2.2
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.11.2
)
```

---

### `Dockerfile`

**Responsabilidade:** Build multi-stage. Compila o binário estático em `golang:1.24-alpine`, copia para `alpine:3.21`.

```dockerfile
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /erp-inventory ./src/cmd/main.go

FROM alpine:3.21

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

COPY --from=builder /erp-inventory .

EXPOSE 8082

CMD ["./erp-inventory"]
```

---

### `src/internal/infrastructure/config/postgres.go`

**Responsabilidade:** Abre e valida a conexão Postgres a partir das variáveis de ambiente.

```go
package config

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func InitPostgres() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
```

---

### `src/internal/infrastructure/shared/id.go`

**Responsabilidade:** Geração de UUIDv7 e derivação do nome de schema tenant.

```go
package shared

import (
	"strings"

	"github.com/google/uuid"
)

func GenerateID() string {
	id, _ := uuid.NewV7()
	return id.String()
}

type UUIDGenerator struct{}

func (g *UUIDGenerator) Generate() string {
	return GenerateID()
}

func SchemaName(tenantID string) string {
	return "t_" + strings.ReplaceAll(tenantID, "-", "")
}
```

---

### `src/internal/infrastructure/tenant/context.go`

**Responsabilidade:** Armazena e recupera o `tenant_id` do contexto Gin.

```go
package tenant

import "github.com/gin-gonic/gin"

const tenantIDKey = "tenant_id"

func SetTenantID(c *gin.Context, tenantID string) {
	c.Set(tenantIDKey, tenantID)
}

func GetTenantID(c *gin.Context) string {
	v, _ := c.Get(tenantIDKey)
	id, _ := v.(string)
	return id
}
```

---

### `src/internal/infrastructure/auth/jwt.go`

**Responsabilidade:** Valida tokens JWT de collaborator com claims customizados.

```go
package auth

import (
	"errors"

	"github.com/golang-jwt/jwt/v5"
)

type CollaboratorClaims struct {
	Sub       string   `json:"sub"`
	Type      string   `json:"type"`
	CompanyID string   `json:"company_id"`
	TenantID  string   `json:"tenant_id"`
	Roles     []string `json:"roles"`
	Status    string   `json:"status"`
	jwt.RegisteredClaims
}

func ValidateCollaboratorToken(tokenString, secret string) (*CollaboratorClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CollaboratorClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CollaboratorClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
```

---

### `src/internal/infrastructure/auth/middleware.go`

**Responsabilidade:** Middleware Gin que extrai e valida o JWT de collaborator, populando o contexto com `actor_id`, `company_id`, `roles`, `collaborator_status` e `tenant_id`.

```go
package auth

import (
	"net/http"
	"strings"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/tenant"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequireCollaboratorAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := extractBearerToken(c)
		if token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		claims, err := ValidateCollaboratorToken(token, jwtSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}
		if claims.Type != "collaborator" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}
		if _, err := uuid.Parse(claims.TenantID); err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "unauthorized"})
			return
		}

		c.Set("actor_id", claims.Sub)
		c.Set("company_id", claims.CompanyID)
		c.Set("roles", claims.Roles)
		c.Set("collaborator_status", claims.Status)
		tenant.SetTenantID(c, claims.TenantID)
		c.Next()
	}
}

func extractBearerToken(c *gin.Context) string {
	auth := c.GetHeader("Authorization")
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return ""
	}

	return parts[1]
}
```

---

### `src/internal/infrastructure/auth/roles.go`

**Responsabilidade:** Funções puras de verificação de roles e middlewares Gin derivados.

```go
package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CanReadFeature(roles []string, feature string) bool {
	if feature == "" {
		return false
	}

	for _, role := range roles {
		if role == "read" || role == "write" || role == feature+".read" || role == feature+".write" {
			return true
		}
	}

	return false
}

func CanWriteFeature(roles []string, feature string) bool {
	if feature == "" {
		return false
	}

	for _, role := range roles {
		if role == "write" || role == feature+".write" {
			return true
		}
	}

	return false
}

func RequireFeatureRead(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CanReadFeature(rolesFromContext(c), feature) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}

		c.Next()
	}
}

func RequireFeatureWrite(feature string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CanWriteFeature(rolesFromContext(c), feature) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "forbidden"})
			return
		}

		c.Next()
	}
}

func rolesFromContext(c *gin.Context) []string {
	raw, ok := c.Get("roles")
	if !ok {
		return make([]string, 0)
	}

	roles, ok := raw.([]string)
	if !ok {
		return make([]string, 0)
	}

	return roles
}
```

---

### `src/internal/infrastructure/featuregate/middleware.go`

**Responsabilidade:** Middleware Gin que verifica se a feature `"inventory"` está habilitada para a company do token. Consulta `public.company_features` via SQL parametrizado.

```go
package featuregate

import (
	"database/sql"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	inventoryFeatureSlug                 = "inventory"
	inventoryFeatureDisabledMessage      = "inventory module not enabled for this company"
	inventoryFeatureInternalErrorMessage = "internal server error"
	hasCompanyFeatureEnabledQuery        = `
		select exists (
			select 1
			from public.company_features cf
			join public.features f on f.id = cf.feature_id
			where cf.company_id = $1
			  and f.title = $2
			limit 1
		)
	`
)

type (
	featureGateRepository interface {
		HasFeature(companyID, featureSlug string) (bool, error)
	}

	postgresFeatureGateRepository struct {
		db *sql.DB
	}
)

func newPostgresFeatureGateRepository(db *sql.DB) *postgresFeatureGateRepository {
	return &postgresFeatureGateRepository{db: db}
}

func (r *postgresFeatureGateRepository) HasFeature(companyID, featureSlug string) (bool, error) {
	if r.db == nil {
		return false, errors.New("database connection not configured")
	}

	var enabled bool
	err := r.db.QueryRow(hasCompanyFeatureEnabledQuery, companyID, featureSlug).Scan(&enabled)
	if err != nil {
		return false, err
	}

	return enabled, nil
}

func RequireInventoryFeature(db *sql.DB) gin.HandlerFunc {
	return requireInventoryFeature(newPostgresFeatureGateRepository(db))
}

func requireInventoryFeature(repository featureGateRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		companyID := c.GetString("company_id")
		if companyID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": inventoryFeatureDisabledMessage})
			return
		}

		enabled, err := repository.HasFeature(companyID, inventoryFeatureSlug)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": inventoryFeatureInternalErrorMessage})
			return
		}
		if !enabled {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": inventoryFeatureDisabledMessage})
			return
		}

		c.Next()
	}
}
```

---

### `src/internal/domain/access/entity_access.go`

**Responsabilidade:** Erro de domínio, constante do módulo, struct `AccessStatus` e seu construtor.

```go
package access

import "errors"

var ErrAccessTenantRequired = errors.New("tenant id is required")

const ModuleInventory = "inventory"

type AccessStatus struct {
	Module              string
	Enabled             bool
	CanRead             bool
	CanWrite            bool
	Ready               bool
	PendingRequirements []string
}

func NewAccessStatus(canRead, canWrite bool) AccessStatus {
	return AccessStatus{
		Module:              ModuleInventory,
		Enabled:             true,
		CanRead:             canRead,
		CanWrite:            canWrite,
		Ready:               true,
		PendingRequirements: []string{},
	}
}
```

---

### `src/internal/domain/access/vo_draft.go`

**Responsabilidade:** Valida que `tenantID` não está vazio antes de criar o Draft de acesso.

```go
package access

import "strings"

type Draft struct {
	TenantID string
	CanRead  bool
	CanWrite bool
}

func NewDraft(tenantID string, canRead, canWrite bool) (Draft, error) {
	draft := Draft{
		TenantID: strings.TrimSpace(tenantID),
		CanRead:  canRead,
		CanWrite: canWrite,
	}

	if draft.TenantID == "" {
		return Draft{}, ErrAccessTenantRequired
	}

	return draft, nil
}
```

---

### `src/internal/domain/access/usecase_check.go`

**Responsabilidade:** Orquestra a verificação de acesso. O domínio `access` não tem repositório — `CheckUseCase` constrói o `AccessStatus` a partir do draft validado.

```go
package access

type CheckUseCase struct{}

func NewCheckUseCase() *CheckUseCase {
	return &CheckUseCase{}
}

func (u *CheckUseCase) Execute(draft Draft) (AccessStatus, error) {
	return NewAccessStatus(draft.CanRead, draft.CanWrite), nil
}
```

---

### `src/internal/domain/access/mock_repository.go`

**Responsabilidade:** Stub vazio. O domínio `access` não declara interface `Repository` pois não persiste estado.

```go
package access
```

---

### `src/internal/infrastructure/dto/access.go`

**Responsabilidade:** Serialização do `AccessStatus` para resposta HTTP.

```go
package dto

import "github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/access"

type AccessResponse struct {
	Module              string   `json:"module"`
	Enabled             bool     `json:"enabled"`
	CanRead             bool     `json:"can_read"`
	CanWrite            bool     `json:"can_write"`
	Ready               bool     `json:"ready"`
	PendingRequirements []string `json:"pending_requirements"`
}

func NewAccessResponse(status access.AccessStatus) AccessResponse {
	return AccessResponse{
		Module:              status.Module,
		Enabled:             status.Enabled,
		CanRead:             status.CanRead,
		CanWrite:            status.CanWrite,
		Ready:               status.Ready,
		PendingRequirements: status.PendingRequirements,
	}
}
```

---

### `src/internal/infrastructure/rest/access.go`

**Responsabilidade:** Handler HTTP para `GET /api/inventories/access`. Extrai roles do contexto, constrói draft, executa use case e responde.

```go
package rest

import (
	"errors"
	"log"
	"net/http"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/domain/access"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/auth"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/dto"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/tenant"
	"github.com/gin-gonic/gin"
)

type accessHttpHandler struct {
	checkUseCase *access.CheckUseCase
}

func newAccessHttpHandler() *accessHttpHandler {
	return &accessHttpHandler{
		checkUseCase: access.NewCheckUseCase(),
	}
}

func (h *accessHttpHandler) HandleCheck(c *gin.Context) {
	roles := rolesFromContext(c)
	draft, err := access.NewDraft(
		tenant.GetTenantID(c),
		auth.CanReadFeature(roles, access.ModuleInventory),
		auth.CanWriteFeature(roles, access.ModuleInventory),
	)
	if err != nil {
		handleAccessError(c, err)
		return
	}

	status, err := h.checkUseCase.Execute(draft)
	if err != nil {
		handleAccessError(c, err)
		return
	}

	buildResponseSuccess(c, http.StatusOK, dto.NewAccessResponse(status))
}

func handleAccessError(c *gin.Context, err error) {
	if errors.Is(err, access.ErrAccessTenantRequired) {
		buildResponseError(c, http.StatusForbidden, access.ErrAccessTenantRequired)
		return
	}

	log.Printf("access handler error: %v", err)
	buildResponseError(c, http.StatusInternalServerError, errors.New("internal server error"))
}
```

---

### `src/internal/infrastructure/rest/router.go`

**Responsabilidade:** Registra todas as rotas. Aplica middlewares de autenticação, feature gate e permissão de role. Contém helpers compartilhados pelos handlers.

```go
package rest

import (
	"database/sql"
	"net/http"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/auth"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/featuregate"
	"github.com/gin-gonic/gin"
)

type router struct {
	Server *gin.Engine
}

func NewRouter(db *sql.DB, jwtSecret string) *router {
	r := gin.Default()

	r.GET("/api/inventories/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "module": "inventory"})
	})

	inventory := r.Group("/api/inventories")
	inventory.Use(auth.RequireCollaboratorAuth(jwtSecret))
	inventory.Use(featuregate.RequireInventoryFeature(db))
	inventory.Use(auth.RequireFeatureRead("inventory"))
	{
		accessHandler := newAccessHttpHandler()
		inventory.GET("/access", accessHandler.HandleCheck)

		productHandler := newProductHttpHandler(db)
		inventory.GET("/products", productHandler.HandleList)
		inventory.GET("/products/:id", productHandler.HandleFindByID)
		inventory.POST("/products", auth.RequireFeatureWrite("inventory"), productHandler.HandleCreate)
		inventory.PUT("/products/:id", auth.RequireFeatureWrite("inventory"), productHandler.HandleUpdate)
		inventory.DELETE("/products/:id", auth.RequireFeatureWrite("inventory"), productHandler.HandleDelete)
	}

	return &router{Server: r}
}

func buildResponseError(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"message": err.Error()})
}

func buildResponseSuccess(c *gin.Context, status int, content any) {
	c.JSON(status, content)
}

func parseStringToInt(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	v := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return defaultVal
		}
		v = v*10 + int(ch-'0')
	}
	return v
}

func actorIDFromContext(c *gin.Context) string {
	return c.GetString("actor_id")
}

func rolesFromContext(c *gin.Context) []string {
	raw, ok := c.Get("roles")
	if !ok {
		return make([]string, 0)
	}
	roles, ok := raw.([]string)
	if !ok {
		return make([]string, 0)
	}
	return roles
}
```

---

### `src/cmd/main.go`

**Responsabilidade:** Wiring de dependências e startup. Falha explicitamente se `JWT_SECRET` estiver ausente.

```go
package main

import (
	"log"
	"os"

	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/config"
	"github.com/camilodsilva/erp-erp-backend-module-inventory/src/internal/infrastructure/rest"
)

func main() {
	postgres, err := config.InitPostgres()
	if err != nil {
		log.Fatal(err)
	}
	defer postgres.Close()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET env var is required")
	}

	rest.NewRouter(postgres, jwtSecret).Server.Run(":8082")
}
```

---

## Arquivos Deletados

Nenhum.

---

## Testes Unitários

### Arquivos a criar

| Arquivo | Pacote | Cenários obrigatórios |
|---------|--------|----------------------|
| `src/internal/domain/access/vo_draft_test.go` | `access` | Happy path com tenantID válido; `ErrAccessTenantRequired` quando tenantID vazio; `ErrAccessTenantRequired` quando tenantID só espaços |
| `src/internal/domain/access/usecase_check_test.go` | `access` | Happy path: `can_read=true`, `can_write=true`; `can_read=true`, `can_write=false`; `can_read=false`, `can_write=false` |

### Naming obrigatório

```
TestAccessDraft_NewDraft_Success
TestAccessDraft_NewDraft_EmptyTenantID
TestAccessDraft_NewDraft_WhitespaceOnlyTenantID
TestCheckAccessUseCase_Execute_FullAccess
TestCheckAccessUseCase_Execute_ReadOnly
TestCheckAccessUseCase_Execute_NoAccess
```

### Exemplo de estrutura (`vo_draft_test.go`)

```go
package access

import "testing"

func TestAccessDraft_NewDraft_Success(t *testing.T) {
    draft, err := NewDraft("tenant-uuid", true, true)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if draft.TenantID != "tenant-uuid" {
        t.Errorf("expected tenant-uuid, got %s", draft.TenantID)
    }
}

func TestAccessDraft_NewDraft_EmptyTenantID(t *testing.T) {
    _, err := NewDraft("", true, true)
    if err != ErrAccessTenantRequired {
        t.Errorf("expected ErrAccessTenantRequired, got %v", err)
    }
}

func TestAccessDraft_NewDraft_WhitespaceOnlyTenantID(t *testing.T) {
    _, err := NewDraft("   ", true, true)
    if err != ErrAccessTenantRequired {
        t.Errorf("expected ErrAccessTenantRequired, got %v", err)
    }
}
```

### Exemplo de estrutura (`usecase_check_test.go`)

```go
package access

import "testing"

func TestCheckAccessUseCase_Execute_FullAccess(t *testing.T) {
    draft, _ := NewDraft("tenant-uuid", true, true)
    uc := NewCheckUseCase()
    status, err := uc.Execute(draft)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if !status.CanRead || !status.CanWrite {
        t.Errorf("expected full access, got can_read=%v can_write=%v", status.CanRead, status.CanWrite)
    }
    if !status.Ready {
        t.Errorf("expected ready=true")
    }
    if len(status.PendingRequirements) != 0 {
        t.Errorf("expected empty pending_requirements")
    }
}

func TestCheckAccessUseCase_Execute_ReadOnly(t *testing.T) {
    draft, _ := NewDraft("tenant-uuid", true, false)
    uc := NewCheckUseCase()
    status, err := uc.Execute(draft)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if !status.CanRead || status.CanWrite {
        t.Errorf("expected read-only, got can_read=%v can_write=%v", status.CanRead, status.CanWrite)
    }
}

func TestCheckAccessUseCase_Execute_NoAccess(t *testing.T) {
    draft, _ := NewDraft("tenant-uuid", false, false)
    uc := NewCheckUseCase()
    status, err := uc.Execute(draft)
    if err != nil {
        t.Fatalf("expected no error, got %v", err)
    }
    if status.CanRead || status.CanWrite {
        t.Errorf("expected no access, got can_read=%v can_write=%v", status.CanRead, status.CanWrite)
    }
}
```

---

## MODELING.md

Nenhuma tabela criada nesta feature. A fundação estabelece apenas a estrutura Go e os endpoints de health/access — sem DDL de tenant. A tabela `inventory_product` é criada na feature-002.

Sem atualização necessária em [MODELING.md](../../../../MODELING.md).

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
# Health (sem autenticação)
curl -i http://localhost:8082/api/inventories/health
# Esperado: 200 {"status":"ok","module":"inventory"}

# Access sem token
curl -i http://localhost:8082/api/inventories/access
# Esperado: 401 {"message":"unauthorized"}

# Access com token válido mas feature desabilitada
curl -i http://localhost:8082/api/inventories/access \
  -H "Authorization: Bearer <TOKEN_SEM_FEATURE>"
# Esperado: 403 {"message":"inventory module not enabled for this company"}

# Access com token válido e feature habilitada
curl -i http://localhost:8082/api/inventories/access \
  -H "Authorization: Bearer <TOKEN_COM_FEATURE>"
# Esperado: 200 {"module":"inventory","enabled":true,"can_read":true,"can_write":true,...}
```
