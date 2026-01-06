---
id: design-ssh-forwarding
title: SSH forwarding and sshd process lifecycle
status: draft
updated: 2026-01-06T22:47:34Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Design: SSH forwarding and sshd process lifecycle

## Overview

This document specifies how ssh-bastion hardens SSH TCP port forwarding by restricting destinations using `sshd_config` `PermitOpen`, including the sshd process lifecycle required to apply changes.

Primary goals:

- Block arbitrary destination IP/ports in SSH TCP forwarding.
- Support non-22 tunnels (e.g., database access).
- Allow switching behavior during live operations, persisted under `${SSHBASTION_DATA_DIR}`.

Assumptions:

- The bastion does not provide shell access.
- The allowlist is global (not per-user/role).

## Admin UI

- Route: `GET /admin/targets`
- Auth: admin-only.
- UI style: guided forms (no textarea).

Page behavior:

- Global mode selector: `any` / `none` / `custom`.
- Custom rules list with per-rule `enable` / `disable` / `remove` actions (same interaction model as SSH keys pages).

## Global mode

A single global switch controls enforcement mode:

- `any` (default): remove all restrictions and permit any TCP forwarding requests.
- `none`: prohibit all TCP forwarding requests.
- `custom`: allow only destinations matching enabled custom target rules.

Rationale:

- Avoid ambiguity from mixing reserved words (`any`/`none`) with `host:port` rules.
- Make the operator intent obvious.

## Custom target rules

Each custom rule is a single destination pattern in `host:port` form:

- `host` may be a DNS name, an IPv4 literal, an IPv6 literal in brackets, or `*`.
- `port` may be a number (1-65535) or `*`.

Examples:

- `db.example.com:5432`
- `10.0.0.10:22`
- `[fd00::1]:5432`
- `*.example.com:5432`
- `db.example.com:*`
- `*:*`

## PermitOpen behavior notes

From `man sshd_config`:

- Multiple `PermitOpen` arguments may be specified by separating them with whitespace.
- Reserved words:
  - `any`: remove all restrictions and permit any forwarding requests.
  - `none`: prohibit all forwarding requests.
- Wildcards:
  - `*` may be used for host or port to allow all hosts or all ports respectively.
  - Otherwise, no pattern matching or address lookups are performed on supplied names.

## Persistence under /data

The SSH forwarding configuration is persisted under `${SSHBASTION_DATA_DIR}` so operators can switch behavior during live operations (e.g., before/after maintenance).

Source of truth:

- `${SSHBASTION_DATA_DIR}/ssh/forwarding.json`

Format (JSON):

```json
{
  "mode": "any",
  "targets": [
    {
      "rule": "db.example.com:5432",
      "enabled": true
    }
  ],
  "restartGeneration": 0
}
```

Defaults:

- If the file does not exist: treat as `mode=any`, `targets=[]`, `restartGeneration=0`.

## sshd lifecycle (propagation, reload, restart)

The sshd container must pick up changes without requiring a Pod restart.

Constraints:

- `${SSHBASTION_DATA_DIR}` may be backed by a remote filesystem (e.g., Azure Files), so relying on inotify-style events is not sufficient.

Approach:

- Poll the persisted configuration in the sshd container.
- Default maximum propagation delay: 5 seconds.

Apply actions:

- **Reload**: generate sshd_config, validate it (e.g., `sshd -t`), then send SIGHUP to the sshd master process to reload.
- **Restart**: support a persisted restart trigger to forcefully disconnect all existing sessions.
  - UI for restart is out of scope.
  - Trigger mechanism: increment `restartGeneration` in `${SSHBASTION_DATA_DIR}/ssh/forwarding.json`.
  - If implemented via container restart (terminate sshd master), ensure the orchestrator restarts the sshd container correctly:
    - Kubernetes: standard Deployment-managed Pods restart crashed containers.
    - docker compose: configure an appropriate restart policy.

## Validation and injection safety

Custom rules must be validated to avoid `sshd_config` injection and ambiguous parsing.

Requirements:

- Reject empty rules.
- Custom rules accept only `host:port` (or bracketed IPv6 `[addr]:port`).
- Reserved words `any`/`none` are not valid custom rules; they are represented only via the global mode switch.
- Reject whitespace characters in rules (spaces, tabs, newlines).
- Reject control characters (including `\r`/`\n`).
- Reject `#` (comment marker).
- Enforce a conservative max length (TBD).

## References

- [task-20260106a-ssh-forwarding-hardening](../tasks/task-20260106a-ssh-forwarding-hardening.md)
- [design-roadmap](design-roadmap.md)
- [design-containers](design-containers.md)
