#!/usr/bin/env bash
set -euo pipefail
# Verifies manual E2E fixtures in a disposable PostgreSQL container only.
# This is NOT native Supabase Auth/FE end-to-end testing.
task_root=$(cd -- "$(dirname -- "$0")/.." && pwd)
task_container="specuse-e2e-seed-test-$$"
task_tmp=$(mktemp -d -t specuse-e2e-fixture-check.XXXXXX)
trap 'docker rm -f "$task_container" >/dev/null 2>&1 || true; rm -rf -- "$task_tmp"' EXIT
docker run --name "$task_container" --network none --tmpfs /var/lib/postgresql/data \
  -e POSTGRES_HOST_AUTH_METHOD=trust -d postgres:17-alpine >/dev/null
task_ready=false
for task_attempt in $(seq 1 30); do
  if docker exec "$task_container" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then task_ready=true; break; fi
  sleep 1
done
if [ "$task_ready" != true ]; then echo 'PostgreSQL not ready' >&2; exit 1; fi
docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 < "$task_root/integration/bootstrap.sql"
for task_migration in "$task_root"/migrations/*.up.sql; do
  docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 < "$task_migration"
done
if task_error=$(docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 < "$task_root/seeds/private_draft_e2e.sql" 2>&1); then
  echo 'Seed unexpectedly accepted without opt-in' >&2; exit 1
fi
if [[ "$task_error" != *"explicit local-test opt-in"* ]]; then echo 'Unexpected seed guard error' >&2; exit 1; fi
# psql -c and -f share one session, which keeps the opt-in flag.
docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 \
  -c "SELECT set_config('app.allow_private_draft_e2e_seed','on',false)" \
  -f /dev/stdin < "$task_root/seeds/private_draft_e2e.sql" >/dev/null
docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 <<'SQL'
DO $$ BEGIN
 IF (SELECT count(*) FROM public.products)<>13 OR (SELECT count(*) FROM public.products WHERE is_active)<>11 THEN RAISE EXCEPTION 'Unexpected catalog count';END IF;
 IF (SELECT count(*) FROM public.ram_models)<>4 THEN RAISE EXCEPTION 'Unexpected RAM model count';END IF;
 IF (SELECT count(*) FROM public.products WHERE name LIKE '[SYNTHETIC E2E ONLY]%')<>13 THEN RAISE EXCEPTION 'Synthetic labels missing';END IF;
 IF EXISTS(SELECT 1 FROM public.products WHERE hardware_specs IS NOT NULL) THEN RAISE EXCEPTION 'Invented hardware evidence';END IF;
 IF EXISTS(SELECT 1 FROM auth.users) OR EXISTS(SELECT 1 FROM public.profiles) OR EXISTS(SELECT 1 FROM public.builds) THEN RAISE EXCEPTION 'Fixture created accounts or builds';END IF;
 IF (SELECT count(*) FROM public.product_rams r JOIN public.products p ON p.id=r.product_id WHERE r.ram_model_id='20202020-0000-4000-8000-000000000002' AND p.is_active)<>0 THEN RAISE EXCEPTION 'Inactive RAM group wrong';END IF;
 IF EXISTS(SELECT 1 FROM public.product_rams WHERE ram_model_id='20202020-0000-4000-8000-000000000003') THEN RAISE EXCEPTION 'Empty RAM group wrong';END IF;
 IF (SELECT count(*) FROM jsonb_array_elements(public.backend_catalog_snapshot()->'products'))<>13 THEN RAISE EXCEPTION 'Snapshot lost fixture products';END IF;
END $$;
SQL
if task_error=$(docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 \
  -c "SELECT set_config('app.allow_private_draft_e2e_seed','on',false)" \
  -f /dev/stdin < "$task_root/seeds/private_draft_e2e.sql" 2>&1); then
  echo 'Seed unexpectedly overwrote existing catalog' >&2; exit 1
fi
if [[ "$task_error" != *"existing data will not be overwritten"* ]]; then echo 'Unexpected nonempty-catalog guard error' >&2; exit 1; fi
echo 'PASS: E2E seed opt-in, empty-catalog guard, 13 synthetic products, 4 RAM models, no users/builds/evidence'

docker exec -i "$task_container" psql -h 127.0.0.1 -qAt -U postgres -v ON_ERROR_STOP=1 \
  -c "SELECT public.backend_catalog_snapshot()->'products'" > "$task_tmp/catalog.json"
(cd "$task_root" && go build -o "$task_tmp/api" ./cmd/api)
python3 "$task_root/integration/private_draft_fixture_prices.py" "$task_root" "$task_tmp/api" "$task_tmp/catalog.json"
