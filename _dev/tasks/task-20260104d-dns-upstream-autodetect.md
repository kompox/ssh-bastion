---
id: task-20260104d-dns-upstream-autodetect
title: DNS proxy - upstream autodetect from resolv.conf
status: in-progress
updated: 2026-01-04T09:38:25Z
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

## Plan & Checklist

- [ ] 1) Define upstream selection order
- [ ] 2) Implement `/etc/resolv.conf` parsing helper
- [ ] 3) Wire selection into `ssh-bastion serve` defaulting behavior
- [ ] 4) Add unit tests for:
  - [ ] IPv4 nameserver
  - [ ] IPv6 nameserver
  - [ ] Missing/empty resolv.conf
- [ ] 5) Ensure docs and flags/env vars remain consistent

## Progress

- 2026-01-04T09:36:58Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

## References

- [design-app-dns] - App DNS (in-process DNS proxy)
- [design-containers] - Containers (image + runtime topology)
- [design-roadmap] - Development Roadmap

[design-app-dns]: ../design/design-app-dns.md
[design-containers]: ../design/design-containers.md
[design-roadmap]: ../design/design-roadmap.md
