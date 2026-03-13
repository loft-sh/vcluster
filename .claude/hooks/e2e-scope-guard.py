#!/usr/bin/env python3
"""
Hook: Block file edits outside the plan-declared directories.
Fires on Edit and Write tool calls via PreToolUse.

Session resolution: reads session_id from the hook input JSON, looks up
AGENT_SESSION in .agent-scratchpad/.active-sessions, then reads
.agent-scratchpad/.scope-guard-<AGENT_SESSION>.

Guard is inactive when no scope-guard file is found. Exit 2 blocks the call.
"""

import json
import os
import sys


def resolve_agent_session(project_dir: str, session_id: str) -> str | None:
    """Look up AGENT_SESSION for the given Claude session ID."""
    active_sessions = os.path.join(project_dir, ".agent-scratchpad", ".active-sessions")
    try:
        with open(active_sessions) as f:
            for line in f:
                line = line.strip()
                if line.startswith(session_id + "="):
                    return line[len(session_id) + 1:]
    except FileNotFoundError:
        pass
    return None


def main():
    project_dir = os.path.abspath(os.environ.get("CLAUDE_PROJECT_DIR", "."))

    try:
        data = json.load(sys.stdin)
    except (json.JSONDecodeError, AttributeError):
        sys.exit(0)

    # Resolve which session's scope guard applies.
    session_id = data.get("session_id", "")
    agent_session = resolve_agent_session(project_dir, session_id) if session_id else None

    if agent_session:
        scope_file = os.path.join(project_dir, ".agent-scratchpad", f".scope-guard-{agent_session}")
    else:
        # Fallback for sessions not registered in .active-sessions.
        scope_file = os.path.join(project_dir, ".agent-scratchpad", ".scope-guard")

    try:
        with open(scope_file) as f:
            scope_guard = f.read().strip()
    except FileNotFoundError:
        sys.exit(0)

    if not scope_guard:
        sys.exit(0)

    file_path = data.get("tool_input", {}).get("file_path", "")
    if not file_path:
        sys.exit(0)

    # Resolve file_path: relative paths are anchored to project_dir.
    if os.path.isabs(file_path):
        abs_path = os.path.normpath(file_path)
    else:
        abs_path = os.path.normpath(os.path.join(project_dir, file_path))

    # .agent-scratchpad/ is always allowed.
    scratchpad = os.path.join(project_dir, ".agent-scratchpad") + os.sep
    if abs_path.startswith(scratchpad):
        sys.exit(0)

    # Check each allowed directory.
    for raw_dir in scope_guard.split(":"):
        raw_dir = raw_dir.strip()
        if not raw_dir:
            continue
        allowed = os.path.normpath(os.path.join(project_dir, raw_dir))
        if abs_path == allowed or abs_path.startswith(allowed + os.sep):
            sys.exit(0)

    pretty_allowed = ", ".join(d.strip() for d in scope_guard.split(":") if d.strip())
    print(f"Scope guard: '{file_path}' is outside the allowed directories.", file=sys.stderr)
    print(f"Allowed: {pretty_allowed}", file=sys.stderr)
    print("STOP — present the proposed change and reason to the user.", file=sys.stderr)
    print("To allow this directory: update '## Allowed Directories' in the plan, rewrite the scope-guard file, then proceed.", file=sys.stderr)
    sys.exit(2)


if __name__ == "__main__":
    main()
