# vCluster — Agent Instructions

## E2E TDD Workflow

When doing test-driven development, ALWAYS follow the workflow in
[.claude/e2e-tdd-workflow.md](.claude/e2e-tdd-workflow.md).
Use the AGENT_SESSION env var to isolate your commands per session.

## CI / GitHub Actions

Shared CI logic lives in [`loft-sh/github-actions`](https://github.com/loft-sh/github-actions);
production repos reuse those actions rather than inlining shell. The
`github-actions-developer` skill (from `vcluster-skills-engineering`) auto-loads
when you touch `.github/` and carries the full convention.
