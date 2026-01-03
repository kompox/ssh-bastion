---
id: design-manual-testing
title: Manual Testing & Local Dev Workflow
status: stable
updated: 2026-01-02T18:16:42Z
---
# Manual Testing & Local Dev Workflow

## Purpose

This document describes the supported manual testing workflows for the ssh-bastion web app, including local development without an auth proxy and verification of generated output files.

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

## Run Locally

### Quick start (browser testing without an auth proxy)

This starts the server in “test mode” (override identity), storing data under `_tmp/data`:

```bash
make run-test-mode
```

### Data directory

```bash
mkdir -p _tmp/data
```

### Environment variables

#### Production/testing with a real auth proxy

```bash
export SSHBASTION_DATA_DIR="_tmp/data"
export SSHBASTION_AUTH_MODE="easy_auth"
```

#### Development/testing without an auth proxy (browser testing)

```bash
export SSHBASTION_DATA_DIR="_tmp/data"
export SSHBASTION_AUTH_MODE="easy_auth"
export SSHBASTION_AUTH_OVERRIDE_USER_ID="test-user-123"
export SSHBASTION_AUTH_OVERRIDE_EMAIL="user@example.com"
```

Security note: the override environment variables are **test mode only** and must never be used in production.

### Start the server

```bash
./ssh-bastion web
```

The server listens on `http://localhost:8080`.

## Manual Browser Testing

### Browser testing in override mode

When `SSHBASTION_AUTH_OVERRIDE_USER_ID` and `SSHBASTION_AUTH_OVERRIDE_EMAIL` are set:

1. Open `http://localhost:8080/`
2. Verify the SSH Keys page loads
3. Add/enable/disable/delete keys
4. Navigate to `DNS Aliases` and add/delete aliases

All operations are associated with the override identity.

### Error UX regression checks (task-20260102b)

Goal: confirm user-facing failures do not render a “plain error text page”.

1. **Keys form error**: submit an invalid public key on `/` and confirm:
     - The page is re-rendered with the normal layout
     - An inline error is shown
     - The text area preserves the submitted value
2. **DNS form error**: submit a duplicate `source` on `/dns` and confirm:
     - The page is re-rendered with the normal layout
     - An inline error is shown
     - Inputs preserve the submitted values
3. **Auth failure (no headers, no overrides)**: in a separate terminal, run:

```bash
SSHBASTION_AUTH_OVERRIDE_USER_ID= SSHBASTION_AUTH_OVERRIDE_EMAIL= \
  curl -i -H "Accept: text/html" http://localhost:8080/
```

Confirm it returns `401` and HTML (so browsers don’t show a plain text error page).

### Browser testing with header injection (no override)

Use a browser header injection tool (e.g., ModHeader) to inject headers and then navigate to `http://localhost:8080/`.

For `easy_auth` defaults:

- `X-MS-CLIENT-PRINCIPAL-ID`: `test-user-123`
- `X-MS-CLIENT-PRINCIPAL-NAME`: `your-email@example.com`

## Manual API Testing (curl)

### Azure Easy Auth mode (default)

```bash
# View SSH keys page
curl -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     http://localhost:8080/

# Add an SSH key
curl -X POST \
     -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     -d "publicKey=ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl test@example.com" \
     http://localhost:8080/keys

# View DNS aliases page
curl -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     http://localhost:8080/dns

# Add a DNS alias
curl -X POST \
     -H "X-MS-CLIENT-PRINCIPAL-ID: test-user-123" \
     -H "X-MS-CLIENT-PRINCIPAL-NAME: user@example.com" \
     -d "source=gitea.example.com&destination=gitea.gitea.svc.cluster.local" \
     http://localhost:8080/dns
```

### oauth2-proxy mode

```bash
export SSHBASTION_AUTH_MODE="oauth2_proxy"
./ssh-bastion web

curl -H "X-Auth-Request-User: test-user-123" \
     -H "X-Auth-Request-Email: user@example.com" \
     http://localhost:8080/
```

## Expected Outputs (Verification)

### authorized_keys

```bash
cat _tmp/data/authorized_keys/jump
```

This file should contain all *enabled* SSH public keys.

### dnsmasq config

```bash
cat _tmp/data/dns/dnsmasq.d/generated.conf
```

This file should contain `cname=` directives for all configured DNS aliases.

## Auth Failure Behavior

If required identity headers are missing (and test mode overrides are not set), requests are rejected.

```bash
curl -v http://localhost:8080/
# Expect HTTP 401 Unauthorized
```

## Running Tests

```bash
# Run all tests
make test

# Alternative (without Makefile)
go test -v ./...

# Per-package
go test -v ./internal/keys
go test -v ./internal/dns
go test -v ./internal/storage
```

## On-disk Layout

After use, `${SSHBASTION_DATA_DIR}` is expected to contain:

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
    ├── aliases.json
    └── dnsmasq.d/
        └── generated.conf
```

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
