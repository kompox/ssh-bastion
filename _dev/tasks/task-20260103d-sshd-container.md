---
id: task-20260103d-sshd-container
title: SSHD container (local compose)
status: in-progress
updated: 2026-01-03T08:17:07Z
---
# Task: SSHD container (local compose)

## Goal

Add an `sshd` container to the local runtime topology, using the existing image and shared `/data` volume, as the next step toward the target multi-container Pod model.

## Scope

### In

- Implement a dedicated `sshd` container in local docker-compose.
- Use the same published image (`ghcr.io/kompox/ssh-bastion`) and shared `/data` volume.
- Define how `sshd` reads authorized keys from `/data` (compatible with the existing “generate files under /data” approach).

### Out

- Production hardening (securityContext/capabilities, read-only rootfs, etc.).
- Kubernetes manifests/Helm.
- SSH forwarding policy and restrictions.

## Spec (summary)

- Roadmap item: “Container development and local testing” in [design-roadmap].
- Container topology and conventions: [design-containers].

## Plan & Checklist

- [ ] Decide file locations under `/data` for `authorized_keys` and host keys
- [ ] Add compose service for `sshd` (from the same image) with correct mounts
- [ ] Ensure `sshd` can start reliably (host keys present/generated)
- [ ] Ensure the `authorized_keys` path wiring matches the future generator behavior
- [ ] Document how to run/test locally (minimal)

## Progress

- 2026-01-03T08:17:07Z
  - Create task document and add roadmap entry

## References

- [design-roadmap] - Development Roadmap
- [design-containers] - Containers (image + runtime topology)

[design-roadmap]: ../design/design-roadmap.md
[design-containers]: ../design/design-containers.md
