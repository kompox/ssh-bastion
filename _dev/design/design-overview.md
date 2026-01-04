---
id: design-overview
title: Kompox ssh-bastion design overview
status: stable
updated: 2026-01-04T18:26:24Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Kompox ssh-bastion design overview

This document is a coding-agent-oriented design doc for a new repository:

- GitHub repo: `kompox/ssh-bastion`
- Container image: `ghcr.io/kompox/ssh-bastion`

Goal: provide a shared SSH bastion (ProxyJump) and a small web UI to manage user SSH public keys and DNS alias rules used only by the bastion.

For container image/runtime topology details (K8s Pod target + docker-compose emulation), see [design-containers].

## Requirements

### Must

- Provide a shared SSH bastion reachable on TCP/22.
- Users authenticate to the bastion with their own SSH keypair (public keys registered via the web UI).
- End-to-end requirement: the final target (e.g., Gitea/GitLab container on port 22) still authenticates the user’s *local* SSH key as-is; the bastion is only a jump host.
- SSO via OIDC is required for accessing the web UI.
  - In this repo, the ssh-bastion app does **not** implement OIDC directly (MVP).
  - Instead, it trusts an external auth proxy (Azure Easy Auth or oauth2-proxy) and reads user identity from HTTP headers.
- Web UI can be built with HTMX.
- DNS aliasing:
  - Source hostnames: externally published FQDNs (e.g., `gitea.example.com`).
  - Destination hostnames: *cluster-internal* service FQDNs (e.g., `gitea.gitea.svc.cluster.local`).
  - Behavior must be “CNAME-like” (do not pin A/AAAA).
  - The DNS is only used by the bastion.

### Should

- File-based persistence (no database) for both keys and DNS alias rules.
- Simple, auditable formats and atomic writes.
- Minimal operational surface area.

### Non-goals

- No SSH private key storage.
- No per-app LoadBalancer.
- No general-purpose cluster DNS management (only bastion-local DNS).
- No heavy features (RBAC UI, groups, approvals, etc.) unless explicitly added later.

## High-level architecture

Run one Kubernetes **Pod** with two containers and shared volumes.

- Container: `ssh-bastion`
  - Provides:
    - OIDC-protected Web UI and API to manage SSH public keys.
    - Web UI and API to manage DNS alias rules.
    - Generation of `authorized_keys` file used by `sshd`.
    - An in-process DNS proxy used only by the bastion to apply DNS alias rules.
- Container: `sshd`
  - Provides the bastion SSH endpoint.
  - Reads host keys and `authorized_keys` from shared storage.

Why two containers (not multiple processes in one container):

- Keeps `sshd` isolated and operable with K8s-native lifecycle/health checks.
- Avoids maintaining a process supervisor (s6/runit) in PID 1.
- Keeps “config passing” as simple files under the shared data volume.

### Pod DNS flow

Default Pod DNS can remain `dnsPolicy: ClusterFirst`.

Resolver intent:

- `ssh-bastion` keeps the platform default resolver (cluster DNS).
- `sshd` is the only container that resolves via the bastion-local DNS proxy (`127.0.0.1:53`), by rewriting `/etc/resolv.conf` in the sshd entrypoint.

Result: the web UI can resolve upstream names without recursion, while the bastion SSH uses the DNS proxy for aliasing.

## SSH bastion behavior

### User model for SSH

- Use a fixed, shared OS user on the bastion (e.g., `jump`).
- Users authenticate via public keys; the username does not identify the user.
  - This avoids having to create OS users per OIDC user.

### sshd configuration principles

- `PasswordAuthentication no`
- `KbdInteractiveAuthentication no`
- `AuthenticationMethods publickey`
- `PermitRootLogin no`
- `AllowTcpForwarding yes` (required for ProxyJump)
- `X11Forwarding no`, `AllowAgentForwarding no` (recommended)
- `PermitTTY no` (recommended)
- Consider restricting where forwarding can go (optional, later): `PermitOpen`.

Note: avoid `ForceCommand` unless proven compatible with the chosen ProxyJump workflow.

### `authorized_keys` generation

- ssh-bastion app writes a single `authorized_keys` file for the `jump` user, containing all enabled keys.
- The key registry uses a stable fingerprint as the primary key.

Performance note: OpenSSH scans keys linearly in `authorized_keys`. If this becomes an issue, consider moving to `AuthorizedKeysCommand` later. For MVP, a generated file is simplest.

## Bastion-local DNS (in-process DNS proxy)

### Desired behavior

- For a configured alias `gitea.example.com -> gitea.gitea.svc.cluster.local`, resolve as CNAME-like.
- Keep target dynamic by letting cluster DNS answer A/AAAA for the service.

### DNS proxy strategy

- Run a lightweight DNS server inside the `ssh-bastion` process.
- The proxy rewrites A/AAAA queries for configured source names to the destination name, forwards to an upstream resolver (cluster DNS), then rewrites the response back to the original name.
- For non-A/AAAA query types (e.g. TXT/MX), return NODATA (NOERROR + empty answer) instead of forwarding.

For details (protocol behavior, upstream wiring, and rationale), see [design-app-dns].

### Known_hosts collision risk

If the same external FQDN is used for both:

- direct external SSH access, and
- bastion-routed internal SSH access,

host keys may differ and clients can hit `known_hosts` conflicts.

Mitigations:

- Prefer a dedicated SSH hostname per service (e.g., `gitea-ssh.example.com`) if external SSH exists.
- Or ensure the same host key is presented in both paths.

## ssh-bastion app (Web UI)

### UX scope (MVP)

- OIDC login.
- Pages:
  - SSH Public Keys
    - List keys
    - Add key (paste)
    - Disable/Enable
    - Delete
  - DNS Aliases
    - List aliases
    - Add alias
    - Delete alias
- Styling:
  - Use a minimal classless CSS framework for speed (recommended: Pico.css).

### Authentication (via trusted headers)

The ssh-bastion app assumes **no anonymous access**: every request must come through an auth proxy that already enforced OIDC login.

Supported auth proxies (MVP):

- Azure Easy Auth (App Service Authentication)
- oauth2-proxy

The app reads identity from HTTP request headers. Both header names are configurable via environment variables.

Default header mapping:

- Azure Easy Auth
  - User ID header: `X-MS-CLIENT-PRINCIPAL-ID`
  - Email header: `X-MS-CLIENT-PRINCIPAL-NAME`
- oauth2-proxy
  - User ID header: `X-Auth-Request-User`
  - Email header: `X-Auth-Request-Email`

Request handling rule:

- If either the resolved **user ID** header value or the resolved **email** header value is empty (after trimming), the app returns **HTTP 401**.

Notes:

- `X-MS-CLIENT-PRINCIPAL` (Base64 JSON) is **Azure Easy Auth only**. Initial implementation ignores it and only reads `X-MS-CLIENT-PRINCIPAL-ID` / `X-MS-CLIENT-PRINCIPAL-NAME`.
- Azure Easy Auth user IDs are commonly GUIDs. The UI should display **email only**; do not display the raw user ID.

### Identity key

Do not key user data by email.

- Key by a stable user identifier provided by the auth proxy (from the configured **user ID header**).
- Persist a safe, filesystem-friendly identifier derived as a UUIDv3 hash:
  - `userDirId = UUIDv3(NAMESPACE_UUID, normalize(userIdHeaderValue))`
  - Normalization: `strings.TrimSpace`, then `strings.ToLower`.
  - Store under `users/<userDirId>/...`.

Namespace UUID (constant for this repo):

- `9f7f2e3a-3c77-4d7b-a4d4-6d5167b1b5ad`

Important: changing the namespace later will change all derived user directory IDs and effectively orphan existing user directories.

UI note: the UI should use the configured **email header** for display and labels. It must not rely on email for storage keys.

## Configuration (environment variables)

All configuration is via environment variables (no config files).

This repository runs two containers in local docker-compose (and in Kubernetes):

- `ssh-bastion` (ssh-bastion app / Web UI + optional in-process DNS proxy)
- `sshd` (bastion SSH)

Each environment variable below states which role/container it configures.

- `SSHBASTION_DATA_DIR`
  - Default: `/data`
  - Purpose: root directory for file-based persistence.
  - Applies to: **ssh-bastion**, **sshd** (shared volume mount location)
- `SSHBASTION_AUTH_MODE`
  - Values: `easy_auth` | `oauth2_proxy`
  - Purpose: selects default header mapping.
  - Applies to: **ssh-bastion**
- `SSHBASTION_AUTH_USER_ID_HEADER`
  - Default: `X-MS-CLIENT-PRINCIPAL-ID` (when `easy_auth`), `X-Auth-Request-User` (when `oauth2_proxy`)
  - Purpose: request header name containing the stable user ID.
  - Applies to: **ssh-bastion**
- `SSHBASTION_AUTH_EMAIL_HEADER`
  - Default: `X-MS-CLIENT-PRINCIPAL-NAME` (when `easy_auth`), `X-Auth-Request-Email` (when `oauth2_proxy`)
  - Purpose: request header name containing the user email (UI display only).
  - Applies to: **ssh-bastion**
- `SSHBASTION_AUTH_OVERRIDE_USER_ID`
  - Default: (empty)
  - Purpose: when set, uses this value instead of reading from the header. For testing/development only.
  - Applies to: **ssh-bastion**
- `SSHBASTION_AUTH_OVERRIDE_EMAIL`
  - Default: (empty)
  - Purpose: when set, uses this value instead of reading from the header. For testing/development only.
  - Applies to: **ssh-bastion**
- `SSHBASTION_ROLE_DEFAULT`
  - Values: `user` | `admin`
  - Default: `user`
  - Purpose: default role for users not listed in `SSHBASTION_ROLE_ADMIN_IDS`.
  - Applies to: **ssh-bastion**
- `SSHBASTION_ROLE_ADMIN_IDS`
  - Default: (empty)
  - Purpose: comma-separated list of admin user IDs (as provided by the auth proxy user ID header).
  - Applies to: **ssh-bastion**
- `SSHBASTION_DNS_UPSTREAM`
  - Default: (empty)
  - Purpose: DNS proxy upstream resolver (`host:port`). Used when `-dns-upstream` is not set; otherwise it auto-detects from `/etc/resolv.conf`.
  - Applies to: **ssh-bastion**
- `SSHBASTION_LOG_LEVEL`
  - Values: `error` | `warn` | `info` | `debug`
  - Default: `info`
  - Purpose: controls server log verbosity.
  - Applies to: **ssh-bastion**
- `SSHBASTION_SSHD_LOG_LEVEL`
  - Values: `QUIET` | `FATAL` | `ERROR` | `INFO` | `VERBOSE` | `DEBUG` | `DEBUG1` | `DEBUG2` | `DEBUG3`
  - Default: `INFO`
  - Purpose: overrides `sshd_config` `LogLevel` for the bastion sshd.
  - Applies to: **sshd**
- `SSHBASTION_SSHD_NAMESERVER`
  - Default: (empty)
  - Purpose: when set, the sshd entrypoint rewrites `/etc/resolv.conf` to use this `nameserver`.
  - Applies to: **sshd**

Validation rules (MVP):

- At startup: `SSHBASTION_DATA_DIR` must be non-empty.
- Per request: if resolved user ID or email is empty after trimming, return `401`.

Test mode:

- When `SSHBASTION_AUTH_OVERRIDE_USER_ID` and `SSHBASTION_AUTH_OVERRIDE_EMAIL` are set, the app ignores request headers and uses these values for all requests.
- This allows browser testing without a reverse proxy or header injection tools.
- **Security note**: These overrides should only be used in development/testing environments. Do not set them in production.

## Storage layout (file-based)

Root: `${SSHBASTION_DATA_DIR}` (default: `/data`)

Expected on-disk layout:

```
${SSHBASTION_DATA_DIR}/
├── content/
│   └── pages/
│       └── home.md
├── users/
│   └── <userDirId>/
│       ├── profile.json
│       └── keys/
│           ├── <fingerprint>.json
│           └── <fingerprint>.pub
├── authorized_keys/
│   └── jump
├── dns/
│   └── aliases.json
└── ssh/
  ├── ssh_host_ed25519_key
  └── ssh_host_rsa_key
```

Notes:

- `content/pages/home.md` is optional; it is read by `GET /` and editable by admins via `/admin/home`.
- `users/<userDirId>/profile.json` stores user display info (e.g. email) for admin views.
- `authorized_keys/jump` is generated from all enabled keys.
- `dns/aliases.json` is the source of truth for DNS aliases (read by the in-process DNS proxy).
- `ssh/ssh_host_*` are bastion host keys generated by the `sshd` container entrypoint and persisted under the shared data dir.

### Fingerprint

- Parse the submitted public key.
- Compute a canonical fingerprint (recommended: SHA256 fingerprint like OpenSSH shows).
- Normalize whitespace and reject unsupported key types.

### Atomic writes

Write to a temp file in the same directory, then rename.

## Deployment (Kubernetes)

### Pod spec summary

- 1 Pod / 2 containers
  - Container: `ghcr.io/kompox/ssh-bastion` running `ssh-bastion serve ...`
  - Container: `ghcr.io/kompox/ssh-bastion` running `sshd ...`
- Shared volumes:
  - `/data` as a PVC
- Ports:
  - `ssh-bastion`: 8080 (Web UI)
  - `ssh-bastion`: 53/udp (DNS proxy; optional)
  - `sshd`: 22 (SSH)

### Security contexts

- `sshd` typically needs root to bind 22 or use `CAP_NET_BIND_SERVICE`.
- If the DNS proxy listens on 53, `ssh-bastion` needs to bind 53, so use `CAP_NET_BIND_SERVICE`.
- Prefer dropping all other capabilities.

## Repository implementation plan

### Tech stack

- Language: Go
- HTTP:
  - `net/http` + a small router (chi) or stdlib mux
  - `html/template`
  - HTMX for interactions
- CSS: Pico.css
- Key parsing: use Go’s SSH parsing (`golang.org/x/crypto/ssh`).

### Executables / commands

The container image supports:

- `ssh-bastion serve` (runs HTTP, and optionally the DNS proxy)
- `ssh-bastion web` (HTTP-only; back-compat alias)

### Milestones

1. **File storage + models**
    - Key model, user model, alias model
    - Fingerprint calc + validation
    - Atomic write helpers
2. **Web UI (no auth)**
    - CRUD pages for keys + aliases
  - Generate authorized_keys
3. **Authentication integration (auth proxy headers)**
    - Implement header-based auth (Azure Easy Auth + oauth2-proxy)
    - Header names configurable via environment variables
    - 401 when user ID or email header is missing/empty
4. **Container image**
    - Multi-stage Docker build
  - Include `sshd`
    - Minimal runtime base
5. **Kubernetes manifests / Helm**
    - Deployment, Service (LB for SSH), Service for Web UI (cluster-internal)
    - PVC
    - Health checks
6. **GitHub Actions**
    - Buildx
    - Push to GHCR

### Acceptance criteria (MVP)

- A user can log in via OIDC (via an auth proxy; Azure Easy Auth or oauth2-proxy is acceptable).
- A user can register an SSH public key via web UI.
- Bastion accepts the user key to log in as `jump`.
- From bastion, `ssh` to `gitea.example.com` resolves to the configured internal service FQDN (CNAME-like), and connection is possible.
- End-to-end authentication at the final target uses the user’s local key.

## Open questions

- Whether external SSH access to the same external FQDN exists (known_hosts collision risk).
- Whether to restrict forwarding targets/ports via `PermitOpen` (security hardening).

## References

- [design-containers] - Containers (image + runtime topology)
- [design-app-http] - App HTTP (Routes & Sitemap)
- External resources:
  - [OpenSSH sshd_config]
  - [oauth2-proxy]
  - [Azure App Service Easy Auth: user identities]
  - [HTMX]
  - [Pico.css]

[design-containers]: ../design/design-containers.md
[design-app-dns]: ../design/design-app-dns.md
[design-app-http]: ../design/design-app-http.md
[OpenSSH sshd_config]: https://man.openbsd.org/sshd_config
[oauth2-proxy]: https://oauth2-proxy.github.io/oauth2-proxy/
[Azure App Service Easy Auth: user identities]: https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-user-identities
[HTMX]: https://htmx.org/
[Pico.css]: https://picocss.com/
