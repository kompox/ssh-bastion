---
id: task-20260104d-dns-upstream-autodetect
title: DNS proxy - upstream autodetect from resolv.conf
status: done
updated: 2026-01-04T09:57:58Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: DNS proxy - upstream autodetect from resolv.conf

## Background

The DNS proxy currently requires an explicit upstream configuration path (flag/env defaulting). For good out-of-the-box behavior across Kubernetes and Docker, the DNS proxy should be able to select an upstream resolver automatically when `SSHBASTION_DNS_UPSTREAM` is not set.

The desired behavior has been documented in:

- [design-app-dns] (DNS proxy behavior)
- [design-containers] (container/runtime wiring)

## Goal

Implement upstream DNS selection logic so the DNS proxy works without explicit upstream configuration:

- Prefer `SSHBASTION_DNS_UPSTREAM` when set.
- Otherwise, auto-detect from `/etc/resolv.conf` (first `nameserver`, add port `:53`).
- IPv6: support bracket form (e.g. `[fd00::1]:53`).
- If no upstream can be determined, fail fast at startup with a clear error.

## Non-goals

- Changing DNS protocol behavior (still UDP).
- Adding caching.
- Changing how `sshd` is pointed at `127.0.0.1`.

## Spec summary

- Upstream selection precedence: `-dns-upstream` flag > `SSHBASTION_DNS_UPSTREAM` env > `/etc/resolv.conf` (first `nameserver`, add `:53`; IPv6 bracket form).
- Startup behavior: if DNS is enabled and no upstream can be determined, fail fast with a clear error.
- Developer helpers: `make run-test-mode` relies on autodetect (no manual `/etc/resolv.conf` parsing).

## Plan & Checklist

- [x] 1) Define upstream selection order
- [x] 2) Implement `/etc/resolv.conf` parsing helper
- [x] 3) Wire selection into `ssh-bastion serve` defaulting behavior
- [x] 4) Add unit tests for:
  - [x] IPv4 nameserver
  - [x] IPv6 nameserver
  - [x] Missing/empty resolv.conf
- [x] 5) Ensure docs and flags/env vars remain consistent
- [x] 6) Refactor `run-test-mode` target to align with autodetect

## Progress

- 2026-01-04T09:36:58Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

- 2026-01-04T09:43:52Z
  - Implemented upstream selection precedence: flag > env > `/etc/resolv.conf`
  - Added resolv.conf parsing + IPv6 bracket formatting
  - Added unit tests and verified with `make test`

- 2026-01-04T09:44:55Z
  - Only resolve upstream when DNS is enabled (HTTP-only mode does not require DNS config)

- 2026-01-04T09:55:00Z
  - Updated `make run-test-mode` to rely on upstream autodetect (no manual resolv.conf parsing)

## References

- [design-app-dns] - App DNS (in-process DNS proxy)
- [design-containers] - Containers (image + runtime topology)
- [design-roadmap] - Development Roadmap

[design-app-dns]: ../design/design-app-dns.md
[design-containers]: ../design/design-containers.md
[design-roadmap]: ../design/design-roadmap.md
