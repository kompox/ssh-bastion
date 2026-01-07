---
id: task-20260106a-ssh-forwarding-hardening
title: Security - SSH forwarding hardening
status: done
updated: 2026-01-07T02:30:25Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Security - SSH forwarding hardening

## Goal

Restrict sshd TCP port forwarding to configured targets only (block arbitrary destination IP/ports) while supporting non-22 tunnels (e.g., DB access).

## Non-Goals

- Providing shell access on the bastion
- Per-user / per-role target allowlists (global allowlist only)
- Building a general-purpose service discovery system (this task is about an allowlist + enforcement)
- Adding a restart control to the admin UI (decide UX/ops in a separate task)

## Plan & Checklist

- [x] Finalize spec in this task file (no implementation before this)
- [x] Review current sshd container configuration and how it loads/reloads sshd_config
- [x] Decide configuration persistence (store global mode under `${SSHBASTION_DATA_DIR}`)
- [x] Decide target storage format and location under `${SSHBASTION_DATA_DIR}`
- [x] Update route list in `_dev/design/design-app-http.md` (add `/admin/targets` endpoints)
- [x] Implement target registry CRUD (storage layer)
- [x] Implement admin UI:
  - [x] `GET /admin/targets` list page
  - [x] Guided add form (no textarea)
  - [x] `POST /admin/targets/add`
  - [x] `POST /admin/targets/{rule}/enable`
  - [x] `POST /admin/targets/{rule}/disable`
  - [x] `POST /admin/targets/{rule}/delete`
  - [x] `POST /admin/targets/mode`
- [x] Implement sshd_config generation using `PermitOpen` allowlist
- [x] Implement safe reload/restart behavior for sshd when config changes
- [x] Add unit tests for parsing/validation
- [x] Add E2E/integration tests to verify forwarding is denied/allowed as expected
- [x] Update design docs and operator docs as needed

## Specification

Specification is maintained in [design-ssh-forwarding].

## Progress

- 2026-01-06T22:04:07Z

  - Created task and moved roadmap item to IN-PROGRESS

- 2026-01-06T22:15:04Z
  - Updated spec: guided forms UI, accept `PermitOpen` rule strings, and support enable/disable/remove

- 2026-01-06T22:17:40Z
  - Updated spec: allow `any`/`none` and `*` wildcards; added explicit anti-injection validation requirements

- 2026-01-06T22:23:14Z
  - Updated spec: add global mode switch (`any`/`none`/`custom`); restrict custom rules to `host:port` only

- 2026-01-06T22:27:45Z
  - Updated spec: persist the global mode under `${SSHBASTION_DATA_DIR}` for operational switching

- 2026-01-06T22:35:22Z
  - Updated spec: polling-based propagation; support reload and restart triggers for operations

- 2026-01-06T22:37:12Z
  - Updated spec: set polling default to 5s; clarified restart depends on K8s/Compose restart configuration

- 2026-01-06T22:47:34Z
  - Decided /data layout: `${SSHBASTION_DATA_DIR}/ssh/forwarding.json` (mode/targets/restartGeneration)

- 2026-01-06T22:54:16Z
  - Task list update: moved inlined specification to design doc reference; added checklist item for updating design-app-http route list

- 2026-01-06T22:57:50Z
  - Normalized reference style: in-body reference uses `[design-ssh-forwarding]` and References display matches other task files

- 2026-01-06T22:59:14Z
  - Normalized References section to match task reference-label style

- 2026-01-06T23:43:09Z
  - Implemented /admin/targets (mode + targets CRUD), forwarding.json registry + validation, sshd PermitOpen generation + polling reload/restart trigger, and updated design-app-http routes; verified make test

- 2026-01-07T01:06:38Z
  - Added E2E scenario to verify sshd port forwarding allow/deny behavior (PermitOpen)

- 2026-01-07T02:30:25Z
  - Marked task DONE (no further doc updates needed for this change)

## References

- [design-ssh-forwarding] - SSH forwarding and sshd process lifecycle
- [design-app-http] - Web app routes
- [design-roadmap] - Development Roadmap
- [design-containers] - Containers (image + runtime topology)
- [design-overview] - Design overview

[design-ssh-forwarding]: ../design/design-ssh-forwarding.md
[design-app-http]: ../design/design-app-http.md
[design-roadmap]: ../design/design-roadmap.md
[design-containers]: ../design/design-containers.md
[design-overview]: ../design/design-overview.md
