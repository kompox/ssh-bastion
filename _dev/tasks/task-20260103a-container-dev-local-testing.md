---
id: task-20260103a-container-dev-local-testing
title: Container development and local testing (Dockerfile + compose)
status: todo
updated: 2026-01-03T04:41:05Z
---
# Task: Container development and local testing (Dockerfile + compose)

## Goal

Create a runnable container image and a minimal local docker-compose setup for development/testing.

This task is derived from the roadmap section “Container development and local testing”.

## Scope

### In

- Add `Dockerfile` that builds `ghcr.io/kompox/ssh-bastion`
  - Include `ssh-bastion` binary
  - Include runtime dependencies for `sshd` and `dnsmasq`
- Add `compose.yml` for local testing
  - Two services: `ssh-bastion` and `dnsmasq` sidecar
  - Shared volume for generated dnsmasq config
  - Use test mode auth via:
    - `SSHBASTION_AUTH_OVERRIDE_USER_ID`
    - `SSHBASTION_AUTH_OVERRIDE_EMAIL`

### Out

- Kubernetes manifests / Helm
- Production hardening beyond what is needed for local testing

## Plan & Checklist

- [ ] Implement `Dockerfile`
  - [ ] Multi-stage build for Go binary
  - [ ] Install/ship `sshd` and `dnsmasq` runtime dependencies
  - [ ] Provide sensible default entrypoint/command (may be overridden by compose)

- [ ] Implement `compose.yml`
  - [ ] Define `ssh-bastion` service (web + sshd as appropriate)
  - [ ] Define `dnsmasq` sidecar service
  - [ ] Wire shared volumes for:
    - [ ] `/data` persistence
    - [ ] generated dnsmasq config path
  - [ ] Expose ports needed for local testing (SSH and web)
  - [ ] Set test mode env vars (`SSHBASTION_AUTH_OVERRIDE_USER_ID`, `SSHBASTION_AUTH_OVERRIDE_EMAIL`)

- [ ] Document minimal local run steps
  - [ ] Add quickstart notes (where to browse, where data lives)

## Progress

- 2026-01-03T04:41:05Z
  - Create task document
