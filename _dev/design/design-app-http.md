---
id: design-app-http
title: App HTTP (Routes & Sitemap)
status: stable
updated: 2026-01-06T23:43:09Z
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

## Roles & Permissions

This section documents intended access rules derived from [design-roadmap].

Roles:

|Role|Key permissions|DNS alias permissions|Home/admin permissions|
|-|-|-|-|
|`admin`|Manage all users’ SSH public keys|Manage DNS alias rules|View and edit home content via `/admin`|
|`user`|Manage own SSH public keys only|No access (cannot view or change)|No access to `/admin`; view home only|

Note: route-level authorization enforcement may lag behind this document.

## Route List

### Pages (GET)

- `GET /`
  - `200 OK`: Home
  - Access: `admin`, `user`
  - Content source: `${SSHBASTION_DATA_DIR}/content/pages/home.md` (default base: `/data`)
  - If missing: render a minimal placeholder (do not break the app)
- `GET /ssh`
  - `200 OK`: SSH Keys page
  - Access: `admin`, `user`
- `GET /admin`
  - `200 OK`: Admin dashboard
  - Access: `admin` only

Admin pages (HTMX HTML pages; no compatibility routes):

- `GET /admin/home`
  - `200 OK`: Home page markdown editor
  - Access: `admin` only
- `GET /admin/users`
  - `200 OK`: Users admin page
  - Access: `admin` only
  - List derived from current key owners
  - Display both `email` (primary) and `userID`
- `GET /admin/keys`
  - `200 OK`: SSH keys admin page (all users’ keys)
  - Access: `admin` only
  - UI: columns `Owner`, `Fingerprint`, `Status`, `Created`
- `GET /admin/dns`
  - `200 OK`: DNS aliases admin page
  - Access: `admin` only
- `GET /admin/targets`
  - `200 OK`: SSH forwarding targets admin page
  - Access: `admin` only

Removed routes (no compatibility):

- `/dns` (no redirect)

### Key operations (POST)

- `POST /ssh/keys`
  - `303`: add key success → redirect to `/ssh`
  - `400`: validation error → render `/ssh` with `flash-error`
  - Access: `admin` (any user), `user` (own keys)
- `POST /ssh/keys/{fingerprint}/enable`
  - `303`: enable success → redirect to `/ssh`
  - `400`: key not found → render `/ssh` with `flash-warning`
  - Access: `admin` (any user), `user` (own keys)
- `POST /ssh/keys/{fingerprint}/disable`
  - `303`: disable success → redirect to `/ssh`
  - `400`: key not found → render `/ssh` with `flash-warning`
  - Access: `admin` (any user), `user` (own keys)
- `POST /ssh/keys/{fingerprint}/delete`
  - `303`: delete success → redirect to `/ssh`
  - `400`: key not found → render `/ssh` with `flash-warning`
  - Access: `admin` (any user), `user` (own keys)

### Admin operations (POST)

Admin operations are grouped per page:

- `POST /admin/home`
  - `303`: save success → redirect to `/admin/home`
  - `400`: validation error → render `/admin/home` with `flash-error`
  - Access: `admin` only
- (none yet) `/admin/users` (GET only)
- (none yet) `/admin/keys` (GET only)
- `POST /admin/dns`
  - `303`: add alias success → redirect to `/admin/dns`
  - `400`: validation error → render `/admin/dns` with `flash-error`
  - Access: `admin` only
- `POST /admin/dns/{source}/delete`
  - `303`: delete success → redirect to `/admin/dns`
  - `400`: alias not found → render `/admin/dns` with `flash-warning`
  - Access: `admin` only

- `POST /admin/targets/mode`
  - `303`: set mode success → redirect to `/admin/targets`
  - `400`: validation error → render `/admin/targets` with `flash-error`
  - Access: `admin` only
- `POST /admin/targets/add`
  - `303`: add target success → redirect to `/admin/targets`
  - `400`: validation error → render `/admin/targets` with `flash-error`
  - Access: `admin` only
- `POST /admin/targets/{rule}/enable`
  - `303`: enable success → redirect to `/admin/targets`
  - `400`: target not found → render `/admin/targets` with `flash-warning`
  - Access: `admin` only
- `POST /admin/targets/{rule}/disable`
  - `303`: disable success → redirect to `/admin/targets`
  - `400`: target not found → render `/admin/targets` with `flash-warning`
  - Access: `admin` only
- `POST /admin/targets/{rule}/delete`
  - `303`: delete success → redirect to `/admin/targets`
  - `400`: target not found → render `/admin/targets` with `flash-warning`
  - Access: `admin` only

## Sitemap

```text
/
  Home (markdown)

/ssh
  (POST /ssh/keys)                Add key
  (POST /ssh/keys/{fp}/enable)    Enable key
  (POST /ssh/keys/{fp}/disable)   Disable key
  (POST /ssh/keys/{fp}/delete)    Delete key

/admin
  Admin dashboard

/admin/home
  (POST /admin/home)               Save home markdown

/admin/users
  GET only

/admin/keys
  GET only

/admin/dns
  (POST /admin/dns)                 Add alias
  (POST /admin/dns/{source}/delete) Delete alias

/admin/targets
  (POST /admin/targets/mode)              Set mode
  (POST /admin/targets/add)               Add target
  (POST /admin/targets/{rule}/enable)     Enable target
  (POST /admin/targets/{rule}/disable)    Disable target
  (POST /admin/targets/{rule}/delete)     Delete target
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
