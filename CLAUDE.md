# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

ccmon is an ambient TUI monitor for Claude Code sessions. It uses Claude Code's command hooks to capture session events into SQLite, then renders a real-time card grid via Bubble Tea. Written in Go with CGO (for go-sqlite3).

## Build & Run

```bash
# Build (CGO required for SQLite)
go build -o ccmon .

# Run TUI
./ccmon tui

# Install hooks into Claude Code settings.json
./ccmon install

# Process a hook event from stdin (called by Claude Code, not manually)
./ccmon hook

# Remove hooks
./ccmon uninstall
```

There are no tests yet. No linter is configured.

## Architecture

### Data Flow

```
Claude Code hooks → ccmon hook (stdin JSON) → SQLite → ccmon tui (300ms poll)
```

### Package Structure

- `cmd/` — Cobra CLI commands (`root`, `tui`, `hook`, `install`/`uninstall`). Each file registers its command in `init()`.
- `internal/config/` — Resolves config dir, source namespace, and DB path from flags/env (`CLAUDE_CONFIG_DIR`).
- `internal/hook/hook.go` — Parses Claude Code hook JSON payloads, extracts preview/headline/status, upserts session and inserts event into DB.
- `internal/hook/install.go` — Merges/removes ccmon hook blocks in `settings.json`. Uses `__ccmon__` marker key to identify owned blocks. Creates `.bak` backup before modifying.
- `internal/db/` — SQLite store with WAL mode. Two tables: `sessions` (keyed by `source_namespace + session_id`) and `events`. Schema migrations are inline in `migrate()` using idempotent `ALTER TABLE ADD COLUMN`.
- `internal/model/` — Domain types (`Session`, `Event`, `Status`, `DotKind`, `HeadlineSource`), status state machine (`TransitionStatus`), and view model builder (`BuildCardVM`). Stale detection: sessions with no events for 90s become `StatusStale`.
- `internal/tui/` — Bubble Tea TUI. `tui.go` is the main model (Init/Update/View loop), `card.go` renders individual session cards, `styles.go` defines the Tokyo Night-inspired color palette.

### Key Design Decisions

- **Source namespace**: Sessions are scoped by namespace (derived from `--source` flag or config dir basename), allowing multiple ccmon instances for different Claude configs (personal/work).
- **Upsert semantics**: `UpsertSession` uses SQL `CASE` expressions to preserve first-set values (e.g., `session_title` from first `UserPromptSubmit`) and avoid overwriting with empty strings.
- **Ghostty integration**: On `SessionStart`, captures the Ghostty terminal ID via AppleScript. TUI's Enter key / mouse click focuses the corresponding Ghostty pane.
- **Hook block ownership**: `install.go` marks hook blocks with `"__ccmon__": true` so `uninstall` only removes ccmon's hooks without touching user-defined ones.
