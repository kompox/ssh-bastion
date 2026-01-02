---
id: task-20260102c-webapp-routing-hardening
title: Harden web routing for path parameters
status: done
updated: 2026-01-02T20:32:00Z
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
- Validate DNS alias inputs as DNS-1123 subdomains (server-side)

### Out

- Changing route structure unless required
- Adding new user-facing features

## Plan & Checklist

- [x] List all mux patterns with `{...}` params
- [x] For each param:
  - [x] Confirm it is encoded when constructing URLs in templates
  - [x] Decode it in the handler and validate expectations
- [x] Add tests for:
  - [x] Key fingerprint (already exists; keep)
  - [x] DNS delete route (`source`)
- [x] Add DNS-1123 validation for DNS aliases:
  - [x] Validate `source` and `destination` as DNS-1123 subdomains
  - [x] Add unit tests for invalid DNS-1123 inputs
  - [x] Add web handler test ensuring invalid input returns `400` and preserves form values
- [x] Run full test suite

## Manual testing

- Start web app (test mode): `make run-test-mode`
- Open:
  - `http://localhost:8080/dns`
- Add alias (valid):
  - Source: `gitea.example.com`
  - Destination: `gitea.gitea.svc.cluster.local`
  - Expect: redirect back to `/dns` and alias shows in list
- Add alias (invalid DNS-1123):
  - Source: `Bad_Name.example.com` (uppercase + underscore)
  - Destination: `dest.example.com`
  - Expect: page re-renders with an error (HTTP `400`), and both input fields keep their values
- Delete alias:
  - Add `foo-bar.example.com` then click Delete
  - Expect: redirect back to `/dns` and alias disappears

## Progress

- 2026-01-02T17:48:53Z
  - Create task document

- 2026-01-02T20:18:00Z
  - Encode DNS `{source}` path value in templates (`/dns/{source}/delete`)
  - Decode `{source}` safely in handler (handle `/`, `+`, `%` reliably)
  - Add tests for template encoding and delete handler with escaped characters
  - Confirm `make test` passes

- 2026-01-02T20:32:00Z
  - Add server-side DNS-1123 validation for DNS alias inputs (source/destination)
  - Add unit + web tests for invalid DNS-1123 input behavior
  - Add short manual testing steps

## References

- [design-overview] - Design overview document
- [design-webapp-routes] - Web app routes & sitemap

[design-overview]: ../design/design-overview.md
[design-webapp-routes]: ../design/design-webapp-routes.md
