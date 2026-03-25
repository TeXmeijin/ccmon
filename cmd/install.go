package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/TeXmeijin/ccmon/internal/config"
	"github.com/TeXmeijin/ccmon/internal/hook"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install ccmon hooks into Claude Code settings.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Resolve(flagConfigDir, flagSource, flagDB)

		// Resolve ccmon binary path
		binary, err := os.Executable()
		if err != nil {
			binary = "ccmon"
		}
		binary, _ = filepath.Abs(binary)

		if err := hook.Install(cfg.ConfigDir, cfg.Source, binary); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}

		fmt.Printf("Hooks installed into %s/settings.json\n", cfg.ConfigDir)
		fmt.Printf("  source: %s\n", cfg.Source)
		fmt.Printf("  db: %s\n", cfg.DBPath)
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove ccmon hooks from Claude Code settings.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Resolve(flagConfigDir, flagSource, flagDB)

		if err := hook.Uninstall(cfg.ConfigDir); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}

		fmt.Printf("Hooks removed from %s/settings.json\n", cfg.ConfigDir)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
