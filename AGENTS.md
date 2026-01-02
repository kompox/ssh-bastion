# ssh-bastion

## Docs for developers and maintainers

- Design docs: `_dev/design/`
- Task files: `_dev/tasks/`

Each document begins with YAML front matter with the following fields:

- id: Unique identifier for the document, equal to the filename without the extension
  - design: `design-<short-slug>`
  - tasks: `task-YYYYMMDDa-<short-slug>` (use `a`, `b`, ... to disambiguate multiple tasks on the same date)
- title: Title of the document
- status: Document status (draft, stable, deprecated, etc.)
- updated: Last updated timestamp (UTC, ISO 8601: YYYY-MM-DDTHH:MM:SSZ)
  - bash: `date -u +"%Y-%m-%dT%H:%M:%SZ"`
  - pwsh: `(Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss'Z'")`

## Design docs

- Use design docs to describe high-level architecture, components, and decisions.
- Use design docs to draft and review designs before implementation. Use `status:` in the front matter to indicate the maturity of the design.

## Task files

- Use task files to plan and track specific work items, such as features, bug fixes, or improvements.
- Use an independent task file for each work item.
- Keep the work progress tracked in task files:
  - Update the `## Plan & Checklist` section with checkboxes as work is completed. If you add tasks later, add them as unchecked items.
  - Update the `## Progress` section with a timestamped entry as work progresses. Do not change existing entries.
