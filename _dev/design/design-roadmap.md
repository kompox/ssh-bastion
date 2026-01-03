---
id: design-roadmap
title: Development Roadmap
status: draft
updated: 2026-01-03T07:14:29Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Development Roadmap

This roadmap complements [design-overview]. Container-specific decisions are tracked in [design-containers].

## IN-PROGRES

### Container development and local testing

- Deliver a runnable container image (`ghcr.io/kompox/ssh-bastion`)
  - Include runtime dependencies for local sidecar emulation (`dnsmasq`) and future SSH work (`openssh-server`)
- Deliver a minimal docker-compose setup for local testing
  - Two containers: `ssh-bastion` (web) + `dnsmasq` sidecar
  - Pod-like sidecar networking (`network_mode: service:ssh-bastion`)
  - Persist `/data` via bind mount to `./_tmp/data` (simplifies testing/inspection)
  - Use `SSHBASTION_AUTH_OVERRIDE_USER_ID` / `SSHBASTION_AUTH_OVERRIDE_EMAIL` for test mode
- Spec details: see [design-containers]
- Task: [task-20260103a-container-dev-local-testing](../tasks/task-20260103a-container-dev-local-testing.md)

## TODO

### Roles: admin and user

- Introduce roles to control access levels
- Configuration (environment variables):
  - `SSHBASTION_ROLE_ADMIN_IDS`: comma-separated list of admin user IDs
  - `SSHBASTION_ROLE_DEFAULT`: default role for users not in the admin list (default: `user`)

|Role|Capabilities|
|-|-|
|`admin`|Full access to all users' keys and DNS aliases|
|`user`|Manage own keys and DNS aliases only|

### Permissions: SSH public keys

- `admin`: manage all users’ SSH public keys
- `user`: manage own SSH public keys only

### Permissions: DNS alias rules

- `admin`: manage DNS alias rules
- `user`: no access (cannot view or change)

### Container entrypoints

- Support primary entrypoints:
  - `ssh-bastion web` (web app + generates files)
  - `ssh-bastion dns` (runs dnsmasq using generated config)
- Optional debug helpers:
  - `ssh-bastion render authorized-keys`
  - `ssh-bastion render dnsmasq-conf`
- Define/implement reload behavior when generated files change:
  - dnsmasq: SIGHUP or safe restart strategy
  - sshd: ensure it picks up updated `authorized_keys` reliably

### Kubernetes manifests / Helm

- Add initial manifests (or a Helm chart):
  - Deployment: 1 Pod / 2 containers (sshd+web, dnsmasq)
  - Services: SSH (LoadBalancer) + web (cluster-internal)
  - PVC for `/data`
  - Health checks/readiness
  - SecurityContext with minimal capabilities (likely `CAP_NET_BIND_SERVICE` for ports 22/53)

### GitHub Actions: Docker build and push

- On push to `main` and `workflow_dispatch`, build and push `ghcr.io/kompox/ssh-bastion`

### Security hardening decisions

- Decide whether to restrict forwarding targets via `PermitOpen` (and configuration approach)
- Decide and document mitigation for `known_hosts` collisions when the same FQDN is reachable externally

### Docs: README.md

- Update `README.md`:
  - Brief project overview and goals
  - Quick links to key design docs and task files
  - Minimal “how to run locally” notes (test mode + data dir)

## DONE

### GitHub Actions: CI (tests)

- Run `make test` on push and pull requests
- Verified CI success on `main`
- Task: [task-20260103b-ci-tests](../tasks/task-20260103b-ci-tests.md)

### Web app: observability (logging)

- Identify current logging gaps (e.g., template failures, silent best-effort behavior)
- Define a minimal, consistent logging schema (prefixes/fields)
- Log key events (avoid sensitive data):
  - Auth decisions (without dumping raw headers)
  - Key operations (fingerprint only; never key material)
  - DNS operations (source/destination)
- Add targeted tests if it’s practical; otherwise verify manually
- Optionally document what operators should expect in logs
- Task: [task-20260102d-webapp-observability](../tasks/task-20260102d-webapp-observability.md)

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
