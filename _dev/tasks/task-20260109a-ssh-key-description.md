---
id: task-20260109a-ssh-key-description
title: UI - SSH key description field
status: in-progress
updated: 2026-01-09T09:27:46Z
assistedBy: github/copilot (vscode) gpt-5.2
---

## Goal

Add an optional human-readable description for SSH keys so users can identify them (e.g., "work laptop").

## Non-Goals

- Editing descriptions after the key is added (add-only)
- Introducing a separate key-metadata database/table

## Specification

Specification is maintained in [design-ssh-keys].

## Plan & Checklist

- [x] Confirm maximum length for description (128)
- [ ] Update key add handler and templates to accept/display description
- [ ] Update authorized_keys generation to include sanitized comment
- [ ] Add unit tests for sanitization/length behavior
- [ ] Update roadmap entry if needed

## Progress

- 2026-01-09T09:13:57Z
  - Created task and captured initial spec decisions (add-only, store in authorized_keys comment, sanitize + length limit)

- 2026-01-09T09:16:42Z
  - Confirmed max length is 128; over-limit is an error

- 2026-01-09T09:24:04Z
  - Spec: clarified sanitization (whitespace normalization + control-char rejection)

- 2026-01-09T09:27:46Z
  - Moved spec into design doc (design-ssh-keys)

## References

- [design-ssh-keys] - SSH keys (UI + storage)

[design-ssh-keys]: ../design/design-ssh-keys.md
