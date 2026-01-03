---
id: task-20260103e-dns-alias-testing
title: DNS alias testing (E2E)
status: in-progress
updated: 2026-01-03T18:00:03Z
---

# Task: DNS alias testing (E2E)

## Goal

Provide an end-to-end test for DNS alias functionality using the local docker-compose topology.

## Scope

### In

- Add an E2E scenario that:
  - Creates a DNS alias via the web API.
  - Verifies DNS resolution through dnsmasq.
  - Deletes the alias and verifies it no longer resolves.
- Avoid relying on external tools like `dig`/`nslookup`.
- Ensure the scenario runs under `make e2e` and follows the E2E conventions.

### Out

- Performance/load testing.
- Testing external/public DNS.

## Spec (summary)

- Roadmap item: “DNS alias testing” in [design-roadmap].
- E2E conventions: numbered scripts under `e2e/scripts/`.

## Implementation (details)

- Go-based DNS query (no `dig`/`nslookup`): use `github.com/miekg/dns` to query `127.0.0.1:5353`.
- Test flow:
  - Add alias via `POST /dns` (expects 303).
  - Verify generated config contains `cname=<source>,<destination>`.
  - Restart dnsmasq (it does not auto-reload `conf-dir`).
  - Resolve check:
    - Query A for `<source>` and accept either a direct A or a CNAME to `<destination>` that resolves to A.
  - Delete alias via `POST /dns/{source}/delete` (expects 303).
  - Verify generated config no longer contains the CNAME line.
  - Restart dnsmasq again.
  - Negative check:
    - Query A for `<source>` with recursion disabled and assert dnsmasq no longer serves a local A/CNAME answer.
    - This intentionally avoids depending on upstream behavior (e.g. `SERVFAIL` vs `NXDOMAIN`).

## Plan & Checklist

- [x] Identify current coverage gaps (API create/delete vs DNS resolution)
- [x] Implement/adjust E2E test(s) for DNS alias add/delete + resolution
- [x] Ensure cleanup is reliable (idempotent if rerun)
- [x] Run `make e2e` locally and confirm green
- [ ] Add E2E scenario to reproduce CNAME->A resolution issue (sshd via ProxyJump)
- [ ] Update relevant design docs if conventions change
- [ ] Move roadmap item to DONE when complete

## Progress

- 2026-01-03T10:29:20Z
  - Task created

- 2026-01-03T10:38:01Z
  - Started: add/delete + DNS resolve/no-resolve E2E via Go (no dig/nslookup); verified with `make e2e`

- 2026-01-03T10:42:42Z
  - Align task doc formatting with other task files; add implementation details and references

- 2026-01-03T18:00:03Z
  - Investigated alias failure `hoge.local -> github.com` for `ssh -J ... git@hoge.local`.
  - Confirmed dnsmasq answers CNAME-only for `hoge.local` until `github.com` is cached; after caching, A may appear in the same response.
  - Observed sshd-side error: `Name has no usable address` when only CNAME is returned.
  - Decided to track container glibc rebuild separately: see [task-20260103f-container-rebuild-glibc].

## References

- [design-roadmap] - Development Roadmap
- [design-e2e-testing] - E2E testing conventions
- [task-20260103f-container-rebuild-glibc] - Container rebuild with glibc (CNAME->A issue)

[design-roadmap]: ../design/design-roadmap.md
[design-e2e-testing]: ../design/design-e2e-testing.md
[task-20260103f-container-rebuild-glibc]: task-20260103f-container-rebuild-glibc.md
