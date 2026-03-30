package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
)

func ParseProvider(raw string) (Provider, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "auto":
		return "", nil
	case string(ProviderClaude):
		return ProviderClaude, nil
	case string(ProviderCodex):
		return ProviderCodex, nil
	default:
		return "", fmt.Errorf("unknown provider %q (expected claude or codex)", raw)
	}
}

func inferProvider(flagConfigDir string) Provider {
	if p := inferProviderFromDir(flagConfigDir); p != "" {
		return p
	}

	if os.Getenv("CLAUDE_CONFIG_DIR") != "" {
		return ProviderClaude
	}
	if os.Getenv("CODEX_HOME") != "" {
		return ProviderCodex
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ProviderClaude
	}

	claudeDir := filepath.Join(home, ".claude")
	if pathExists(claudeDir) {
		return ProviderClaude
	}

	codexDir := filepath.Join(home, ".codex")
	if pathExists(codexDir) {
		return ProviderCodex
	}

	return ProviderClaude
}

func inferProviderFromDir(configDir string) Provider {
	if configDir == "" {
		return ""
	}

	base := strings.ToLower(filepath.Base(configDir))
	switch base {
	case ".claude", "claude":
		return ProviderClaude
	case ".codex", "codex":
		return ProviderCodex
	}

	lower := strings.ToLower(filepath.Clean(configDir))
	switch {
	case strings.HasSuffix(lower, string(filepath.Separator)+".claude"):
		return ProviderClaude
	case strings.HasSuffix(lower, string(filepath.Separator)+".codex"):
		return ProviderCodex
	}

	return ""
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
