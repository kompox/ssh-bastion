---
id: task-20260103c-docker-push
title: Docker build and push (GitHub Actions)
status: done
updated: 2026-01-03T08:08:09Z
---
# Task: Docker build and push (GitHub Actions)

## Goal

Build and push the container image `ghcr.io/kompox/ssh-bastion` via GitHub Actions.

## Scope

### In

- GitHub Actions workflow under `.github/workflows/`
- Trigger on:
  - `push` to `main`
  - `push` of tags matching `v*`
- Build and push a multi-architecture image using QEMU:
  - `linux/amd64`
  - `linux/arm64`
- Push image to `ghcr.io/kompox/ssh-bastion`

### Out

- Release tagging/versioning strategy
- Signing/attestations (cosign, provenance)

## Spec (summary)

- Roadmap item: “GitHub Actions: Docker build and push” in [design-roadmap].

## Plan & Checklist

- [x] Define image tags:
  - `main` on pushes to `main`
  - `latest` on pushes of `v*` tags
  - `v*` ref tag
- [x] Add workflow with build+push steps
- [x] Set up QEMU + Buildx in workflow
- [x] Use `GITHUB_TOKEN` and `packages: write` permissions (or PAT if required)
- [x] Ensure the Dockerfile build is reproducible in Actions
- [x] Verify the image appears in GHCR after a `main` push and a `v*` tag push

## Progress

- 2026-01-03T07:19:41Z
  - Create task document and move roadmap item to IN-PROGRES

- 2026-01-03T07:28:07Z
  - Update spec: multi-arch build via QEMU (linux/amd64, linux/arm64) and tag triggers (v*)

- 2026-01-03T07:34:39Z
  - Update spec: tag policy (`main` for main branch, `latest` for v* tags; no sha tags)

- 2026-01-03T08:08:09Z
  - Confirm workflow succeeded and `ghcr.io/kompox/ssh-bastion:main` is published

## References

- [design-roadmap] - Development Roadmap
- Reference workflow: https://github.com/yaegashi/p4p-docker/blob/master/.github/workflows/build.yml

[design-roadmap]: ../design/design-roadmap.md
