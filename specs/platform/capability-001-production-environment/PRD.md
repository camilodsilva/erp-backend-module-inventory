# PRD — Ambiente de produção

**ID:** `inventory.platform.capability.001`
**Status:** `prd`

## Contexto

Deploy do módulo de inventário no ambiente de homologação da VPS.

## Checklist

- [ ] Secrets configurados no GitHub: `VPS_HOST`, `VPS_USER`, `VPS_SSH_KEY`, `GHCR_TOKEN`
- [ ] Environment `homologacao` criado no repositório GitHub
- [ ] `/app/.env` na VPS atualizado com variáveis do inventário (nenhuma adicional além das padrão)
- [ ] `/app/docker-compose.yml` na VPS com serviço `module-inventory` (porta interna 8082)
- [ ] `/app/nginx/conf.d/api.conf` com `location /api/inventories/` → `module-inventory:8082`
- [ ] Nginx recarregado
- [ ] Push na branch `main` acionando CI/CD
- [ ] `docker compose ps` mostrando `module-inventory` Up

## Configuração Nginx

```nginx
location /api/inventories/ {
    proxy_pass http://module-inventory:8082;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;
}
```

Inserir antes do `location /api/` existente.

## docker-compose.yml (trecho)

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
```
