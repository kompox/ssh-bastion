---
id: design-roadmap
title: Development Roadmap
status: draft
updated: 2026-01-09T09:32:53Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Development Roadmap

This roadmap complements [design-overview]. Container-specific decisions are tracked in [design-containers].

## IN-PROGRESS

### Kubernetes manifests / Helm

- Add initial manifests (or a Helm chart):
  - Deployment: 1 Pod / 2 containers (sshd + ssh-bastion)
  - Services: SSH (LoadBalancer) + web (cluster-internal)
  - PVC for `/data`
  - Health checks/readiness
  - SecurityContext with minimal capabilities (likely `CAP_NET_BIND_SERVICE` for ports 22/53)
- Chart name: `kompox-ssh-bastion` (avoid collisions with existing `ssh-bastion` charts)
- Task: [task-20260104i-k8s-manifests-helm](../tasks/task-20260104i-k8s-manifests-helm.md)

### UI: SSH key description field

- Optional description field for SSH keys so users can easily identify them ("work laptop", "home desktop", etc.).
- No edit UI required (set on add only).
- Storage: store as the authorized_keys comment (not a separate metadata DB).
- Sanitization: trim surrounding whitespace; normalize whitespace; reject control chars/newlines.
- Length limit: max 128 chars after sanitization; over-limit is an error.
- Design: [design-ssh-keys]
- Task: [task-20260109a-ssh-key-description](../tasks/task-20260109a-ssh-key-description.md)

## TODO

### Container entrypoints

- Support primary entrypoints:
  - `ssh-bastion serve` (runs selected services in one process)
- Optional debug helpers:
  - `ssh-bastion render authorized-keys`
- Define/implement reload behavior when generated files change:
  - DNS proxy: changes take effect without reload (reads alias registry at query time)
  - sshd: ensure it picks up updated `authorized_keys` reliably

### Security hardening decisions

- Decide whether to restrict forwarding targets via `PermitOpen` (and configuration approach)
- Decide and document mitigation for `known_hosts` collisions when the same FQDN is reachable externally

## DONE

### Security: SSH forwarding hardening

- Added settings to restrict sshd TCP port forwarding to configured targets only (block arbitrary destination IP/ports).
- Admin UI: manage allowlisted targets (host:port) via `/admin/targets`.
- Enforcement: generate sshd_config with `PermitOpen` allowlist and apply safely via reload/restart triggers.
- Tests: unit tests + E2E scenario verifying allow/deny behavior.
- Task: [task-20260106a-ssh-forwarding-hardening](../tasks/task-20260106a-ssh-forwarding-hardening.md)

### Security: DNS proxy hardening

- Added `SSHBASTION_DNS_ALIASES_ONLY` setting to restrict DNS proxy resolution to configured aliases only.
- When enabled: return NXDOMAIN for non-aliased A/AAAA queries.
- Admin UI: `/admin/dns` displays status in the page title (display-only; no toggle).
- Task: [task-20260105a-dns-proxy-hardening](../tasks/task-20260105a-dns-proxy-hardening.md)

### Docs: README.md

- Updated `README.md` as the primary entry point (Features, multi-site example, and Developer's guide links)
- Highlighted shared public IP / LoadBalancer rule benefit (single SSH entrypoint)
- Updated the Gitea Kubernetes example and Mermaid diagram for the gitea1/2/3 multi-site use case
- Task: [task-20260104h-docs-readme](../tasks/task-20260104h-docs-readme.md)

### Admin: Home page markdown editing

- Implement admin-only markdown editor for the home page content (`/admin/home`)
- Edit/save `${SSHBASTION_DATA_DIR}/content/pages/home.md` with basic size limit and success message
- Task: [task-20260104g-admin-home-editing](../tasks/task-20260104g-admin-home-editing.md)

### Permissions: SSH keys, DNS, admin

- Define and enforce route-level permissions for the web app.
- Spec reference: [design-app-http]
- Task: [task-20260104f-permissions-enforcement](../tasks/task-20260104f-permissions-enforcement.md)

Admin endpoints (initial scope):

- `GET /admin` dashboard
- `GET /admin/users` (list current key owners; no POST)
- `GET /admin/keys` (all users’ keys list; no POST)
- `GET /admin/dns` + `POST /admin/dns/*` (migrated from `/dns/*`)
- `/dns` is removed (no redirect)
- Pagination/search: out of scope (separate tasks)

|Area|`admin`|`user`|
|-|-|-|
|SSH public keys (`/ssh/keys`)|Manage all users’ SSH public keys|Manage own SSH public keys only|
|DNS alias rules (`/admin/dns`)|Manage DNS alias rules|No access (cannot view or change)|
|Admin dashboard (`/admin`)|Manage the home page content etc.|No access|

### Roles: admin and user

- Introduce roles to control access levels
- Configuration (environment variables):
  - `SSHBASTION_ROLE_ADMIN_IDS`: comma-separated list of admin user IDs
  - `SSHBASTION_ROLE_DEFAULT`: default role for users not in the admin list (default: `user`)
- Task: [task-20260104e-roles-admin-user](../tasks/task-20260104e-roles-admin-user.md)

|Role|Capabilities|
|-|-|
|`admin`|Full access to all users' keys and DNS aliases|
|`user`|Manage own keys and DNS aliases only|

### DNS proxy: upstream autodetect

- Implemented upstream DNS selection logic so it works without explicit config:
  - Prefer `-dns-upstream` flag when set
  - Otherwise prefer `SSHBASTION_DNS_UPSTREAM` when set
  - Otherwise auto-detect from `/etc/resolv.conf` (first `nameserver`, add `:53`; IPv6 bracket form)
- Task: [task-20260104d-dns-upstream-autodetect](../tasks/task-20260104d-dns-upstream-autodetect.md)

### Docs: design docs update

- Updated design docs to reflect the in-process DNS proxy (no DNS sidecar container)
- Documented upstream DNS selection: `SSHBASTION_DNS_UPSTREAM` or `/etc/resolv.conf` autodetect
- Updated remaining route references to `/ssh` and `/ssh/keys`
- Added `design-app-dns.md` and renamed routes doc to `design-app-http.md`
- Task: [task-20260104c-design-docs-update](../tasks/task-20260104c-design-docs-update.md)

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
- Eliminate the sidecar DNS container in docker-compose
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
- [design-app-http] - App HTTP (Routes & Sitemap)
- [design-ssh-keys] - SSH keys (UI + storage)

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-app-http]: ../design/design-app-http.md
[design-ssh-keys]: ../design/design-ssh-keys.md
