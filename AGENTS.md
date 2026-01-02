# ssh-bastion

## Docs for developers and maintainers

- Design docs: `_dev/design/`
- Task files: `_dev/tasks/`

Each document begins with YAML front matter with the following fields:

- id: Unique identifier for the document, equal to the filename without the extension
  - design: `design-<short-slug>`
  - tasks: `task-YYYYMMDDa-<short-slug>` (use `a`, `b`, ... to disambiguate multiple tasks on the same date)
- title: Title of the document
- status: Document status (see below)
- updated: Last updated timestamp (UTC, ISO 8601: YYYY-MM-DDTHH:MM:SSZ)
  - bash: `date -u +"%Y-%m-%dT%H:%M:%SZ"`
  - pwsh: `(Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ss'Z'")`

## Design docs

- Use design docs to describe high-level architecture, components, and decisions.
- Use design docs to draft and review designs before implementation. Use `status:` in the front matter to indicate the maturity of the design.
- Possible statuses:
  - draft: Initial design proposal, subject to change.
  - stable: Design has been reviewed and approved; implementation can proceed.
  - deprecated: Design is outdated or replaced; do not implement new work based on this design.

## Task files

- Use task files to plan and track specific work items, such as features, bug fixes, or improvements.
- Use an independent task file for each work item.
- Keep the work progress tracked in task files:
  - Update the `## Plan & Checklist` section with checkboxes as work is completed. If you add tasks later, add them as unchecked items.
  - Update the `## Progress` section with a timestamped entry as work progresses. Do not change existing entries.
- Possible statuses:
  - todo: Initial task definition, may evolve.
  - in-progress: Work on the task has started.
  - done: Task is complete and verified.
  - blocked: Task cannot proceed due to external dependencies or issues.
  - cancelled: Task has been abandoned and will not be completed.