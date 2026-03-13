# Task Context

You are currently working on **{{.TaskID}}: {{.Title}}**

## Task Description

{{.Description}}

## Acceptance Criteria

{{range .AcceptanceCriteria}}
- [ ] {{.}}
{{else}}
- [ ] Define acceptance criteria
{{end}}

## Current Status

**Status:** {{.Status}}
**Created:** {{.CreatedAt}}
{{if .UpdatedAt}}**Updated:** {{.UpdatedAt}}{{end}}

## Workspace

All work for this task should be done within the `tickets/{{.TaskID}}/` directory.

- **status.yaml** - Task metadata and status
- **context.md** - Detailed task context and requirements
- **notes.md** - Working notes and observations
- **design.md** - Design decisions and architecture
- **sessions/** - Session logs and artifacts
- **knowledge/** - Knowledge base and decisions

## Important

Always refer to the task directory structure when working on this task. Update documentation as you make progress.
