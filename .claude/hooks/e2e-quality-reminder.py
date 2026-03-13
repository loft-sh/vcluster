#!/usr/bin/env python3
"""
PostToolUse hook — remind agent to run the quality checklist after e2e-next
test runs via Justfile.agent.
"""

import json
import re
import sys


def main():
    try:
        data = json.load(sys.stdin)
    except (json.JSONDecodeError, AttributeError):
        sys.exit(0)

    command = data.get("tool_input", {}).get("command", "")

    # Match: just -f Justfile.agent test (with optional AGENT_SESSION prefix)
    # Use (?!\S) to avoid matching test-focus, test-anything, etc.
    if re.search(
        r"(just\s+-f\s+Justfile\.agent\s+test(?!\S)|AGENT_SESSION=\S+\s+just\s+-f\s+Justfile\.agent\s+test(?!\S))",
        command,
    ):
        print(json.dumps({
            "additionalContext": (
                "IMPORTANT: Before declaring this test done, verify your code against ALL 9 items in "
                ".claude/rules/e2e-quality-checklist.md (auto-loaded — already in context). Walk through each item "
                "against the code you wrote. Pay special attention to: cleanup error handling (no swallowed "
                "errors), DeferCleanup ordering, specific error assertions, and lint "
                "(run `just -f Justfile.agent lint ./e2e-next/...` — compile-check is NOT sufficient)."
            )
        }))


if __name__ == "__main__":
    main()
