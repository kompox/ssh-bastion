---
id: design-manual-testing
title: Manual Testing & Local Dev Workflow
status: stable
updated: 2026-01-02T17:09:09Z
---
# Manual Testing & Local Dev Workflow

## Purpose

This document describes the supported manual testing workflows for the ssh-bastion web app, including local development without an auth proxy and verification of generated output files.

## Assumptions

- The web app is a server-rendered HTML app.
- Authentication is *trusted-header based* (an external auth proxy injects identity headers).
- Persistent state is file-based under a configured data directory.

## Build

```bash
go build -o ssh-bastion ./cmd/ssh-bastion
```

## Run Locally

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
go test ./...

# Verbose
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
