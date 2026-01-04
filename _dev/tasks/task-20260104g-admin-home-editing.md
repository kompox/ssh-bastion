---
id: task-20260104g-admin-home-editing
title: Admin: Home page markdown editing
status: in-progress
updated: 2026-01-04T12:44:46Z
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
- [ ] 2) Add GET handler to read current markdown and render edit form
- [ ] 3) Add POST handler to save updated markdown
- [ ] 4) Add basic validation (e.g., file size limit)
- [ ] 5) Update templates and navigation
- [ ] 6) Add unit tests for new handlers
- [ ] 7) Verify locally (test mode with admin role)

## Progress

- 2026-01-04T12:39:35Z
  - Task created and moved to IN-PROGRESS in roadmap

- 2026-01-04T12:44:46Z
  - Decided route: `/admin/home`

## References

- [design-roadmap] - Development Roadmap
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-roadmap]: ../design/design-roadmap.md
[design-app-http]: ../design/design-app-http.md
