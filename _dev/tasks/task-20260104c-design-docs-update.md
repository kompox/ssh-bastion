---
id: task-20260104c-design-docs-update
title: Docs - update design docs (DNS proxy, /ssh routes, rename HTTP doc)
status: in-progress
updated: 2026-01-04T08:50:20Z
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

- [ ] 1) Inventory remaining `dnsmasq` mentions
  - [ ] `grep -R dnsmasq _dev/design/` and classify each mention as: remove / rewrite / historical note

- [ ] 2) Update design docs for `/ssh/keys`
  - [ ] Replace any `http://.../keys` and `POST /keys` references with `/ssh/keys`
  - [ ] Ensure route/testing docs reference `/ssh` and `/ssh/keys/...` consistently

- [ ] 3) Update `design-overview`
  - [ ] Replace the “dnsmasq” container/topology description with the DNS proxy approach
  - [ ] Remove `dnsmasq` config generation/reload strategy

- [ ] 4) Update `design-containers`
  - [ ] Remove/replace sections describing `dnsmasq` as a sidecar service
  - [ ] Update DNS routing/resolver notes to match the in-process DNS proxy

- [ ] 5) Update testing docs that reference `dnsmasq`
  - [ ] Ensure app/testing docs no longer reference `dnsmasq` config generation

- [ ] 6) Create `design-app-dns.md` (DNS server implementation details)
  - [ ] Add DNS proxy implementation details (listeners, rewrite logic, upstream forwarding, response rewriting)
  - [ ] Document on-disk state (alias registry files) and interaction with the web app

- [ ] 7) Record the rationale for dropping dnsmasq (in `design-app-dns.md`)
  - [ ] Originally: dnsmasq as a CNAME-only DNS server; 3-container architecture
  - [ ] Problem: CNAME -> A resolution not automatically performed
  - [ ] Confirmed: neither musl nor glibc stub resolvers resolve CNAME -> A in this setup
  - [ ] Decision: drop the CNAME-server approach; implement DNS proxy; move to 2-container architecture

- [ ] 8) Rename `design-webapp-routes.md` to `design-app-http.md`
  - [ ] Create the new doc (same content + updated naming)
  - [ ] Update all references to point at `design-app-http`

- [ ] 9) Validation
  - [ ] Confirm design docs no longer describe `dnsmasq` as an active sidecar design
  - [ ] Keep any unavoidable historical mentions explicitly labeled as historical

- [ ] 10) Editorial pass (read-through)
  - [ ] For each touched design doc, read the full document and adjust section structure so it reads naturally
  - [ ] Remove duplicated explanations and consolidate fragmented content into a single appropriate section

## Progress

- 2026-01-04T08:35:40Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-app-testing] - App testing design doc
- [design-webapp-routes] - Web App Routes & Sitemap
- [task-20260104a-dns-query-rewrite-proxy] - DNS alias: query rewrite proxy

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-app-testing]: ../design/design-app-testing.md
[design-webapp-routes]: ../design/design-webapp-routes.md
[task-20260104a-dns-query-rewrite-proxy]: task-20260104a-dns-query-rewrite-proxy.md
