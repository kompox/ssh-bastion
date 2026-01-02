---
id: task-20260102b-webapp-error-pages
title: Improve web UX for errors (no plain error pages)
status: done
updated: 2026-01-02T18:44:19Z
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

## References

- [design-overview] - Design overview document

[design-overview]: ../design/design-overview.md
