---
id: design-webapp-routes
title: Web App Routes & Sitemap
status: stable
updated: 2026-01-02T18:56:16Z
---
# Web App Routes & Sitemap

## Purpose

Document the ssh-bastion web app’s HTTP routes (GET/POST) and provide a simple sitemap for manual testing and maintenance.

This document complements [design-overview].

## Notes

- The app is server-rendered HTML.
- All routes are wrapped by the auth middleware (trusted-header based). See auth behavior in other docs; this doc focuses on routing.
- Some actions accept a path parameter that may be URL-escaped in the client (e.g. `fingerprint`).

### Status code conventions

- Auth failures: `401` (HTML for browser requests; plain/text for non-HTML clients).
- Success for form actions: `303 See Other` redirect back to the relevant page.
- Validation / not found: `400 Bad Request` with the same page rendered and a flash message.
- Unexpected server errors: `500 Internal Server Error` with the same page rendered and a flash message.

## Route List

### Pages (GET)

- `GET /`
  - `200 OK`: SSH Keys page
- `GET /dns`
  - `200 OK`: DNS Aliases page

### Key operations (POST)

- `POST /keys`
  - `303`: add key success → redirect to `/`
  - `400`: validation error → render `/` with `flash-error`
- `POST /keys/{fingerprint}/enable`
  - `303`: enable success → redirect to `/`
  - `400`: key not found → render `/` with `flash-warning`
- `POST /keys/{fingerprint}/disable`
  - `303`: disable success → redirect to `/`
  - `400`: key not found → render `/` with `flash-warning`
- `POST /keys/{fingerprint}/delete`
  - `303`: delete success → redirect to `/`
  - `400`: key not found → render `/` with `flash-warning`

### DNS operations (POST)

- `POST /dns`
  - `303`: add alias success → redirect to `/dns`
  - `400`: validation error → render `/dns` with `flash-error`
- `POST /dns/{source}/delete`
  - `303`: delete success → redirect to `/dns`
  - `400`: alias not found → render `/dns` with `flash-warning`

## Sitemap

```text
/
  (POST /keys)                Add key
  (POST /keys/{fp}/enable)    Enable key
  (POST /keys/{fp}/disable)   Disable key
  (POST /keys/{fp}/delete)    Delete key

/dns
  (POST /dns)                 Add alias
  (POST /dns/{source}/delete) Delete alias
```

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)

- Source of truth: `internal/web/server.go`
- Templates:
  - `web/templates/layout.html`
  - `web/templates/keys.html`
  - `web/templates/dns.html`

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
