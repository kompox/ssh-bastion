---
id: design-containers
title: Containers (image + runtime topology)
status: draft
updated: 2026-01-04T09:09:31Z
updated: 2026-01-04T09:24:31Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: Containers (image + runtime topology)

This doc describes:

- the intended `ghcr.io/kompox/ssh-bastion` container image contents and conventions
- the target runtime topology as a Kubernetes Pod (primary target)
- a local docker-compose setup that emulates the Pod behavior closely enough for development/testing

This design complements the high-level architecture in [design-overview].

## Goals

- Single build artifact: publish one image (`ghcr.io/kompox/ssh-bastion`) and run multiple containers from it by overriding `command`/`args`.
- Pod-style networking: run `sshd` in the same Pod/network namespace so it can use a bastion-local DNS proxy at `127.0.0.1:53`.
- Simple operations: log to stdout; configs are files under a shared `/data` volume.
- Good local parity: docker-compose should emulate the K8s Pod approach (sidecar networking + shared volume).

## Non-goals

- Implementing or hardening SSH daemon behavior (sshd startup/config, user/authorized_keys wiring, port 22 exposure).
- Shipping Kubernetes manifests/Helm from this document.
- Production-grade hardening (capabilities, AppArmor/SELinux, read-only rootfs, resource limits, network policies), beyond what is required to develop locally.

## Container image design

### Base image

- Runtime base: Alpine (current implementation uses `alpine:3.20`).
- Rationale:
  - `ssh-bastion` is built with `CGO_ENABLED=0` and is suitable as a static binary.
  - `openssh-server` is available as an Alpine package.
- Escape hatch: switch to Debian/Ubuntu if we later need glibc compatibility or encounter musl-specific issues.

### Contents

The image should contain:

- `/usr/local/bin/ssh-bastion` (the Go binary)
- `/app/web/` (templates and static assets required by the web server)
- runtime packages:
  - `openssh-server` (used by the `sshd` container)
  - `ca-certificates`

### Defaults and overridability

- Default entrypoint should be the `ssh-bastion` binary.
- Default command should start the web server (for local usability).
- Other roles (notably `sshd`) are expected to be run by overriding `entrypoint`/`command` at the orchestrator level.

Note: this is a deliberate trade-off to keep “one image, multiple containers” without introducing a process supervisor.

## Target runtime topology (Kubernetes)

### Pod structure

Target a single Pod with multiple containers and a shared writable volume:

- Container: `ssh-bastion`
  - Runs: `ssh-bastion web ...`
  - Reads/writes under `/data`:
    - DNS alias rules (e.g. `/data/dns/aliases.json`)
    - key registry / generated authorized_keys
- Container: `sshd`
  - Runs: `sshd -D -e ...` (exact command/config is out of scope here)
  - Reads generated `authorized_keys` and host keys from shared storage

### Resolver wiring (Kubernetes)

- Default: the Pod uses cluster DNS (e.g. CoreDNS) via the normal `dnsPolicy: ClusterFirst` behavior.
- `ssh-bastion` should keep that default resolver.
- `sshd` is the only container intended to use the bastion-local DNS proxy (`127.0.0.1:53`), by rewriting `/etc/resolv.conf` in the sshd entrypoint (under the assumptions documented in [DNS (nameserver routing)](#dns-nameserver-routing)).

### Shared volume

- A single shared volume mounted at `/data` in all containers.
- The application treats `/data` as the persistence root (configurable by `SSHBASTION_DATA_DIR`).

## Local runtime topology (docker-compose)

### Design intent

docker-compose should emulate two key Pod properties:

1. Shared network namespace between the `ssh-bastion` container and `sshd`.
2. Shared writable volume for `/data`.

### Sidecar networking emulation

- Use `network_mode: service:ssh-bastion` on the `sshd` service.
- As a result:
  - `ssh-bastion` can bind a DNS listener on `127.0.0.1:53`
  - `sshd` can reach the bastion-local DNS proxy at `127.0.0.1:53`

Important compose behavior:

- Port publishing must be done on the “network namespace owner” service (`ssh-bastion`).
- Published DNS ports (53/udp,tcp) must be declared on `ssh-bastion`.

### Resolver wiring in Docker

In Docker, upstream DNS is provided by the engine via an embedded resolver:

- Docker embedded DNS IP: `127.0.0.11`

Resolver wiring intent:

- `ssh-bastion` should keep Docker’s default resolver (`127.0.0.11`).
- DNS proxy upstream selection:
  - If `SSHBASTION_DNS_UPSTREAM` is set, use it.
  - Otherwise, auto-detect from `/etc/resolv.conf` (in Docker this is typically `127.0.0.11:53`).
- `sshd` is the only container intended to use the DNS proxy as its resolver (`127.0.0.1:53`).

The key constraint is avoiding a recursion loop where Docker’s resolver ends up forwarding back to `127.0.0.1`. Keep the “network namespace owner” on Docker’s default DNS, and only opt `sshd` into the DNS proxy.

### Current local contract

- Web UI served on `:8080`.
- Optional DNS exposure to the host via a mapped port (e.g. `5353 -> 53`) for inspection/testing.
  - Containers that opt into the DNS proxy (intended: `sshd`) resolve via `127.0.0.1:53` within the shared network namespace.
- Data persisted via a bind mount to `./_tmp/data`.

## DNS (nameserver routing)

This section consolidates the DNS resolver routing design for both supported environments.

### Design intent

- The DNS proxy answers configured CNAME-like aliases and forwards everything else to an upstream resolver.
- Only the `sshd` container is expected to use the DNS proxy as its resolver (`127.0.0.1:53`).
- The `ssh-bastion` container should keep the platform-default resolver so it can resolve upstream names without recursion.

### Kubernetes (Pod)

Constraints:

- Pod DNS is configured at the Pod level (all containers get the same default resolver config).
- Per-container DNS is not a first-class feature.

Supported approaches:

**Option A (simple, current MVP intent): per-container resolver override**

- Keep the Pod DNS policy as the default (`dnsPolicy: ClusterFirst`).
- Run the `ssh-bastion` container with the DNS proxy enabled, listening on `:53` inside the Pod network namespace (reachable at `127.0.0.1:53`).
- In the `sshd` container entrypoint, overwrite `/etc/resolv.conf` to use `nameserver 127.0.0.1`.

Upstream selection (DNS proxy):

- If `SSHBASTION_DNS_UPSTREAM` is not set, the DNS proxy should use the `nameserver` from `/etc/resolv.conf`.
  - In Kubernetes, this is typically the cluster DNS service IP.

Assumptions:

- `sshd` runs as root.
- Read-only root filesystem is not enabled for the `sshd` container.
- `/etc/resolv.conf` is writable in the `sshd` container.

**Option B (strict, Pod-wide): DNS proxy as the Pod resolver**

- `dnsPolicy: None`
- `dnsConfig.nameservers: ["127.0.0.1"]`
- DNS proxy upstream: the cluster DNS service IP (e.g. CoreDNS)

This makes every container use the DNS proxy (not currently required). It is more uniform, but requires careful upstream wiring.

### Docker Compose

Constraints:

- sshd is run as a “sidecar” sharing the `ssh-bastion` network namespace (`network_mode: service:ssh-bastion`).
- Docker does not allow setting `dns:` on a service that uses `network_mode: service:...`.
- Port publishing must be done on the “namespace owner” service (`ssh-bastion`).

Recommended behavior:

- `ssh-bastion`: keep the default Docker resolver (`127.0.0.11`).
- DNS proxy (in `ssh-bastion`): if `SSHBASTION_DNS_UPSTREAM` is not set, use `/etc/resolv.conf` (in Docker typically `127.0.0.11:53`).
- `sshd` (sidecar): overwrite `/etc/resolv.conf` at start to `nameserver 127.0.0.1` so only sshd resolves via the DNS proxy.

Pitfall: recursion loop

- If the “namespace owner” container (`ssh-bastion`) is configured to use `127.0.0.1` as its DNS server, Docker’s embedded DNS may forward external lookups back to `127.0.0.1`.
- If the DNS proxy forwards upstream to Docker’s resolver (`127.0.0.11:53`), this creates a loop:
  - `dns-proxy -> 127.0.0.11 -> 127.0.0.1 -> dns-proxy`
- Symptoms include `SERVFAIL`, timeouts, and high query concurrency.

## Operational notes

### CI build (multi-architecture) and publishing (GHCR)

- The image is published to GHCR as `ghcr.io/kompox/ssh-bastion`.
- CI builds a multi-architecture image using QEMU + Buildx:
  - `linux/amd64`
  - `linux/arm64`
- Triggers:
  - push to `main`
  - push of tags matching `v*`
- Tags:
  - `main` (updated on pushes to `main`)
  - `latest` (updated on pushes of `v*` tags)
  - `v*` (the pushed Git tag)

Operator notes:

- GHCR package page: https://github.com/kompox/ssh-bastion/pkgs/container/ssh-bastion
- Pull examples:
  - `docker pull ghcr.io/kompox/ssh-bastion:main`
  - `docker pull ghcr.io/kompox/ssh-bastion:latest`
  - `docker pull ghcr.io/kompox/ssh-bastion:vX.Y.Z`

### DNS proxy updates

The DNS proxy reads the alias registry at query time, so updates take effect without a reload.

### Logging

- All processes should log to stdout/stderr.

## Open questions / follow-up work

- Document and validate how `sshd` is started (separate container from the same image) and how it consumes generated `authorized_keys`.
- Add explicit multi-role entrypoints to the Go binary (e.g. `ssh-bastion dns` / `ssh-bastion sshd`) to reduce reliance on compose `entrypoint` overrides.
- Production hardening: securityContext/capabilities for binding to port 53, and a minimal-permission setup for sshd.

## References

- [design-overview] - Design overview document
- [design-app-http] - App HTTP (Routes & Sitemap)
- Workflow: [docker-build.yml](../../.github/workflows/docker-build.yml)
- [design-roadmap] - Development Roadmap

[design-overview]: ../design/design-overview.md
[design-app-http]: ../design/design-app-http.md
[design-roadmap]: ../design/design-roadmap.md
