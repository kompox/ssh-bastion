---
id: design-roadmap
title: Development Roadmap
status: draft
updated: 2026-01-04T08:35:40Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Development Roadmap

This roadmap complements [design-overview]. Container-specific decisions are tracked in [design-containers].

## IN-PROGRESS

### Docs: design docs update

- Eliminate mentions of dnsmasq sidecar in design docs
- Task: [task-20260104c-design-docs-update](../tasks/task-20260104c-design-docs-update.md)

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
  - `ssh-bastion serve` (runs selected services in one process)
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

### Web app: site map and routing update

- Site map:
  - `/` home (reads markdown from `${SSHBASTION_DATA_DIR}/content/pages/home.md`; default base: `/data`; missing file shows placeholder)
  - `/ssh` SSH key management (key endpoints under `/ssh/keys/...`)
  - `/dns` DNS alias management (no change until roles/permissions exist)
  - `/admin` placeholder (future use)
- Approach: HTMX + server-rendered HTML for MVP; defer adding `/api/*` until a real REST/JSON need emerges
- Task: [task-20260104b-webapp-routing-update](../tasks/task-20260104b-webapp-routing-update.md)

### DNS alias testing

- E2E tests for DNS alias functionality using docker-compose setup
- Add E2E test to verify aliases resolve to A/AAAA answers (no client-side CNAME chasing)
- Task: [task-20260103e-dns-alias-testing](../tasks/task-20260103e-dns-alias-testing.md)

### DNS alias: query rewrite proxy

- Run a minimal DNS forwarder inside ssh-bastion (in-process): listen on UDP :53
- Rewrite only QNAME based on aliases configured in the web app (e.g. `hoge.local` -> `github.com`)
- Forward the rewritten query to an upstream DNS server and return the response
- Rewrite the owner name back to the original QNAME for `A`/`AAAA` records only
- Eliminate the dnsmasq sidecar container in docker-compose
- Expose service selection via `ssh-bastion serve` flags (web only, DNS only, or both)
- Task: [task-20260104a-dns-query-rewrite-proxy](../tasks/task-20260104a-dns-query-rewrite-proxy.md)

### Container development and local testing

- Spec summary:
  - One published image (`ghcr.io/kompox/ssh-bastion`) supports the local topology
  - Local compose emulates a multi-container Pod with shared `/data`
  - Runtime dependencies for sidecar DNS and SSH are included in the image
- Spec details: see [design-containers]
- Tasks:
  - [task-20260103a-container-dev-local-testing](../tasks/task-20260103a-container-dev-local-testing.md)
  - [task-20260103d-sshd-container](../tasks/task-20260103d-sshd-container.md)

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

## CANCELED

### Container: rebuild with glibc

- Rebuild the container image using a glibc-based base image (e.g., `debian:stable-slim` or `ubuntu:latest`)
- Ensure that sshd can resolve CNAME to A records correctly
- Task: [task-20260103f-container-rebuild-glibc](../tasks/task-20260103f-container-rebuild-glibc.md)

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-webapp-routes] - Web App Routes & Sitemap

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-webapp-routes]: ../design/design-webapp-routes.md
