#!/usr/bin/env bash
set -euo pipefail
# Creates only a disposable local Docker container, never connects to a cloud DB.
task_root=$(cd -- "$(dirname -- "$0")/.." && pwd)
task_container="specuse-postgres-test-$$"
trap 'docker rm -f "$task_container" >/dev/null 2>&1 || true' EXIT
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
docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 < "$task_root/integration/permissions.sql"

docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 <<'SQL' &
BEGIN;
UPDATE public.products SET name='Concurrent update' WHERE id='44444444-4444-4444-8444-444444444444';
SELECT pg_advisory_xact_lock(1947);
SELECT pg_sleep(5);
COMMIT;
SQL
task_writer=$!
task_locked=false
for task_attempt in $(seq 1 50); do
  if [ "$(docker exec "$task_container" psql -h 127.0.0.1 -U postgres -Atc 'SELECT NOT pg_try_advisory_lock(1947)')" = "t" ]; then
    task_locked=true
    break
  fi
  sleep 0.1
done
if [ "$task_locked" != true ]; then echo 'Concurrent writer did not acquire lock' >&2; wait "$task_writer"; exit 1; fi
docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 <<'SQL'
DO $$
DECLARE original jsonb;
BEGIN
 original := public.backend_product_json('44444444-4444-4444-8444-444444444444');
 IF original->>'name' <> 'Updated board' THEN RAISE EXCEPTION 'Writer committed before concurrent read'; END IF;
 BEGIN
  PERFORM public.backend_product_save_checked('22222222-2222-4222-8222-222222222222',
   jsonb_set(original,'{price_thb}','8000'),false,original);
  RAISE EXCEPTION 'Concurrent stale update accepted';
 EXCEPTION WHEN SQLSTATE 'PT409' THEN NULL;
 END;
 IF (SELECT name FROM public.products WHERE id='44444444-4444-4444-8444-444444444444') <> 'Concurrent update'
    OR (SELECT price_thb FROM public.products WHERE id='44444444-4444-4444-8444-444444444444') <> 7000 THEN
  RAISE EXCEPTION 'Concurrent update was overwritten';
 END IF;
END $$;
SQL
wait "$task_writer"
echo 'PASS: concurrent product update rejected without overwriting'

docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 < "$task_root/integration/member_builds.sql"
