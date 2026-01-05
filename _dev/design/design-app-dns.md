---
id: design-app-dns
title: App DNS (in-process DNS proxy)
status: draft
updated: 2026-01-05T23:30:13Z
assistedBy: github/copilot (vscode) gpt-5.2
---
# Design: App DNS (in-process DNS proxy)

## Purpose

This document describes the bastion-local DNS behavior implemented by `ssh-bastion`.

Scope:

- DNS alias rules ("CNAME-like") used only by the bastion.
- The in-process DNS proxy behavior and resolver wiring.

Non-scope:

- General-purpose DNS management.
- Exposing DNS as a cluster-wide service.

For container topology and how `sshd` is wired to use the proxy, see [design-containers].

## Goals

- Enable SSH-style name resolution for selected external FQDNs by rewriting them to cluster-internal service names.
- Keep destination resolution dynamic (do not pin A/AAAA at configuration time).
- Keep persistence file-based and auditable.
- Avoid a separate DNS sidecar container.

## Non-goals

- Supporting arbitrary DNS record types.
- Implementing a full caching resolver.
- Acting as the Pod-wide resolver for all containers.

## Data model

Source of truth:

- `${SSHBASTION_DATA_DIR}/dns/aliases.json`

Each alias maps a source hostname to a destination hostname.

Example:

```json
[
  {
    "source": "gitea.example.com",
    "destination": "gitea.gitea.svc.cluster.local"
  }
]
```

Rules:

- Matching is case-insensitive on hostnames.
- Hostnames may be stored with or without a trailing dot; comparisons treat them as FQDNs.

## Runtime behavior

### Where it runs

The DNS proxy runs inside the `ssh-bastion` process (enabled via `ssh-bastion serve -dns=true`).

- Listen address: `-dns-addr` (default `:53`)
- Upstream resolver: `-dns-upstream`

### Upstream resolver selection (desired behavior)

To work out-of-the-box in both Kubernetes and Docker, the DNS proxy should resolve its upstream in this order:

1. CLI flag: `-dns-upstream` (explicit)
2. Environment variable: `SSHBASTION_DNS_UPSTREAM`
3. Auto-detect from `/etc/resolv.conf`:
  - read the first `nameserver` entry and use `:53`
  - for IPv6, use bracket form (e.g. `[fd00::1]:53`)

If no upstream can be determined (e.g. no `nameserver` entries), the DNS proxy should fail fast at startup with a clear error.

Notes:

- Docker typically writes `nameserver 127.0.0.11` into `/etc/resolv.conf`.
- Kubernetes typically writes the cluster DNS service IP into `/etc/resolv.conf`.

### Resolver wiring intent

- `ssh-bastion` keeps the platform default resolver for its own non-proxy DNS needs.
- `sshd` is the only container that points `/etc/resolv.conf` at the proxy (`nameserver 127.0.0.1`).

This avoids recursion loops and keeps the blast radius of DNS aliasing limited to SSH traffic.

## DNS protocol behavior

### Transport

- UDP only.

This matches the needs of SSH hostname lookups and keeps the implementation minimal.

### Query types

- The proxy only answers/forwards `A` and `AAAA` queries.
- For other QTYPEs (e.g. TXT/MX), it returns NODATA (NOERROR + empty answer) instead of forwarding.

Rationale: aliases are intended to make SSH-style `A/AAAA` name resolution reliable, and forwarding unrelated QTYPEs can surface upstream resolver quirks.

### Rewrite algorithm

For a single-question DNS query:

1. If QNAME does not match any configured alias source:
  - If `SSHBASTION_DNS_ALIASES_ONLY` is enabled, return NXDOMAIN.
  - Otherwise, forward the query unchanged to the upstream resolver.
2. If QNAME matches an alias source:
   - Rewrite QNAME to the alias destination.
   - Forward to the upstream resolver.
   - Rewrite the response owner names for `A`/`AAAA` records back to the original QNAME.

For multi-question queries, the proxy forwards the query unchanged.

### Response shape

The proxy aims to be "CNAME-like" in configuration, but return direct `A/AAAA` answers with the original owner name.

This is specifically to avoid relying on client-side CNAME chasing during SSH hostname resolution.

## Rationale (why a DNS proxy)

Earlier designs used a DNS sidecar and CNAME-based answers. In practice, relying on client-side CNAME chasing for SSH-style resolution proved unreliable across environments (including differences between libc stub resolvers).

The DNS proxy approach:

- Makes the bastion behavior deterministic by returning `A/AAAA` answers directly.
- Eliminates the need to generate sidecar-specific config files and reload a separate DNS daemon.
- Reduces the operational surface area (2 containers instead of 3).

## Operational notes

- Alias updates take effect without a DNS reload: the proxy reads alias state at query time.
- If DNS is exposed to the host for debugging (e.g. `5353/udp`), treat it as a development aid; the intended consumer is the `sshd` container.

## References

- [design-overview] - overall architecture
- [design-containers] - container/runtime topology

[design-overview]: ../design/design-overview.md
[design-containers]: ../design/design-containers.md
