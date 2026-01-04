#!/usr/bin/env bash
set -euo pipefail

# Preflight port checks for E2E scenarios.
# NOTE: Keep in sync with docker-compose / E2E assumptions.

if ! command -v ss >/dev/null 2>&1; then
  echo "WARN: ss not found; skipping E2E port preflight check" >&2
  exit 0
fi

in_use=0

check_tcp() {
  local p="$1"
  if ss -H -lnt | awk -v p=":${p}$" '$4 ~ p {exit 0} END{exit 1}'; then
    echo "ERROR: TCP port ${p} is already in use (E2E requires localhost:${p})." >&2
    in_use=1
  fi
}

check_udp() {
  local p="$1"
  if ss -H -lnu | awk -v p=":${p}$" '$4 ~ p {exit 0} END{exit 1}'; then
    echo "ERROR: UDP port ${p} is already in use (E2E requires 127.0.0.1:${p}/udp)." >&2
    in_use=1
  fi
}

check_tcp 8080
check_tcp 2222
check_udp 5353

if [ "$in_use" -ne 0 ]; then
  echo "Hint: stop any local dev servers (e.g. make run-test-mode) before running E2E." >&2
  exit 1
fi
