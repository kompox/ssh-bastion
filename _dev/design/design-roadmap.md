---
id: design-roadmap
title: Development Roadmap
status: draft
updated: 2026-01-03T08:25:50Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Development Roadmap

This roadmap complements [design-overview]. Container-specific decisions are tracked in [design-containers].

## IN-PROGRES

### Container development and local testing

- Spec summary:
  - One published image (`ghcr.io/kompox/ssh-bastion`) supports the local topology
  - Local compose emulates a multi-container Pod with shared `/data`
  - Runtime dependencies for sidecar DNS and SSH are included in the image
- Spec details: see [design-containers]
- Tasks:
  - [task-20260103a-container-dev-local-testing](../tasks/task-20260103a-container-dev-local-testing.md)
  - [task-20260103d-sshd-container](../tasks/task-20260103d-sshd-container.md)

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

### Security hardening decisions

- Decide whether to restrict forwarding targets via `PermitOpen` (and configuration approach)
- Decide and document mitigation for `known_hosts` collisions when the same FQDN is reachable externally

### Docs: README.md

- Update `README.md`:
  - Brief project overview and goals
  - Quick links to key design docs and task files
  - Minimal “how to run locally” notes (test mode + data dir)

## DONE

### GitHub Actions: Docker build and push

- On push to `main` and tags `v*`, build and push multi-arch images (`linux/amd64`, `linux/arm64`) to `ghcr.io/kompox/ssh-bastion`
- Verified `ghcr.io/kompox/ssh-bastion:main` is published
- Task: [task-20260103c-docker-push](../tasks/task-20260103c-docker-push.md)

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
