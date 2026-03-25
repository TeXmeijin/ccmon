package cmd

import (
	"github.com/spf13/cobra"
)

var (
	flagConfigDir string
	flagSource    string
	flagDB        string
)

var rootCmd = &cobra.Command{
	Use:   "ccmon",
	Short: "Claude Code session monitor TUI",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flagConfigDir, "config-dir", "", "Claude config directory (default: $CLAUDE_CONFIG_DIR or ~/.claude)")
	rootCmd.PersistentFlags().StringVar(&flagSource, "source", "", "Source namespace label (default: basename of config-dir)")
	rootCmd.PersistentFlags().StringVar(&flagDB, "db", "", "SQLite database path (default: <config-dir>/ccmon/ccmon.db)")
}

func Execute() error {
	return rootCmd.Execute()
}
