#!/bin/bash
# ccmon demo script
# Feeds fake hook events to demonstrate all session states.
# Uses a completely isolated temp directory — never touches real config or data.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
CCMON="${SCRIPT_DIR}/../ccmon"
DEMO_DIR=$(mktemp -d /tmp/ccmon-demo.XXXXXX)

cleanup() {
  rm -rf "$DEMO_DIR"
  echo ""
  echo "Demo cleanup: removed $DEMO_DIR"
}
trap cleanup EXIT

echo "ccmon demo"
echo "Demo dir: $DEMO_DIR"
echo ""
echo "Start the TUI in another terminal:"
echo "  $CCMON tui --config-dir $DEMO_DIR --source demo"
echo ""
read -p "Press Enter when TUI is ready..."

send() {
  local json="$1"
  echo "$json" | "$CCMON" hook --source demo --config-dir "$DEMO_DIR"
}

S1="demo-sess-aaa-frontend"
S2="demo-sess-bbb-backend"
S3="demo-sess-ccc-docs"

echo "[0s] Session 1: frontend app starts"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"SessionStart\",\"cwd\":\"/home/dev/projects/acme-frontend\"}"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"UserPromptSubmit\",\"prompt\":\"Add dark mode toggle to the header component\"}"
sleep 2

echo "[2s] Session 2: backend API starts"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"SessionStart\",\"cwd\":\"/home/dev/projects/acme-api\"}"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"UserPromptSubmit\",\"prompt\":\"Fix the N+1 query in /users endpoint\"}"
sleep 1

echo "[3s] Session 1: reading files"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Read\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-frontend/src/components/Header.tsx\"},\"cwd\":\"/home/dev/projects/acme-frontend\"}"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Read\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-frontend/src/components/Header.tsx\"},\"cwd\":\"/home/dev/projects/acme-frontend\"}"
sleep 1

echo "[4s] Session 2: running query analysis"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"grep -rn 'User.find' app/controllers/\"},\"cwd\":\"/home/dev/projects/acme-api\"}"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"grep -rn 'User.find' app/controllers/\"},\"cwd\":\"/home/dev/projects/acme-api\"}"
sleep 2

echo "[6s] Session 1: editing component"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-frontend/src/components/Header.tsx\"},\"cwd\":\"/home/dev/projects/acme-frontend\"}"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-frontend/src/components/Header.tsx\"},\"cwd\":\"/home/dev/projects/acme-frontend\"}"
sleep 1

echo "[7s] Session 3: docs session starts"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"SessionStart\",\"cwd\":\"/home/dev/projects/acme-docs\"}"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"UserPromptSubmit\",\"prompt\":\"Review the API migration guide for v2\"}"
sleep 2

echo "[9s] Session 1: permission prompt!"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"npm run build\"},\"cwd\":\"/home/dev/projects/acme-frontend\"}"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"Notification\",\"notification_type\":\"permission_prompt\",\"message\":\"Claude needs your permission to use Bash\",\"cwd\":\"/home/dev/projects/acme-frontend\"}"
sleep 3

echo "[12s] Session 2: editing files"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-api/app/controllers/users_controller.rb\"},\"cwd\":\"/home/dev/projects/acme-api\"}"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Edit\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-api/app/controllers/users_controller.rb\"},\"cwd\":\"/home/dev/projects/acme-api\"}"
sleep 1

echo "[13s] Session 1: permission granted, running build"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"npm run build\"},\"cwd\":\"/home/dev/projects/acme-frontend\"}"
sleep 2

echo "[15s] Session 3: reading docs"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Read\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-docs/migration-v2.md\"},\"cwd\":\"/home/dev/projects/acme-docs\"}"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"PostToolUse\",\"tool_name\":\"Read\",\"tool_input\":{\"file_path\":\"/home/dev/projects/acme-docs/migration-v2.md\"},\"cwd\":\"/home/dev/projects/acme-docs\"}"
sleep 2

echo "[17s] Session 2: completed"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"Stop\",\"stop_reason\":\"end_turn\",\"last_assistant_message\":\"Fixed the N+1 query by adding eager loading with includes(:posts). The /users endpoint should now make 2 queries instead of N+1.\",\"cwd\":\"/home/dev/projects/acme-api\"}"
sleep 2

echo "[19s] Session 3: build failure"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"PreToolUse\",\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"mdbook build\"},\"cwd\":\"/home/dev/projects/acme-docs\"}"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"PostToolUseFailure\",\"tool_name\":\"Bash\",\"tool_input\":{\"command\":\"mdbook build\"},\"error\":\"exit code 1\",\"cwd\":\"/home/dev/projects/acme-docs\"}"
sleep 2

echo "[21s] Session 1: completed"
send "{\"session_id\":\"$S1\",\"hook_event_name\":\"Stop\",\"stop_reason\":\"end_turn\",\"last_assistant_message\":\"Dark mode toggle added to Header. The component now uses useTheme() hook and renders a sun/moon icon button.\",\"cwd\":\"/home/dev/projects/acme-frontend\"}"
sleep 3

echo "[24s] Session 3: stop with failure context"
send "{\"session_id\":\"$S3\",\"hook_event_name\":\"Stop\",\"stop_reason\":\"end_turn\",\"last_assistant_message\":\"The build failed due to a broken link in chapter 3. I've fixed the reference.\",\"cwd\":\"/home/dev/projects/acme-docs\"}"
sleep 2

echo "[26s] Session 2: ended"
send "{\"session_id\":\"$S2\",\"hook_event_name\":\"SessionEnd\",\"cwd\":\"/home/dev/projects/acme-api\"}"

echo ""
echo "Demo complete! The TUI should now show:"
echo "  - Session 1 (acme-frontend): DONE"
echo "  - Session 2 (acme-api): END"
echo "  - Session 3 (acme-docs): DONE"
echo ""
echo "Wait ~90s to see IDLE (stale) state, or press Ctrl+C to exit."
read -p "Press Enter to cleanup..."
