---
id: task-20260102b-webapp-error-pages
title: Improve web UX for errors (no plain error pages)
status: todo
updated: 2026-01-02T17:48:53Z
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

- [ ] Inventory remaining `http.Error(...)` paths for user-facing routes
- [ ] Decide consistent UX:
  - [ ] For form submissions: re-render same page with inline error + preserved inputs
  - [ ] For non-form actions (enable/disable/delete): redirect back with a visible error (or re-render page)
- [ ] Implement error rendering for:
  - [ ] Key enable/disable/delete
  - [ ] DNS delete
  - [ ] Auth 401 (show a helpful page and keep 401 status)
- [ ] Add regression tests for representative failures
- [ ] Update manual testing notes if needed

## Progress

- 2026-01-02T17:48:53Z
  - Create task document
