---
id: task-20260102d-webapp-observability
title: Improve web observability (logging and error context)
status: todo
updated: 2026-01-02T17:48:53Z
---
# Task: Improve web observability (logging and error context)

## Goal

Make operational debugging easier by ensuring server logs capture consistent, actionable context for failures while keeping user-facing messages simple.

## Scope

### In

- Normalize logging for key user actions (add/toggle/delete keys, add/delete aliases)
- Ensure errors include enough context (route/action, user dir ID, relevant identifiers)
- Ensure internal errors are logged when returning 4xx/5xx

### Out

- External telemetry systems (OpenTelemetry, tracing backends)
- PII-heavy logging (avoid logging raw public keys)

## Plan & Checklist

- [ ] Identify current logging gaps (silent template errors, best-effort renders, etc.)
- [ ] Decide a minimal logging schema (consistent prefixes/fields)
- [ ] Add logging for:
  - [ ] Auth decisions (without leaking sensitive headers)
  - [ ] Key operations (log fingerprint, not key material)
  - [ ] DNS operations (log source/destination)
- [ ] Add regression tests if appropriate (or keep as manual verification)
- [ ] Document expectations in a short note (optional)

## Progress

- 2026-01-02T17:48:53Z
  - Create task document
