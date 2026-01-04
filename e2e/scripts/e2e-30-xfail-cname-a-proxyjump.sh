#!/usr/bin/env bash
set -euo pipefail

# XFAIL E2E scenario: reproduce the CNAME->A resolution issue via sshd + ProxyJump.
#
# What this tests
# - Create DNS alias: hoge.local -> github.com
# - (Previously) restart dnsmasq (cache cold)
# - Attempt: ssh -J (via local sshd container) to git@hoge.local
#
# Expected current behavior (known issue)
# - Fails with: "Name has no usable address" (sshd cannot use CNAME-only answer)
#
# Expected behavior after fix (e.g., glibc rebuild)
# - Does NOT fail with that DNS error; connection proceeds to GitHub and typically
#   fails later at authentication (e.g. "Permission denied (publickey)."), which is OK.

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

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
  docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*' >/dev/null 2>&1 || true
  if [ -n "${tmp_root:-}" ]; then
    rm -rf "${tmp_root}" >/dev/null 2>&1 || true
  fi
}
trap cleanup EXIT

# Clean bind-mounted data via a helper container so the host isn't blocked
# by root-owned files created inside containers.
docker compose run --rm -T --entrypoint sh ssh-bastion -c 'rm -rf /data/*'

echo "[1/7] Starting compose (build if needed)…" >&2
docker compose up -d --build --force-recreate

echo "[2/7] Waiting for web server…" >&2
wait_for "web server (GET / -> 200)" 30 sh -c 'test "$(curl -sS -o /dev/null -w "%{http_code}" http://localhost:8080/)" = "200"'

key_prefix="${tmp_root}/id_e2e_cname_a"
priv_key="${key_prefix}"
pub_key="${key_prefix}.pub"

echo "[3/7] Generating SSH keypair for bastion login…" >&2
rm -f "$priv_key" "$pub_key"
ssh-keygen -q -t ed25519 -N "" -f "$priv_key"
chmod 600 "$priv_key"

if [ ! -s "$pub_key" ]; then
  echo "ERROR: public key not created: ${pub_key}" >&2
  exit 1
fi

echo "[4/7] Registering public key via web app…" >&2
status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST --data-urlencode "publicKey@${pub_key}" http://localhost:8080/keys)"
if [ "$status" != "303" ]; then
  echo "ERROR: expected POST /keys to return 303, got ${status}" >&2
  exit 1
fi

expected_line="$(cat "$pub_key")"
wait_for "authorized_keys contains newly added key" 30 grep -Fxq "$expected_line" "${host_data_dir}/authorized_keys/jump"

echo "[5/7] Creating DNS alias: hoge.local -> github.com …" >&2
status="$(curl -sS -o /dev/null -w "%{http_code}" -X POST --data-urlencode "source=hoge.local" --data-urlencode "destination=github.com" http://localhost:8080/dns)"
if [ "$status" != "303" ]; then
  echo "ERROR: expected POST /dns to return 303, got ${status}" >&2
  exit 1
fi

aliases_json="${host_data_dir}/dns/aliases.json"
wait_for "aliases.json contains hoge.local alias" 30 grep -Fq "hoge.local" "${aliases_json}"

known_hosts_file="${tmp_root}/known_hosts"
ssh_config="${tmp_root}/ssh_config_e2e_cname_a"
cat >"$ssh_config" <<EOF
Host bastion
  HostName 127.0.0.1
  Port 2222
  User jump
  IdentityFile ${priv_key}
  IdentitiesOnly yes
  BatchMode yes
  StrictHostKeyChecking no
  UserKnownHostsFile ${known_hosts_file}
  LogLevel INFO
  ConnectTimeout 10
  ConnectionAttempts 1

Host target
  HostName hoge.local
  User git
  ProxyJump bastion
  BatchMode yes
  StrictHostKeyChecking no
  UserKnownHostsFile ${known_hosts_file}
  LogLevel INFO
  ConnectTimeout 10
  ConnectionAttempts 1
EOF

# Sanity: bastion login must work.
echo "[6/7] Sanity-check: SSH login to bastion…" >&2
wait_for "ssh login to bastion succeeds" 30 ssh -4 -F "$ssh_config" bastion true

# Repro: ProxyJump to the alias.
echo "[7/7] Repro: ssh -J bastion git@hoge.local (expected to fail today)…" >&2
set +e
out="$(ssh -4 -F "$ssh_config" target 2>&1)"
rc=$?
set -e

# This is the known failure mode we want to gate on.
if echo "$out" | grep -Fq "Name has no usable address"; then
  echo "XFAIL confirmed: DNS alias returned CNAME-only; sshd failed to connect: Name has no usable address" >&2
  echo "$out" >&2
  exit 1
fi

# If it didn't hit the DNS error, consider the issue fixed (even if auth fails).
# A typical post-fix outcome is an auth error against GitHub.
if [ $rc -ne 0 ]; then
  echo "OK: did not hit DNS resolution error; ssh exited non-zero (likely auth), which is acceptable." >&2
  echo "$out" >&2
  exit 0
fi

echo "OK: ssh succeeded unexpectedly (still acceptable)." >&2
exit 0
