---
id: task-20260104f-permissions-enforcement
title: Permissions: SSH keys, DNS, admin
status: done
updated: 2026-01-04T12:30:22Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Permissions: SSH keys, DNS, admin

## Background

We have role definitions (admin/user) but permissions are not fully defined and/or enforced across routes.

This work is tracked in [design-roadmap].

## Goal

Define and implement route-level permissions for the web app, based on roles.

## Requirements

Permissions matrix (source of truth: [design-roadmap]):

|Area|`admin`|`user`|
|-|-|-|
|SSH public keys (`/ssh/keys`)|Manage all users’ SSH public keys|Manage own SSH public keys only|
|DNS alias rules (`/admin/dns`)|Manage DNS alias rules|No access (cannot view or change)|
|Admin dashboard (`/admin`)|Manage the home page content etc.|No access|

Implementation scope (agreed for 20260104f):

- `GET /admin`
  - Admin dashboard top page (links to sub pages)
- `GET /admin/users`
  - List users who have registered at least one key
  - UI: show both `email` (primary) and `userID`
  - User list definition: derive from current key owners (if a user deletes all keys, they disappear from the list; enable/disable alone does not remove them)
  - No `POST` in this scope
- `GET /admin/keys`
  - Show all users’ keys (key list)
  - UI: columns `Owner`, `Fingerprint`, `Status`, `Created`
  - No `POST` in this scope
- `GET /admin/dns`
  - Migrate from `/dns`
  - `POST` operations exist under `/admin/dns/*` (same routes and behavior as the previous `POST /dns/*`)

General constraints:

- Pagination: out of scope (separate task)
- Search: out of scope (separate task)

Routing compatibility:

- `/dns` is removed (no redirect)

## Non-goals

- Building a full RBAC system.
- Per-resource authorization beyond the route-level rules above.
- Pagination.
- Search.

## Plan & Checklist

- [x] 1) Update [design-app-http] to reflect per-route access rules
- [x] 2) Add admin pages and routing:
  - [x] `GET /admin`
  - [x] `GET /admin/users` (GET only)
  - [x] `GET /admin/keys` (GET only)
  - [x] `GET /admin/dns` and `POST /admin/dns/*`
  - [x] Remove `/dns` (no redirect)
- [x] 3) Implement route-level authorization checks per the matrix
- [x] 4) Add/adjust unit tests for authorization behavior
- [x] 5) Verify locally (test mode)

## Progress

- 2026-01-04T11:17:27Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

- 2026-01-04T11:32:39Z
  - Decided admin route structure and removed `/dns` in favor of `/admin/dns` (no compatibility)

- 2026-01-04T11:40:29Z
  - Recorded the agreed implementation scope for 20260104f (no pagination/search; `/dns` removed; admin pages split by endpoint)

- 2026-01-04T11:48:20Z
  - Clarified UI and data rules for `/admin/users` (show both email+userID; list derived from current key owners)
  - Confirmed DNS POST operations move from `/dns/*` to `/admin/dns/*` as-is

- 2026-01-04T12:05:06Z
  - Implemented `/admin/*` pages and admin-only enforcement
  - Migrated DNS management from `/dns` to `/admin/dns` and removed `/dns`
  - Updated unit tests and E2E to use `/admin/dns`

- 2026-01-04T12:30:22Z
  - Corrected `/admin/keys` behavior: it is a key list page (Owner/Fingerprint/Status/Created), not a users overview

## References

- [design-roadmap] - Development Roadmap
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-roadmap]: ../design/design-roadmap.md
[design-app-http]: ../design/design-app-http.md
