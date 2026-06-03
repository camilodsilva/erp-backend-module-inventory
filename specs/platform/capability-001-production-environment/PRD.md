# PRD — Ambiente de Produção (Homologação)

**ID:** `inventory.platform.capability.001`
**Status:** `prd`

## Contexto

O módulo `erp-backend-module-inventory` existe como código funcional e testado, mas não está sendo servido em nenhum ambiente acessível externamente. Sem um ambiente de homologação em execução, o frontend não pode integrar com o módulo, e os testes de ponta a ponta com dados reais de produção simulada não são viáveis.

Os demais módulos do ERP (common, fiscal) já estão configurados na VPS de homologação. O inventário precisa ser incluído nessa infraestrutura para completar o ciclo de desenvolvimento.

## Objetivo de Negócio

Disponibilizar o módulo de inventário no ambiente de homologação da VPS de forma que:
- O frontend possa se comunicar com os endpoints do módulo via HTTPS
- O CI/CD realize deploys automáticos a cada push na branch `main`
- O módulo esteja acessível sob o mesmo domínio dos demais módulos, via proxy Nginx

## Personas e Permissões de Negócio

| Persona | Responsabilidade |
|---------|----------------|
| Engenheiro responsável pelo deploy | Configura os secrets no GitHub, o `docker-compose.yml` e o Nginx na VPS |
| Pipeline de CI/CD (GitHub Actions) | Executa build, push da imagem Docker e deploy automatizado após aprovação |
| Frontend e QA | Consome o módulo em homologação para validar integrações |

## Escopo

### Dentro do escopo

- Criação do environment `homologacao` no repositório GitHub com os secrets necessários para CI/CD
- Adição do serviço `module-inventory` ao `docker-compose.yml` da VPS
- Configuração do bloco `location /api/inventories/` no Nginx para fazer proxy para o módulo
- Verificação de que as variáveis de ambiente necessárias (`POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`, `JWT_SECRET`) estão presentes no `.env` compartilhado da VPS
- Validação de saúde pós-deploy via endpoint `GET /api/inventories/health`

### Fora do escopo

- Criação do workflow de CI/CD — o arquivo `.github/workflows/deploy.yml` já existe no repositório
- Configurações de DNS ou certificados SSL — já gerenciados para o domínio existente
- Ambiente de produção definitivo — esta capability é de homologação
- Monitoramento e alertas de produção

## Regras de Negócio

| Código | Regra | Comportamento esperado |
|--------|-------|----------------------|
| RN-001 | O módulo de inventário deve usar o mesmo `JWT_SECRET` compartilhado pelos demais módulos — sem exceção | **Dado que** o `JWT_SECRET` do `.env` da VPS é o mesmo usado pelo módulo common, **Quando** o módulo inventory inicializa, **Então** aceita tokens emitidos pelo common sem rejeição |
| RN-002 | O endpoint `GET /api/inventories/health` deve responder com 200 após o deploy, confirmando que o módulo está em execução | **Dado que** o deploy foi concluído com sucesso, **Quando** um agente externo acessa o endpoint de saúde via HTTPS, **Então** recebe 200 com `{"status":"ok","module":"inventory"}` |
| RN-003 | O módulo não deve expor sua porta diretamente ao público — o acesso deve ser exclusivamente via proxy Nginx | **Dado que** o módulo está em execução na VPS, **Quando** um agente externo tenta acessar a porta `8082` diretamente, **Então** o acesso é bloqueado (firewall/rede Docker) |
| RN-004 | O módulo deve reiniciar automaticamente em caso de falha | **Dado que** o processo do módulo termina inesperadamente, **Quando** o Docker detecta a falha, **Então** o contêiner é reiniciado automaticamente (`restart: always`) |
| RN-005 | Qualquer push na branch `main` deve acionar o pipeline de CI/CD completo: testes, build da imagem Docker e deploy na VPS | **Dado que** um engenheiro faz push na branch `main`, **Quando** o GitHub Actions executa, **Então** os jobs `test`, `build-and-push` e `deploy` completam com sucesso |

## Estados e Transições

| Estado atual | Ação | Próximo estado | Quem pode executar | Condição |
|-------------|------|---------------|-------------------|---------|
| Não configurado | Configuração manual (secrets, VPS, Nginx) | Configurado, aguardando deploy | Engenheiro responsável | Todos os pré-requisitos cumpridos |
| Configurado, aguardando deploy | Push na `main` aciona CI/CD | Em execução (homologação) | Pipeline de CI/CD | Testes passam, build da imagem tem sucesso |
| Em execução | Novo push na `main` | Em execução (atualizado) | Pipeline de CI/CD | Testes passam, nova imagem disponível |

## Fluxos de Negócio

### Fluxo de configuração inicial

1. O engenheiro cria o environment `homologacao` no GitHub com os quatro secrets necessários
2. O engenheiro adiciona o bloco do serviço `module-inventory` ao `docker-compose.yml` da VPS
3. O engenheiro insere o bloco `location /api/inventories/` no Nginx antes do `location /api/` genérico
4. O engenheiro valida a configuração do Nginx e o recarrega
5. Um push na branch `main` aciona o primeiro deploy automático
6. O engenheiro valida o endpoint de saúde externamente para confirmar o deploy

### Fluxo de deploy contínuo (recorrente após configuração inicial)

1. Engenheiro faz push na branch `main` com novas features ou correções
2. O GitHub Actions executa os testes unitários
3. Se os testes passam, a imagem Docker é construída e publicada no GHCR
4. O pipeline faz SSH na VPS e executa `docker compose up -d --no-deps --force-recreate module-inventory`
5. O novo contêiner sobe com a imagem atualizada; o antigo é substituído

## Critérios de Aceite

> **Dado que** os secrets estão configurados e a VPS está preparada, **Quando** um push na branch `main` é feito, **Então** todos os jobs do GitHub Actions completam com status verde. _(RN-005)_

> **Dado que** o deploy foi concluído com sucesso, **Quando** um agente externo acessa `https://<dominio>/api/inventories/health`, **Então** recebe 200 com `{"status":"ok","module":"inventory"}`. _(RN-002)_

> **Dado que** o módulo está em execução, **Quando** um agente tenta acessar `https://<dominio>/api/inventories/products` sem token JWT, **Então** recebe 401 — confirmando que o roteamento Nginx → módulo → middleware de autenticação está funcionando. _(RN-001, RN-003)_

> **Dado que** o processo do módulo é encerrado inesperadamente, **Quando** o Docker detecta a falha, **Então** o contêiner é reiniciado automaticamente dentro de segundos. _(RN-004)_

## Dependências de Negócio

- O repositório `camilodsilva/erp-backend-module-inventory` deve existir no GitHub com o workflow `.github/workflows/deploy.yml` já implementado
- A VPS de homologação deve estar em execução com Docker, Docker Compose e Nginx configurados para os demais módulos
- O banco de dados PostgreSQL deve estar em execução na VPS, acessível pelo contêiner de inventário
- As migrations do módulo de inventário (`2001_inventory_product.sql`, `2002_inventory_product_fiscal_fields.sql`) devem ter sido executadas nos schemas dos tenants existentes na VPS — responsabilidade do módulo common ao provisionar empresas

## Análise de Segurança e LGPD

**Autenticação/Autorização:** não há endpoints públicos além de `/health`. Todos os demais exigem JWT válido — garantido pelos middlewares já implementados.

**Autorização por recurso/tenant:** o isolamento multi-tenant é garantido pela implementação do módulo — não há configuração adicional na infraestrutura.

**Validação de entrada:** não aplicável nesta capability — sem lógica de negócio nova.

**Proteção contra mass assignment:** não aplicável.

**Minimização de dados:** não aplicável.

**SQL Injection:** garantido pela implementação — queries parametrizadas.

**Isolamento de tenant:** garantido pela implementação — schemas por tenant.

**Concorrência e idempotência:** o deploy usa `--no-deps --force-recreate` para garantir que a instância mais recente está em execução sem downtime prolongado.

**Auditoria:** logs do contêiner disponíveis via `docker compose logs module-inventory`. Logs de CI/CD registrados no GitHub Actions.

**Logs e observabilidade:** logs do módulo são enviados ao stdout do contêiner — sem PII ou segredos nos logs (garantido pela implementação).

**Segredos e credenciais:** `VPS_SSH_KEY`, `GHCR_TOKEN`, `JWT_SECRET`, `POSTGRES_PASSWORD` são armazenados como secrets do GitHub — nunca expostos em código ou logs. O `.env` na VPS contém valores de ambiente — protegido pelas permissões de filesystem do servidor.

**Rate limit e abuso:** o Nginx é o ponto de entrada — rate limiting global pode ser configurado no Nginx se necessário. Não implementado nesta capability.

**Dados pessoais (LGPD):** o ambiente de homologação deve usar dados sintéticos ou anonimizados — não dados de pessoas físicas reais.

## Riscos e Pontos de Atenção

- **`JWT_SECRET` divergente:** se o `.env` da VPS tiver um `JWT_SECRET` diferente do usado pelo módulo common em produção, todos os tokens serão rejeitados. Verificar que o valor é idêntico antes do deploy.
- **Ordem do bloco Nginx:** o bloco `location /api/inventories/` deve ser inserido **antes** de qualquer `location /api/` genérico. Se inserido depois, o Nginx não alcançará o bloco específico do inventário.
- **Migrations não executadas:** se as migrations de inventário não tiverem sido executadas para os tenants existentes na VPS, as operações de produto falharão com erro de tabela não encontrada. Verificar que o módulo common executa as migrations ao provisionar empresas.
- **Imagem não publicada antes do deploy:** o job `deploy` depende de `build-and-push`. Em casos de falha parcial do pipeline, a VPS pode tentar fazer pull de uma imagem que não existe no GHCR.
- **Dados de homologação:** não usar dados pessoais reais de clientes na VPS de homologação. Usar apenas dados sintéticos.

## Evolução da Feature

Não aplicável. Esta é a PRD inicial desta capability.
