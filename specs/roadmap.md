# Roadmap — Módulo Inventory

Roadmap incremental do módulo de inventário do ERP CDStudio. Este arquivo é a fonte canônica de progresso do módulo; as specs em `specs/` guardam documentação executável, não status implícito.

---

## Organização

| Tipo | Diretório | Uso |
|---|---|---|
| Business feature | `specs/business/feature-*/` | Capacidades de inventário percebidas no produto ou operação do tenant |
| Platform capability | `specs/platform/capability-*/` | Infraestrutura técnica, readiness operacional e fundação interna |

Cada entrega deve conter ou declarar `manifest.yaml`, `PRD.md` e, quando pronta para implementação, `SPEC.md`. O status vive no `manifest.yaml` e neste roadmap.

## Status

### Business Features

| ID | Entrega | Status | Specs | Evidência |
|---|---|---|---|---|
| `inventory.business.feature.001` | Fundação do módulo de inventário | `in_progress` | [specs/business/feature-001-foundation](specs/business/feature-001-foundation/) | `src/cmd/main.go`, `/api/inventories/health`, router base |
| `inventory.business.feature.002` | CRUD de produtos | `in_progress` | [specs/business/feature-002-product-crud](specs/business/feature-002-product-crud/) | Domínio `product`, rotas `/api/inventories/products` |
| `inventory.business.feature.003` | Feature gate do inventário | `prd` | [specs/business/feature-003-feature-gate](specs/business/feature-003-feature-gate/) | PRD criada; SPEC pendente |
| `inventory.business.feature.004` | Endpoint de busca para integração fiscal | `prd` | [specs/business/feature-004-fiscal-integration](specs/business/feature-004-fiscal-integration/) | PRD criada; SPEC pendente |

### Platform Capabilities

| ID | Entrega | Status | Specs | Evidência |
|---|---|---|---|---|
| `inventory.platform.capability.001` | Ambiente de produção | `specified` | [specs/platform/capability-001-production-environment](specs/platform/capability-001-production-environment/) | SPEC criada; configuração externa GitHub/VPS pendente |

## Estados Permitidos

| Status | Significado |
|---|---|
| `idea` | Necessidade identificada, ainda sem PRD formal |
| `prd` | PRD criada, SPEC ainda pendente |
| `specified` | PRD e SPEC prontas para implementação |
| `in_progress` | Implementação em andamento |
| `implemented` | Código entregue, ainda sem verificação final registrada |
| `verified` | Implementado e validado por evidência técnica |
| `deprecated` | Entrega obsoleta, mantida apenas por histórico |
| `superseded` | Substituída por outra entrega |

## Backlog Pós-MVP

- Movimentações de estoque (entradas, saídas, ajustes com histórico).
- Categorias e atributos de produto.
- Importação em lote (CSV/planilha).
- Alertas de estoque mínimo.
- Integração com módulo de billing para baixa automática de estoque.
- Rastreabilidade por lote/série.
