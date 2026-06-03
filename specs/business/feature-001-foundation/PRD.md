# PRD — Fundação do Módulo de Inventário

**ID:** `inventory.business.feature.001`
**Status:** `implemented`

## Contexto

O ERP CDStudio precisa de um módulo dedicado ao gerenciamento do catálogo de produtos que cada cliente (tenant) comercializa. Para que esse módulo possa existir como serviço independente e seguro dentro da arquitetura multi-tenant do ERP, é necessário primeiro construir a fundação: estrutura do projeto Go, conectividade com o banco de dados, autenticação JWT de colaboradores, controle de acesso por feature e por role, e os endpoints mínimos de saúde e verificação de acesso.

Sem essa fundação, nenhuma feature de produto (CRUD, busca, integração fiscal) pode ser implementada com segurança ou de forma rastreável.

## Objetivo de Negócio

Disponibilizar o módulo `erp-backend-module-inventory` como serviço HTTP operacional, capaz de:

- Receber requisições autenticadas de colaboradores de qualquer tenant
- Validar se o tenant tem a feature `"inventory"` habilitada antes de processar qualquer operação
- Confirmar para o frontend o status de acesso do colaborador autenticado (pode ler, pode escrever, módulo pronto para uso)
- Persistir dados no schema isolado do tenant no PostgreSQL

O módulo estará pronto para receber features de produto quando este conjunto de capacidades estiver operacional e testado.

## Personas e Permissões de Negócio

| Persona | Contexto de uso | Permissões |
|---------|----------------|------------|
| Colaborador com `inventory.read` | Usuário autenticado de uma empresa que tem o módulo inventory habilitado | Pode consultar status de acesso e listar/buscar produtos |
| Colaborador com `inventory.write` | Usuário autenticado de uma empresa que tem o módulo inventory habilitado | Pode consultar status de acesso, listar, criar, alterar e remover produtos |
| Colaborador sem feature inventory | Usuário autenticado de empresa sem o módulo contratado | Bloqueado em todas as operações com 403 |
| Colaborador sem token JWT válido | Qualquer usuário sem sessão ou com token expirado | Bloqueado com 401 em todos os endpoints protegidos |

Managers do backoffice não acessam este módulo diretamente. A habilitação da feature `"inventory"` por empresa é feita no módulo common pelo backoffice de Manager.

## Escopo

### Dentro do escopo

- Estrutura do projeto Go: `go.mod`, `Dockerfile`, `.env.example`, estrutura de diretórios conforme padrão CDStudio
- Conexão com PostgreSQL configurável via variáveis de ambiente
- Autenticação JWT de colaborador com verificação de `type = "collaborator"`, `tenant_id` válido e extração de roles
- Feature gate: rejeição de requests de tenants sem a feature `"inventory"` habilitada
- Controle de acesso por role: `inventory.read` (leitura) e `inventory.write` (escrita)
- Endpoint de saúde: `GET /api/inventories/health` sem autenticação
- Endpoint de acesso: `GET /api/inventories/access` com autenticação e feature gate, retornando status de acesso do colaborador
- Migration SQL inicial (`2001_inventory_product.sql`) registrada no módulo common para criar a tabela de produtos no schema do tenant
- Testes unitários do domínio `access` (draft e use case)
- Script de teste integrado automatizado da fundação

### Fora do escopo

- CRUD de produtos (feature-002)
- Busca textual de produtos (feature-004)
- Interface de administração ou backoffice própria
- Mensageria assíncrona (RabbitMQ)
- Configurações de deploy e infraestrutura de produção (capability-001)

## Regras de Negócio

| Código | Regra | Comportamento esperado |
|--------|-------|----------------------|
| RN-001 | Apenas colaboradores autenticados com JWT válido podem acessar endpoints protegidos do módulo | **Dado que** um usuário envia uma requisição sem token ou com token inválido, **Quando** o sistema processa a requisição, **Então** o acesso é negado com status 401 |
| RN-002 | O JWT deve ser do tipo `"collaborator"` — tokens de manager são rejeitados | **Dado que** um manager autentica com seu token e tenta acessar o módulo de inventário, **Quando** o sistema valida o tipo do token, **Então** o acesso é negado com status 403 |
| RN-003 | O `tenant_id` extraído do JWT deve ser um UUID válido | **Dado que** um token é emitido com `tenant_id` inválido ou ausente, **Quando** o sistema extrai o `tenant_id`, **Então** a requisição é rejeitada com status 401 |
| RN-004 | A empresa do colaborador deve ter a feature `"inventory"` habilitada para acessar qualquer endpoint do módulo | **Dado que** um colaborador autenticado pertence a uma empresa sem a feature `"inventory"` habilitada, **Quando** o sistema verifica o acesso, **Então** todas as operações são bloqueadas com status 403 |
| RN-005 | Um colaborador com role `inventory.read` pode ler; com `inventory.write` pode escrever; sem nenhum dos dois, é bloqueado nas operações correspondentes | **Dado que** um colaborador tem apenas `inventory.read`, **Quando** tenta criar ou alterar um produto, **Então** o acesso é negado com status 403 |
| RN-006 | O endpoint de saúde `GET /api/inventories/health` não requer autenticação | **Dado que** qualquer agente faz uma requisição ao endpoint de health, **Quando** o módulo está em execução, **Então** recebe status 200 com confirmação do módulo |
| RN-007 | O endpoint `GET /api/inventories/access` responde com o status real de acesso do colaborador autenticado: se pode ler, escrever, se o módulo está pronto e quais requisitos pendentes existem | **Dado que** um colaborador autenticado com feature `"inventory"` habilitada e role `inventory.read` e `inventory.write` acessa `/access`, **Quando** o sistema processa a requisição, **Então** retorna `enabled: true`, `can_read: true`, `can_write: true`, `ready: true`, `pending_requirements: []` |
| RN-008 | Para o MVP do módulo inventory, não há pré-requisitos além de a feature estar habilitada. `pending_requirements` é sempre vazio e `ready` é sempre `true` quando o acesso é permitido | **Dado que** a feature `"inventory"` está habilitada para a empresa, **Quando** o colaborador consulta `/access`, **Então** `ready: true` e `pending_requirements: []` sempre |

## Estados e Transições

Sem estados ou transições de negócio relevantes nesta feature. A fundação não gerencia ciclo de vida de entidades — apenas estabelece a estrutura de controle de acesso.

## Fluxos de Negócio

### Fluxo principal — verificação de acesso ao módulo

1. O colaborador autentica no módulo common e recebe um JWT com `type = "collaborator"`, `tenant_id`, `company_id`, `roles` e status
2. O colaborador envia uma requisição ao módulo de inventário com o JWT no header `Authorization: Bearer`
3. O módulo valida o JWT: assinatura, expiração, tipo e `tenant_id` como UUID válido
4. O módulo verifica se a empresa do colaborador tem a feature `"inventory"` habilitada
5. Se a feature não estiver habilitada, o acesso é bloqueado antes de qualquer processamento de negócio
6. Se a feature estiver habilitada, o módulo verifica se o colaborador tem as roles necessárias para a operação solicitada
7. O endpoint `/access` retorna ao frontend o estado completo: módulo habilitado, capacidade de leitura e escrita, se está pronto para uso

### Fluxo alternativo — health check

1. Qualquer agente (monitor, load balancer, CI/CD) faz um `GET /api/inventories/health`
2. O módulo responde imediatamente com o identificador do módulo e status `"ok"`, sem autenticação

## Critérios de Aceite

> **Dado que** o módulo está em execução e o banco de dados está acessível, **Quando** qualquer agente acessa `GET /api/inventories/health`, **Então** recebe 200 com `{"status":"ok","module":"inventory"}`. _(RN-006)_

> **Dado que** um colaborador envia uma requisição sem token JWT, **Quando** o sistema processa a requisição em qualquer endpoint protegido, **Então** recebe 401 com `{"message":"unauthorized"}`. _(RN-001)_

> **Dado que** um manager envia sua sessão JWT para o módulo de inventário, **Quando** o sistema valida o tipo do token, **Então** recebe 403 com `{"message":"forbidden"}`. _(RN-002)_

> **Dado que** um colaborador autenticado pertence a uma empresa sem a feature `"inventory"` habilitada, **Quando** acessa qualquer endpoint do módulo, **Então** recebe 403 com `{"message":"inventory module not enabled for this company"}`. _(RN-004)_

> **Dado que** um colaborador autenticado pertence a uma empresa com feature `"inventory"` habilitada e tem as roles `inventory.read` e `inventory.write`, **Quando** acessa `GET /api/inventories/access`, **Então** recebe 200 com `enabled: true`, `can_read: true`, `can_write: true`, `ready: true`, `pending_requirements: []`. _(RN-007, RN-008)_

> **Dado que** um colaborador autenticado tem apenas a role `inventory.read`, **Quando** acessa `GET /api/inventories/access`, **Então** recebe 200 com `can_read: true` e `can_write: false`. _(RN-005, RN-007)_

## Dependências de Negócio

- O módulo `erp-backend-module-common` deve estar em execução e acessível ao módulo de inventário, pois:
  - O JWT é emitido e assinado pelo common (chave compartilhada via `JWT_SECRET`)
  - A tabela `public.company_features` é consultada pelo feature gate do inventário
- A feature `"inventory"` deve estar cadastrada na tabela `public.features` do common
- O `JWT_SECRET` deve ser idêntico entre o common e o inventory — qualquer divergência invalida todos os tokens
- O banco de dados PostgreSQL deve estar acessível na inicialização do módulo

## Análise de Segurança e LGPD

**Autenticação/Autorização por role:** Todos os endpoints protegidos exigem JWT de colaborador válido. As operações de leitura exigem role `inventory.read`; as de escrita exigem `inventory.write`. O endpoint de health é público.

**Autorização por recurso/tenant:** O `tenant_id` e o `company_id` são extraídos exclusivamente do JWT assinado — nunca do body ou de query params. O feature gate consulta o banco usando o `company_id` do token, sem possibilidade de spoofing pelo cliente.

**Validação de entrada:** O único input validado nesta feature é o `tenant_id` do JWT (formato UUID). A validação ocorre no middleware de autenticação.

**Proteção contra mass assignment:** Não há campos de entrada nesta feature. O endpoint `/access` não aceita body.

**Minimização de dados expostos:** O endpoint `/access` expõe apenas metadados de permissão — `module`, `enabled`, `can_read`, `can_write`, `ready`, `pending_requirements`. Nenhum dado pessoal ou sensível é exposto.

**SQL Injection:** A única query SQL desta feature (verificação de feature gate) usa parâmetros posicionais (`$1`, `$2`). Sem concatenação de strings SQL.

**Isolamento de tenant:** O feature gate filtra por `company_id` do token. O `tenant_id` é validado como UUID antes de ser armazenado no contexto e usado para derivar o nome do schema em features subsequentes.

**Concorrência e idempotência:** O endpoint `/access` é idempotente (leitura pura). O feature gate é uma leitura sem efeito colateral.

**Auditoria:** Esta feature não modifica dados — não há necessidade de auditoria de operações. A estrutura de auditoria (`created_by`, `updated_by`) é estabelecida na migration da tabela de produtos para uso nas features seguintes.

**Logs e observabilidade:** Nenhuma informação sensível (token, senha, PII) deve aparecer em logs. Apenas categorias de erro são logadas.

**Segredos e credenciais:** `JWT_SECRET` é configurado exclusivamente via variável de ambiente. Nunca em código ou banco de dados.

**Rate limit e abuso:** Não aplicável para o MVP desta feature. O endpoint de health e de access são de baixo risco. Endpoints de escrita serão protegidos por roles.

**Dados pessoais (LGPD):** Esta feature não coleta nem processa dados pessoais. O `tenant_id` e `company_id` são identificadores técnicos de empresa, não de pessoa física.

## Riscos e Pontos de Atenção

- **`JWT_SECRET` desatualizado:** se o secret mudar no common sem ser atualizado no inventory, todos os tokens válidos serão rejeitados com 401. Essa situação é operacionalmente invisível para o usuário final.
- **Feature `"inventory"` não cadastrada:** se a feature não existir na tabela `public.features` do common, o feature gate sempre bloqueará o acesso mesmo para tenants que deveriam ter acesso. O cadastro da feature é responsabilidade do backoffice de Manager.
- **Schema do tenant não criado:** se a migration `2001_inventory_product.sql` não tiver sido executada para o tenant, operações de produto falharão com erro de banco. O módulo common é responsável por executar as migrations na criação da empresa.
- **Conflito de roles:** o sistema aceita roles genéricas `read` e `write` além de `inventory.read` e `inventory.write`. Essa flexibilidade pode criar comportamentos inesperados se roles genéricas forem atribuídas sem intenção.

## Evolução da Feature

Não aplicável. Esta é a PRD inicial da feature de fundação.
