# PRD — Endpoint de busca para integração fiscal

**ID:** `inventory.business.feature.004`
**Status:** `prd`

## Contexto

Quando o frontend de emissão de NF-e for integrado ao inventário (frontend feature `frontend.tax.business.feature.011`), ele precisará buscar produtos por SKU ou nome para pré-preencher os campos do item da NF-e.

Este endpoint complementa o CRUD básico da feature-002 com capacidade de busca textual, permitindo ao frontend do tax module encontrar produtos sem conhecer o UUID antecipadamente.

## Endpoint previsto

```
GET /api/inventories/products/search?q=camiseta&page=1&size=10
```

Response: mesmo formato paginado do `GET /api/inventories/products`, mas filtrado por `title` ou `sku` contendo `q` (case-insensitive).

## Dados retornados por produto (relevantes para NF-e)

- `id` — para armazenar como `product_external_id` no item da NF-e
- `title` / `description` — para preencher a descrição do item
- `sku` — para preencher o código do produto
- `ean` — para preencher `cEAN` / `cEANTrib` no XML da NF-e
- `unit` — para preencher a unidade do item
- `unit_price` — para sugerir o preço unitário
- `fiscal_profile_external_id` — para auto-sugerir o perfil fiscal no formulário de NF-e

## Dependências

- feature-002 implementada e verificada
- Frontend feature `frontend.tax.business.feature.011` em progresso

## Decisão de implementação pendente

Avaliar se o endpoint de busca deve ser implementado como:
a) Query param `?q=` no `GET /api/inventories/products` existente
b) Endpoint separado `GET /api/inventories/products/search`

Preferência: (a) para manter o contrato simples, adicionando `?q=` como filtro opcional ao endpoint de listagem já existente.
