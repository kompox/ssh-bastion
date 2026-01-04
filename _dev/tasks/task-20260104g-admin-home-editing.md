---
id: task-20260104g-admin-home-editing
title: Admin: Home page markdown editing
status: done
updated: 2026-01-04T13:03:47Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Task: Admin: Home page markdown editing

## Background

The home page (`/`) currently reads markdown from `${SSHBASTION_DATA_DIR}/content/pages/home.md` and renders it. There is no UI to edit this content; admins must manually edit the file on disk or via kubectl/docker exec.

This work is tracked in [design-roadmap].

## Goal

Provide a web-based form on `/admin/home` for admins to edit the home page markdown content.

## Requirements

- `GET /admin/home`
  - Show current markdown content in a textarea
  - Access: `admin` only
- `POST /admin/home`
  - Save the updated markdown to `${SSHBASTION_DATA_DIR}/content/pages/home.md`
  - Validate basic constraints (e.g., file size limit)
  - Access: `admin` only
  - Success: redirect back to `/admin/home` with success message
  - Error: re-render form with error message

## Non-goals

- Markdown preview (out of scope; can be added later)
- Version control / history (out of scope)
- Multi-page CMS (only home page for now)

## Plan & Checklist

- [x] 1) Decide route: use `/admin/home`
- [x] 2) Add GET handler to read current markdown and render edit form
- [x] 3) Add POST handler to save updated markdown
- [x] 4) Add basic validation (e.g., file size limit)
- [x] 5) Update templates and navigation
- [x] 6) Add unit tests for new handlers
- [x] 7) Verify locally (test mode with admin role)

## Progress

- 2026-01-04T12:39:35Z
  - Task created and moved to IN-PROGRESS in roadmap

- 2026-01-04T12:44:46Z
  - Decided route: `/admin/home`

- 2026-01-04T12:53:35Z
  - Implemented `GET /admin/home` and `POST /admin/home` (admin-only) for editing home markdown
  - Added templates and admin dashboard link
  - Updated [design-app-http] and added unit tests

- 2026-01-04T13:03:47Z
  - Moved roadmap item to DONE and refreshed summaries

## References

- [design-roadmap] - Development Roadmap
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-roadmap]: ../design/design-roadmap.md
[design-app-http]: ../design/design-app-http.md
