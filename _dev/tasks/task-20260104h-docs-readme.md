---
id: task-20260104h-docs-readme
title: Docs: README.md
status: done
updated: 2026-01-04T21:22:30Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Docs: README.md

## Background

The repository has working dev/test workflows and a set of design docs under `_dev/design/`, but `README.md` is not yet organized as the primary entry point for new users and contributors.

This work is tracked in [design-roadmap].

## Goal

Update `README.md` so it provides a minimal, accurate entry point:

- Project overview and goals
- Quick links to key design docs and task files
- Minimal “how to run locally” notes (test mode + data dir)

## Requirements

- Keep the README concise (MVP documentation)
- Ensure examples match the current routes and commands
- Prefer `make` targets where available

## Non-goals

- Full end-user documentation
- Kubernetes/Helm docs (separate task)
- Detailed operational runbooks

## Plan & Checklist

- [x] 1) Review current `README.md` and identify gaps
- [x] 2) Add brief overview + goals
- [x] 3) Add quick links to design docs and recent task files
- [x] 4) Add minimal local run notes (test mode + data dir)
- [x] 5) Sanity-check commands/paths referenced

## Progress

- 2026-01-04T13:07:46Z
  - Task created and moved to IN-PROGRESS in roadmap

- 2026-01-04T13:15:46Z
  - Updated `README.md` with overview, quick links, local dev notes, and Mermaid diagram for Gitea SSH access in Kubernetes

- 2026-01-04T21:03:28Z
  - README.md: Added Features section highlighting shared public IP / LoadBalancer rule benefit, Gitea use case with DNS aliasing, Architecture diagram (gitea1/2/3 multi-site example), connection examples, Developer's guide with design docs links
  - LICENSE: Added MIT license
  - [design-overview]: Updated env var list and storage layout for consistency with implementation
  - [design-app-testing]: Canonicalized storage layout reference, updated routes and run instructions
  - Makefile: Added DATA_DIR and DNS_UPSTREAM variables for test-mode flexibility

## References

- [design-overview] - Kompox ssh-bastion design overview
- [design-roadmap] - Development Roadmap
- [design-app-testing] - App Testing (HTTP + DNS)

[design-overview]: ../design/design-overview.md
[design-roadmap]: ../design/design-roadmap.md
[design-app-testing]: ../design/design-app-testing.md
