---
id: task-20260104e-roles-admin-user
title: Roles: admin and user
status: done
updated: 2026-01-04T11:07:34Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Roles: admin and user

## Background

The app currently does not distinguish between admins and regular users. We want roles to control which features are accessible.

This work is tracked in [design-roadmap].

## Goal

Define roles and configuration behavior, and make the current user's role visible in the web UI.

Configuration (environment variables):

- `SSHBASTION_ROLE_ADMIN_IDS`: comma-separated list of admin user IDs
- `SSHBASTION_ROLE_DEFAULT`: default role for users not in the admin list (default: `user`)

Roles and intended capabilities:

|Role|Capabilities|
|-|-|
|`admin`|Full access to all users' keys and DNS aliases|
|`user`|Manage own SSH public keys only; no DNS access|

UI requirement:

- In the web UI header (top-right username display), show the current user's role (e.g. `alice (admin)` or `alice [admin]`).

## Non-goals

- Implementing detailed/individual resource protection logic (e.g. per-endpoint/per-resource permission enforcement).
- Building a full RBAC system.
- Changing auth provider behavior.

## Plan & Checklist

- [x] 1) Implement role config/env parsing (`SSHBASTION_ROLE_DEFAULT`, `SSHBASTION_ROLE_ADMIN_IDS`)
- [x] 2) Propagate role via auth middleware context
- [x] 3) Update web UI header: show role next to username (top-right)
- [x] 4) Document Roles/Permissions in [design-app-http]:
  - [x] Add Roles/Permissions section
  - [x] For each route, state access derived from [design-roadmap] (who can view/operate)
- [x] 5) Update design docs / README if needed

## Progress

- 2026-01-04T10:03:27Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

- 2026-01-04T10:08:49Z
  - Adjusted scope: no implementation in this task update; exclude detailed per-resource protection
  - Added UI requirement: show current user's role next to username

- 2026-01-04T10:20:17Z
  - Added doc-only requirement: record Roles/Permissions in [design-app-http] and annotate each route with access derived from [design-roadmap]

- 2026-01-04T10:28:07Z
  - Implemented roles config + propagation and enforced admin-only DNS/admin routes
  - Updated layout header to display role and hide DNS link for non-admins
  - Verified with `make test`

- 2026-01-04T10:57:56Z
  - Implemented role derivation (admin/user) and role display in the header (`email (role)`)
  - Verified behavior manually; unit tests passing (`make test`)
  - Note: authorization enforcement is not treated as completed in the current implementation

- 2026-01-04T11:07:34Z
  - Updated [design-app-http] with Roles/Permissions and per-route access
  - Marked this task as done

## References

- [design-roadmap] - Development Roadmap
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-roadmap]: ../design/design-roadmap.md
[design-app-http]: ../design/design-app-http.md
