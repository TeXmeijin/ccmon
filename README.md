# ccmon

Ambient TUI monitor for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) sessions.

Watch all your Claude Code sessions at a glance — which ones are running, which need permission, and what each is working on — without reading logs or switching panes.

<!-- TODO: Add demo GIF here -->
<!-- ![ccmon demo](./assets/demo.gif) -->

## Features

- **Card-based TUI** — Each Claude Code session gets a visual card with status, directory, and activity
- **Real-time updates** — Cards update as Claude works via command hooks
- **Status at a glance** — Color-coded badges: `RUN`, `WAIT`, `PERM`, `DONE`, `FAIL`, `IDLE`, `END`
- **Session title** — First user prompt shown alongside directory name for quick identification
- **Headline with source** — See who said what: `>` user, `<` assistant, `!` notification, `~` summary
- **Activity dots** — Recent event history as colored dots
- **Responsive layout** — Auto-switches between 1/2/3 column grid based on terminal width
- **Multi-config support** — Run separate instances for personal/work Claude configs via `CLAUDE_CONFIG_DIR`
- **Safe hook install** — Non-destructive merge into `settings.json`, preserving all existing hooks

## Install

```bash
go install github.com/TeXmeijin/ccmon@latest
```

Requires Go 1.21+ and CGO enabled (for SQLite).

## Quick Start

```bash
# 1. Install hooks into your Claude Code config
ccmon install

# 2. Launch the monitor
ccmon tui
```

That's it. Open Claude Code sessions in other terminals and watch them appear.

### With custom config directory

If you use `CLAUDE_CONFIG_DIR` to separate personal/work configs:

```bash
# Personal
ccmon install --config-dir ~/.claude-personal --source personal
ccmon tui --config-dir ~/.claude-personal --source personal

# Work
ccmon install --config-dir ~/.claude-work --source work
ccmon tui --config-dir ~/.claude-work --source work
```

## How It Works

```
Claude Code ──hook──> ccmon hook ──> SQLite
                                       │
                          ccmon tui <───┘
```

1. `ccmon install` adds [command hooks](https://docs.anthropic.com/en/docs/claude-code/hooks) to your Claude Code `settings.json`
2. When Claude Code fires events (tool use, notifications, stop, etc.), the hooks pipe JSON to `ccmon hook` via stdin
3. `ccmon hook` parses the payload and writes a lightweight summary to a local SQLite database
4. `ccmon tui` reads the database and renders the card grid, refreshing every 300ms

### What gets stored

- Session ID, working directory, status
- Tool names and short previews (file paths, truncated commands)
- Notification messages, first-line summaries

### What does NOT get stored

- Full conversation transcripts
- Complete file contents
- Full command output
- Tool response bodies

## CLI Reference

| Command | Description |
|---|---|
| `ccmon tui` | Launch the TUI monitor |
| `ccmon install` | Install hooks into Claude Code settings.json |
| `ccmon uninstall` | Remove only ccmon hooks (preserves other hooks) |
| `ccmon hook` | Process a hook event from stdin (called by Claude Code) |

### Global Flags

| Flag | Description | Default |
|---|---|---|
| `--config-dir` | Claude config directory | `$CLAUDE_CONFIG_DIR` or `~/.claude` |
| `--source` | Source namespace label | basename of config dir |
| `--db` | SQLite database path | `<config-dir>/ccmon/ccmon.db` |

## TUI Keys

| Key | Action |
|---|---|
| `q` / `Ctrl+C` | Quit |
| `j` / `k` / arrows | Move selection |
| `h` / `l` / arrows | Move selection horizontally |
| `g` / `G` | Jump to first / last |
| `r` | Force reload |

## Card Layout

```
╭──────────────────────────────────────────────╮
│ [RUN] ⠋ my-project  Add dark mode...    5m  │
│ > Add dark mode toggle to the header         │
│ Edit: src/components/Header.tsx               │
│ ● ● ● ● ● ● ● ●                   bcd94834 │
╰──────────────────────────────────────────────╯
```

- **Row 1**: Status badge + spinner + directory + session title + elapsed time
- **Row 2**: Headline with source prefix (`>` user / `<` assistant / `!` notification / `~` summary)
- **Row 3**: Current tool action
- **Row 4**: Activity dots + short session ID

## Demo

Run the included demo script to see ccmon in action with simulated sessions:

```bash
# Terminal 1: Start TUI
ccmon tui --config-dir /tmp/ccmon-demo --source demo

# Terminal 2: Run demo
./examples/demo.sh
```

## Uninstall

```bash
# Remove hooks from settings.json
ccmon uninstall

# Optionally delete the database
rm -rf ~/.claude/ccmon/
```

## License

MIT
