---
id: design-e2e-testing
title: E2E / integration testing (docker-compose + published ports)
status: draft
updated: 2026-01-03T06:41:25Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: E2E / integration testing (docker-compose)

This document defines the E2E/integration test approach for local development and CI.

The E2E tests are intentionally “black-box”: they interact only with:

- published ports from `docker compose up -d`
  - HTTP: `http://localhost:8080`
  - DNS: `127.0.0.1:5353` (TCP/UDP)
- the bind-mounted data directory `./_tmp/data`

Container topology and networking details are specified in [design-containers].

## Goals

- Provide a repeatable E2E test flow that validates the system as deployed by docker-compose.
- Validate:
  - web server is reachable on `:8080`
  - DNS sidecar answers on `:5353`
  - the web app generates expected files under `./_tmp/data`
- Keep tests independent from internal Go packages (black-box behavior).

## Non-goals

- SSH (port 22) E2E testing (not in current compose scope).
- White-box tests that import internal registries or storage.

## Test topology

- Runtime: `docker compose up -d --build`
- Persistence: bind mount `./_tmp/data:/data`
- Auth: docker-compose runs the web app in test mode via `SSHBASTION_AUTH_OVERRIDE_USER_ID` / `SSHBASTION_AUTH_OVERRIDE_EMAIL`

## Test implementation

### Go tests

- Location: `./e2e/e2e_test.go`
- Package: `e2e_test`
- Build tag: `e2e` (tests do not run under default `go test ./...`)
- Behavior:
  - interacts with `http://localhost:8080` using `net/http`
  - interacts with `127.0.0.1:5353` using a DNS client
  - inspects `./_tmp/data` contents on disk

### Makefile orchestration

The Makefile owns the public targets, but the orchestration script lives under `./e2e/scripts/`.

- `make e2e-clean`
  - removes contents under `./_tmp/data` (ensures no state leakage)
- `make e2e-up`
  - starts services with `docker compose up -d --build`
- `make e2e-down`
  - stops services with `docker compose down`
- `make e2e`
  - runs the full flow, including:
    - bringing up docker-compose
    - waiting for HTTP readiness
    - running the E2E test phase that creates DNS aliases and verifies generated files
    - restarting dnsmasq (since it does not auto-reload config)
    - running the E2E test phase that verifies DNS resolution via port 5353

Implementation note:

- `make e2e` calls `bash ./e2e/scripts/run.sh`.
- If scenarios grow, add scripts under `./e2e/scripts/` (e.g. `./e2e/scripts/e2e-<scenario>.sh`) and keep assertions in Go tests.

## What we verify

### HTTP (8080)

- `GET /` returns `200` (readiness)

### Generated files (./_tmp/data)

- `./_tmp/data/dns/dnsmasq.d/generated.conf` exists and contains the expected `cname=` line.
- `./_tmp/data/dns/aliases.json` contains the created alias.

### DNS (5353)

- A query to `127.0.0.1:5353` for the E2E source name returns either:
  - at least one A record, or
  - a CNAME pointing at the configured destination (and that destination resolves to an A record)

Note: dnsmasq reload is currently modeled as “restart the container” in docker-compose.

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-webapp-routes] - Web app routes & sitemap

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-webapp-routes]: ../design/design-webapp-routes.md
