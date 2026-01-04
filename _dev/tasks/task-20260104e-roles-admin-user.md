---
id: task-20260104e-roles-admin-user
title: Roles: admin and user
status: in-progress
updated: 2026-01-04T10:21:30Z
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

- [ ] 1) Specify role resolution rules (user id -> role)
- [ ] 2) Specify config parsing + validation rules for role env vars
- [ ] 3) Specify authorization policy at a high level (what we eventually want to protect)
- [ ] 4) Update web UI spec: show role next to username (top-right)
- [ ] 5) Document Roles/Permissions in [design-app-http]:
  - [ ] Add Roles/Permissions section
  - [ ] For each route, state access derived from [design-roadmap] (who can view/operate)
- [ ] 6) Update design docs / README if needed

## Progress

- 2026-01-04T10:03:27Z
  - Task created (moved from roadmap TODO to IN-PROGRESS)

- 2026-01-04T10:08:49Z
  - Adjusted scope: no implementation in this task update; exclude detailed per-resource protection
  - Added UI requirement: show current user's role next to username

- 2026-01-04T10:20:17Z
  - Added doc-only requirement: record Roles/Permissions in [design-app-http] and annotate each route with access derived from [design-roadmap]

## References

- [design-roadmap] - Development Roadmap
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-roadmap]: ../design/design-roadmap.md
[design-app-http]: ../design/design-app-http.md
