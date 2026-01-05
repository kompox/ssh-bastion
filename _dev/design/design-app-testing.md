---
id: design-app-testing
title: App Testing (HTTP + DNS)
status: stable
updated: 2026-01-05T23:30:13Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# App Testing (HTTP + DNS)

## Purpose

This document describes the supported manual testing workflows for the ssh-bastion app.

Scope:

- This document covers testing the application behavior (HTTP UI/API and the in-process DNS proxy).
- This document does not cover end-to-end container testing via `compose.yml`.
- There are no manual tests for SSH functionality in this document.

For architecture context, see [design-overview]. For container/local topology, see [design-containers].

## Assumptions

- The web app is a server-rendered HTML app.
- Authentication is *trusted-header based* (an external auth proxy injects identity headers).
- Persistent state is file-based under a configured data directory.

## Makefile targets (recommended)

If available, prefer these targets over running raw `go` commands:

```bash
make clean
make build
make test
make run-test-mode
```

## Build

```bash
make build
```

## Unit Test

```bash
make test
```

## Run Server for Manual Testing

Use `make run-test-mode` to run HTTP + DNS locally with a test identity (no auth proxy required):

```bash
make run-test-mode
```

It starts `./ssh-bastion serve` with:

- Services listening on the following `addr:port/proto`:
  - HTTP: `:8080/tcp` (Open in browser at `http://localhost:8080/`)
  - DNS: `:5353/udp` (Query via `dig @localhost -p 5353 example.com A +short`)
- DNS upstream: auto-detected from `/etc/resolv.conf`
  - `SSHBASTION_DNS_UPSTREAM=$(DNS_UPSTREAM)` (default: empty)
- Optional DNS hardening:
  - `SSHBASTION_DNS_ALIASES_ONLY=false` (set to `true` to return NXDOMAIN for non-aliased A/AAAA queries)
- Data directory: `_tmp/data`
  - `SSHBASTION_DATA_DIR=$(DATA_DIR)` (default: `_tmp/data`)
  - Expected on-disk outputs: see [design-overview] (Storage layout section).
- Auth and roles are set via Makefile variables:
  - `SSHBASTION_AUTH_OVERRIDE_USER_ID=$(ID)` (default: `test-user-123`)
  - `SSHBASTION_AUTH_OVERRIDE_EMAIL=$(EMAIL)` (default: `developer@localhost`)
  - `SSHBASTION_ROLE_DEFAULT=$(ROLE)` (default: `user`)
  - `SSHBASTION_ROLE_ADMIN_IDS=$(ADMINS)` (default: empty)

You can override Makefile variables on invocation like this:

```bash
make run-test-mode ID=test-user-123 EMAIL=developer@localhost ADMINS=test-user-123,test-user-456
make run-test-mode ROLE=admin
make run-test-mode DATA_DIR=_tmp/data DNS_UPSTREAM=127.0.0.11:53
```

Stop the server with `Ctrl+C`.

## Manual HTTP Server Testing

These checks assume the server is running via `make run-test-mode`.

### Browser testing (override mode)

1. Open `http://localhost:8080/`.
2. Verify the Home page loads.
3. Navigate to `/ssh` and add/enable/disable/delete keys.
4. (Admin only) Navigate to `/admin/dns` and add/delete aliases.
5. (Admin only) Navigate to `/admin/home` and edit/save the home page markdown.

To test admin-only pages in test mode, start the server with an admin role, e.g.:

```bash
make run-test-mode ROLE=admin
```

### Error UX regression checks (task-20260102b)

Goal: confirm user-facing failures do not render a plain error text page.

1. Keys form error: submit an invalid public key on `/ssh` and confirm:
    - The page is re-rendered with the normal layout.
    - An inline error is shown.
    - The text area preserves the submitted value.
2. DNS form error (admin only): submit a duplicate `source` on `/admin/dns` and confirm:
    - The page is re-rendered with the normal layout.
    - An inline error is shown.
    - Inputs preserve the submitted values.

### Manual API checks (curl)

In test mode, you can still hit endpoints directly:

```bash
curl http://localhost:8080/

curl http://localhost:8080/ssh

curl -X POST \
    -d "publicKey=ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com" \
    http://localhost:8080/ssh/keys

curl http://localhost:8080/admin/dns

curl -X POST \
    -d "source=gitea.example.com&destination=gitea.gitea.svc.cluster.local" \
    http://localhost:8080/admin/dns
```

### Auth failure behavior (when not in test mode)

If required identity headers are missing (and test mode overrides are not set), requests are rejected.

```bash
curl -v http://localhost:8080/
# Expect HTTP 401 Unauthorized
```

## Manual DNS Server Testing

These checks assume the DNS server is running via `make run-test-mode`.

### Create an alias (via web UI)

1. (Admin only) Open `http://localhost:8080/admin/dns`.
2. Add `source`: `hoge.local`.
3. Add `destination`: `example.com`.

### Query DNS

Use `dig` (or `drill`) from your host:

```bash
dig @127.0.0.1 -p 5353 hoge.local A +short
```

Expected: an IPv4 address is returned.

To check the owner name, run without `+short`:

```bash
dig @127.0.0.1 -p 5353 hoge.local A
```

Expected: the answer owner name should be `hoge.local.` (not `example.com.`).

### Delete alias and verify it stops resolving

1. Delete the `hoge.local` alias from `http://localhost:8080/admin/dns`.
2. Query again:

```bash
dig @127.0.0.1 -p 5353 hoge.local A
```

Expected: no `A` answer is returned.

## Configuration Matrix

| Variable | Default (easy_auth) | Default (oauth2_proxy) | Description |
|----------|---------------------|------------------------|-------------|
| `SSHBASTION_DATA_DIR` | `/data` | `/data` | Root directory for file storage |
| `SSHBASTION_AUTH_MODE` | `easy_auth` | - | Authentication mode |
| `SSHBASTION_AUTH_USER_ID_HEADER` | `X-MS-CLIENT-PRINCIPAL-ID` | `X-Forwarded-User` | Header containing user ID |
| `SSHBASTION_AUTH_EMAIL_HEADER` | `X-MS-CLIENT-PRINCIPAL-NAME` | `X-Forwarded-Email` | Header containing email |
| `SSHBASTION_AUTH_OVERRIDE_USER_ID` | (empty) | (empty) | TEST MODE: override user ID (ignores headers) |
| `SSHBASTION_AUTH_OVERRIDE_EMAIL` | (empty) | (empty) | TEST MODE: override email (ignores headers) |
| `SSHBASTION_ROLE_DEFAULT` | `user` | `user` | Default role (set to `admin` to access `/admin/*`) |
| `SSHBASTION_ROLE_ADMIN_IDS` | (empty) | (empty) | Comma-separated admin user IDs (alternative to `SSHBASTION_ROLE_DEFAULT=admin`) |
| `SSHBASTION_DNS_UPSTREAM` | (optional; from `/etc/resolv.conf`) | (optional; from `/etc/resolv.conf`) | DNS proxy upstream resolver (e.g. `127.0.0.11:53` in Docker) |
| `SSHBASTION_DNS_ALIASES_ONLY` | `false` | `false` | When `true`, DNS proxy answers aliases only; other A/AAAA queries return NXDOMAIN |

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-app-http]: ../design/design-app-http.md
