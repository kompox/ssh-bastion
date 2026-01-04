---
id: design-app-testing
title: App Testing (HTTP + DNS)
status: stable
updated: 2026-01-04T04:36:00Z
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

Use a single local startup method:

```bash
make run-test-mode
```

It starts `ssh-bastion serve` with:

- HTTP: `http://localhost:8080`
- DNS: UDP `127.0.0.1:5353`
- Data directory: `_tmp/data`
- Auth: test-mode identity override (no auth proxy required)

Stop with `Ctrl+C`.

### Expected on-disk outputs

After using the app, `_tmp/data/` is expected to contain:

```
_tmp/data/
├── users/
│   └── <user-uuid>/
│       └── keys/
│           ├── <fingerprint>.json
│           └── <fingerprint>.pub
├── authorized_keys/
│   └── jump
└── dns/
    └── aliases.json
```

Notes:

- `authorized_keys/jump` contains all enabled SSH public keys.
- `dns/aliases.json` contains configured DNS aliases.
- The app no longer generates dnsmasq config files.

## Manual HTTP Server Testing

These checks assume the server is running via `make run-test-mode`.

### Browser testing (override mode)

1. Open `http://localhost:8080/`.
2. Verify the SSH Keys page loads.
3. Add/enable/disable/delete keys.
4. Navigate to the DNS Aliases page and add/delete aliases.

### Error UX regression checks (task-20260102b)

Goal: confirm user-facing failures do not render a plain error text page.

1. Keys form error: submit an invalid public key on `/` and confirm:
    - The page is re-rendered with the normal layout.
    - An inline error is shown.
    - The text area preserves the submitted value.
2. DNS form error: submit a duplicate `source` on `/dns` and confirm:
    - The page is re-rendered with the normal layout.
    - An inline error is shown.
    - Inputs preserve the submitted values.

### Manual API checks (curl)

In test mode, you can still hit endpoints directly:

```bash
curl http://localhost:8080/

curl -X POST \
    -d "publicKey=ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com" \
    http://localhost:8080/keys

curl http://localhost:8080/dns

curl -X POST \
    -d "source=gitea.example.com&destination=gitea.gitea.svc.cluster.local" \
    http://localhost:8080/dns
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

1. Open `http://localhost:8080/dns`.
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

1. Delete the `hoge.local` alias from `http://localhost:8080/dns`.
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
| `SSHBASTION_AUTH_USER_ID_HEADER` | `X-MS-CLIENT-PRINCIPAL-ID` | `X-Auth-Request-User` | Header containing user ID |
| `SSHBASTION_AUTH_EMAIL_HEADER` | `X-MS-CLIENT-PRINCIPAL-NAME` | `X-Auth-Request-Email` | Header containing email |
| `SSHBASTION_AUTH_OVERRIDE_USER_ID` | (empty) | (empty) | TEST MODE: override user ID (ignores headers) |
| `SSHBASTION_AUTH_OVERRIDE_EMAIL` | (empty) | (empty) | TEST MODE: override email (ignores headers) |

## References

- [design-overview] - Design overview document
- [design-containers] - Containers (image + runtime topology)
- [design-webapp-routes] - Web app routes & sitemap

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
[design-webapp-routes]: ../design/design-webapp-routes.md
