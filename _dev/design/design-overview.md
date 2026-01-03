---
id: design-overview
title: kompox-ssh-bastion (design + implementation plan)
status: stable
updated: 2026-01-03T04:31:37Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: kompox-ssh-bastion (design + implementation plan)

This document is a coding-agent-oriented design doc for a new repository:

- GitHub repo: `kompox/ssh-bastion`
- Container image: `ghcr.io/kompox/ssh-bastion`

Goal: provide a shared SSH bastion (ProxyJump) and a small web UI to manage user SSH public keys and DNS alias rules used only by the bastion.

## Requirements

### Must

- Provide a shared SSH bastion reachable on TCP/22.
- Users authenticate to the bastion with their own SSH keypair (public keys registered via the web UI).
- End-to-end requirement: the final target (e.g., Gitea/GitLab container on port 22) still authenticates the user’s *local* SSH key as-is; the bastion is only a jump host.
- SSO via OIDC is required for accessing the web UI.
  - In this repo, the web app does **not** implement OIDC directly (MVP).
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

- Container A: `sshd + web app`
  - `sshd` provides the bastion.
  - Web app provides:
    - OIDC-protected UI and API to manage SSH public keys.
    - UI and API to manage DNS alias rules.
    - Generation of `authorized_keys` file used by `sshd`.
    - Generation of dnsmasq config snippets.
- Container B: `dnsmasq`
  - Listens on `127.0.0.1:53` within the Pod network namespace.
  - Provides CNAME-like aliasing for selected external FQDNs.
  - Forwards everything else to cluster DNS.

Why two containers (not three processes in one container):

- Keeps K8s-native lifecycle/health checks and failure isolation.
- Avoids maintaining a process supervisor (s6/runit) in PID 1.
- Still avoids “hard” config passing: config is just files in a shared volume + SIGHUP.

### Pod DNS flow

- Pod `dnsPolicy: None`
- Pod `dnsConfig.nameservers: ["127.0.0.1"]`

Then dnsmasq forwards to the cluster DNS service (kube-dns/CoreDNS) IP.

Result: name resolution inside the Pod is controlled by bastion-local dnsmasq, while still resolving normal cluster names.

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

- Web app writes a single `authorized_keys` file for the `jump` user, containing all enabled keys.
- The key registry uses a stable fingerprint as the primary key.

Performance note: OpenSSH scans keys linearly in `authorized_keys`. If this becomes an issue, consider moving to `AuthorizedKeysCommand` later. For MVP, a generated file is simplest.

## Bastion-local DNS (dnsmasq)

### Desired behavior

- For a configured alias `gitea.example.com -> gitea.gitea.svc.cluster.local`, resolve as CNAME-like.
- Keep target dynamic by letting cluster DNS answer A/AAAA for the service.

### dnsmasq configuration strategy

Generate config under a shared directory (e.g., `/etc/dnsmasq.d/generated/*.conf`).

Use `cname=` directives:

- `cname=gitea.example.com,gitea.gitea.svc.cluster.local`

Then forward all other queries to cluster DNS:

- `server=<cluster-dns-ip>`

Reload on updates:

- send `SIGHUP` to dnsmasq (or restart the dnsmasq container if that is simpler/safer).

### Known_hosts collision risk

If the same external FQDN is used for both:

- direct external SSH access, and
- bastion-routed internal SSH access,

host keys may differ and clients can hit `known_hosts` conflicts.

Mitigations:

- Prefer a dedicated SSH hostname per service (e.g., `gitea-ssh.example.com`) if external SSH exists.
- Or ensure the same host key is presented in both paths.

## Web app

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

The web app assumes **no anonymous access**: every request must come through an auth proxy that already enforced OIDC login.

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

- `SSHBASTION_DATA_DIR`
  - Default: `/data`
  - Purpose: root directory for file-based persistence.
- `SSHBASTION_AUTH_MODE`
  - Values: `easy_auth` | `oauth2_proxy`
  - Purpose: selects default header mapping.
- `SSHBASTION_AUTH_USER_ID_HEADER`
  - Default: `X-MS-CLIENT-PRINCIPAL-ID` (when `easy_auth`), `X-Auth-Request-User` (when `oauth2_proxy`)
  - Purpose: request header name containing the stable user ID.
- `SSHBASTION_AUTH_EMAIL_HEADER`
  - Default: `X-MS-CLIENT-PRINCIPAL-NAME` (when `easy_auth`), `X-Auth-Request-Email` (when `oauth2_proxy`)
  - Purpose: request header name containing the user email (UI display only).
- `SSHBASTION_AUTH_OVERRIDE_USER_ID`
  - Default: (empty)
  - Purpose: when set, uses this value instead of reading from the header. For testing/development only.
- `SSHBASTION_AUTH_OVERRIDE_EMAIL`
  - Default: (empty)
  - Purpose: when set, uses this value instead of reading from the header. For testing/development only.

- `SSHBASTION_LOG_LEVEL`
  - Values: `error` | `warn` | `info` | `debug`
  - Default: `info`
  - Purpose: controls server log verbosity.

Validation rules (MVP):

- At startup: `SSHBASTION_DATA_DIR` must be non-empty.
- Per request: if resolved user ID or email is empty after trimming, return `401`.

Test mode:

- When `SSHBASTION_AUTH_OVERRIDE_USER_ID` and `SSHBASTION_AUTH_OVERRIDE_EMAIL` are set, the app ignores request headers and uses these values for all requests.
- This allows browser testing without a reverse proxy or header injection tools.
- **Security note**: These overrides should only be used in development/testing environments. Do not set them in production.

## Storage layout (file-based)

Root: `${SSHBASTION_DATA_DIR}` (default: `/data`)

- `${SSHBASTION_DATA_DIR}/users/<userDirId>/keys/<fingerprint>.pub`
- `${SSHBASTION_DATA_DIR}/users/<userDirId>/keys/<fingerprint>.json` (metadata)
- `${SSHBASTION_DATA_DIR}/authorized_keys/jump` (generated)
- `${SSHBASTION_DATA_DIR}/dns/aliases.json` (source of truth)
- `${SSHBASTION_DATA_DIR}/dns/dnsmasq.d/generated.conf` (generated)

### Fingerprint

- Parse the submitted public key.
- Compute a canonical fingerprint (recommended: SHA256 fingerprint like OpenSSH shows).
- Normalize whitespace and reject unsupported key types.

### Atomic writes

Write to a temp file in the same directory, then rename.

## Deployment (Kubernetes)

### Pod spec summary

- 1 Pod / 2 containers
  - Container A: `ghcr.io/kompox/ssh-bastion` running `sshd` + web
  - Container B: `ghcr.io/kompox/ssh-bastion` running `dnsmasq` (same image, different args)
- Shared volumes:
  - `/data` as a PVC
  - `/etc/dnsmasq.d/generated` (can live under `/data` and bind-mount into dnsmasq)
- Ports:
  - Container A: 22 (sshd), 8080 (web)
  - Container B: 53/udp + 53/tcp (dns)

### Security contexts

- `sshd` typically needs root to bind 22 or use `CAP_NET_BIND_SERVICE`.
- `dnsmasq` needs to bind 53, so use `CAP_NET_BIND_SERVICE`.
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

The container image should support two entrypoints:

- `ssh-bastion web` (runs web app + writes generated files)
- `ssh-bastion dns` (runs dnsmasq with generated config)

Optionally also:

- `ssh-bastion render authorized-keys` (debug)
- `ssh-bastion render dnsmasq-conf` (debug)

### Milestones

1. **File storage + models**
   - Key model, user model, alias model
   - Fingerprint calc + validation
   - Atomic write helpers

2. **Web UI (no auth)**
   - CRUD pages for keys + aliases
   - Generate authorized_keys + dnsmasq conf

3. **Authentication integration (auth proxy headers)**
  - Implement header-based auth (Azure Easy Auth + oauth2-proxy)
  - Header names configurable via environment variables
  - 401 when user ID or email header is missing/empty

4. **Container image**
   - Multi-stage Docker build
   - Include `sshd` and `dnsmasq`
   - Minimal runtime base

5. **Kubernetes manifests / Helm**
   - Deployment, Service (LB for SSH), Service for web (cluster-internal)
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

- See: [design-webapp-routes] - Route list (GET/POST) and simple sitemap for manual testing and maintenance.
- Reference: [OpenSSH sshd_config]
- Reference: [dnsmasq]
- Reference: [oauth2-proxy]
- Reference: [Azure App Service Easy Auth: user identities]
- Reference: [HTMX]
- Reference: [Pico.css]

[design-webapp-routes]: ./design-webapp-routes.md
[OpenSSH sshd_config]: https://man.openbsd.org/sshd_config
[dnsmasq]: https://thekelleys.org.uk/dnsmasq/doc.html
[oauth2-proxy]: https://oauth2-proxy.github.io/oauth2-proxy/
[Azure App Service Easy Auth: user identities]: https://learn.microsoft.com/en-us/azure/app-service/configure-authentication-user-identities
[HTMX]: https://htmx.org/
[Pico.css]: https://picocss.com/
