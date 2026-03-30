package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Provider  Provider
	ConfigDir string
	Source    string
	DBPath    string
}

// Resolve determines the config directory and DB path from CLI flags and environment.
func Resolve(flagConfigDir, flagSource, flagDB, flagProvider string) (Config, error) {
	provider, err := ParseProvider(flagProvider)
	if err != nil {
		return Config{}, err
	}
	if provider == "" {
		provider = inferProvider(flagConfigDir)
	}

	configDir := flagConfigDir
	if configDir == "" {
		configDir = defaultConfigDir(provider)
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
		Provider:  provider,
		ConfigDir: configDir,
		Source:    source,
		DBPath:    dbPath,
	}, nil
}

func (c Config) HookConfigPath() string {
	switch c.Provider {
	case ProviderCodex:
		return filepath.Join(c.ConfigDir, "hooks.json")
	default:
		return filepath.Join(c.ConfigDir, "settings.json")
	}
}

func defaultConfigDir(provider Provider) string {
	home, _ := os.UserHomeDir()

	switch provider {
	case ProviderCodex:
		if configDir := os.Getenv("CODEX_HOME"); configDir != "" {
			return configDir
		}
		return filepath.Join(home, ".codex")
	default:
		if configDir := os.Getenv("CLAUDE_CONFIG_DIR"); configDir != "" {
			return configDir
		}
		return filepath.Join(home, ".claude")
	}
}
