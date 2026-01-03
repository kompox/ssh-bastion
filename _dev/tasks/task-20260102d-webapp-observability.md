---
id: task-20260102d-webapp-observability
title: Improve web observability (logging and error context)
status: done
updated: 2026-01-03T04:31:37Z
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

## Spec

### Logging format

- Log lines use a simple key/value style suitable for grep.
- Each log includes a severity prefix (`ERROR`, `WARN`, `INFO`) and an `op` field where applicable.

### Log levels

- `SSHBASTION_LOG_LEVEL` controls verbosity.
  - Values: `error` | `warn` | `info` | `debug`
  - Default: `info`
- `error`: only errors
- `warn`: warnings + errors
- `info`: info + warnings + errors
- `debug`: reserved for future request-level diagnostics (no extra debug logs yet)

### Privacy / PII

- Never log raw public key material.
- Avoid logging raw auth header values; log only header names and whether overrides are enabled.

## Plan & Checklist

- [x] Identify current logging gaps (silent template errors, best-effort renders, etc.)
- [x] Decide a minimal logging schema (consistent prefixes/fields)
- [x] Add logging for:
  - [x] Auth decisions (without leaking sensitive headers)
  - [x] Key operations (log fingerprint, not key material)
  - [x] DNS operations (log source/destination)
- [x] Add regression tests if appropriate (or keep as manual verification)
- [x] Document expectations in a short note (optional)

## Progress

- 2026-01-02T17:48:53Z
  - Create task document

- 2026-01-03T04:31:37Z
  - Add consistent key/value logging for web handlers (keys + DNS) and template failures
  - Add auth unauthorized logging (without leaking header values)
  - Add `SSHBASTION_LOG_LEVEL` to control verbosity
  - Confirm `make test` passes

## References

- [design-overview] - Design overview document

[design-overview]: ../design/design-overview.md
