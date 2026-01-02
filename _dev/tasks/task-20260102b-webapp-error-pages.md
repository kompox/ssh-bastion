---
id: task-20260102b-webapp-error-pages
title: Improve web UX for errors (no plain error pages)
status: done
updated: 2026-01-02T20:07:00Z
---
# Task: Improve web UX for errors (no plain error pages)

## Goal

Remove “plain error text page” responses from user-facing flows and present errors consistently using the existing layout and pages.

## Scope

### In

- Key operations: enable/disable/delete
- DNS operations: delete
- Auth failure: missing headers (401)
- Common internal errors: template execution failures, storage read/write failures

### Out

- New pages, modals, or new UI patterns beyond simple inline error text
- Complex error taxonomy or localization

## Plan & Checklist

- [x] Inventory remaining `http.Error(...)` paths for user-facing routes
- [x] Decide consistent UX:
  - [x] For form submissions: re-render same page with inline error + preserved inputs
  - [x] For non-form actions (enable/disable/delete): redirect back with a visible error (or re-render page)
- [x] Implement error rendering for:
  - [x] Key enable/disable/delete
  - [x] DNS delete
  - [x] Auth 401 (show a helpful page and keep 401 status)
- [x] Add regression tests for representative failures
- [x] Update manual testing notes if needed

### Follow-ups captured

- [x] Align DNS delete missing behavior with keys (`400` not `303`)
- [x] Document expected HTTP statuses in routes design doc

## Progress

- 2026-01-02T17:48:53Z
  - Create task document

- 2026-01-02T18:11:28Z
  - Make user-facing auth 401 return HTML when appropriate
  - Reduce remaining plain `http.Error(...)` responses in web flows
  - Add regression tests and confirm `make test` passes
  - Fix a test compilation failure (duplicate `package auth` line)

- 2026-01-02T18:44:19Z
  - Align delete-key missing behavior with enable/disable (`400 Key not found`)
  - Manually verify curl cases behave as intended

- 2026-01-02T18:51:59Z
  - Make error messages more prominent with a page-wide flash banner

- 2026-01-02T19:37:05Z
  - Move flash banner under forms (better visibility in flows)
  - Remove flash title bar and add 4 kinds (success/error/warning/info)

- 2026-01-02T19:47:19Z
  - Manually verify flash banner placement and visibility in browser

- 2026-01-02T20:07:00Z
  - Align DNS delete missing behavior to return `400` (no silent `303` redirect)
  - Add regression tests for missing DNS alias delete
  - Document expected HTTP statuses in routes design doc

## References

- [design-overview] - Design overview document
- [design-webapp-routes] - Web app routes & sitemap

[design-overview]: ../design/design-overview.md
[design-webapp-routes]: ../design/design-webapp-routes.md
