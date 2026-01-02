---
id: task-20260102a-webapp
title: Implement ssh-bastion web app (keys + DNS)
status: stable
updated: 2026-01-02T16:58:38Z
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

- [x] Confirm MVP scope from design overview
  - [x] Pages: SSH Public Keys / DNS Aliases
  - [x] Auth: return 401 if user ID or email header missing/empty

- [x] Define public surfaces (routes + templates)
  - [x] Routes for keys CRUD (list/add/enable-disable/delete)
  - [x] Routes for alias CRUD (list/add/delete)
  - [x] HTMX partials (post-redirect-get or swap fragments)

- [x] Implement auth middleware (trusted headers)
  - [x] Env vars: `SSHBASTION_AUTH_MODE`, `SSHBASTION_AUTH_USER_ID_HEADER`, `SSHBASTION_AUTH_EMAIL_HEADER`
  - [x] Derive stable storage key from user ID header
  - [x] Store by `UUIDv3(namespace, normalize(userId))` per design

- [x] Implement storage primitives
  - [x] Env var: `SSHBASTION_DATA_DIR` (non-empty)
  - [x] Directory layout creation on demand
  - [x] Atomic write helper (temp + rename in same dir)

- [x] Implement SSH public key registry
  - [x] Parse submitted public key (reject unsupported/invalid)
  - [x] Compute canonical fingerprint (SHA256 as OpenSSH shows)
  - [x] Persist per-user key material + metadata
  - [x] Enable/disable and delete operations

- [x] Implement DNS alias registry
  - [x] Validate alias inputs (FQDN-like; simple sanity checks)
  - [x] Persist aliases in a single source-of-truth file
  - [x] Generate dnsmasq config using `cname=` directives

- [x] Implement render/generation outputs
  - [x] Render `authorized_keys/jump` by aggregating enabled keys
  - [x] Render `dns/dnsmasq.d/generated.conf` from alias rules
  - [x] Ensure generation is idempotent and safe on concurrent writes

- [x] Implement UI pages (minimal)
  - [x] SSH Public Keys page (list/add/toggle/delete)
  - [x] DNS Aliases page (list/add/delete)
  - [x] Display user email from header (do not store by email)
  - [x] Fix template name collision that prevented DNS page rendering

- [x] Testing + manual verification
  - [x] Unit tests: key parsing + fingerprinting + atomic writes
  - [x] Unit tests: alias validation + dnsmasq conf generation
  - [x] Fix action URLs for fingerprints containing `/` (URL-encode in template + decode in handler)
  - [x] Add regression test for URL-encoded fingerprint actions
  - [x] Render Add Key/Add Alias errors inline (avoid plain error pages)
  - [x] Add regression tests for inline form errors
  - [x] Manual run notes (how to set headers + data dir for local testing)

## Progress

- 2026-01-02T11:03:00Z
  - Create task document

- 2026-01-02T12:03:07Z
  - Align scope/checklist with design overview (trusted-header auth, keys, DNS)

- 2026-01-02T12:23:00Z
  - Complete initial implementation
  - Implement config, auth middleware, storage primitives
  - Implement SSH key registry with fingerprinting
  - Implement DNS alias registry with dnsmasq config generation
  - Create HTML templates with Pico.css and HTMX
  - Implement web server with all routes
  - Write unit tests for storage, keys, and DNS packages
  - All tests passing

- 2026-01-02T13:22:00Z
  - Add test mode for browser testing
  - Add `SSHBASTION_AUTH_OVERRIDE_USER_ID` and `SSHBASTION_AUTH_OVERRIDE_EMAIL` environment variables
  - When both are set, app ignores request headers and uses override values
  - Update design doc and manual testing documentation
  - Useful for local development and integration testing

- 2026-01-02T16:24:42Z
  - Fix key Disable/Delete actions for fingerprints containing `/` by URL-encoding fingerprints in templates and decoding in handlers
  - Add regression test for URL-encoded fingerprint action URLs
  - Run full test suite (passing)

- 2026-01-02T16:58:38Z
  - Fix DNS page rendering by removing shared template name collisions
  - Improve UX: render Add Alias/Add Key errors inline and preserve form inputs
  - Add regression tests for DNS/keys inline error rendering
  - Run full test suite (passing)

## References

- [design-overview] - Design overview document

[design-overview]: ../design/design-overview.md
