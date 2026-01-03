---
id: task-20260103a-container-dev-local-testing
title: Container development and local testing (Dockerfile + compose)
status: done
updated: 2026-01-03T06:41:25Z
---
# Task: Container development and local testing (Dockerfile + compose)

## Goal

Create a runnable container image and a minimal local docker-compose setup for development/testing.

This task is derived from the roadmap section “Container development and local testing”.

## Scope

### In

- Provide a runnable container image for local development/testing
  - `Dockerfile` builds `ghcr.io/kompox/ssh-bastion`
  - Default: start web server (`ssh-bastion web`)
  - Include runtime packages needed for local DNS sidecar emulation (`dnsmasq`) and future SSH work (`openssh-server`)
- Provide a minimal local “Pod-like” runtime topology via Docker Compose
  - `compose.yml` runs two containers: web + dnsmasq (sidecar)
  - Persist `/data` via a bind mount to `./_tmp/data` (simplifies testing/inspection)
  - Auth runs in test mode via `SSHBASTION_AUTH_OVERRIDE_USER_ID` / `SSHBASTION_AUTH_OVERRIDE_EMAIL`

### Out

- Kubernetes manifests / Helm
- `sshd` startup, configuration, and SSH connectivity (port 22) in Docker Compose
  - No `sshd_config` generation, host keys management, user/authorized_keys wiring, or security hardening
- Adding new runtime entrypoints/commands (e.g. `ssh-bastion dns`, `ssh-bastion sshd`)
  - Compose may override the image entrypoint to run `dnsmasq` for local emulation
- Production hardening beyond what is needed for local testing
  - e.g. K8s `securityContext`, capabilities, read-only rootfs, probes, resource limits, network policies

## Spec (summary)

Detailed container/runtime decisions (image contents, Pod topology, compose sidecar emulation, DNS resolver wiring) are documented in [design-containers].

E2E/integration testing against the docker-compose setup (published ports `:8080` / `:5353` + `./_tmp/data` inspection) is documented in [design-e2e-testing].

This task document intentionally keeps only the “deliverables contract” above.

## Plan & Checklist

- [x] Implement `Dockerfile`
  - [x] Multi-stage build for Go binary
  - [x] Install/ship `sshd` and `dnsmasq` runtime dependencies
  - [x] Provide sensible default entrypoint/command (may be overridden by compose)

- [x] Implement `compose.yml`
  - [x] Define `ssh-bastion` service (web)
  - [x] Define `dnsmasq` sidecar service
  - [x] Wire shared volumes for:
    - [x] `/data` persistence (bind mount `./_tmp/data`)
    - [x] generated dnsmasq config path
  - [x] Expose ports needed for local testing
  - [x] Set test mode env vars (`SSHBASTION_AUTH_OVERRIDE_USER_ID`, `SSHBASTION_AUTH_OVERRIDE_EMAIL`)

- [x] Document minimal local run steps
  - [x] Add quickstart notes (where to browse, where data lives)

## Quickstart (local)

### Run with Docker Compose

```bash
docker compose up --build
```

- Web UI: http://localhost:8080/
- dnsmasq (optional from host): localhost port 5353 (TCP/UDP)
- Data is stored under `./_tmp/data` (bind mount).

Notes:

- Auth is in test mode via `SSHBASTION_AUTH_OVERRIDE_USER_ID` / `SSHBASTION_AUTH_OVERRIDE_EMAIL`.
- dnsmasq reads config from `/data/dns/dnsmasq.d/*.conf`.
  - The web app generates `/data/dns/dnsmasq.d/generated.conf`.
  - Restart the `dnsmasq` service to pick up changes.
- The `ssh-bastion` container uses `127.0.0.1:53` for DNS (sidecar emulation).

## Progress

- 2026-01-03T04:41:05Z
  - Create task document

- 2026-01-03T04:45:46Z
  - Add `Dockerfile` (multi-stage build) with `openssh-server` and `dnsmasq` installed
  - Add `compose.yml` for local testing (web + dnsmasq sharing `/data`)
  - Add quickstart instructions

- 2026-01-03T05:00:46Z
  - Switch runtime base image to Alpine for a smaller image
  - Update compose to emulate Pod sidecar networking (`network_mode: service:ssh-bastion`)
  - Configure container resolver to use dnsmasq at `127.0.0.1:53`

- 2026-01-03T05:34:12Z
  - Switch docker-compose persistence from a named volume to a bind mount (`./_tmp/data`)

- 2026-01-03T06:41:25Z
  - Move `make e2e` orchestration script to `./e2e/scripts/run.sh`

## References

- [design-containers] - Containers (image + runtime topology)
- [design-overview] - Design overview document
- [design-e2e-testing] - E2E / integration testing (docker-compose + published ports)

[design-containers]: ../design/design-containers.md
[design-overview]: ../design/design-overview.md
[design-e2e-testing]: ../design/design-e2e-testing.md
