# PRD — CRUD de Produtos

**ID:** `inventory.business.feature.002`
**Status:** `implemented`

## Contexto

Com a fundação do módulo de inventário estabelecida (feature-001), o próximo passo é permitir que colaboradores gerenciem o catálogo de produtos da empresa. Sem essa capacidade, o módulo de inventário não tem utilidade operacional — o cliente não consegue registrar, consultar, atualizar ou remover os produtos que comercializa.

O produto é a entidade central do módulo: é através dele que o estoque é controlado, a integração fiscal é realizada (via campos de classificação tributária) e os itens das notas fiscais são preenchidos. Esta feature entrega o CRUD completo de produtos com suporte a campos fiscais obrigatórios desde a criação.

## Objetivo de Negócio

Permitir que colaboradores autorizados cadastrem e mantenham o catálogo de produtos da empresa, com dados descritivos, operacionais e fiscais suficientes para suportar a emissão de NF-e. O catálogo deve ser isolado por tenant — produtos de uma empresa nunca são visíveis para outra.

## Personas e Permissões de Negócio

| Persona | Ação permitida | Condição |
|---------|---------------|---------|
| Colaborador com `inventory.write` | Criar, atualizar e remover produtos | Feature `"inventory"` habilitada na empresa |
| Colaborador com `inventory.read` | Listar e consultar produtos | Feature `"inventory"` habilitada na empresa |
| Colaborador sem role adequada | Nenhuma operação de inventário | — |
| Colaborador de outra empresa | Nenhum acesso ao catálogo | Isolamento por tenant |

## Escopo

### Dentro do escopo

- Criação de produto com dados descritivos (`title`, `description`, `sku`, `ean`, `unit`, `unit_price`, `stock_quantity`) e dados fiscais obrigatórios (`ncm`, `origin`) e opcional (`cest`)
- Referência opaca a perfil fiscal do módulo tax via `fiscal_profile_external_id` (UUID, sem FK cross-module)
- Listagem paginada de produtos ativos do tenant
- Busca por ID de produto
- Atualização de todos os campos do produto
- Remoção lógica (soft delete): produto removido não aparece em listagens nem em buscas
- SKU único por tenant (insensível a maiúsculas/minúsculas, normalizado para uppercase)
- Produto inicia sempre como ativo (`is_active: true`)
- Campos de auditoria (`created_by`, `updated_by`) extraídos do JWT — nunca do body da requisição
- Testes unitários do domínio e teste integrado do CRUD completo

### Fora do escopo

- Busca textual por nome ou SKU (feature-004)
- Controle de estoque com movimentações (entrada/saída) — apenas o campo `stock_quantity` é gerenciado como dado descritivo
- Ativação/desativação de produto como transição de estado gerenciada — `is_active` reflete apenas o estado de deleção lógica
- Validação de existência do `fiscal_profile_external_id` no módulo tax — referência é tratada como UUID opaco
- Importação em lote de produtos
- Histórico de alterações de produto

## Regras de Negócio

| Código | Regra | Comportamento esperado |
|--------|-------|----------------------|
| RN-001 | `title` é obrigatório e deve ter no máximo 120 caracteres | **Dado que** um colaborador tenta criar um produto sem título, **Quando** o sistema valida a requisição, **Então** retorna erro de validação indicando que o título é obrigatório |
| RN-002 | `sku` é obrigatório, deve ter no máximo 60 caracteres, e é normalizado para uppercase antes da persistência | **Dado que** um colaborador cadastra um produto com SKU `cam-bra-p`, **Quando** o sistema persiste o produto, **Então** o SKU armazenado e retornado é `CAM-BRA-P` |
| RN-003 | O SKU deve ser único por tenant entre produtos ativos (não deletados) | **Dado que** já existe um produto ativo com SKU `CAM-BRA-P` no tenant, **Quando** um colaborador tenta criar outro produto com o mesmo SKU, **Então** o sistema rejeita a operação com conflito |
| RN-004 | `unit` é obrigatório e deve ter no máximo 6 caracteres; é normalizado para uppercase | **Dado que** um colaborador cria um produto com unidade `un`, **Quando** o sistema persiste o produto, **Então** a unidade armazenada é `UN` |
| RN-005 | `unit_price` é obrigatório e deve ser maior ou igual a zero | **Dado que** um colaborador tenta criar um produto sem informar `unit_price`, **Quando** o sistema valida a requisição, **Então** retorna erro indicando que o preço unitário é obrigatório |
| RN-006 | `stock_quantity`, quando informado, deve ser maior ou igual a zero. Quando omitido, assume 0 | **Dado que** um colaborador cria um produto com quantidade em estoque negativa, **Quando** o sistema valida a requisição, **Então** retorna erro de validação |
| RN-007 | `ean`, quando informado, deve conter exatamente 8, 13 ou 14 dígitos numéricos | **Dado que** um colaborador informa um EAN com 10 dígitos, **Quando** o sistema valida a requisição, **Então** retorna erro indicando o formato inválido |
| RN-008 | `fiscal_profile_external_id`, quando informado, deve ser um UUID válido | **Dado que** um colaborador informa `fiscal_profile_external_id` com valor que não é UUID, **Quando** o sistema valida a requisição, **Então** retorna erro de validação |
| RN-009 | `ncm` é obrigatório e deve conter exatamente 8 dígitos numéricos (Nomenclatura Comum do Mercosul) | **Dado que** um colaborador tenta criar um produto sem NCM ou com NCM de 7 dígitos, **Quando** o sistema valida a requisição, **Então** retorna erro indicando que o NCM deve ter exatamente 8 dígitos |
| RN-010 | `origin` é obrigatório e deve ser um único dígito entre 0 e 8 (origem da mercadoria conforme tabela ICMS) | **Dado que** um colaborador informa `origin: 9`, **Quando** o sistema valida a requisição, **Então** retorna erro indicando que a origem deve ser um dígito entre 0 e 8 |
| RN-011 | `cest`, quando informado, deve conter exatamente 7 dígitos numéricos (Código Especificador de Substituição Tributária) | **Dado que** um colaborador informa um CEST com 6 dígitos, **Quando** o sistema valida a requisição, **Então** retorna erro indicando o formato inválido |
| RN-012 | Um produto removido não pode ser encontrado por ID nem aparecer em listagens; um novo produto pode ser criado com o mesmo SKU do produto removido | **Dado que** um produto com SKU `CAM-BRA-P` foi removido, **Quando** um colaborador busca esse produto por ID, **Então** recebe 404; e **Quando** cria um novo produto com o mesmo SKU, **Então** a criação é aceita |
| RN-013 | A remoção de produto é lógica (soft delete): o produto não é destruído do banco de dados, permitindo rastreabilidade histórica | **Dado que** um colaborador remove um produto, **Quando** o sistema processa a remoção, **Então** o produto deixa de aparecer em listagens e buscas mas seu registro permanece no banco para auditoria |
| RN-014 | Os campos `created_by` e `updated_by` são extraídos do JWT do colaborador autenticado — nunca do body da requisição | **Dado que** um colaborador cria ou atualiza um produto, **Quando** o sistema persiste a operação, **Então** `created_by` e `updated_by` refletem o UUID do colaborador autenticado, independente de qualquer valor enviado no body |
| RN-015 | Erros de validação de payload devem ser reportados todos de uma vez, não um por vez | **Dado que** um produto tem título, SKU e unidade inválidos simultaneamente, **Quando** o sistema valida a requisição, **Então** a resposta de erro lista todos os problemas de uma vez |

## Estados e Transições

| Estado atual | Ação | Próximo estado | Quem pode executar | Condição |
|-------------|------|---------------|-------------------|---------|
| — | Criação | Ativo | Colaborador com `inventory.write` | Dados válidos, SKU único no tenant |
| Ativo | Atualização | Ativo (atualizado) | Colaborador com `inventory.write` | Produto existe e não foi removido; dados válidos |
| Ativo | Remoção | Removido (soft delete) | Colaborador com `inventory.write` | Produto existe e não foi removido |
| Removido | Qualquer operação | — | — | Produto removido é invisível — retorna 404 |

> Um produto removido nunca volta ao estado ativo. Não há operação de restauração nesta feature.

## Fluxos de Negócio

### Fluxo de criação de produto

1. Colaborador autenticado com `inventory.write` envia os dados do produto, incluindo `title`, `sku`, `unit`, `unit_price`, `ncm` e `origin` (obrigatórios)
2. O sistema valida todos os campos de uma vez, reportando quaisquer erros encontrados
3. Se os dados são válidos, o sistema verifica que o SKU (normalizado para uppercase) não existe no catálogo ativo do tenant
4. Se o SKU já existe, o sistema rejeita a criação com conflito
5. Se tudo está válido, o produto é criado como ativo com os campos de auditoria preenchidos automaticamente
6. O sistema retorna os dados completos do produto criado, incluindo o ID gerado

### Fluxo de consulta de produto

1. Colaborador com `inventory.read` solicita a lista de produtos ou busca por ID específico
2. Apenas produtos não removidos (`deleted_at IS NULL`) são retornados
3. A listagem é paginada com parâmetros de `page` e `size` configuráveis

### Fluxo de atualização de produto

1. Colaborador com `inventory.write` envia os dados atualizados para um produto existente
2. O sistema valida todos os campos e verifica que o produto existe e não foi removido
3. Se o SKU for alterado, verifica que o novo SKU não conflita com outro produto ativo no tenant
4. O sistema atualiza todos os campos do produto, incluindo `updated_by` com o UUID do colaborador
5. O sistema retorna os dados atualizados do produto

### Fluxo de remoção de produto

1. Colaborador com `inventory.write` solicita a remoção de um produto por ID
2. O sistema verifica que o produto existe e não foi removido
3. O sistema marca o produto como removido (`deleted_at`, `deleted_by`), preservando o registro histórico
4. O produto deixa de aparecer em listagens e buscas a partir da remoção
5. O sistema confirma a remoção sem retornar dados do produto

## Critérios de Aceite

> **Dado que** um colaborador com `inventory.write` envia dados válidos com SKU único, NCM válido e origin válida, **Quando** cria um produto, **Então** o sistema retorna 201 com o produto criado, incluindo ID gerado, SKU normalizado para uppercase e `is_active: true`. _(RN-001 a RN-011, RN-014)_

> **Dado que** um colaborador envia dados com múltiplos campos inválidos (título vazio, SKU vazio, NCM inválido), **Quando** tenta criar o produto, **Então** o sistema retorna 400 com todos os erros de validação de uma vez. _(RN-015)_

> **Dado que** já existe um produto ativo com SKU `CAM-BRA-P` no tenant, **Quando** um colaborador tenta criar outro produto com SKU `cam-bra-p`, **Então** o sistema retorna 409 com mensagem de conflito de SKU. _(RN-003)_

> **Dado que** um colaborador com `inventory.read` lista os produtos, **Quando** solicita a listagem paginada, **Então** recebe 200 com a lista de produtos ativos do tenant na estrutura paginada. _(RN-013)_

> **Dado que** um produto existe e não foi removido, **Quando** um colaborador com `inventory.read` busca por ID, **Então** recebe 200 com os dados completos do produto. _(RN-013)_

> **Dado que** um produto não existe ou foi removido, **Quando** um colaborador busca por ID, **Então** recebe 404. _(RN-012)_

> **Dado que** um colaborador com `inventory.write` remove um produto, **Quando** tenta criar um novo produto com o mesmo SKU, **Então** a criação é aceita e retorna 201. _(RN-012)_

> **Dado que** um colaborador cria ou atualiza um produto, **Quando** o sistema persiste a operação, **Então** os campos `created_by` e `updated_by` refletem o UUID do colaborador do JWT, não qualquer valor enviado no body. _(RN-014)_

## Dependências de Negócio

- feature-001 (Fundação) implementada e verificada: autenticação JWT, feature gate e controle de roles devem estar operacionais
- Estrutura de banco de dados de produtos criada no schema isolado do tenant para cada empresa que usará o módulo (gerenciada pelo módulo common no provisionamento da empresa)
- Estrutura de banco de dados atualizada com campos fiscais nos schemas de tenants existentes
- Feature `"inventory"` habilitada no cadastro da empresa via backoffice de Manager
- Colaborador deve ter as roles `inventory.read` ou `inventory.write` atribuídas via backoffice

## Análise de Segurança e LGPD

**Autenticação/Autorização por role:** Operações de leitura (listagem, busca por ID) exigem a role `inventory.read`. Operações de escrita (criação, atualização, remoção) exigem a role `inventory.write`. Todos os endpoints passam pelo controle de autenticação e pelo feature gate antes de qualquer processamento.

**Autorização por recurso/tenant:** O identificador do tenant é derivado exclusivamente do token assinado. Um colaborador nunca acessa produtos de outro tenant — o isolamento é garantido em nível de banco de dados.

**Validação de entrada:** Todos os campos são validados pelo sistema antes de qualquer persistência. Erros são reportados de uma vez. Campos controlados pelo sistema como identificador do produto, autor da criação/atualização, status ativo e timestamps são definidos pelo servidor — nunca aceitos como entrada do cliente.

**Proteção contra mass assignment:** O identificador do produto (gerado pelo sistema como UUID), o estado ativo do produto, os campos de auditoria e os timestamps de criação, atualização e remoção são controlados exclusivamente pelo servidor — nunca podem ser enviados ou manipulados pelo cliente.

**Minimização de dados expostos:** Campos de auditoria (`created_by`, `updated_by`) não são expostos no response JSON público do produto, mantendo o contrato mínimo do MVP.

**Proteção contra injeção:** todas as consultas ao banco de dados devem usar mecanismos seguros de parametrização — sem concatenação de valores de usuário em consultas. O identificador do tenant é derivado exclusivamente do token assinado, nunca de valores fornecidos pelo cliente.

**Isolamento de tenant:** todo acesso a dados de produto é restrito ao espaço de dados do tenant autenticado. Operações de atualização e remoção verificam que o produto pertence ao tenant correto e está no estado esperado (não removido) antes de persistir qualquer alteração.

**Concorrência e idempotência:** a unicidade de SKU por tenant deve ser garantida pelo banco de dados, de forma que tentativas concorrentes de criar o mesmo SKU resultem em conflito — sem necessidade de lock em nível de aplicação.

**Auditoria:** Todos os produtos têm `created_by`, `updated_by`, `created_at`, `updated_at`. Produtos removidos têm `deleted_by` e `deleted_at` registrados. Registros não são destruídos, garantindo rastreabilidade histórica.

**Logs e observabilidade:** Erros de banco são logados por categoria sem expor dados do produto, queries ou PII.

**Segredos e credenciais:** Não aplicável — nenhum segredo adicional além do `JWT_SECRET` da fundação.

**Rate limit e abuso:** Não implementado no MVP. A criação e remoção de produtos são operações protegidas por autenticação e role, o que limita o vetor de abuso.

**Dados pessoais (LGPD):** Os produtos são dados de negócio do tenant, não dados pessoais de pessoas físicas. O `created_by` e `updated_by` armazenam UUIDs de colaboradores — são identificadores técnicos, não PII direta. O módulo não coleta dados pessoais de clientes finais.

## Riscos e Pontos de Atenção

- **Atualização de produtos existentes com campos fiscais:** ao adicionar campos fiscais obrigatórios (`ncm` e `origin`) ao cadastro de produto, produtos cadastrados antes dessa mudança terão esses campos em branco no banco de dados — inválidos pela regra de negócio atual. É necessário atualizar os produtos legados com valores corretos antes de usá-los em integrações fiscais.
- **SKU duplicado em inserções concorrentes:** em ambientes com alto volume de cadastros simultâneos, é possível que duas requisições tentem criar produtos com o mesmo SKU ao mesmo tempo. O banco de dados garante a unicidade e rejeita a segunda tentativa com conflito — a operação deve ser retentada pelo cliente.
- **Referência ao perfil fiscal sem validação cruzada:** o campo de referência ao perfil fiscal é um identificador opaco — o sistema de inventário não verifica se o perfil fiscal referenciado ainda existe no módulo fiscal. Se o perfil for removido do módulo fiscal, o produto continuará com uma referência inválida. O frontend deve tratar essa situação na tela de cadastro e emissão.
- **Quantidade em estoque sem rastreabilidade de movimentações:** o campo de quantidade em estoque reflete apenas o valor informado pelo colaborador, sem histórico de entradas e saídas. Divergências entre o estoque físico e o sistema são de responsabilidade operacional do cliente — nesta feature não há controle transacional de estoque.
- **Campos fiscais obrigatórios ao atualizar produtos:** qualquer atualização de produto (mesmo que seja apenas para corrigir o título) exige que `ncm` e `origin` sejam enviados com valores válidos. Sistemas que integrarem com o módulo de inventário precisam estar cientes dessa obrigatoriedade.

## Evolução da Feature

### Emenda — Campos Fiscais Fixos no Produto

**Comportamento anterior:** o produto tinha apenas dados descritivos e operacionais (`title`, `sku`, `unit`, `unit_price`, `stock_quantity`, `ean`, `description`, `fiscal_profile_external_id`). Não havia campos de classificação tributária diretamente no produto.

**Novo comportamento:** o produto passou a incluir `ncm` (obrigatório, 8 dígitos), `origin` (obrigatório, dígito 0–8) e `cest` (opcional, 7 dígitos). Esses campos permitem que a NF-e seja emitida sem depender de um perfil fiscal configurado.

**O que permanece igual:** todos os campos descritivos e operacionais anteriores; a lógica de soft delete; o isolamento por tenant; os campos de auditoria.

**O que deixa de ser válido:** criar ou atualizar um produto sem informar `ncm` e `origin` — esses campos passaram a ser obrigatórios.

**Motivo da mudança:** integração fiscal requer que cada produto tenha sua classificação tributária para emissão de NF-e. Centralizar `ncm`, `origin` e `cest` no produto simplifica o preenchimento de itens da NF-e e reduz a dependência de perfis fiscais.
