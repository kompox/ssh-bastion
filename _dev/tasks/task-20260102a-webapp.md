---
id: task-20260102a-webapp
title: Implement ssh-bastion web app (keys + DNS)
status: stable
updated: 2026-01-02T12:03:07Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Task: Implement ssh-bastion web app (keys + DNS)

## Goal

Implement the `ssh-bastion web` executable described in the design overview:

- Header-based auth (trust external auth proxy; no direct OIDC/OAuth implementation)
- UI + API to manage SSH public keys (per-user)
- UI + API to manage bastion-local DNS alias rules
- File-based persistence + atomic writes
- Render outputs consumed by sshd/dnsmasq (authorized_keys + dnsmasq conf)

## Spec Summary

- Language: Go
- UI: server-rendered HTML + HTMX
- CSS: Pico.css
- Auth: trusted headers (Azure Easy Auth / oauth2-proxy), configurable header names
- Storage: file-based, simple auditable formats, atomic writes

## Scope

### In

- HTTP server for UI + API
- Authentication + request identity derived from headers
- Data model + persistence for:
  - SSH public keys
  - DNS alias rules
- Rendering/generation of:
  - `${SSHBASTION_DATA_DIR}/authorized_keys/jump`
  - `${SSHBASTION_DATA_DIR}/dns/dnsmasq.d/generated.conf`

### Out

- Container/Dockerfile work
- Kubernetes manifests / Helm chart
- sshd and dnsmasq runtime wiring (signals/reload) beyond file generation

## Plan & Checklist

- [ ] Confirm MVP scope from design overview
  - [ ] Pages: SSH Public Keys / DNS Aliases
  - [ ] Auth: return 401 if user ID or email header missing/empty

- [ ] Define public surfaces (routes + templates)
  - [ ] Routes for keys CRUD (list/add/enable-disable/delete)
  - [ ] Routes for alias CRUD (list/add/delete)
  - [ ] HTMX partials (post-redirect-get or swap fragments)

- [ ] Implement auth middleware (trusted headers)
  - [ ] Env vars: `SSHBASTION_AUTH_MODE`, `SSHBASTION_AUTH_USER_ID_HEADER`, `SSHBASTION_AUTH_EMAIL_HEADER`
  - [ ] Derive stable storage key from user ID header
  - [ ] Store by `UUIDv3(namespace, normalize(userId))` per design

- [ ] Implement storage primitives
  - [ ] Env var: `SSHBASTION_DATA_DIR` (non-empty)
  - [ ] Directory layout creation on demand
  - [ ] Atomic write helper (temp + rename in same dir)

- [ ] Implement SSH public key registry
  - [ ] Parse submitted public key (reject unsupported/invalid)
  - [ ] Compute canonical fingerprint (SHA256 as OpenSSH shows)
  - [ ] Persist per-user key material + metadata
  - [ ] Enable/disable and delete operations

- [ ] Implement DNS alias registry
  - [ ] Validate alias inputs (FQDN-like; simple sanity checks)
  - [ ] Persist aliases in a single source-of-truth file
  - [ ] Generate dnsmasq config using `cname=` directives

- [ ] Implement render/generation outputs
  - [ ] Render `authorized_keys/jump` by aggregating enabled keys
  - [ ] Render `dns/dnsmasq.d/generated.conf` from alias rules
  - [ ] Ensure generation is idempotent and safe on concurrent writes

- [ ] Implement UI pages (minimal)
  - [ ] SSH Public Keys page (list/add/toggle/delete)
  - [ ] DNS Aliases page (list/add/delete)
  - [ ] Display user email from header (do not store by email)

- [ ] Testing + manual verification
  - [ ] Unit tests: key parsing + fingerprinting + atomic writes
  - [ ] Unit tests: alias validation + dnsmasq conf generation
  - [ ] Manual run notes (how to set headers + data dir for local testing)

## Progress

- 2026-01-02T11:03:00Z
  - Create task document

- 2026-01-02T12:03:07Z
  - Align scope/checklist with design overview (trusted-header auth, keys, DNS)

## References

- [design-overview] - Design overview document

[design-overview]: ../design/design-overview.md
