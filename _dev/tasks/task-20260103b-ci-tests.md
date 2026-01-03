---
id: task-20260103b-ci-tests
title: CI tests (GitHub Actions)
status: done
updated: 2026-01-03T07:14:29Z
---
# Task: CI tests (GitHub Actions)

## Goal

Add a GitHub Actions workflow that runs the project test suite on pushes and pull requests.

## Scope

### In

- Add GitHub Actions workflow(s) under `.github/workflows/`
- Run `make test`
- Trigger on:
  - `push`
  - `pull_request`

### Out

- Running docker-compose based E2E tests in CI (`make e2e`)
- Container build/push workflows

## Spec (summary)

- Roadmap item: “GitHub Actions: CI (tests)” in [design-roadmap].
- Unit tests should be runnable with `make test`.

## Plan & Checklist

- [x] Add workflow file (e.g. `.github/workflows/ci.yml`)
- [x] Use a Go toolchain compatible with `go.mod`
- [x] Run `make test`
- [x] Verify it triggers on PRs and pushes

## Progress

- 2026-01-03T06:57:40Z
  - Create task document

- 2026-01-03T07:00:28Z
  - Add GitHub Actions workflow to run `make test` on push/PR

- 2026-01-03T07:14:29Z
  - Confirm CI succeeded on `main` via `gh run view`

## References

- [design-roadmap] - Development Roadmap

[design-roadmap]: ../design/design-roadmap.md
