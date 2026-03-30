# ccmon

Ambient TUI monitor for [Claude Code](https://docs.anthropic.com/en/docs/claude-code) and [Codex](https://developers.openai.com/codex/hooks) sessions.

Watch your coding-agent sessions at a glance — which ones are running, which need permission, and what each is working on — without reading logs or switching panes.

<!-- TODO: Add demo GIF here -->
<!-- ![ccmon demo](./assets/demo.gif) -->

## Features

- **Card-based TUI** — Each Claude Code or Codex session gets a visual card with status, directory, and activity
- **Real-time updates** — Cards update via provider command hooks
- **Status at a glance** — Color-coded badges: `RUN`, `WAIT`, `PERM`, `DONE`, `FAIL`, `IDLE`, `END`
- **Session title** — First user prompt shown alongside directory name for quick identification
- **Headline with source** — See who said what: `>` user, `<` assistant, `!` notification, `~` summary
- **Activity dots** — Recent event history as colored dots
- **Responsive layout** — Auto-switches between 1/2/3 column grid based on terminal width
- **Multi-config support** — Run separate instances for personal/work configs via `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, or explicit flags
- **Safe hook install** — Non-destructive merge into the provider's hook config, preserving existing hooks

## Install

```bash
go install github.com/TeXmeijin/ccmon@latest
```

Requires Go 1.21+ and CGO enabled (for SQLite).

## Quick Start

```bash
# 1. Install hooks into your provider config
ccmon install

# 2. Launch the monitor
ccmon tui
```

That's it. Open Claude Code or Codex sessions in other terminals and watch them appear.

### Explicit provider selection

Claude remains the default when auto-detection is ambiguous. Use `--provider codex` to target Codex explicitly.

```bash
# Claude Code
ccmon install --provider claude
ccmon tui --provider claude

# Codex
ccmon install --provider codex
ccmon tui --provider codex
```

### With custom config directory

If you use custom config directories:

```bash
# Claude personal
ccmon install --config-dir ~/.claude-personal --source personal
ccmon tui --config-dir ~/.claude-personal --source personal

# Claude work
ccmon install --config-dir ~/.claude-work --source work
ccmon tui --config-dir ~/.claude-work --source work

# Codex personal
ccmon install --provider codex --config-dir ~/.codex-personal --source personal
ccmon tui --provider codex --config-dir ~/.codex-personal --source personal
```

## How It Works

```
Claude Code / Codex ──hook──> ccmon hook ──> SQLite
                                                │
                                   ccmon tui <───┘
```

1. `ccmon install` adds provider-specific command hooks
2. When the provider fires events (tool use, notifications, stop, etc.), the hooks pipe JSON to `ccmon hook` via stdin
3. `ccmon hook` parses the payload and writes a lightweight summary to a local SQLite database
4. `ccmon tui` reads the database and renders the card grid, refreshing every 300ms

### Provider notes

- Claude Code installs into `settings.json`.
- Codex installs into `hooks.json` and enables `codex_hooks = true` in `config.toml` if needed.
- Current Codex hook support tracks `SessionStart`, `UserPromptSubmit`, `PreToolUse`, `PostToolUse`, and `Stop`.
- Current Codex hook docs only expose `Bash` for `PreToolUse` / `PostToolUse`, so fine-grained `Read` / `Edit` / `Write` activity remains Claude-only for now.

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
| `ccmon install` | Install hooks into the selected provider config |
| `ccmon uninstall` | Remove only ccmon hooks (preserves other hooks) |
| `ccmon hook` | Process a provider hook event from stdin |

### Global Flags

| Flag | Description | Default |
|---|---|---|
| `--provider` | Session provider | auto |
| `--config-dir` | Provider config directory | Claude: `$CLAUDE_CONFIG_DIR` or `~/.claude`; Codex: `$CODEX_HOME` or `~/.codex` |
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
# Remove hooks from the provider config
ccmon uninstall --provider claude
ccmon uninstall --provider codex

# Optionally delete the database
rm -rf ~/.claude/ccmon/
rm -rf ~/.codex/ccmon/
```

## License

MIT
