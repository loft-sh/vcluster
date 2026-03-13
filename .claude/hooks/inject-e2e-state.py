#!/usr/bin/env python3
"""
Hook: Inject E2E implementation state after context compaction.
Fires on SessionStart with trigger=compact. Reads the state file for the
current session and outputs it as additionalContext so the agent resumes
with full situational awareness.
"""

import json
import os
import sys


def main():
    try:
        data = json.load(sys.stdin)
    except (json.JSONDecodeError, AttributeError):
        sys.exit(0)

    # Only act on post-compaction session starts
    if data.get("trigger") != "compact":
        sys.exit(0)

    session_id = data.get("session_id", "")
    if not session_id:
        sys.exit(0)

    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", "")
    if not project_dir:
        sys.exit(0)

    active_sessions = os.path.join(project_dir, ".agent-scratchpad", ".active-sessions")
    if not os.path.isfile(active_sessions):
        sys.exit(0)

    # Find the AGENT_SESSION name for this Claude session ID
    agent_session = None
    with open(active_sessions) as f:
        for line in f:
            line = line.strip()
            if line.startswith(session_id + "="):
                agent_session = line[len(session_id) + 1:]
                break

    if not agent_session:
        sys.exit(0)

    state_file = os.path.join(project_dir, ".agent-scratchpad", f"state-{agent_session}.md")
    if not os.path.isfile(state_file):
        sys.exit(0)

    content = open(state_file).read()

    # Skip if already completed
    if "## Status: COMPLETED" in content:
        sys.exit(0)

    print(json.dumps({
        "hookSpecificOutput": {
            "hookEventName": "SessionStart",
            "additionalContext": (
                f"=== E2E IMPLEMENTATION STATE (restored after compaction) ===\n"
                f"{content}\n"
                f"=== END STATE ==="
            ),
        }
    }))


if __name__ == "__main__":
    main()
