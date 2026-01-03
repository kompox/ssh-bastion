---
id: task-20260103f-container-rebuild-glibc
title: Container rebuild with glibc (CNAME->A issue)
status: in-progress
updated: 2026-01-03T17:55:31Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Container rebuild with glibc (CNAME->A issue)

## Background

In the current local docker-compose topology, DNS aliasing is implemented via dnsmasq `cname=...`.

Observed issue:

- When an alias is configured (e.g. `hoge.local -> github.com`) and the SSH ProxyJump target is `git@hoge.local`, the jump-side sshd sometimes fails with `Name has no usable address`.
- After `git@github.com` is resolved once, `git@hoge.local` starts working.

Working hypothesis:

- The sshd container's resolver behavior (Alpine/musl) does not reliably chase CNAME -> A/AAAA in this environment.
- Rebuilding the image on a glibc-based distro is likely the most direct path to correct resolver behavior.

## Goal

- Make `git@<alias>` work reliably when the alias is a CNAME to an external name.
- Use a glibc-based container image to ensure CNAME -> A resolution behaves as expected for sshd.

## Scope

### In

- Confirm and document a reproducible scenario for the failure (see checklist).
- Switch base image to a glibc-based distribution (e.g. `debian:stable-slim` or Ubuntu).
- Validate that the reproduction scenario no longer fails.

### Out

- Replacing dnsmasq with a different DNS server.
- Adding new UX or web endpoints.

## Plan & Checklist

- [ ] Document the minimal reproduction steps and expected/actual behavior.
- [ ] Add/adjust E2E coverage that reproduces the failure deterministically (or as a stable flake repro).
- [ ] Rebuild container image with glibc base.
- [ ] Re-run E2E scenario(s) and confirm the failure is gone.
- [ ] Update design docs if the container/base-image decision changes other assumptions.

## Progress

- 2026-01-03T17:55:31Z
  - Task created

## References

- [design-roadmap] - Development Roadmap
- [design-containers] - Containers (image + runtime topology)
- [design-e2e-testing] - E2E / integration testing (docker-compose)
- [task-20260103e-dns-alias-testing] - DNS alias testing (E2E)

[design-roadmap]: ../design/design-roadmap.md
[design-containers]: ../design/design-containers.md
[design-e2e-testing]: ../design/design-e2e-testing.md
[task-20260103e-dns-alias-testing]: task-20260103e-dns-alias-testing.md
