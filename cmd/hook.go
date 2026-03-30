package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/TeXmeijin/ccmon/internal/config"
	"github.com/TeXmeijin/ccmon/internal/db"
	"github.com/TeXmeijin/ccmon/internal/hook"
)

var flagDebugDir string

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Process a provider hook event from stdin",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(flagConfigDir, flagSource, flagDB, flagProvider)
		if err != nil {
			return err
		}

		store, err := db.Open(cfg.DBPath)
		if err != nil {
			return fmt.Errorf("opening database: %w", err)
		}
		defer store.Close()

		bindingLogPath := filepath.Join(cfg.ConfigDir, "ccmon", "ghostty-binding.jsonl")
		return hook.Process(os.Stdin, store, cfg.Source, flagDebugDir, bindingLogPath)
	},
}

func init() {
	hookCmd.Flags().StringVar(&flagDebugDir, "debug-dir", "", "Directory to dump raw payloads for debugging")
	rootCmd.AddCommand(hookCmd)
}
