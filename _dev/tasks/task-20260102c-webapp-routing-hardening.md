---
id: task-20260102c-webapp-routing-hardening
title: Harden web routing for path parameters
status: todo
updated: 2026-01-02T17:48:53Z
---
# Task: Harden web routing for path parameters

## Goal

Ensure all dynamic route parameters are safe and reliable when represented in URLs (e.g., values containing `/`, `+`, `%`, etc.), and that handlers decode them correctly.

## Motivation

We already fixed fingerprints containing `/` by URL-encoding in templates and decoding in handlers. This task generalizes the approach to other routes.

## Scope

### In

- Audit all routes using `{param}` path variables
- Ensure templates encode path variables
- Ensure handlers decode path variables (`url.PathUnescape`) and validate
- Add tests that cover problematic characters

### Out

- Changing route structure unless required
- Adding new user-facing features

## Plan & Checklist

- [ ] List all mux patterns with `{...}` params
- [ ] For each param:
  - [ ] Confirm it is encoded when constructing URLs in templates
  - [ ] Decode it in the handler and validate expectations
- [ ] Add tests for:
  - [ ] Key fingerprint (already exists; keep)
  - [ ] DNS delete route (`source`)
- [ ] Run full test suite

## Progress

- 2026-01-02T17:48:53Z
  - Create task document
