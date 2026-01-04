#!/usr/bin/env bash

set -euo pipefail

repo_root() {
  local script_dir
  script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  # e2e/scripts -> repo root
  (cd "${script_dir}/../.." && pwd)
}

main() {
  local root http_base
  root="$(repo_root)"
  http_base="${SSHBASTION_E2E_HTTP_BASE:-http://localhost:8080}"

  cd "${root}"

  # E2E runs create a unique docker-compose project (and thus a unique network).
  # If prior runs were interrupted (Ctrl+C, crash), those networks can accumulate
  # and eventually exhaust Docker's default address pools.
  # Best-effort cleanup: remove stale E2E networks from previous runs.
  docker network ls --format '{{.Name}}' \
    | grep -E '^ssh-bastion-e2e-' \
    | xargs -r docker network rm >/dev/null 2>&1 || true

  local tmp_root host_data_dir project
  tmp_root="$(mktemp -d "${root}/_tmp/e2e-XXXXXXXX")"
  host_data_dir="${tmp_root}/data"
  mkdir -p "${host_data_dir}"

  project="ssh-bastion-e2e-$(basename "${tmp_root}" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_-')"

  export SSHBASTION_HOST_DATA_DIR="${host_data_dir}"
  export SSHBASTION_E2E_DATA_DIR="${host_data_dir}"
  export COMPOSE_PROJECT_NAME="${project}"

  cleanup() {
    docker compose down >/dev/null 2>&1 || true
    # Clean bind-mounted data via a helper container so the host isn't blocked
    # by root-owned files created inside containers.
    docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*' >/dev/null 2>&1 || true
    if [ -n "${tmp_root:-}" ]; then
      rm -rf "${tmp_root}" >/dev/null 2>&1 || true
    fi
  }
  trap cleanup EXIT

  # Ensure bind-mounted data is clean.
  docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

  docker compose up -d --build --force-recreate

  # Wait for HTTP readiness.
  for _ in $(seq 1 60); do
    if curl -fsS "${http_base}/" >/dev/null; then
      break
    fi
    sleep 1
  done
  curl -fsS "${http_base}/" >/dev/null

  go test -tags=e2e -v ./e2e -count=1 -run TestE2E_AliasIsPersisted -timeout 2m

  go test -tags=e2e -v ./e2e -count=1 -run TestE2E_DNSResolvesAlias -timeout 2m

  go test -tags=e2e -v ./e2e -count=1 -run TestE2E_DeleteDnsAlias -timeout 2m

  go test -tags=e2e -v ./e2e -count=1 -run TestE2E_DNSDoesNotResolveAlias -timeout 2m
}

main "$@"
