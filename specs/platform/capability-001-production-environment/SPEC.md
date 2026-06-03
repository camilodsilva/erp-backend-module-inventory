# SPEC — Ambiente de Produção (Homologação)

## Resumo Executivo

Configura o deploy do `erp-backend-module-inventory` na VPS de homologação via CI/CD existente. O workflow `.github/workflows/deploy.yml` já está implementado no repositório — o que resta são exclusivamente configurações manuais no GitHub e na VPS: secrets, environment, entradas no `docker-compose.yml` e `nginx.conf` do servidor. Nenhum arquivo do repositório precisa ser criado ou modificado.

---

## Impacto em Segurança e LGPD

- **Autenticação/Autorização por role:** garantida pela implementação do módulo — nenhuma configuração adicional de infraestrutura necessária.
- **Segredos e credenciais:** `VPS_SSH_KEY`, `GHCR_TOKEN` armazenados como secrets do environment `homologacao` no GitHub — nunca expostos em código ou logs. `JWT_SECRET` e `POSTGRES_PASSWORD` no `.env` da VPS — protegidos por permissões de filesystem.
- **Isolamento de tenant:** garantido pela implementação — schemas Postgres por tenant.
- **Logs e observabilidade:** logs do contêiner via stdout — sem PII ou segredos nos logs (garantido pela implementação).
- **Dados pessoais (LGPD):** ambiente de homologação deve usar exclusivamente dados sintéticos ou anonimizados. Não usar dados de pessoas físicas reais.
- **Exposição de porta:** módulo não expõe porta pública — acesso exclusivamente via proxy Nginx na rede Docker interna.
- **Rate limit e abuso:** não implementado nesta capability. Nginx é o ponto de entrada — rate limiting global pode ser adicionado se necessário.

---

## Decisões de Domínio e Clean Architecture

Esta capability é de infraestrutura operacional — sem código de domínio, use cases, repositórios ou handlers criados ou modificados.

**Decisão técnica: sem mudanças no repositório.** O workflow CI/CD já existe em `.github/workflows/deploy.yml`. Toda a configuração é externa ao código: secrets do GitHub, arquivos na VPS e configuração do Nginx.

**Checklist de Qualidade Arquitetural:**
- N/A — capability de infraestrutura sem código de aplicação

---

## Débitos Técnicos da Feature

| Código | Origem | Débito técnico | Camada | Arquivos previstos | Verificação |
|--------|--------|----------------|--------|--------------------|-------------|
| DT-001 | RN-005 | Criar environment `homologacao` no GitHub com os 4 secrets | Config/GitHub | (externo ao repo) | Pipeline verde em push na main |
| DT-002 | RN-001, RN-003, RN-004 | Adicionar serviço `module-inventory` ao `docker-compose.yml` da VPS | Config/VPS | `/app/docker-compose.yml` (na VPS) | `docker compose ps module-inventory` → Up |
| DT-003 | RN-003 | Inserir bloco `location /api/inventories/` no Nginx antes do `location /api/` genérico | Config/VPS | `/app/nginx/conf.d/api.conf` (na VPS) | `docker exec nginx nginx -t` → OK |
| DT-004 | RN-001 | Verificar que `JWT_SECRET` no `.env` da VPS é idêntico ao do módulo common | Config/VPS | `/app/.env` (na VPS) | Token do common aceito pelo inventory |
| DT-005 | RN-002 | Smoke test pós-deploy no endpoint de saúde | Verificação | Externo | `curl https://<dominio>/api/inventories/health` → 200 |

---

## ⚙️ Raciocínio Arquitetural

**Problema:** O módulo inventory existe como código mas não possui infraestrutura de deploy configurada.

**Estado atual:** O workflow CI/CD já está implementado em `.github/workflows/deploy.yml`. Ele depende dos secrets `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `GHCR_TOKEN` e do environment `homologacao` no repositório GitHub. Nenhum desses artefatos existe ainda.

**Decisão:** Toda a mudança é de configuração externa (GitHub + VPS). Nenhum arquivo do repositório precisa ser criado ou modificado. A ordem de execução importa: secrets → environment → VPS → nginx → push.

---

## Ordem de Execução

### 1. GitHub — Criar Environment `homologacao`

No repositório `camilodsilva/erp-backend-module-inventory`:

```
Settings → Environments → New environment
Nome: homologacao
```

Não é necessário configurar reviewers ou regras de proteção para homologação.

---

### 2. GitHub — Configurar Secrets do Environment

Em `Settings → Environments → homologacao → Environment secrets`:

| Secret | Valor |
|--------|-------|
| `VPS_HOST` | IP ou hostname da VPS |
| `VPS_USER` | Usuário SSH com acesso ao Docker (ex: `deploy`) |
| `VPS_SSH_KEY` | Chave privada SSH (conteúdo completo do arquivo `id_rsa` ou `id_ed25519`) |
| `GHCR_TOKEN` | Personal Access Token com escopo `read:packages` |

> `GITHUB_TOKEN` é injetado automaticamente pelo runner — não criar como secret.

---

### 3. VPS — Atualizar `/app/.env`

Nenhuma variável adicional além das já presentes no `.env` compartilhado é necessária. O serviço usa `POSTGRES_HOST`, `POSTGRES_PORT`, `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` e `JWT_SECRET` — todas devem já existir.

Verificar:

```bash
grep -E 'POSTGRES_|JWT_SECRET' /app/.env
```

Se alguma estiver ausente, adicionar:

```bash
# /app/.env (adicionar apenas as ausentes)
POSTGRES_HOST=postgres
POSTGRES_PORT=5432
POSTGRES_USER=<valor>
POSTGRES_PASSWORD=<valor>
POSTGRES_DB=<valor>
JWT_SECRET=<mínimo 64 bytes>
```

---

### 4. VPS — Adicionar serviço ao `/app/docker-compose.yml`

Abrir `/app/docker-compose.yml` na VPS e adicionar o bloco abaixo dentro da chave `services:`:

```yaml
  module-inventory:
    image: ghcr.io/camilodsilva/erp-backend-module-inventory:latest
    env_file: .env
    environment:
      POSTGRES_HOST: postgres
      POSTGRES_PORT: 5432
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
      JWT_SECRET: ${JWT_SECRET}
    depends_on:
      - postgres
    restart: always
    networks:
      - shared_network
```

> O serviço não expõe porta pública. O Nginx faz o proxy internamente via rede Docker `shared_network`.

---

### 5. VPS — Configurar Nginx

Arquivo: `/app/nginx/conf.d/api.conf`

Inserir o bloco abaixo **antes** do `location /api/` existente:

```nginx
location /api/inventories/ {
    proxy_pass http://module-inventory:8082;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

> A ordem importa no Nginx: blocos `location` mais específicos devem vir antes dos mais genéricos. `/api/inventories/` deve preceder `/api/`.

---

### 6. VPS — Recarregar Nginx

```bash
docker exec nginx nginx -t && docker exec nginx nginx -s reload
```

O primeiro comando valida a configuração antes de aplicar. Se retornar erro, revisar o bloco inserido.

---

### 7. Trigger — Push na branch `main`

Com todas as configurações acima prontas, fazer push em `main`:

```bash
git push origin main
```

O workflow executa automaticamente:

1. `test` — `go test ./...`
2. `build-and-push` — build da imagem Docker + push para `ghcr.io/camilodsilva/erp-backend-module-inventory:latest`
3. `deploy` — SSH na VPS, pull da imagem, `docker compose up -d --no-deps --force-recreate module-inventory`

---

## Ajustes no banco de dados

Sem ajustes de banco de dados e sem alteração em `MODELING.md`. As migrations de tenant já existem e são executadas pelo módulo common no provisionamento de empresas.

---

## Arquivos Modificados

Nenhum arquivo do repositório é modificado por esta capability. Toda a configuração é externa ao código.

---

## Arquivos Criados

Nenhum. O workflow CI/CD já existe em:

```
erp-backend-module-inventory/.github/workflows/deploy.yml
```

---

## Arquivos Deletados

Nenhum.

---

## Checklist de Verificação

### Pré-deploy

```bash
# Na VPS: validar nginx antes do reload
docker exec nginx nginx -t

# Na VPS: verificar variáveis presentes
grep -E 'POSTGRES_HOST|POSTGRES_PORT|POSTGRES_USER|POSTGRES_PASSWORD|POSTGRES_DB|JWT_SECRET' /app/.env
```

### Pós-deploy

```bash
# Na VPS: confirmar que o serviço está Up
docker compose -f /app/docker-compose.yml ps module-inventory

# Na VPS: verificar logs do container
docker compose -f /app/docker-compose.yml logs --tail=50 module-inventory

# Externamente: smoke test de saúde (ajustar host)
curl -i https://<dominio>/api/inventories/
# Esperado: 401 Unauthorized (endpoint existe, mas requer JWT)
# Inaceitável: 502 Bad Gateway ou connection refused
```

### Verificar CI/CD no GitHub

```
Actions → Build & Deploy → último run → todos os jobs verdes
```
