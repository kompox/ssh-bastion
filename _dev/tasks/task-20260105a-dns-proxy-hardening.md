---
id: task-20260105a-dns-proxy-hardening
title: Security - DNS proxy hardening
status: done
updated: 2026-01-05T23:54:48Z
assistedBy: github/copilot (vscode) gpt-5.2
---

## Goal

Add configuration to restrict DNS proxy resolution to configured aliases only (return NXDOMAIN for all other queries).

## Non-Goals

- UI changes beyond what is required to support the setting
- Adding new DNS features (e.g., wildcard aliases) unless explicitly scoped

## Plan & Checklist

- [x] Finalize spec in this task file (no implementation before this)
- [x] Review current DNS proxy behavior for non-aliased queries
- [x] Decide configuration surface (env var) and defaults
- [x] Implement allowlist-only behavior when enabled
- [x] Add/update unit tests for NXDOMAIN behavior
- [x] Update design docs (at least: `_dev/design/design-app-dns.md`, `_dev/design/design-overview.md`, `_dev/design/design-containers.md`)
- [x] Update testing docs if needed (e.g. `_dev/design/design-app-testing.md`)
- [ ] Update `README.md` if needed

## Specification

- Configuration: environment variable `SSHBASTION_DNS_ALIASES_ONLY` (boolean)
- Default: `false` (preserve current behavior)
- When enabled (`true`):
	- If the query name (QNAME) is not a configured alias, return NXDOMAIN.
	- If the query name is a configured alias, resolve via the existing alias rewrite + upstream forwarding path.
	- For non-A/AAAA query types, keep the current behavior of not forwarding upstream (the proxy is intended for SSH-style A/AAAA resolution).

### Admin UI (status display only)

- The `/admin/dns` page should indicate whether `SSHBASTION_DNS_ALIASES_ONLY` is enabled.
- Display: page title `DNS Aliases (aliases-only)` when enabled; otherwise `DNS Aliases (unrestricted)`.
- This is display-only; the admin UI must not provide a toggle.

## Progress

- 2026-01-05T23:10:08Z
	- Moved "Security: DNS proxy hardening" to IN-PROGRESS and created this task file

- 2026-01-05T23:14:05Z
	- Documented initial spec: use env var `SSHBASTION_DNS_ALIASES_ONLY` to toggle aliases-only behavior

- 2026-01-05T23:22:22Z
	- Added “finalize spec before coding” gate, expanded doc-update checklist, and specified admin UI status display (no toggle)

- 2026-01-05T23:29:07Z
	- Implemented `SSHBASTION_DNS_ALIASES_ONLY` config parsing and aliases-only DNS behavior (NXDOMAIN for non-aliased A/AAAA)
	- Admin UI: `/admin/dns` title shows restriction status (`aliases-only` vs `unrestricted`)
	- Verified `make test`

- 2026-01-05T23:31:46Z
	- Updated design docs to include `SSHBASTION_DNS_ALIASES_ONLY` in configuration lists and describe its behavior

- 2026-01-05T23:35:24Z
	- Updated checklist to reflect completed implementation, tests, and design-doc updates

- 2026-01-05T23:54:48Z
	- Marked task as done

## References

- [_dev/design/design-app-dns.md](../design/design-app-dns.md)
- [_dev/design/design-overview.md](../design/design-overview.md)
- [internal/dnsproxy/server.go](../../internal/dnsproxy/server.go)
- [internal/dns/registry.go](../../internal/dns/registry.go)
- [internal/config/config.go](../../internal/config/config.go)
