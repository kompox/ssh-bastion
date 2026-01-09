---
id: design-ssh-keys
title: SSH keys (UI + storage)
status: draft
updated: 2026-01-09T09:27:46Z
assistedBy: github/copilot (vscode) gpt-5.2
---

# Design: SSH keys (UI + storage)

This document defines the UI and persistence rules for SSH public key management.
Routes and access control are described in [design-app-http].

## Goals

- Provide a simple UI for users to add and manage their SSH public keys.
- Generate an `authorized_keys` file consumed by `sshd`.
- Support an optional, human-readable key description to help users identify keys.

## Non-Goals

- Editing descriptions after a key is added (add-only)
- Introducing a separate key-metadata database/table

## UI specification

### Add key

- The add form includes:
  - `publicKey` (required)
  - `description` (optional)
- Description is set on add only (no edit UI / endpoint).

### List keys

- Key listing displays the description (if present).

## Persistence / file format

### authorized_keys generation

- The system generates `authorized_keys` for the `jump` user.
- For a key with a non-empty description, the description is stored as the comment at the end of the authorized_keys line.
- No separate metadata store is introduced for descriptions.

## Description sanitization and validation

### Sanitization

- Trim surrounding whitespace.
- Whitespace normalization: replace any run of whitespace (spaces, tabs, newlines) with a single ASCII space (`' '`).

### Validation

- Reject any control characters (including newlines). If any are present, return an error.
- Maximum length: 128 characters after sanitization.
  - If length exceeds 128, return an error (do not truncate).
- Empty result is allowed (treat as no description / no comment).

## References

- [design-overview] - Design overview document
- [design-app-http] - App HTTP (Routes & Sitemap)

[design-overview]: design-overview.md
[design-app-http]: design-app-http.md
