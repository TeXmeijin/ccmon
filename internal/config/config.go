package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ConfigDir string
	Source    string
	DBPath   string
}

// Resolve determines the config directory and DB path from CLI flags and environment.
func Resolve(flagConfigDir, flagSource, flagDB string) Config {
	configDir := flagConfigDir
	if configDir == "" {
		configDir = os.Getenv("CLAUDE_CONFIG_DIR")
	}
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".claude")
	}

	source := flagSource
	if source == "" {
		source = filepath.Base(configDir)
	}

	dbPath := flagDB
	if dbPath == "" {
		dbPath = filepath.Join(configDir, "ccmon", "ccmon.db")
	}

	return Config{
		ConfigDir: configDir,
		Source:    source,
		DBPath:   dbPath,
	}
}
