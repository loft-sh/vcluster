#!/usr/bin/env python3
"""
PostToolUse hook — remind agent to update the state file after SP completion points.
Fires after: just -f Justfile.agent test, test-focus, compile-check
"""

import json
import os
import re
import sys


def resolve_agent_session(project_dir: str, data: dict) -> str:
    """
    Resolve AGENT_SESSION via three methods in priority order:
    1. .active-sessions registry (session_id -> AGENT_SESSION)
    2. Inline env prefix in the bash command (AGENT_SESSION=foo just ...)
    3. AGENT_SESSION process environment variable (fallback)
    """
    session_id = data.get("session_id", "")
    if session_id:
        active_sessions = os.path.join(project_dir, ".agent-scratchpad", ".active-sessions")
        try:
            with open(active_sessions) as f:
                for line in f:
                    line = line.strip()
                    if line.startswith(session_id + "="):
                        return line[len(session_id) + 1:]
        except FileNotFoundError:
            pass

    command = data.get("tool_input", {}).get("command", "")
    m = re.search(r"AGENT_SESSION=(\S+)", command)
    if m:
        return m.group(1)

    return os.environ.get("AGENT_SESSION", "")


def main():
    try:
        data = json.load(sys.stdin)
    except (json.JSONDecodeError, AttributeError):
        sys.exit(0)

    command = data.get("tool_input", {}).get("command", "")

    # Match SP completion points: test, test-focus, compile-check
    if not re.search(r"just\s+-f\s+Justfile\.agent\s+(test(-focus)?|compile-check)", command):
        sys.exit(0)

    project_dir = os.path.abspath(os.environ.get("CLAUDE_PROJECT_DIR", "."))
    agent_session = resolve_agent_session(project_dir, data)
    if not agent_session:
        sys.exit(0)

    state_file = os.path.join(project_dir, ".agent-scratchpad", f"state-{agent_session}.md")

    if not os.path.isfile(state_file):
        sys.exit(0)

    content = open(state_file).read()

    if "## Status: COMPLETED" in content:
        sys.exit(0)

    if "PENDING" not in content:
        sys.exit(0)

    # Find the first pending SP for a specific nudge
    match = re.search(r"(SP-\d+)[^\n]*PENDING", content)
    pending_sp = match.group(1) if match else "the current SP"

    print(json.dumps({
        "additionalContext": (
            f"STATE FILE UPDATE REQUIRED: Before proceeding, update {state_file} — "
            f"mark {pending_sp} as ✓ PASSED or ✗ FAILED, update ## Progress → Current SP, "
            f"and ## Next Steps. Do this now, then continue to the next sub-problem."
        )
    }))


if __name__ == "__main__":
    main()
