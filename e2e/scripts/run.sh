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

  trap 'docker compose down' EXIT

  # Clean bind-mounted data via a helper container so the host isn't blocked
  # by root-owned files created inside containers.
  mkdir -p ./_tmp/data
  docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

  docker compose up -d --build

  # Wait for HTTP readiness.
  for _ in $(seq 1 60); do
    if curl -fsS "${http_base}/" >/dev/null; then
      break
    fi
    sleep 1
  done
  curl -fsS "${http_base}/" >/dev/null

  go test -tags=e2e -v ./e2e -count=1 -run TestE2E_GenerateDnsmasqConf -timeout 2m

  # dnsmasq does not auto-reload config; restart to pick up regenerated *.conf.
  docker compose restart dnsmasq

  go test -tags=e2e -v ./e2e -count=1 -run TestE2E_DNSResolvesAlias -timeout 2m
}

main "$@"
