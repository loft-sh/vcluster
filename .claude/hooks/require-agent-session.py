#!/usr/bin/env python3
"""
Hook: Require AGENT_SESSION for all Justfile.agent commands.
Prevents cross-contamination between parallel agents sharing the same
image tag, build artifact, and report file.

Side effect: writes session_id=AGENT_SESSION to .active-sessions so that
other hooks (inject-e2e-state, e2e-scope-guard, update-state-reminder) can
resolve AGENT_SESSION from the Claude session ID without relying on env vars.
"""

import json
import os
import re
import sys


def main():
    try:
        data = json.load(sys.stdin)
    except (json.JSONDecodeError, AttributeError):
        sys.exit(0)

    command = data.get("tool_input", {}).get("command", "")

    # Only check commands that invoke Justfile.agent
    if not re.search(r"just\s+(-f\s+Justfile\.agent|--justfile\s+Justfile\.agent)", command):
        sys.exit(0)

    # Extract AGENT_SESSION from inline prefix (AGENT_SESSION=foo just ...) or env
    m = re.search(r"AGENT_SESSION=(\S+)", command)
    agent_session = m.group(1) if m else os.environ.get("AGENT_SESSION", "")

    if not agent_session:
        print("AGENT_SESSION is not set. Export it before running Justfile.agent commands:", file=sys.stderr)
        print('  export AGENT_SESSION="<unique-name>"', file=sys.stderr)
        sys.exit(2)

    # Register session_id -> AGENT_SESSION mapping for hook lookups.
    session_id = data.get("session_id", "")
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", "")
    if session_id and project_dir:
        active_sessions = os.path.join(project_dir, ".agent-scratchpad", ".active-sessions")
        os.makedirs(os.path.dirname(active_sessions), exist_ok=True)

        # Upsert: remove any existing entry for this session, then append fresh one.
        lines: list[str] = []
        if os.path.isfile(active_sessions):
            with open(active_sessions) as f:
                lines = [l for l in f.readlines() if not l.startswith(session_id + "=")]

        lines.append(f"{session_id}={agent_session}\n")
        with open(active_sessions, "w") as f:
            f.writelines(lines)

    sys.exit(0)


if __name__ == "__main__":
    main()
