---
id: design-containers
title: Containers (image + runtime topology)
status: draft
updated: 2026-01-03T18:12:41Z
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
- Pod-style networking: run dnsmasq as a sidecar in the same network namespace so the sshd process can use `127.0.0.1:53`.
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
  - `dnsmasq` and `openssh-server` are available as Alpine packages.
- Escape hatch: switch to Debian/Ubuntu if we later need glibc compatibility or encounter musl-specific issues.

### Contents

The image should contain:

- `/usr/local/bin/ssh-bastion` (the Go binary)
- `/app/web/` (templates and static assets required by the web server)
- runtime packages:
  - `dnsmasq` (required for sidecar DNS)
  - `openssh-server` (used by the `sshd` container)
  - `ca-certificates`

### Defaults and overridability

- Default entrypoint should be the `ssh-bastion` binary.
- Default command should start the web server (for local usability).
- Other roles (dnsmasq, sshd) are expected to be run by overriding `entrypoint`/`command` at the orchestrator level:
  - docker-compose: `entrypoint: ["dnsmasq"]`
  - Kubernetes: `command: ["dnsmasq", ...]` or `command: ["ssh-bastion", "dns", ...]` (future)

Note: this is a deliberate trade-off to keep “one image, multiple containers” without introducing a process supervisor.

## Target runtime topology (Kubernetes)

### Pod structure

Target a single Pod with multiple containers and a shared writable volume:

- Container: `ssh-bastion`
  - Runs: `ssh-bastion web ...`
  - Reads/writes under `/data`:
    - generated dnsmasq config snippets (e.g. `/data/dns/dnsmasq.d/*.conf`)
    - key registry / generated authorized_keys
- Container: `dnsmasq` (sidecar)
  - Runs: `dnsmasq ...`
  - Listens on `:53` within the Pod network namespace (reachable at `127.0.0.1:53`)
  - Reads config from the shared `/data` volume
- Container: `sshd`
  - Runs: `sshd -D -e ...` (exact command/config is out of scope here)
  - Reads generated `authorized_keys` and host keys from shared storage

### Resolver wiring (Kubernetes)

- Default: the Pod uses cluster DNS (e.g. CoreDNS) via the normal `dnsPolicy: ClusterFirst` behavior.
- `ssh-bastion` and `dnsmasq` should keep that default resolver.
- `sshd` is the only container intended to use dnsmasq (`127.0.0.1:53`), by rewriting `/etc/resolv.conf` in the sshd entrypoint (under the assumptions documented in [DNS (nameserver routing)](#dns-nameserver-routing)).

### Shared volume

- A single shared volume mounted at `/data` in all containers.
- The application treats `/data` as the persistence root (configurable by `SSHBASTION_DATA_DIR`).

## Local runtime topology (docker-compose)

### Design intent

docker-compose should emulate two key Pod properties:

1. Shared network namespace between the `ssh-bastion` container and dnsmasq.
2. Shared writable volume for `/data`.

### Sidecar networking emulation

- Use `network_mode: service:ssh-bastion` on the `dnsmasq` service.
- As a result:
  - dnsmasq can bind to `127.0.0.1:53`
  - containers in the shared network namespace can reach dnsmasq at `127.0.0.1:53`

Important compose behavior:

- Port publishing must be done on the “network namespace owner” service (`ssh-bastion`).
- Even if dnsmasq is a separate service, published DNS ports (53/udp,tcp) must be declared on `ssh-bastion`.

### Resolver wiring in Docker

In Docker, upstream DNS is provided by the engine via an embedded resolver:

- Docker embedded DNS IP: `127.0.0.11`

Resolver wiring intent:

- `ssh-bastion` should keep Docker’s default resolver (`127.0.0.11`).
- `dnsmasq` should forward non-alias queries to an upstream resolver that can reach the Internet (in compose: Docker’s embedded resolver).
- `sshd` is the only container intended to use dnsmasq as its resolver (`127.0.0.1:53`).

The key constraint is avoiding a recursion loop where Docker’s resolver ends up forwarding back to dnsmasq. Keep the “network namespace owner” on Docker’s default DNS, and only opt `sshd` into dnsmasq.

### Current local contract

- Web UI served on `:8080`.
- Optional DNS exposure to the host via a mapped port (e.g. `5353 -> 53`) for inspection/testing.
  - For Docker port publishing to work, dnsmasq must also listen on the container interface (e.g. `--listen-address=0.0.0.0`).
    Containers that opt into dnsmasq (intended: `sshd`) can still resolve via `127.0.0.1:53` within the shared network namespace.
- Data persisted via a bind mount to `./_tmp/data`.

## DNS (nameserver routing)

This section consolidates the DNS resolver routing design for both supported environments.

### Design intent

- dnsmasq answers configured CNAME-like aliases and forwards everything else to an upstream resolver.
- Only the `sshd` container is expected to use dnsmasq as its resolver (`127.0.0.1:53`).
- The `ssh-bastion` and `dnsmasq` containers should keep the platform-default resolver so they can resolve upstream names without recursion.

### Kubernetes (Pod)

Constraints:

- Pod DNS is configured at the Pod level (all containers get the same default resolver config).
- Per-container DNS is not a first-class feature.

Supported approaches:

**Option A (simple, current MVP intent): per-container resolver override**

- Keep the Pod DNS policy as the default (`dnsPolicy: ClusterFirst`).
- Run dnsmasq as a sidecar that binds to `0.0.0.0:53` (and optionally `127.0.0.1:53`) inside the Pod network namespace.
- In the `sshd` container entrypoint, overwrite `/etc/resolv.conf` to use `nameserver 127.0.0.1`.

Assumptions:

- `sshd` runs as root.
- Read-only root filesystem is not enabled for the `sshd` container.
- `/etc/resolv.conf` is writable in the `sshd` container.

**Option B (strict, Pod-wide): dnsmasq as the Pod resolver**

- `dnsPolicy: None`
- `dnsConfig.nameservers: ["127.0.0.1"]`
- dnsmasq upstream: the cluster DNS service IP (e.g. CoreDNS)

This makes every container use dnsmasq (not currently required). It is more uniform, but requires careful upstream wiring.

### Docker Compose

Constraints:

- dnsmasq and sshd are run as “sidecars” sharing the `ssh-bastion` network namespace (`network_mode: service:ssh-bastion`).
- Docker does not allow setting `dns:` on a service that uses `network_mode: service:...`.
- Port publishing must be done on the “namespace owner” service (`ssh-bastion`).

Recommended behavior:

- `ssh-bastion`: keep the default Docker resolver (`127.0.0.11`).
- `dnsmasq` (sidecar): forward non-alias queries to Docker’s resolver (`127.0.0.11`).
- `sshd` (sidecar): overwrite `/etc/resolv.conf` at start to `nameserver 127.0.0.1` so only sshd resolves via dnsmasq.

Pitfall: recursion loop

- If the “namespace owner” container (`ssh-bastion`) is configured to use `127.0.0.1` as its DNS server, Docker’s embedded DNS may forward external lookups back to `127.0.0.1`.
- If dnsmasq also uses Docker’s resolver (`127.0.0.11`) as its upstream, this creates a loop:
  - `dnsmasq -> 127.0.0.11 -> 127.0.0.1 -> dnsmasq`
- Symptoms include `SERVFAIL`, timeouts, and `Maximum number of concurrent DNS queries reached` in dnsmasq logs.

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

### Reloading dnsmasq after changes

- Local docker-compose (MVP): restart the `dnsmasq` service to pick up regenerated `*.conf` files.
- Kubernetes (target): prefer SIGHUP to dnsmasq or restart the container; exact mechanism depends on how we choose to supervise changes.

### Logging

- All processes should log to stdout/stderr.
- dnsmasq should use `--log-facility=-` in local compose.

## Open questions / follow-up work

- Document and validate how `sshd` is started (separate container from the same image) and how it consumes generated `authorized_keys`.
- Add explicit multi-role entrypoints to the Go binary (e.g. `ssh-bastion dns` / `ssh-bastion sshd`) to reduce reliance on compose `entrypoint` overrides.
- Decide the exact reload strategy for dnsmasq and sshd.
- Production hardening: securityContext/capabilities for binding to port 53, and a minimal-permission setup for sshd.

## References

- [design-overview] - Design overview document
- [design-webapp-routes] - Web app routes & sitemap
- Workflow: [docker-build.yml](../../.github/workflows/docker-build.yml)
- [design-roadmap] - Development Roadmap

[design-overview]: ../design/design-overview.md
[design-webapp-routes]: ../design/design-webapp-routes.md
[design-roadmap]: ../design/design-roadmap.md
