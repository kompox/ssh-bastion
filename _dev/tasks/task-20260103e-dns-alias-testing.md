---
id: task-20260103e-dns-alias-testing
title: DNS alias testing (E2E)
status: done
updated: 2026-01-04T06:51:36Z
---

# Task: DNS alias testing (E2E)

## Goal

Provide an end-to-end test for DNS alias functionality using the local docker-compose topology.

## Scope

### In

- Add an E2E scenario that:
  - Creates a DNS alias via the web API.
  - Verifies DNS resolution through the in-process DNS proxy.
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
  - Verify `aliases.json` contains the alias.
  - Resolve check:
    - Query A for `<source>` and expect an A answer with owner name matching `<source>`.
  - Delete alias via `POST /dns/{source}/delete` (expects 303).
  - Verify `aliases.json` no longer contains the alias.
  - Negative check:
    - Query A for `<source>` with recursion disabled and assert the DNS proxy does not return any address answers.
    - This intentionally avoids depending on upstream behavior (e.g. `SERVFAIL` vs `NXDOMAIN`).

## Plan & Checklist

- [x] Identify current coverage gaps (API create/delete vs DNS resolution)
- [x] Implement/adjust E2E test(s) for DNS alias add/delete + resolution
- [x] Ensure cleanup is reliable (idempotent if rerun)
- [x] Run `make e2e` locally and confirm green
- [x] Add E2E scenario to reproduce ProxyJump resolution issue (sshd via ProxyJump)
- [x] Update relevant design docs if conventions change
- [x] Move roadmap item to DONE when complete

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

- 2026-01-03T18:46:25Z
  - Added an XFAIL E2E scenario to reproduce the ProxyJump resolution issue via sshd.
  - Script: `e2e/scripts/e2e-30-xfail-cname-a-proxyjump.sh`
  - Note: this was excluded from `make e2e` while the issue was unresolved.

- 2026-01-03T19:17:12Z
  - Documented E2E script naming conventions (`e2e-NN-xfail-*`, `e2e-NN-skip-*`) in [design-e2e-testing].

- 2026-01-04T06:51:36Z
  - Updated E2E docs and harness for the in-process DNS proxy (removed dnsmasq-specific checks).
  - ProxyJump scenario is now stable; renamed and enabled under `make e2e`.
  - Script: `e2e/scripts/e2e-30-proxyjump-dns-alias.sh`

## References

- [design-roadmap] - Development Roadmap
- [design-e2e-testing] - E2E testing conventions
- [task-20260103f-container-rebuild-glibc] - Container rebuild with glibc (CNAME->A issue)

[design-roadmap]: ../design/design-roadmap.md
[design-e2e-testing]: ../design/design-e2e-testing.md
[task-20260103f-container-rebuild-glibc]: task-20260103f-container-rebuild-glibc.md
