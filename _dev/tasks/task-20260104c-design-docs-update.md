---
id: task-20260104c-design-docs-update
title: Docs - update design docs (DNS proxy, /ssh routes, rename HTTP doc)
status: done
updated: 2026-01-04T09:32:12Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Docs - update design docs (DNS proxy, /ssh routes, rename HTTP doc)

## Background

The project has moved away from the `dnsmasq` sidecar approach (see completed work in `task-20260104a-dns-query-rewrite-proxy`).

However, several design documents still describe a topology and behaviors based on a `dnsmasq` sidecar (compose/Kubernetes, resolver rewrites, dnsmasq config generation/reload).

## Goal

- Eliminate mentions of a `dnsmasq` sidecar as an active/target design in `_dev/design/*`.
- Update the design docs to reflect the current direction: in-process DNS query rewrite proxy (`internal/dnsproxy`) and `ssh-bastion serve` service selection.
- Update design docs for the routing change: `/keys` -> `/ssh/keys`.

## Non-goals

- No code changes in this task.
- No changes to runtime behavior.
- No new architecture beyond aligning docs with current implementation decisions.

## Scope

### In

- Update design docs to remove/replace `dnsmasq` sidecar references.
- Ensure the docs consistently describe:
  - DNS aliasing implemented by the in-process DNS proxy.
  - Service topology without a `dnsmasq` container.
  - Any remaining “two containers” topology references updated accordingly.

### Out

- Rewriting unrelated sections.
- Changing the roadmap beyond moving this task between states.

## Plan & Checklist

- [x] 1) Inventory remaining `dnsmasq` mentions
  - [x] `grep -R dnsmasq _dev/design/` and classify each mention as: remove / rewrite / historical note

- [x] 2) Update design docs for `/ssh/keys`
  - [x] Replace any `http://.../keys` and `POST /keys` references with `/ssh/keys`
  - [x] Ensure route/testing docs reference `/ssh` and `/ssh/keys/...` consistently

- [x] 3) Update `design-overview`
  - [x] Replace the “dnsmasq” container/topology description with the DNS proxy approach
  - [x] Remove `dnsmasq` config generation/reload strategy

- [x] 4) Update `design-containers`
  - [x] Remove/replace sections describing `dnsmasq` as a sidecar service
  - [x] Update DNS routing/resolver notes to match the in-process DNS proxy

- [x] 5) Update testing docs that reference `dnsmasq`
  - [x] Ensure app/testing docs no longer reference `dnsmasq` config generation

- [x] 6) Create `design-app-dns.md` (DNS server implementation details)
  - [x] Add DNS proxy implementation details (listeners, rewrite logic, upstream forwarding, response rewriting)
  - [x] Document on-disk state (alias registry files) and interaction with the web app

- [x] 7) Record the rationale for dropping dnsmasq (in `design-app-dns.md`)
  - [x] Originally: dnsmasq as a CNAME-only DNS server; 3-container architecture
  - [x] Problem: CNAME -> A resolution not automatically performed
  - [x] Confirmed: neither musl nor glibc stub resolvers resolve CNAME -> A in this setup
  - [x] Decision: drop the CNAME-server approach; implement DNS proxy; move to 2-container architecture

- [x] 8) Rename routes doc to `design-app-http.md`
  - [x] Create the new doc (same content + updated naming)
  - [x] Update all references to point at `design-app-http`

- [x] 9) Validation
  - [x] Confirm design docs no longer describe `dnsmasq` as an active sidecar design
  - [x] Keep any unavoidable historical mentions explicitly labeled as historical

- [x] 10) Editorial pass (read-through)
  - [x] For each touched design doc, read the full document and adjust section structure so it reads naturally
  - [x] Remove duplicated explanations and consolidate fragmented content into a single appropriate section

## Progress

- 2026-01-04T08:35:40Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

- 2026-01-04T09:09:31Z
  - Updated design docs to remove DNS sidecar assumptions and align to the in-process DNS proxy
  - Updated remaining routing references to `/ssh` and `/ssh/keys`
  - Added `design-app-dns.md` and renamed routes doc to `design-app-http.md`

- 2026-01-04T09:32:12Z
  - Documented DNS proxy upstream autodetect behavior (`SSHBASTION_DNS_UPSTREAM` or `/etc/resolv.conf`) in design docs
  - Normalized `assistedBy` across design docs and cleaned up roadmap formatting

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-app-testing] - App testing design doc
- [design-app-http] - App HTTP (Routes & Sitemap)
- [task-20260104a-dns-query-rewrite-proxy] - DNS alias: query rewrite proxy

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-app-testing]: ../design/design-app-testing.md
[design-app-http]: ../design/design-app-http.md
[task-20260104a-dns-query-rewrite-proxy]: task-20260104a-dns-query-rewrite-proxy.md
