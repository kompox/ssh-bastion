---
id: design-e2e-testing
title: E2E / integration testing (docker-compose + published ports)
status: draft
updated: 2026-01-04T09:09:31Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: E2E / integration testing (docker-compose)

This document defines the E2E/integration test approach for local development and CI.

The E2E tests are intentionally “black-box”: they interact only with:

- published ports from `docker compose up -d`
  - HTTP: `http://localhost:8080`
  - DNS: `127.0.0.1:5353` (UDP)
  - SSH: `ssh -p 2222 jump@127.0.0.1`
- the bind-mounted data directory (per-run) under `./_tmp/`
  - by default, E2E scripts use `mktemp -d ./_tmp/e2e-XXXXXXXX` and bind-mount `./_tmp/e2e-XXXXXXXX/data` to `/data`.
  - for ad-hoc/manual runs, the compose file supports a default of `./_tmp/data`.

Container topology and networking details are specified in [design-containers].

## Goals

- Provide a repeatable E2E test flow that validates the system as deployed by docker-compose.
- Validate:
  - web server is reachable on `:8080`
  - DNS server answers on `:5353`
  - SSH login works via the sshd sidecar on `:2222`
  - the web app generates expected files under the bind-mounted data directory
- Keep tests independent from internal Go packages (black-box behavior).

## Non-goals
- White-box tests that import internal registries or storage.

## Test topology

- Runtime: `docker compose up -d --build --force-recreate`
  - This repo uses a sidecar topology where some services share a network namespace via `network_mode: service:ssh-bastion`.
  - If `ssh-bastion` is recreated without recreating the sidecars, the sidecars may remain attached to the *previous* container network namespace, causing DNS/SSH flakiness.
- Persistence: bind mount `${SSHBASTION_HOST_DATA_DIR}:/data`
  - E2E scripts should set `SSHBASTION_HOST_DATA_DIR` to a per-run directory under `./_tmp/`.
- Auth: docker-compose runs the web app in test mode via `SSHBASTION_AUTH_OVERRIDE_USER_ID` / `SSHBASTION_AUTH_OVERRIDE_EMAIL`

## Assumptions / constraints

- The E2E suite assumes the fixed published ports are available on the host:
  - TCP 8080 (HTTP)
  - UDP 5353 (DNS)
  - TCP 2222 (SSH)
- If you are running local dev (e.g. `make run-test-mode`), it may occupy UDP 5353 and cause E2E to fail. Stop it before running `make e2e`.

## Test implementation

## Conventions

- Run from the repository root.
  - All `make` targets are expected to be invoked from the repo root.
  - Each `e2e/scripts/e2e-NN-*.sh` script should also be runnable standalone.
- Temporary files live under `./_tmp/`.
  - Each E2E script should create its own per-run directory with `mktemp -d ./_tmp/e2e-XXXXXXXX`.
  - Bind-mounted test state: `./_tmp/e2e-XXXXXXXX/data/` (bind-mounted to `/data` in containers)
  - SSH E2E keys/config (if generated): under that per-run directory
  - Scripts must clean up their per-run directory on exit.

### Script naming (skip / xfail)

- Default runner scope: `make e2e` discovers `./e2e/scripts/e2e-NN-*.sh` and runs them in lexicographic order.
- Scripts with special markers in their filename are intentionally *skipped* by `make e2e`:
  - `e2e-NN-xfail-*.sh`: known-broken scenario that is expected to fail until a tracked fix lands.
  - `e2e-NN-skip-*.sh`: scenario that is intentionally not run by default (e.g. manual-only).
  - (also recognized for convenience: `-known-fail-`, `-quarantine-`)
- Skipped scripts should still be runnable manually (e.g. `bash e2e/scripts/e2e-30-xfail-...sh`).
- Note: If an `-xfail-` scenario becomes stable, rename it to remove the marker so it runs under `make e2e`.

### Go tests

- Location: `./e2e/e2e_test.go`
- Package: `e2e_test`
- Build tag: `e2e` (tests do not run under default `go test ./...`)
- Behavior:
  - interacts with `http://localhost:8080` using `net/http`
  - interacts with `127.0.0.1:5353` using a DNS client
  - inspects E2E data directory contents on disk (via `SSHBASTION_E2E_DATA_DIR`)

### Makefile orchestration

The Makefile owns the public targets, but the orchestration script lives under `./e2e/scripts/`.

- `make e2e-clean`
  - removes contents under `./_tmp/data` (the default data dir for ad-hoc/manual runs)
- `make e2e-up`
  - starts services with `docker compose up -d --build --force-recreate`
- `make e2e-down`
  - stops services with `docker compose down`
- `make e2e`
  - runs all `./e2e/scripts/e2e-NN-*.sh` scripts in order.

Implementation note:

- `make e2e` runs all `./e2e/scripts/e2e-NN-*.sh` scripts in order.
  - `e2e-10-go-test.sh`: docker-compose + Go `-tags=e2e` black-box checks (HTTP/DNS/files)
  - `e2e-20-ssh-login.sh`: OpenSSH-based login scenario (ssh-keygen/register/ssh)
- Convention: `e2e-NN-*.sh` scripts must be runnable standalone (they handle setup and cleanup).
- Convention: helper sub-scripts may live under `./e2e/scripts/steps/` if needed.

## What we verify

### HTTP (8080)

- `GET /` returns `200` (readiness)

### Generated files (data dir)

- `<data-dir>/dns/aliases.json` contains the created alias.

Where `<data-dir>` is the host directory bind-mounted to `/data` (typically `./_tmp/e2e-XXXXXXXX/data`).

### DNS (5353)

- A query to `127.0.0.1:5353` for the E2E source name returns at least one A record with the owner name matching the queried name.
- Non-address queries (anything other than A/AAAA) are intentionally not supported by the DNS proxy and should return NODATA (NOERROR + empty answer).

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-app-http]: ../design/design-app-http.md
