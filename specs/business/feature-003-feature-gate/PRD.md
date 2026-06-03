# PRD — Feature Gate do Inventário

**ID:** `inventory.business.feature.003`
**Status:** `implemented`

## Contexto

O ERP CDStudio é um SaaS multi-tenant onde diferentes clientes (empresas) contratam diferentes módulos. O módulo de inventário não deve estar disponível para todas as empresas — apenas para aquelas que efetivamente contrataram esse serviço. Além disso, dentro de uma empresa com o módulo contratado, os colaboradores podem ter diferentes níveis de acesso: alguns apenas leem o catálogo, outros podem criar e modificar produtos.

Essa feature documenta o comportamento de controle de acesso ao módulo de inventário, que na prática foi implementado como parte da fundação (feature-001). A separação em feature própria serve para deixar explícita a regra de negócio do feature gate e do endpoint de status de acesso, que serão consumidos diretamente pelo frontend.

## Objetivo de Negócio

Garantir que:
1. Apenas empresas com a feature `"inventory"` contratada e habilitada no backoffice possam acessar qualquer operação do módulo de inventário
2. O frontend possa consultar o status de acesso do colaborador autenticado para exibir ou ocultar funcionalidades de inventário na interface

## Personas e Permissões de Negócio

| Persona | Acesso ao módulo | Permissões de operação |
|---------|-----------------|----------------------|
| Colaborador de empresa com feature habilitada e role `inventory.read` | Permitido | Pode listar, buscar e consultar status de acesso |
| Colaborador de empresa com feature habilitada e role `inventory.write` | Permitido | Pode criar, alterar e remover produtos (implica leitura) |
| Colaborador de empresa sem a feature `"inventory"` habilitada | Bloqueado | Nenhuma operação — 403 em todos os endpoints |
| Colaborador sem role `inventory.read` nem `inventory.write` | Bloqueado nas operações correspondentes | Não pode executar operações de leitura ou escrita de produtos |
| Manager do backoffice | Controla se a feature está habilitada por empresa | Age no módulo common, não no inventory |

## Escopo

### Dentro do escopo

- Verificação de feature gate antes de qualquer operação do módulo: a feature `"inventory"` deve estar habilitada para a empresa do colaborador autenticado
- Endpoint `GET /api/inventories/access` que informa ao frontend: se o módulo está habilitado, se o colaborador pode ler, se pode escrever, se o módulo está pronto para uso e se há requisitos pendentes
- Para o MVP: `ready: true` e `pending_requirements: []` sempre que o acesso é permitido

### Fora do escopo

- Gestão de features por empresa (feita no módulo common pelo backoffice de Manager)
- Controle de acesso por recurso individual (todos os produtos de um tenant são acessíveis para o colaborador com a role correta)
- Funcionalidades específicas de produto — apenas o controle de acesso ao módulo é abordado aqui

## Regras de Negócio

| Código | Regra | Comportamento esperado |
|--------|-------|----------------------|
| RN-001 | A feature `"inventory"` deve estar habilitada para a empresa do colaborador antes de qualquer operação do módulo ser processada | **Dado que** um colaborador autenticado pertence a uma empresa sem a feature `"inventory"` habilitada, **Quando** acessa qualquer endpoint do módulo de inventário, **Então** recebe 403 com mensagem `"inventory module not enabled for this company"` |
| RN-002 | O endpoint `/access` retorna o status real de acesso: se o módulo está habilitado, se pode ler e se pode escrever | **Dado que** um colaborador com feature habilitada e role `inventory.read` acessa `/access`, **Quando** o sistema processa a requisição, **Então** recebe `enabled: true`, `can_read: true`, `can_write: false` |
| RN-003 | O endpoint `/access` é protegido por autenticação JWT e pelo próprio feature gate — apenas colaboradores de empresas com a feature habilitada chegam ao handler | **Dado que** um colaborador sem token tenta acessar `/access`, **Quando** o sistema processa a requisição, **Então** recebe 401 antes de qualquer verificação de feature |
| RN-004 | Para o MVP do módulo inventory, não há pré-requisitos operacionais além da feature estar habilitada. `pending_requirements` é sempre `[]` e `ready` é sempre `true` quando o acesso é concedido | **Dado que** a feature `"inventory"` está habilitada para a empresa do colaborador, **Quando** o colaborador acessa `/access`, **Então** `ready: true` e `pending_requirements: []` |
| RN-005 | A verificação de feature é feita consultando `public.company_features` usando o `company_id` do token — nunca um valor fornecido pelo cliente | **Dado que** um cliente tenta manipular o `company_id` para simular acesso a outra empresa, **Quando** o sistema verifica a feature, **Então** usa exclusivamente o `company_id` do JWT assinado |

## Estados e Transições

Sem estados ou transições de negócio relevantes nesta feature. O feature gate é uma verificação estática a cada request — não há ciclo de vida de acesso gerenciado aqui.

## Fluxos de Negócio

### Fluxo de verificação de feature gate (toda requisição ao módulo)

1. O colaborador envia uma requisição autenticada para qualquer endpoint do módulo de inventário
2. O sistema verifica o JWT e extrai o `company_id`
3. O sistema consulta se a empresa tem a feature `"inventory"` habilitada
4. Se não habilitada: rejeita a requisição com 403 antes de qualquer processamento de negócio
5. Se habilitada: a requisição prossegue para o handler correspondente

### Fluxo de consulta de status de acesso (`GET /api/inventories/access`)

1. O frontend autentica o colaborador e envia o token para `GET /api/inventories/access`
2. O sistema valida o JWT, extrai o `company_id`, verifica a feature (fluxo acima), verifica roles
3. O sistema determina se o colaborador pode ler (`inventory.read` ou `inventory.write`) e se pode escrever (`inventory.write`)
4. Retorna o status completo ao frontend: `module`, `enabled`, `can_read`, `can_write`, `ready`, `pending_requirements`
5. O frontend usa essa resposta para exibir ou ocultar funcionalidades de inventário

## Critérios de Aceite

> **Dado que** uma empresa não tem a feature `"inventory"` habilitada e um colaborador dessa empresa está autenticado, **Quando** acessa qualquer endpoint do módulo de inventário, **Então** recebe 403 com `{"message":"inventory module not enabled for this company"}`. _(RN-001)_

> **Dado que** um colaborador autenticado tem a feature habilitada e apenas a role `inventory.read`, **Quando** consulta `GET /api/inventories/access`, **Então** recebe 200 com `enabled: true`, `can_read: true`, `can_write: false`, `ready: true`, `pending_requirements: []`. _(RN-002, RN-004)_

> **Dado que** um colaborador autenticado tem a feature habilitada e as roles `inventory.read` e `inventory.write`, **Quando** consulta `GET /api/inventories/access`, **Então** recebe 200 com `enabled: true`, `can_read: true`, `can_write: true`, `ready: true`, `pending_requirements: []`. _(RN-002, RN-004)_

> **Dado que** qualquer agente acessa `GET /api/inventories/access` sem token JWT, **Quando** o sistema processa a requisição, **Então** recebe 401 com `{"message":"unauthorized"}`. _(RN-003)_

## Dependências de Negócio

- Feature `"inventory"` deve estar cadastrada na tabela `public.features` do módulo common
- A habilitação ou desabilitação da feature por empresa é feita no backoffice de Manager (módulo common)
- O módulo common deve estar em execução e acessível para que o feature gate possa consultar `public.company_features`
- O `JWT_SECRET` deve ser compartilhado entre o módulo common (emissor do token) e o módulo inventory (validador)

## Análise de Segurança e LGPD

**Autenticação/Autorização por role:** o feature gate é verificado após a autenticação JWT. O endpoint `/access` exige adicionalmente role `inventory.read` para ser processado. A sequência de middlewares é: autenticação → feature gate → role check.

**Autorização por recurso/tenant:** o `company_id` usado na consulta de feature gate vem exclusivamente do JWT — sem possibilidade de spoofing pelo cliente.

**Validação de entrada:** o endpoint `/access` não aceita body — sem validação de entrada necessária além do JWT.

**Proteção contra mass assignment:** não aplicável — sem body de entrada.

**Minimização de dados expostos:** o response de `/access` expõe apenas metadados de permissão — sem dados pessoais ou informações do catálogo.

**SQL Injection:** a query de verificação de feature usa parâmetros posicionais (`$1`, `$2`).

**Isolamento de tenant:** a verificação usa o `company_id` do token — garantia de que a verificação é sempre feita para a empresa correta.

**Concorrência e idempotência:** verificação é leitura pura — idempotente.

**Auditoria:** não aplicável — sem operações de escrita.

**Logs e observabilidade:** erros de banco no feature gate são logados com status 500 — sem exposição de dados sensíveis.

**Segredos e credenciais:** não aplicável — nenhum segredo adicional além do `JWT_SECRET`.

**Rate limit e abuso:** não implementado no MVP. O feature gate em si tem baixo risco.

**Dados pessoais (LGPD):** nenhum dado pessoal coletado ou processado.

## Riscos e Pontos de Atenção

- **Falha no banco ao verificar feature gate:** se o banco estiver indisponível durante a consulta de `company_features`, o feature gate retorna 500 — o módulo inteiro fica inacessível. Não há fallback ou cache de features implementado no MVP.
- **Feature não cadastrada no common:** se a feature `"inventory"` não estiver cadastrada na tabela `public.features`, a query retornará `false` para todas as empresas. Isso bloquearia o acesso mesmo para empresas habilitadas. O setup inicial da feature é responsabilidade do processo de onboarding.
- **Roles genéricas (`read`, `write`):** o sistema aceita roles genéricas além das específicas por módulo. Isso pode conceder acesso involuntário se um colaborador tiver apenas a role genérica `read` sem intenção de acessar o módulo de inventário.

## Evolução da Feature

Não aplicável. Esta é a PRD inicial. A implementação aconteceu integralmente como parte da feature-001 (Fundação).
