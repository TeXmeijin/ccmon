package cmd

import (
	"github.com/spf13/cobra"
)

var (
	flagConfigDir string
	flagSource    string
	flagDB        string
	flagProvider  string
)

var rootCmd = &cobra.Command{
	Use:   "ccmon",
	Short: "Claude Code / Codex session monitor TUI",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagProvider, "provider", "", "Session provider: claude or codex (default: auto)")
	rootCmd.PersistentFlags().StringVar(&flagConfigDir, "config-dir", "", "Provider config directory (Claude: $CLAUDE_CONFIG_DIR or ~/.claude; Codex: $CODEX_HOME or ~/.codex)")
	rootCmd.PersistentFlags().StringVar(&flagSource, "source", "", "Source namespace label (default: basename of config-dir)")
	rootCmd.PersistentFlags().StringVar(&flagDB, "db", "", "SQLite database path (default: <config-dir>/ccmon/ccmon.db)")
}

func Execute() error {
	return rootCmd.Execute()
}
