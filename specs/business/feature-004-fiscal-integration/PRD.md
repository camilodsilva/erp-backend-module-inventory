# PRD — Busca de Produtos para Integração Fiscal

**ID:** `inventory.business.feature.004`
**Status:** `implemented`

## Contexto

O frontend de emissão de NF-e (módulo fiscal) precisa buscar produtos do catálogo para pré-preencher os campos de item da nota fiscal. Sem essa capacidade de busca, o operador precisa conhecer previamente o ID ou SKU exato do produto, o que é inviável em catálogos com dezenas ou centenas de itens.

A busca textual por nome ou SKU, já implementada como extensão do endpoint de listagem (`?q=`), permite que o frontend ofereça um autocomplete ou campo de busca que localiza produtos sem exigir conhecimento prévio do UUID.

## Objetivo de Negócio

Permitir que o frontend do módulo fiscal localize produtos do catálogo por nome ou SKU para pré-preencher os campos do item da NF-e, reduzindo erros de digitação e acelerando o processo de emissão de notas fiscais.

## Personas e Permissões de Negócio

| Persona | Ação | Condição |
|---------|------|---------|
| Colaborador com `inventory.read` ou `inventory.write` | Buscar produtos por nome ou SKU | Feature `"inventory"` habilitada na empresa |
| Frontend do módulo fiscal | Consultar produtos via `?q=` para pré-preencher itens de NF-e | Usando o token JWT do colaborador autenticado |

A busca é uma operação de leitura — exige apenas `inventory.read`.

## Escopo

### Dentro do escopo

- Filtro textual opcional `?q=` no endpoint existente `GET /api/inventories/products`
- Busca case-insensitive em `title` e `sku` do produto
- Quando `?q=` está ausente ou é string vazia, o comportamento é idêntico ao endpoint de listagem original (retrocompatível)
- Resultados paginados com os mesmos parâmetros `page` e `size` do endpoint de listagem
- Retorno dos campos relevantes para emissão de NF-e: `id`, `title`, `description`, `sku`, `ean`, `unit`, `unit_price`, `fiscal_profile_external_id`, `ncm`, `origin`, `cest`
- Apenas produtos ativos (não deletados) aparecem nos resultados

### Fora do escopo

- Endpoint separado `GET /api/inventories/products/search` — optou-se por `?q=` no endpoint existente
- Busca por outros campos além de `title` e `sku`
- Relevância ou ranqueamento de resultados
- Busca com suporte a acentos/diacríticos (o `ILIKE` do PostgreSQL não normaliza acentos por padrão)
- Ordenação personalizada de resultados

## Regras de Negócio

| Código | Regra | Comportamento esperado |
|--------|-------|----------------------|
| RN-001 | O parâmetro `?q=` é opcional — quando ausente ou vazio (incluindo apenas espaços), a listagem retorna todos os produtos ativos do tenant paginados | **Dado que** um colaborador acessa `GET /api/inventories/products` sem `?q=`, **Quando** o sistema processa a requisição, **Então** retorna a mesma resposta paginada do comportamento anterior, sem filtragem |
| RN-002 | Quando `?q=` contém um valor não vazio (após trim de espaços), a listagem filtra produtos cujo `title` ou `sku` contenha o termo informado, sem distinção de maiúsculas/minúsculas | **Dado que** existem produtos com título `"Camiseta Branca P"` e `"Calça Jeans 38"` no tenant, **Quando** um colaborador busca `?q=camiseta`, **Então** recebe apenas o produto com título contendo `"camiseta"` (case-insensitive) |
| RN-003 | A busca por `sku` também é case-insensitive — colaboradores podem buscar por partes do SKU em qualquer capitalização | **Dado que** existe um produto com SKU `CAL-JEA-38`, **Quando** um colaborador busca `?q=cal`, **Então** o produto é retornado pelo match no SKU |
| RN-004 | A paginação se aplica aos resultados filtrados — `total` e `total_pages` refletem a contagem de produtos que atendem ao filtro `?q=`, não o total geral do catálogo | **Dado que** existem 10 produtos e apenas 2 contêm `"camiseta"` no título ou SKU, **Quando** um colaborador busca `?q=camiseta`, **Então** `total: 2` e `total_pages` calculado sobre 2 itens |
| RN-005 | Quando a busca não encontra nenhum produto correspondente, o sistema retorna uma lista vazia paginada com `total: 0` — sem erro | **Dado que** nenhum produto do tenant tem `"xyz_inexistente"` no título ou SKU, **Quando** um colaborador busca `?q=xyz_inexistente`, **Então** recebe `data: []`, `total: 0` com status 200 |
| RN-006 | Apenas produtos não deletados (`deleted_at IS NULL`) aparecem nos resultados da busca | **Dado que** um produto com SKU `CAM-BRA-P` foi removido, **Quando** um colaborador busca `?q=CAM`, **Então** o produto removido não aparece nos resultados |

## Estados e Transições

Sem estados ou transições de negócio relevantes. A busca é uma operação de leitura pura sem ciclo de vida de recurso.

## Fluxos de Negócio

### Fluxo principal — busca de produto para NF-e

1. O frontend do módulo fiscal exibe um campo de busca de produto ao operador
2. O operador digita parte do nome ou SKU do produto
3. O frontend envia `GET /api/inventories/products?q=<termo>&page=1&size=10` com o JWT do colaborador
4. O módulo de inventário filtra os produtos ativos do tenant cujo título ou SKU contém o termo informado
5. O frontend exibe a lista paginada de resultados
6. O operador seleciona o produto desejado
7. O frontend usa os dados retornados (especialmente `id`, `title`, `sku`, `ean`, `unit`, `unit_price`, `ncm`, `origin`, `cest`, `fiscal_profile_external_id`) para pré-preencher os campos do item da NF-e

### Fluxo alternativo — busca sem resultado

1. O operador digita um termo que não corresponde a nenhum produto
2. O sistema retorna `data: []` com `total: 0`
3. O frontend informa ao operador que nenhum produto foi encontrado

### Fluxo retrocompatível — listagem sem filtro

1. Qualquer chamada existente ao endpoint `GET /api/inventories/products` sem `?q=` continua funcionando identicamente ao comportamento anterior à feature-004

## Critérios de Aceite

> **Dado que** um colaborador acessa `GET /api/inventories/products` sem `?q=`, **Quando** o sistema processa a requisição, **Então** retorna todos os produtos ativos do tenant paginados, exatamente como antes desta feature. _(RN-001)_

> **Dado que** existem produtos com título `"Camiseta Branca P"` e `"Calça Jeans 38"` no tenant, **Quando** um colaborador busca `?q=camiseta`, **Então** recebe 200 com `total: 1` e apenas o produto com `"Camiseta"` no título. _(RN-002)_

> **Dado que** existe um produto com SKU `CAL-JEA-38`, **Quando** um colaborador busca `?q=CAL`, **Então** o produto é retornado com `total: 1`. _(RN-003)_

> **Dado que** existem 10 produtos e apenas 2 contêm `"camisa"`, **Quando** um colaborador busca `?q=camisa&page=1&size=5`, **Então** `total: 2`, `total_pages: 1`, `data` contém no máximo 2 itens. _(RN-004)_

> **Dado que** nenhum produto do tenant contém `"xyz_inexistente"`, **Quando** um colaborador busca `?q=xyz_inexistente`, **Então** recebe 200 com `data: []` e `total: 0`. _(RN-005)_

> **Dado que** um produto com SKU `CAM-BRA-P` foi removido, **Quando** um colaborador busca `?q=CAM`, **Então** o produto removido não aparece nos resultados. _(RN-006)_

> **Dado que** `?q=` contém apenas espaços (ex: `?q=%20%20`), **Quando** o sistema processa a requisição, **Então** trata como busca vazia e retorna todos os produtos ativos paginados. _(RN-001)_

## Dependências de Negócio

- feature-002 (CRUD de Produtos) implementada e verificada: o catálogo de produtos deve existir para que a busca tenha resultados
- feature-001 (Fundação) operacional: autenticação JWT, feature gate e controle de roles ativos
- Frontend da integração do módulo fiscal com inventário (`frontend.tax.business.feature.011`) para consumir esta capacidade de busca — a feature de backend pode existir antes do frontend estar pronto

## Análise de Segurança e LGPD

**Autenticação/Autorização por role:** busca é operação de leitura — exige `inventory.read`. Todos os middlewares da fundação se aplicam.

**Autorização por recurso/tenant:** a busca filtra apenas produtos do tenant derivado do JWT. Sem possibilidade de acessar produtos de outros tenants.

**Validação de entrada:** o parâmetro `?q=` é extraído, trimado de espaços e passado como parâmetro posicional para o banco — sem risco de injeção.

**Proteção contra mass assignment:** não aplicável — operação de leitura.

**Minimização de dados:** response idêntico ao `FindAll` — sem campos adicionais expostos.

**SQL Injection:** `q` é passado como parâmetro posicional (`$3`) com o padrão `%q%` montado em Go antes do bind. O `%` é um literal Go, não SQL — sem risco de injeção.

**Isolamento de tenant:** queries filtram pelo schema do tenant derivado do token.

**Concorrência e idempotência:** leitura pura — sem efeito colateral.

**Auditoria:** sem operações de escrita.

**Logs e observabilidade:** sem PII exposto.

**Segredos e credenciais:** não aplicável.

**Rate limit e abuso:** um `?q=a` retorna todos os produtos que contêm a letra `a` — pode gerar resultados grandes. Não implementado no MVP. O limite de `size` máximo de 100 mitiga parcialmente o risco de abuso.

**Dados pessoais (LGPD):** produtos são dados de negócio — sem PII.

## Riscos e Pontos de Atenção

- **Performance com catálogos grandes:** `ILIKE` com `%q%` (contém) não usa índice `B-tree` padrão. Para catálogos com milhares de produtos, pode causar full table scan. Para o MVP, aceitável. Para escala, considerar índice `GIN` com `pg_trgm`.
- **Busca sem normalização de acentos:** `ILIKE` no PostgreSQL é case-insensitive mas não normaliza acentos. `?q=calcao` não encontrará `"Calção"`. Esta limitação é conhecida e aceitável para o MVP.
- **Espaços em `?q=`:** strings com apenas espaços são tratadas como busca vazia (trim em Go antes do bind). Comportamento intencional e documentado.
- **Retrocompatibilidade:** chamadas sem `?q=` devem continuar funcionando exatamente como antes. Qualquer alteração na assinatura de `Repository.FindAll` afeta todos os use cases e mocks que implementam a interface.

## Evolução da Feature

Não aplicável. Esta é a PRD inicial da feature de busca para integração fiscal.
