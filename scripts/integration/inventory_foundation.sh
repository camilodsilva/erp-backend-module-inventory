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
TENANT_WITH_FEATURE_ID="01980f01-0000-7000-8000-000000000011"
TENANT_WITHOUT_FEATURE_ID="01980f01-0000-7000-8000-000000000022"
SYSTEM_ACTOR_ID="00000000-0000-0000-0000-000000000000"

psql_exec() { docker exec "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"; }
psql_file() { docker exec -i "$POSTGRES_CONTAINER" psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"; }

fail()                 { printf 'FAIL: %s\n' "$1" >&2; exit 1; }
assert_status()        { [ "$2" = "$3" ] || fail "$1 expected HTTP $2, got $3"; printf 'PASS: %s\n' "$1"; }
assert_body_contains() { printf '%s' "$2" | grep -F -q "$3" || fail "$1 missing fragment: $3"; printf 'PASS: %s\n' "$1"; }
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
VALUES
  ('01980f01-0000-7000-8000-000000000001', 'Empresa Inventory Habilitada',
   '$TENANT_WITH_FEATURE_ID', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f01-0000-7000-8000-000000000002', 'Empresa Inventory Bloqueada',
   '$TENANT_WITHOUT_FEATURE_ID', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.roles (id, role, created_by, updated_by)
VALUES
  ('01980f01-0000-7000-8000-000000000003', 'inventory.read', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f01-0000-7000-8000-000000000004', 'inventory.write', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.features (id, title, created_by, updated_by)
VALUES ('01980f01-0000-7000-8000-000000000005', 'inventory', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_features (id, company_id, feature_id, created_by, updated_by)
VALUES ('01980f01-0000-7000-8000-000000000006',
        '01980f01-0000-7000-8000-000000000001',
        '01980f01-0000-7000-8000-000000000005',
        '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_collaborators (id, company_id, email, password, is_active, status, created_by, updated_by)
VALUES
  ('01980f01-0000-7000-8000-000000000007',
   '01980f01-0000-7000-8000-000000000001',
   'inventory@empresa.com', '$collab_hash', true, 'READY', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f01-0000-7000-8000-000000000008',
   '01980f01-0000-7000-8000-000000000002',
   'sem-inventory@empresa.com', '$collab_hash', true, 'READY', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');

INSERT INTO public.company_collaborator_roles (id, company_collaborator_id, role_id, created_by, updated_by)
VALUES
  ('01980f01-0000-7000-8000-000000000009',
   '01980f01-0000-7000-8000-000000000007',
   '01980f01-0000-7000-8000-000000000003', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f01-0000-7000-8000-000000000010',
   '01980f01-0000-7000-8000-000000000007',
   '01980f01-0000-7000-8000-000000000004', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID'),
  ('01980f01-0000-7000-8000-000000000012',
   '01980f01-0000-7000-8000-000000000008',
   '01980f01-0000-7000-8000-000000000003', '$SYSTEM_ACTOR_ID', '$SYSTEM_ACTOR_ID');
" >/dev/null
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
  local name="$1"
  local email="$2"
  local response
  local body
  local http_code

  response="$(curl -s -w '\n%{http_code}' -X POST "$COMMON_URL/api/auth/login" \
    -H 'Content-Type: application/json' \
    -d "{\"email\":\"$email\",\"password\":\"SenhaCollab123!\"}")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"

  assert_status "$name login" "200" "$http_code"
  TOKEN="$(json_field "$body" token)"
  [ -n "$TOKEN" ] || fail "$name token missing"
}

main() {
  local response
  local body
  local http_code
  local token_with_feature
  local token_without_feature

  assert_services
  reset_database

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/health")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-01 health sem autenticação" "200" "$http_code"
  assert_body_contains "INT-01 health module inventory" "$body" '"module":"inventory"'
  assert_body_contains "INT-01 health status ok" "$body" '"status":"ok"'

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/access")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-02 access sem token" "401" "$http_code"
  assert_body_contains "INT-02 mensagem unauthorized" "$body" '"message":"unauthorized"'

  login_collaborator "INT-03 collaborator sem feature" "sem-inventory@empresa.com"
  token_without_feature="$TOKEN"

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/access" \
    -H "Authorization: Bearer $token_without_feature")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-04 access com feature desabilitada" "403" "$http_code"
  assert_body_contains "INT-04 mensagem feature desabilitada" "$body" '"message":"inventory module not enabled for this company"'

  login_collaborator "INT-05 collaborator com feature" "inventory@empresa.com"
  token_with_feature="$TOKEN"

  response="$(curl -s -w '\n%{http_code}' "$BASE_URL/api/inventories/access" \
    -H "Authorization: Bearer $token_with_feature")"
  http_code="$(printf '%s' "$response" | tail -n1)"
  body="$(printf '%s' "$response" | sed '$d')"
  assert_status "INT-06 access com feature habilitada" "200" "$http_code"
  assert_body_contains "INT-06 module inventory" "$body" '"module":"inventory"'
  assert_body_contains "INT-06 enabled true" "$body" '"enabled":true'
  assert_body_contains "INT-06 can_read true" "$body" '"can_read":true'
  assert_body_contains "INT-06 can_write true" "$body" '"can_write":true'
  assert_body_contains "INT-06 ready true" "$body" '"ready":true'

  printf '\nTodos os testes de integração da foundation do inventory passaram.\n'
}

main "$@"
