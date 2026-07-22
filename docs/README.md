# Project knowledge base

This directory is the versioned source of truth for project knowledge that an
engineer or coding agent needs beyond the local code. Keep it concise,
current, and linked from this index; avoid duplicating the same rule in several
documents.

## Read by task

| Need | Read |
| --- | --- |
| Package ownership or architectural decision | [Architecture](ARCHITECTURE.md) and [root architecture map](../ARCHITECTURE.md) |
| Runtime inputs, Docker, streams, or incident response | [Operations](OPERATIONS.md) |
| Tests, checks, and completion criteria | [Quality](QUALITY.md) |
| A complex or ongoing change | [Execution plans](plans/README.md) |

## Maintenance rules

- Update the document that owns a changed decision in the same pull request.
- Prefer links to code and a short statement of the invariant over a long
  narrative that can drift.
- Put active decisions and progress logs in `plans/active/`; retain useful
  history in `plans/completed/`.
- `make docs-check` verifies this index and the required document structure.
