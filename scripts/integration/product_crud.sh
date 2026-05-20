#!/usr/bin/env bash

set -euo pipefail

MODULE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
COMMON_DIR="$(cd "$MODULE_DIR/../erp-backend-module-common" && pwd)"

BASE_URL="${BASE_URL:-http://localhost:8082}"
COMMON_URL="${COMMON_URL:-http://localhost:8080}"
TENANT_ID="01980f02-0000-7000-8000-000000000011"
TENANT_SCHEMA="t_01980f02000070008000000000000011"
SYSTEM_ACTOR_ID="00000000-0000-0000-0000-000000000000"

fail()                 { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
assert_status()        { [ "$2" = "$3" ] || fail "$1 expected HTTP $2, got $3"; printf 'PASS: %s\n' "$1"; }
assert_body_contains() { printf '%s' "$2" | grep -F -q "$3" || fail "$1 missing fragment: $3"; printf 'PASS: %s\n' "$1"; }
assert_body_absent()   { ! printf '%s' "$2" | grep -F -q "$3" || fail "$1 unexpected fragment: $3"; printf 'PASS: %s\n' "$1"; }
json_field()           { printf '%s' "$1" | sed -n "s/.*\"$2\":\"\([^\"]*\)\".*/\1/p" | head -1; }

reset_database() {
  docker exec postgres psql -U postgres -d erp_common -c "
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

  for schema in $(docker exec postgres psql -U postgres -d erp_common -Atc \
    "SELECT schema_name FROM information_schema.schemata WHERE schema_name LIKE 't\\_%' ESCAPE '\\';"); do
    docker exec postgres psql -U postgres -d erp_common -c "DROP SCHEMA IF EXISTS \"$schema\" CASCADE;" >/dev/null
  done

  docker exec -i postgres psql -U postgres -d erp_common < "$COMMON_DIR/data/init.sql" >/dev/null

  local collab_hash
  collab_hash="$(cd "$COMMON_DIR" && go run ./src/cmd/gen_hash/main.go "SenhaCollab123!" | tr -d '\n')"

  docker exec postgres psql -U postgres -d erp_common -c "
INSERT INTO public.companies (id, name, tenant_id, created_by, updated_by)
VALUES ('01980f02-0000-7000-8000-000000000001', 'Empresa Product CRUD',
        '$TENANT_ID', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.roles (id, role, created_by, updated_by)
VALUES
  ('01980f02-0000-7000-8000-000000000003', 'inventory.read', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f02-0000-7000-8000-000000000004', 'inventory.write', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.features (id, title, created_by, updated_by)
VALUES ('01980f02-0000-7000-8000-000000000005', 'inventory', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_features (id, company_id, feature_id, created_by, updated_by)
VALUES ('01980f02-0000-7000-8000-000000000006',
        '01980f02-0000-7000-8000-000000000001',
        '01980f02-0000-7000-8000-000000000005',
        '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_collaborators (id, company_id, email, password, is_active, status, created_by, updated_by)
VALUES ('01980f02-0000-7000-8000-000000000007',
        '01980f02-0000-7000-8000-000000000001',
        'product-crud@empresa.com', '$collab_hash', true, 'READY',
        '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_collaborator_roles (id, company_collaborator_id, role_id, created_by, updated_by)
VALUES
  ('01980f02-0000-7000-8000-000000000009',
   '01980f02-0000-7000-8000-000000000007',
   '01980f02-0000-7000-8000-000000000003', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f02-0000-7000-8000-000000000010',
   '01980f02-0000-7000-8000-000000000007',
   '01980f02-0000-7000-8000-000000000004', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

CREATE SCHEMA IF NOT EXISTS $TENANT_SCHEMA;
" >/dev/null

  sed "s/{{schema}}/$TENANT_SCHEMA/g" "$COMMON_DIR/data/migrations/tenant/2001_inventory_product.sql" \
    | docker exec -i postgres psql -U postgres -d erp_common >/dev/null
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
    -d '{"email":"product-crud@empresa.com","password":"SenhaCollab123!"}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"

  assert_status "INT-PRODUCT-01 login collaborator" "200" "$http_code"
  TOKEN="$(json_field "$body" token)"
  [ -n "$TOKEN" ] || fail "collaborator token missing"
}

main() {
  local token
  local response
  local body
  local http_code
  local product_id

  assert_services
  reset_database
  login_collaborator
  token="$TOKEN"

  response="$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/api/inventories/products" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"title":"Produto sem preco","sku":"NO-PRICE","unit":"UN"}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-02 unit_price obrigatorio" "400" "$http_code"
  assert_body_contains "INT-PRODUCT-02 mensagem unit_price" "$body" '"message":"unit_price is required"'

  response="$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/api/inventories/products" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"title":"Camiseta Branca P","description":"Camiseta 100% algodao","sku":"cam-bra-p","ean":"7891234567890","unit":"un","unit_price":49.90,"stock_quantity":100}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-03 criar produto" "201" "$http_code"
  assert_body_contains "INT-PRODUCT-03 sku normalizado" "$body" '"sku":"CAM-BRA-P"'
  assert_body_contains "INT-PRODUCT-03 unit normalizada" "$body" '"unit":"UN"'
  assert_body_absent "INT-PRODUCT-03 sem created_by" "$body" '"created_by"'
  assert_body_absent "INT-PRODUCT-03 sem updated_by" "$body" '"updated_by"'
  product_id="$(json_field "$body" id)"
  [ -n "$product_id" ] || fail "created product id missing"

  response="$(curl -s -w '\n%{http_code}' -X POST "$BASE_URL/api/inventories/products" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"title":"SKU duplicado","sku":"cam-bra-p","unit":"UN","unit_price":10}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-04 bloquear SKU duplicado" "409" "$http_code"
  assert_body_contains "INT-PRODUCT-04 mensagem duplicado" "$body" '"message":"product with this SKU already exists"'

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products?page=1&size=10" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-05 listar produtos" "200" "$http_code"
  assert_body_contains "INT-PRODUCT-05 total um" "$body" '"total":1'

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products/not-a-uuid" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-06 id invalido retorna 400" "400" "$http_code"
  assert_body_contains "INT-PRODUCT-06 mensagem id invalido" "$body" '"message":"product id is not a valid UUID"'

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products/$product_id" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-07 buscar por id" "200" "$http_code"
  assert_body_contains "INT-PRODUCT-07 id retornado" "$body" "\"id\":\"$product_id\""

  response="$(curl -s -w '\n%{http_code}' -X PUT "$BASE_URL/api/inventories/products/$product_id" \
    -H "Authorization: Bearer $token" \
    -H 'Content-Type: application/json' \
    -d '{"title":"Camiseta Branca P Atualizada","sku":"cam-bra-p","unit":"un","unit_price":59.90,"stock_quantity":80}')"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-08 atualizar produto" "200" "$http_code"
  assert_body_contains "INT-PRODUCT-08 titulo atualizado" "$body" '"title":"Camiseta Branca P Atualizada"'

  response="$(curl -s -w '\n%{http_code}' -X DELETE "$BASE_URL/api/inventories/products/$product_id" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  assert_status "INT-PRODUCT-09 deletar produto" "204" "$http_code"

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/products/$product_id" \
    -H "Authorization: Bearer $token")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-PRODUCT-10 buscar deletado" "404" "$http_code"
  assert_body_contains "INT-PRODUCT-10 mensagem not found" "$body" '"message":"product not found"'

  printf '\nTodos os testes de integração do CRUD de produtos passaram.\n'
}

main "$@"
