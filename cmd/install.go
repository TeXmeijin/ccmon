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
	Short: "Install ccmon hooks into the provider config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(flagConfigDir, flagSource, flagDB, flagProvider)
		if err != nil {
			return err
		}

		// Resolve ccmon binary path
		binary, err := os.Executable()
		if err != nil {
			binary = "ccmon"
		}
		binary, _ = filepath.Abs(binary)

		if err := hook.Install(cfg.Provider, cfg.ConfigDir, cfg.Source, binary); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}

		fmt.Printf("Hooks installed for %s into %s\n", cfg.Provider, cfg.HookConfigPath())
		fmt.Printf("  source: %s\n", cfg.Source)
		fmt.Printf("  db: %s\n", cfg.DBPath)
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove only ccmon hooks from the provider config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Resolve(flagConfigDir, flagSource, flagDB, flagProvider)
		if err != nil {
			return err
		}

		if err := hook.Uninstall(cfg.Provider, cfg.ConfigDir); err != nil {
			return fmt.Errorf("uninstall failed: %w", err)
		}

		fmt.Printf("Hooks removed for %s from %s\n", cfg.Provider, cfg.HookConfigPath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
	rootCmd.AddCommand(uninstallCmd)
}
