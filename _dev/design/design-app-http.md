---
id: design-app-http
title: App HTTP (Routes & Sitemap)
status: stable
updated: 2026-01-04T09:09:31Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# App HTTP (Routes & Sitemap)

## Purpose

Document the ssh-bastion web app’s HTTP routes (GET/POST) and provide a simple sitemap for manual testing and maintenance.

This document complements [design-overview].

## Notes

- The app is server-rendered HTML.
- All routes are wrapped by the auth middleware (trusted-header based). See auth behavior in other docs; this doc focuses on routing.
- Some actions accept a path parameter that may be URL-escaped in the client (e.g. `fingerprint`).
- MVP posture: HTMX-style HTML endpoints first; add `/api/*` only if a real REST/JSON consumer appears.

### Status code conventions

- Auth failures: `401` (HTML for browser requests; plain/text for non-HTML clients).
- Success for form actions: `303 See Other` redirect back to the relevant page.
- Validation / not found: `400 Bad Request` with the same page rendered and a flash message.
- Unexpected server errors: `500 Internal Server Error` with the same page rendered and a flash message.

## Route List

### Pages (GET)

- `GET /`
  - `200 OK`: Home
  - Content source: `${SSHBASTION_DATA_DIR}/content/pages/home.md` (default base: `/data`)
  - If missing: render a minimal placeholder (do not break the app)
- `GET /ssh`
  - `200 OK`: SSH Keys page
- `GET /dns`
  - `200 OK`: DNS Aliases page
- `GET /admin`
  - `200 OK`: Placeholder page (future use)

### Key operations (POST)

- `POST /ssh/keys`
  - `303`: add key success → redirect to `/ssh`
  - `400`: validation error → render `/ssh` with `flash-error`
- `POST /ssh/keys/{fingerprint}/enable`
  - `303`: enable success → redirect to `/ssh`
  - `400`: key not found → render `/ssh` with `flash-warning`
- `POST /ssh/keys/{fingerprint}/disable`
  - `303`: disable success → redirect to `/ssh`
  - `400`: key not found → render `/ssh` with `flash-warning`
- `POST /ssh/keys/{fingerprint}/delete`
  - `303`: delete success → redirect to `/ssh`
  - `400`: key not found → render `/ssh` with `flash-warning`

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
  Home (markdown)

/ssh
  (POST /ssh/keys)                Add key
  (POST /ssh/keys/{fp}/enable)    Enable key
  (POST /ssh/keys/{fp}/disable)   Disable key
  (POST /ssh/keys/{fp}/delete)    Delete key

/dns
  (POST /dns)                 Add alias
  (POST /dns/{source}/delete) Delete alias

/admin
  Placeholder (future use)
```

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-roadmap] - Development roadmap
- [task-20260104b-webapp-routing-update] - Web app site map and routing update

- Source of truth: `internal/web/server.go`
- Templates:
  - `web/templates/layout.html`
  - `web/templates/keys.html`
  - `web/templates/dns.html`

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-roadmap]: ../design/design-roadmap.md
[task-20260104b-webapp-routing-update]: ../tasks/task-20260104b-webapp-routing-update.md
