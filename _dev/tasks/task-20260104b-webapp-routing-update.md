---
id: task-20260104b-webapp-routing-update
title: Web app site map and routing update
status: done
updated: 2026-01-04T08:31:10Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Web app site map and routing update

## Background

The web app currently serves SSH key and DNS alias management pages, but the site map and routing are not yet aligned with the intended structure.

## Goal

Update the web app routes to match the roadmap site map:

- `/` → Home (show markdown page authored by admin)
- `/ssh` → SSH key management
- `/dns` → DNS alias management (no change in this task)
- `/admin` → Admin dashboard (future use)

Additionally, move existing SSH key management endpoints under `/ssh` (key operations under `/ssh/keys/...`).

Define where admin-authored markdown content is stored under `/data` and how the server reads it.

## Scope

### In

- Implement routing so the paths above are available.
- Keep existing SSH key management and DNS alias management functionality working.
- Move the existing SSH key management endpoints under `/ssh`.
- Define a stable, extensible on-disk layout under `/data` for admin-authored markdown content (starting with the home page).

### Out

- Building a full admin dashboard UI (beyond a minimal placeholder).
- Adding new auth providers or role models beyond what exists today.
- Adding additional pages beyond the site map described above.
- Enforcing admin-only access for `/dns` (blocked until roles/permissions are implemented).
- Adding REST/JSON endpoints under `/api/*` in parallel with HTMX endpoints (defer until needed).

## Spec: Admin-authored markdown storage

This task does not implement the Web UI editor yet, but it must define the storage location and read behavior so future Web UI work can write to the same place.

### Storage base

- Base directory: `${SSHBASTION_DATA_DIR}/content/`
  - `${SSHBASTION_DATA_DIR}` defaults to `/data`.
- All admin-authored markdown files live under `${SSHBASTION_DATA_DIR}/content/pages/`.

### Home

- File path: `${SSHBASTION_DATA_DIR}/content/pages/home.md`
- Encoding: UTF-8.
- Missing file behavior:
  - If the file does not exist, the home page should still render, but with a minimal placeholder message indicating it is not configured.
  - This avoids breaking the app on first boot.

### Extensibility (not limited to home)

- Additional customizable pages can be added as separate files under `${SSHBASTION_DATA_DIR}/content/pages/`.
- Naming convention (proposed): `${pageId}.md` where `pageId` is a stable identifier (e.g. `home`, `help`, `about`).
- The server must treat `pageId` as an identifier (not a raw path) to prevent path traversal; only resolve to a filename under `content/pages/`.

### Read behavior

- The app reads markdown from disk at request time (simplest behavior) or with a small in-memory cache keyed by file mtime.
  - The choice can be implementation-defined later, but the observable behavior must be: edits to the file become visible without changing the storage location.
- The renderer must be safe for untrusted content (no raw HTML injection). Exact rendering library is implementation-defined.

## Plan & Checklist

- [x] 1) Update routing documentation
  - [x] Update [design-webapp-routes] to reflect the new sitemap and paths

- [x] 2) Specify markdown storage under `/data`
  - [x] Adopt `${SSHBASTION_DATA_DIR}/content/pages/home.md` as the home source
  - [x] Define missing-file behavior and extensibility rules

- [x] 3) Move SSH key management under `/ssh`
  - [x] Change existing handlers/routes from `/keys` to `/ssh/keys` (and related redirects)
  - [x] Ensure templates still render correctly
  - [x] Update/adjust unit tests accordingly

- [x] 4) Add `/` home route
  - [x] Render an admin-authored markdown page
  - [x] Add basic tests for rendering

- [x] 5) Add `/admin` placeholder route
  - [x] Minimal handler that returns a stub page (future use)

- [x] 6) Verify
  - [x] `make test`

- [x] 7) Document API posture
  - [x] Keep HTMX (HTML) endpoints as the MVP; add `/api/*` only if a real REST/JSON consumer appears

- [x] 8) E2E follow-ups
  - [x] Update E2E scripts for routing changes (`/keys` -> `/ssh/keys`)
  - [x] Add best-effort cleanup for stale `ssh-bastion-e2e-*` docker networks
  - [x] `make e2e`

## Progress

- 2026-01-04T07:22:07Z
  - Task created and moved to IN-PROGRESS in roadmap

- 2026-01-04T07:25:01Z
  - Added doc update step and reference for `design-webapp-routes`

- 2026-01-04T07:28:19Z
  - Reordered and renumbered checklist; added policy to move SSH key endpoints under `/ssh/*`

- 2026-01-04T07:31:07Z
  - Dropped `/dns` changes from scope (admin-only requires roles/permissions)

- 2026-01-04T07:34:50Z
  - Defined `/data` markdown storage layout for home and future pages

- 2026-01-04T07:54:06Z
  - Noted MVP approach: HTMX first; `/api/*` deferred until needed

- 2026-01-04T08:03:59Z
  - Implemented routing updates (`/` home, `/ssh` keys, `/ssh/keys/...` key operations, `/admin` placeholder) and updated docs/templates/tests; verified `make test`

- 2026-01-04T08:21:30Z
  - Standardized home naming and source path (`${SSHBASTION_DATA_DIR}/content/pages/home.md`) across docs and templates; verified `make test`

- 2026-01-04T08:28:40Z
  - Fixed E2E scripts for routing changes: `POST /keys` -> `POST /ssh/keys`
  - Added best-effort cleanup for stale `ssh-bastion-e2e-*` docker networks (prevents Docker address pool exhaustion)
  - Verified `make e2e`

## References

- [design-roadmap] - Development roadmap
- [design-webapp-routes] - Web App Routes & Sitemap

[design-roadmap]: ../design/design-roadmap.md
[design-webapp-routes]: ../design/design-webapp-routes.md
