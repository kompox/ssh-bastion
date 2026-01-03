---
id: design-roadmap
title: Development Roadmap
status: draft
updated: 2026-01-03T04:42:10Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Development Roadmap

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

### Container development and local testing

- Add `Dockerfile` (build `ghcr.io/kompox/ssh-bastion`)
  - Include `ssh-bastion`, `sshd`, and `dnsmasq` runtime dependencies
- Add `compose.yml` for local testing
  - Two services: `ssh-bastion` and `dnsmasq` sidecar
  - Shared volume for generated dnsmasq config
  - Use `SSHBASTION_AUTH_OVERRIDE_USER_ID` and `SSHBASTION_AUTH_OVERRIDE_EMAIL` for test mode
- Task: [task-20260103a-container-dev-local-testing](../tasks/task-20260103a-container-dev-local-testing.md)

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

### GitHub Actions: CI (tests)

- Run `make test` on push and pull requests

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
