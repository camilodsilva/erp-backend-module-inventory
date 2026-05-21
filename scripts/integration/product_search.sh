#!/usr/bin/env bash

set -euo pipefail

MODULE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMMON_DIR="$(cd "$MODULE_DIR/../erp-backend-module-common" && pwd)"

BASE_URL="${BASE_URL:-http://localhost:8082}"
COMMON_URL="${COMMON_URL:-http://localhost:8080}"
POSTGRES_CONTAINER=postgres-erp-it
POSTGRES_DB=erp_common
POSTGRES_USER=postgres
POSTGRES_PORT=55432
TENANT_ID="01980f04-0000-7000-8000-000000000011"
TENANT_SCHEMA="t_01980f04000070008000000000000011"
SYSTEM_ACTOR_ID="00000000-0000-0000-0000-000000000000"

psql_exec() { docker exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"; }
psql_file() { docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"; }

fail()                 { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
assert_status()        { [ "$2" = "$3" ] || fail "$1 expected HTTP $2, got $3"; printf 'PASS: %s\n' "$1"; }
assert_body_contains() { printf '%s' "$2" | grep -F -q "$3" || fail "$1 missing fragment: $3"; printf 'PASS: %s\n' "$1"; }
assert_body_absent()   { ! printf '%s' "$2" | grep -F -q "$3" || fail "$1 unexpected fragment: $3"; printf 'PASS: %s\n' "$1"; }
json_field()           { printf '%s' "$1" | sed -n "s/.*\"$2\":\"\([^\"]*\)\".*/\1/p" | head -1; }

reset_database() {
  psql_exec -c "
TRUNCATE TABLE
  public.manager_roles,
  public.company_collaborator_roles,
  public.company_collaborators,
  public.company_features,
  public.features,
  public.roles,
  public.companies,
  public.managers
RESTART IDENTITY CASCADE;
" >/dev/null

  for schema in $(psql_exec -Atc \
    "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 't\\_%' ESCAPE '\\';"); do
    psql_exec -c "DROP SCHEMA IF EXISTS \"$schema\" CASCADE;" >/dev/null
  done

  psql_file < "$COMMON_DIR/data/init.sql" >/dev/null

  local collab_hash
  collab_hash="$(cd "$COMMON_DIR" && go run ./src/cmd/gen_hash/main.go "SenhaCollab123!" | tr -d '\n')"

  psql_exec -c "
INSERT INTO public.companies (id, name, tenant_id, created_by, updated_by)
VALUES ('01980f04-0000-7000-8000-000000000001', 'Empresa Product Search',
        '$TENANT_ID', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.roles (id, role, created_by, updated_by)
VALUES
  ('01980f04-0000-7000-8000-000000000003', 'inventory.read', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f04-0000-7000-8000-000000000004', 'inventory.write', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.features (id, title, created_by, updated_by)
VALUES ('01980f04-0000-7000-8000-000000000005', 'inventory', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_features (id, company_id, feature_id, created_by, updated_by)
VALUES ('01980f04-0000-7000-8000-000000000006',
        '01980f04-0000-7000-8000-000000000001',
        '01980f04-0000-7000-8000-000000000005',
        '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_collaborators (id, company_id, email, password, is_active, status, created_by, updated_by)
VALUES ('01980f04-0000-7000-8000-000000000007',
        '01980f04-0000-7000-8000-000000000001',
        'product-search@empresa.com', '$collab_hash', true, 'READY',
        '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_collaborator_roles (id, company_collaborator_id, role_id, created_by, updated_by)
VALUES
  ('01980f04-0000-7000-8000-000000000009',
   '01980f04-0000-7000-8000-000000000007',
   '01980f04-0000-7000-8000-000000000003', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f04-0000-7000-8000-000000000010',
   '01980f04-0000-7000-8000-000000000007',
   '01980f04-0000-7000-8000-000000000004', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

CREATE SCHEMA IF NOT EXISTS $TENANT_SCHEMA;
" >/dev/null

  sed "s/{{schema}}/$TENANT_SCHEMA/g" "$COMMON_DIR/data/migrations/tenant/2001_inventory_product.sql" \
    | psql_file >/dev/null
}

assert_services() {
  local common_status
  local inventory_status

  common_status="$(curl -s -o /dev/null -w '%{http_code}' -X POST "$COMMON_URL/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{}' || true)"
  inventory_status="$(curl -s -o /dev/null -w '%{http_code}' "$BASE_URL/api/inventories/health" || true)"

  case "$common_status" in
    200|400|401) ;;
    *) fail "common module must be running on $COMMON_URL" ;;
  esac
  [ "$inventory_status" = "200" ] || fail "inventory module must be running on $BASE_URL"
}

login_collaborator() {
  local response
  local body
  local http_code

  response="$(curl -s -w '\n%{http_code}' -X POST "$COMMON_URL/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d '{"email":"product-search@empresa.com","password":"SenhaCollab123!"}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"

  assert_status "INT-SEARCH-01 login collaborator" "200" "$http_code"
  TOKEN="$(json_field "$body" token)"
  [ -n "$TOKEN" ] || fail "collaborator token missing"
}

main() {
  local token
  local response
  local body
  local http_code

  assert_services
  reset_database
  login_collaborator
  token="$TOKEN"

  # seed: cria dois produtos para os testes de busca
  response="$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/api/inventories/products" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"title":"Camiseta Branca P","sku":"CAM-BRA-P","unit":"UN","unit_price":49.90}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  assert_status "INT-SEARCH-02 seed camiseta" "201" "$http_code"

  response="$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/api/inventories/products" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"title":"Calca Jeans 38","sku":"CAL-JEA-38","unit":"UN","unit_price":129.90}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  assert_status "INT-SEARCH-03 seed calca" "201" "$http_code"

  # listar sem filtro: comportamento original preservado
  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-SEARCH-04 listar sem filtro" "200" "$http_code"
  assert_body_contains "INT-SEARCH-04 total dois" "$body" '"total":2'
  assert_body_contains "INT-SEARCH-04 camiseta presente" "$body" '"title":"Camiseta Branca P"'
  assert_body_contains "INT-SEARCH-04 calca presente" "$body" '"title":"Calca Jeans 38"'

  # busca por titulo (case-insensitive)
  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products?q=camiseta" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-SEARCH-05 busca por titulo" "200" "$http_code"
  assert_body_contains "INT-SEARCH-05 total um" "$body" '"total":1'
  assert_body_contains "INT-SEARCH-05 camiseta retornada" "$body" '"title":"Camiseta Branca P"'
  assert_body_absent  "INT-SEARCH-05 calca ausente" "$body" '"title":"Calca Jeans 38"'

  # busca por SKU (case-insensitive)
  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products?q=CAL" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-SEARCH-06 busca por sku" "200" "$http_code"
  assert_body_contains "INT-SEARCH-06 total um" "$body" '"total":1'
  assert_body_contains "INT-SEARCH-06 sku retornado" "$body" '"sku":"CAL-JEA-38"'
  assert_body_absent  "INT-SEARCH-06 camiseta ausente" "$body" '"title":"Camiseta Branca P"'

  # busca sem resultado
  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products?q=xyz_inexistente" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-SEARCH-07 busca sem resultado" "200" "$http_code"
  assert_body_contains "INT-SEARCH-07 total zero" "$body" '"total":0'

  # busca com paginacao: apenas 1 por pagina, total deve ser >= 1
  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products?q=a&page=1&size=1" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-SEARCH-08 busca paginada" "200" "$http_code"
  assert_body_contains "INT-SEARCH-08 size um" "$body" '"size":1'
  assert_body_absent  "INT-SEARCH-08 total nao zero" "$body" '"total":0'

  # q com espacos em branco deve ser tratado como vazio
  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products?q=%20%20" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-SEARCH-09 q espacos retorna todos" "200" "$http_code"
  assert_body_contains "INT-SEARCH-09 total dois" "$body" '"total":2'

  printf '\nTodos os testes de integração da busca de produtos passaram.\n'
}

main "$@"
