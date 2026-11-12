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

docker exec -i "$task_container" psql -h 127.0.0.1 -q -U postgres -v ON_ERROR_STOP=1 < "$task_root/integration/member_builds.sql"
