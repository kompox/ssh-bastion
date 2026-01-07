#!/usr/bin/env bash
set -euo pipefail

# E2E scenario: sshd port forwarding is denied/allowed based on PermitOpen.
#
# What this tests
# - Register an SSH public key via the web app.
# - Set forwarding mode to "none" and verify that local port forwarding is denied.
# - Set forwarding mode to "custom" with a single allowed target (127.0.0.1:8080)
#   and verify:
#   - forwarding to 127.0.0.1:8080 is allowed and usable
#   - forwarding to 127.0.0.1:22 is denied
#
# Notes
# - docker compose provides:
#   - web UI at http://localhost:8080 (test mode overrides enabled)
#   - sshd at localhost:2222 (published via compose)
# - sshd shares a network namespace with ssh-bastion, so 127.0.0.1:8080 is reachable
#   from the sshd container.

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

# E2E runs create a unique docker-compose project (and thus a unique network).
# If prior runs were interrupted, those networks can accumulate and exhaust
# Docker's default address pools. Best-effort cleanup of stale E2E networks.
docker network ls --format '{{.Name}}' \
  | grep -E '^ssh-bastion-e2e-' \
  | xargs -r docker network rm >/dev/null 2>&1 || true

tmp_root="$(mktemp -d "${repo_root}/_tmp/e2e-XXXXXXXX")"
host_data_dir="${tmp_root}/data"
mkdir -p "${host_data_dir}"

export SSHBASTION_HOST_DATA_DIR="${host_data_dir}"
project="ssh-bastion-e2e-$(basename "${tmp_root}" | tr '[:upper:]' '[:lower:]' | tr -cd 'a-z0-9_-')"
export COMPOSE_PROJECT_NAME="${project}"

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

echo "[1/9] Starting compose (build if needed)…" >&2
docker compose up -d --build --force-recreate

echo "[2/9] Waiting for web server…" >&2
wait_for "web server (GET / -> 200)" 30 sh -c 'test "$(curl -sS -o /dev/null -w "%{http_code}" http://localhost:8080/)" = "200"'

key_prefix="${tmp_root}/id_e2e_forwarding"
priv_key="${key_prefix}"
pub_key="${key_prefix}.pub"

echo "[3/9] Generating SSH keypair for bastion login…" >&2
rm -f "$priv_key" "$pub_key"
ssh-keygen -q -t ed25519 -N "" -f "$priv_key"
chmod 600 "$priv_key"

if [ ! -s "$pub_key" ]; then
  echo "ERROR: public key not created: ${pub_key}" >&2
  exit 1
fi

echo "[4/9] Registering public key via web app…" >&2
status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST --data-urlencode "publicKey@${pub_key}" http://localhost:8080/ssh/keys)"
if [ "$status" != "303" ]; then
  echo "ERROR: expected POST /ssh/keys to return 303, got ${status}" >&2
  exit 1
fi

expected_line="$(cat "$pub_key")"
wait_for "authorized_keys contains newly added key" 30 grep -Fxq "$expected_line" "${host_data_dir}/authorized_keys/jump"

# Sanity: SSH login must work.
echo "[5/9] Sanity-check: SSH login to bastion…" >&2
wait_for "ssh login succeeds" 30 ssh -4 -p 2222 \
  -i "$priv_key" \
  -o BatchMode=yes \
  -o StrictHostKeyChecking=no \
  -o UserKnownHostsFile=/dev/null \
  -o LogLevel=ERROR \
  jump@127.0.0.1 true

# Choose local ports for forwarding. Check they are not already in use.
local_allow_port=18080
local_deny_port=18081

check_tcp_port_free() {
  local p="$1"
  if command -v ss >/dev/null 2>&1; then
    if ss -H -lnt | awk -v p=":${p}$" '$4 ~ p {exit 0} END{exit 1}'; then
      echo "ERROR: TCP port ${p} is already in use (needed for port forwarding E2E)." >&2
      exit 1
    fi
  fi
}

check_tcp_port_free "$local_allow_port"
check_tcp_port_free "$local_deny_port"

start_forward_master() {
  local control_socket="$1"
  local local_port="$2"
  local remote_host="$3"
  local remote_port="$4"

  rm -f "$control_socket" >/dev/null 2>&1 || true

  ssh -4 -p 2222 \
    -i "$priv_key" \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o LogLevel=ERROR \
    -o ConnectTimeout=10 \
    -o ConnectionAttempts=1 \
    -M -S "$control_socket" -f -N \
    -L "127.0.0.1:${local_port}:${remote_host}:${remote_port}" \
    jump@127.0.0.1
}

stop_forward_master() {
  local control_socket="$1"

  ssh -4 -p 2222 \
    -i "$priv_key" \
    -o BatchMode=yes \
    -o StrictHostKeyChecking=no \
    -o UserKnownHostsFile=/dev/null \
    -o LogLevel=ERROR \
    -S "$control_socket" -O exit \
    jump@127.0.0.1 >/dev/null 2>&1 || true

  rm -f "$control_socket" >/dev/null 2>&1 || true
}

http_over_tunnel_is_ok() {
  local control_socket="$1"
  start_forward_master "$control_socket" "$local_allow_port" "127.0.0.1" "8080"

  set +e
  local code
  code="$(curl -sS -o /dev/null -w "%{http_code}" --connect-timeout 2 --max-time 3 "http://127.0.0.1:${local_allow_port}/")"
  local rc=$?
  set -e

  stop_forward_master "$control_socket"

  if [ $rc -eq 0 ] && [ "$code" = "200" ]; then
    return 0
  fi
  return 1
}

http_over_tunnel_is_denied() {
  local control_socket="$1"
  start_forward_master "$control_socket" "$local_allow_port" "127.0.0.1" "8080"

  set +e
  local code
  code="$(curl -sS -o /dev/null -w "%{http_code}" --connect-timeout 2 --max-time 3 "http://127.0.0.1:${local_allow_port}/")"
  local rc=$?
  set -e

  stop_forward_master "$control_socket"

  # Treat anything except a clean 200 as "denied" (PermitOpen blocks at connect time).
  if [ $rc -eq 0 ] && [ "$code" = "200" ]; then
    return 1
  fi
  return 0
}

ssh_banner_over_tunnel_is_denied() {
  local control_socket="$1"
  start_forward_master "$control_socket" "$local_deny_port" "127.0.0.1" "22"

  # If forwarding is allowed, we should receive the OpenSSH banner quickly.
  # If denied, the channel open fails and no banner arrives.
  local banner=""
  set +e
  exec 3<>"/dev/tcp/127.0.0.1/${local_deny_port}"
  local conn_rc=$?
  if [ $conn_rc -eq 0 ]; then
    banner="$(timeout 2 dd bs=1 count=4 <&3 2>/dev/null || true)"
    exec 3<&- 3>&-
  fi
  set -e

  stop_forward_master "$control_socket"

  if echo "$banner" | grep -q '^SSH-'; then
    return 1
  fi
  return 0
}

post_mode() {
  local mode="$1"
  local status
  status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST --data-urlencode "mode=${mode}" http://localhost:8080/admin/targets/mode)"
  if [ "$status" != "303" ]; then
    echo "ERROR: expected POST /admin/targets/mode to return 303, got ${status}" >&2
    return 1
  fi
}

post_add_target() {
  local host="$1"
  local port="$2"
  local status
  status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST --data-urlencode "host=${host}" --data-urlencode "port=${port}" http://localhost:8080/admin/targets/add)"
  if [ "$status" != "303" ]; then
    echo "ERROR: expected POST /admin/targets/add to return 303, got ${status}" >&2
    return 1
  fi
}

# Mode=none: forwarding should be denied.
echo "[6/9] Setting mode=none and verifying forwarding is denied…" >&2
post_mode "none"

control_socket_none="${tmp_root}/ctl_none"
wait_for "sshd denies port forwarding in mode=none" 30 http_over_tunnel_is_denied "$control_socket_none"

# Mode=custom + allow 127.0.0.1:8080: forwarding to 8080 should be allowed.
echo "[7/9] Setting mode=custom and adding allowed target 127.0.0.1:8080…" >&2
post_mode "custom"
post_add_target "127.0.0.1" "8080"

# Use the forwarding channel: HTTP GET / should return 200.
echo "[8/9] Verifying the allowed tunnel works (curl via localhost:${local_allow_port})…" >&2
control_socket_allow="${tmp_root}/ctl_allow"
wait_for "HTTP over tunnel (GET / -> 200)" 30 http_over_tunnel_is_ok "$control_socket_allow"

# Still in custom mode (only 8080 allowed): forwarding to 22 must be denied.
echo "[9/9] Verifying forwarding to disallowed target 127.0.0.1:22 is denied…" >&2
control_socket_deny="${tmp_root}/ctl_deny"
wait_for "sshd denies forwarding to 127.0.0.1:22 in custom mode" 30 ssh_banner_over_tunnel_is_denied "$control_socket_deny"

echo "OK: sshd port forwarding allow/deny behavior is enforced" >&2
