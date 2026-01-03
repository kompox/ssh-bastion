#!/usr/bin/env bash
set -euo pipefail

# E2E scenario: ssh-keygen -> register public key -> ssh login
#
# Assumptions:
# - docker compose provides:
#   - web UI at http://localhost:8080 (test mode overrides enabled)
#   - sshd at localhost:2222 (published via compose)
# - Data is persisted under ./_tmp/data (bind mount)

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

key_prefix="${1:-_tmp/id_e2e}"
priv_key="${key_prefix}"
pub_key="${key_prefix}.pub"

wait_for() {
  local desc="$1"
  local timeout_s="$2"
  shift 2

  local start
  start="$(date +%s)"
  while true; do
    if "$@" >/dev/null 2>&1; then
      return 0
    fi

    local now
    now="$(date +%s)"
    if (( now - start >= timeout_s )); then
      echo "ERROR: timed out waiting for: ${desc}" >&2
      return 1
    fi

    sleep 0.5
  done
}

http_status() {
  curl -sS -o /dev/null -w "%{http_code}" "$1"
}

trap 'docker compose down' EXIT

mkdir -p ./_tmp/data
docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

echo "[1/5] Starting compose (build if needed)…" >&2
docker compose up -d --build

echo "[2/5] Waiting for web server…" >&2
wait_for "web server (GET / -> 200)" 30 sh -c 'test "$(curl -sS -o /dev/null -w "%{http_code}" http://localhost:8080/)" = "200"'

echo "[3/5] Generating SSH keypair: ${priv_key} …" >&2
rm -f "$priv_key" "$pub_key"
ssh-keygen -q -t ed25519 -N "" -f "$priv_key"
chmod 600 "$priv_key"

if [ ! -s "$pub_key" ]; then
  echo "ERROR: public key not created: ${pub_key}" >&2
  exit 1
fi

echo "[4/5] Registering public key via web app…" >&2
status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST --data-urlencode "publicKey@${pub_key}" http://localhost:8080/keys)"
if [ "$status" != "303" ]; then
  echo "ERROR: expected POST /keys to return 303, got ${status}" >&2
  exit 1
fi

expected_line="$(cat "$pub_key")"
wait_for "authorized_keys contains newly added key" 30 grep -Fxq "$expected_line" _tmp/data/authorized_keys/jump

echo "[5/5] SSH login smoke test…" >&2
wait_for "ssh login succeeds" 30 ssh -4 -p 2222 \
  -i "$priv_key" \
  -o BatchMode=yes \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  jump@127.0.0.1 true

echo "OK: sshd E2E scenario succeeded" >&2
