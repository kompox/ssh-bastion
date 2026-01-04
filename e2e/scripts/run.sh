#!/usr/bin/env bash
set -euo pipefail

scripts_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

bash "${scripts_dir}/steps/precheck.sh"

# Run all numbered end-to-end scenarios.
while IFS= read -r f; do
  case "$f" in
    *'-skip-'*.sh|*'-xfail-'*.sh|*'-known-fail-'*.sh|*'-quarantine-'*.sh)
      echo "==> SKIP $f"
      ;;
    *)
      echo "==> $f"
      bash "$f"
      ;;
  esac
done < <(ls -1 "${scripts_dir}"/e2e-[0-9][0-9]-*.sh | LC_ALL=C sort)
