#!/usr/bin/env python3
"""
Hook: Validate migration plan completeness before writing.
Fires on Write tool calls via PreToolUse.

When the agent writes a migration-*.md file, spawns `claude -p` to review
the plan for completeness. Blocks the write if the review fails.

Failure conditions:
- Bootstrap Requirements: no without a concrete provisioning design
- Deferral language in Bootstrap section ("implementing agent should", TBD, etc.)
- Missing required sections
- Validation section has no `just -f Justfile.agent` command
"""

import json
import os
import re
import subprocess
import sys

REVIEW_PROMPT = """\
You are a migration plan reviewer for e2e test migrations to the e2e-next framework.
Analyze the migration plan below and return ONLY a JSON object — no prose, no markdown, just the JSON.

Check ALL of the following:

1. **Required sections present**: The plan must contain all of:
   - ## Bootstrap Requirements
   - ## Sub-Problems
   - ## Structure
   - ## Allowed Directories
   - ## Validation

2. **Bootstrap completeness**: If the Bootstrap Requirements section says \
"Standard bootstrap works: no" (or equivalent), it must contain a concrete \
provisioning design — references to BeforeAll, setuphelm, Upgrade, port-forward, \
cluster setup, chart installation, or equivalent framework primitives. \
Applying `labels.NonDefault` alone is NOT a provisioning plan.

3. **No deferral language**: The Bootstrap Requirements section must NOT contain \
phrases like "implementing agent should", "implementing agent must", \
"must verify that", "left to the implementing agent", "TBD", "to be determined", \
"unclear", or "defer to". Gaps must be resolved in the plan, not passed on.

4. **Infra sub-problem or appendix covers provisioning**: If bootstrap is \
non-standard, there must be either a [infra] sub-problem addressing service \
installation OR an ## Appendix section with a provisioning design.

5. **CI orchestration consulted** (only when bootstrap is non-standard): If the \
Bootstrap Requirements section indicates non-standard bootstrap (external services \
required), check whether `.github/workflows/e2e.yaml` or the old test's `values.yaml` \
is mentioned anywhere in the plan (Problem Summary, Bootstrap Requirements, Design \
Decisions, or Sub-Problems). Old tests with external service dependencies relied on \
CI pre-provisioning that the test file alone does not reveal. If neither \
`.github/workflows/e2e.yaml` nor `values.yaml` is mentioned and bootstrap is \
non-standard, flag as error: "Non-standard bootstrap requires consulting \
.github/workflows/e2e.yaml and the old test's values.yaml — CI-provisioned \
infrastructure may be missing from the plan."

6. **Validation command present**: The ## Validation section must contain a \
`just -f Justfile.agent` command.

Return ONLY this JSON (no other text):
{"pass": true, "findings": []}
or
{"pass": false, "findings": [{"severity": "error", "check": "<name>", "detail": "<specific issue>"}]}

Use severity "error" for blocking issues (plan must not proceed), "warn" for suggestions.

Plan to review:
"""


def is_migration_plan(file_path: str) -> bool:
    project_dir = os.path.abspath(os.environ.get("CLAUDE_PROJECT_DIR", "."))
    abs_path = os.path.normpath(os.path.join(project_dir, file_path) if not os.path.isabs(file_path) else file_path)
    scratchpad = os.path.join(project_dir, ".agent-scratchpad")
    basename = os.path.basename(abs_path)
    return abs_path.startswith(scratchpad + os.sep) and basename.startswith("migration-") and basename.endswith(".md")


def run_review(content: str) -> dict:
    prompt = REVIEW_PROMPT + content
    try:
        result = subprocess.run(
            ["claude", "-p", prompt, "--model", "claude-haiku-4-5-20251001"],
            capture_output=True,
            text=True,
            timeout=120,  # 30s under hook timeout to ensure clean exit
        )
    except subprocess.TimeoutExpired:
        # Timeout — warn but allow rather than block indefinitely
        return {
            "pass": True,
            "findings": [{"severity": "warn", "check": "timeout", "detail": "Review timed out — plan write allowed. Run validate manually."}],
        }
    except FileNotFoundError:
        # claude not on PATH — allow
        return {"pass": True, "findings": []}

    output = result.stdout.strip()
    match = re.search(r"\{.*\}", output, re.DOTALL)
    if not match:
        return {
            "pass": True,
            "findings": [{"severity": "warn", "check": "parse", "detail": "Could not parse review response — plan write allowed."}],
        }

    try:
        return json.loads(match.group())
    except json.JSONDecodeError:
        return {
            "pass": True,
            "findings": [{"severity": "warn", "check": "parse", "detail": "Could not parse review JSON — plan write allowed."}],
        }


def format_output(review: dict) -> str:
    findings = review.get("findings", [])
    errors = [f for f in findings if f.get("severity") == "error"]
    warnings = [f for f in findings if f.get("severity") == "warn"]

    lines = ["Migration plan review FAILED. Resolve these issues before writing:\n"]
    for f in errors:
        lines.append(f"  [ERROR] {f.get('check', 'unknown')}: {f.get('detail', '')}")
    for f in warnings:
        lines.append(f"  [WARN]  {f.get('check', 'unknown')}: {f.get('detail', '')}")
    lines.append(
        "\nFix the plan and retry. See .claude/references/e2e-framework-suite-deps.md "
        "for the self-contained provisioning pattern."
    )
    return "\n".join(lines)


def main():
    try:
        data = json.load(sys.stdin)
    except (json.JSONDecodeError, AttributeError):
        sys.exit(0)

    file_path = data.get("tool_input", {}).get("file_path", "")
    if not file_path or not is_migration_plan(file_path):
        sys.exit(0)

    content = data.get("tool_input", {}).get("content", "")
    if not content:
        sys.exit(0)

    review = run_review(content)

    findings = review.get("findings", [])
    has_errors = any(f.get("severity") == "error" for f in findings)
    has_warnings = any(f.get("severity") == "warn" for f in findings)

    if has_warnings and not has_errors:
        # Print warnings as additionalContext but allow the write
        print(json.dumps({"additionalContext": format_output(review).replace("FAILED", "passed with warnings")}))
        sys.exit(0)

    if not review.get("pass", True) or has_errors:
        print(format_output(review), file=sys.stderr)
        sys.exit(2)

    sys.exit(0)


if __name__ == "__main__":
    main()
