package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/TeXmeijin/ccmon/internal/config"
	"github.com/TeXmeijin/ccmon/internal/db"
	"github.com/TeXmeijin/ccmon/internal/hook"
)

var flagDebugDir string

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Process a Claude Code hook event from stdin",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Resolve(flagConfigDir, flagSource, flagDB)

		store, err := db.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer store.Close()

		return hook.Process(os.Stdin, store, cfg.Source, flagDebugDir)
	},
}

func init() {
	hookCmd.Flags().StringVar(&flagDebugDir, "debug-dir", "", "Directory to dump raw payloads for debugging")
	rootCmd.AddCommand(hookCmd)
}
