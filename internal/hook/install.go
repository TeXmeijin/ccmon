package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const ccmonMarker = "__ccmon__"

// Install adds ccmon hook blocks to the settings.json at the given config dir.
// It preserves all existing hooks and settings.
//
// Claude Code hooks format:
//
//	{
//	  "hooks": {
//	    "EventName": [
//	      {
//	        "matcher": "",
//	        "__ccmon__": true,
//	        "hooks": [
//	          { "type": "command", "command": "ccmon hook ..." }
//	        ]
//	      }
//	    ]
//	  }
//	}
func Install(configDir, source, ccmonBinary string) error {
	settingsPath := filepath.Join(configDir, "settings.json")

	settings, err := readSettingsJSON(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings: %w", err)
	}

	if err := backup(settingsPath); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	type hookDef struct {
		Event   string
		Matcher string
	}

	hookDefs := []hookDef{
		{"SessionStart", ""},
		{"UserPromptSubmit", ""},
		{"PreToolUse", ""},
		{"PostToolUse", ""},
		{"PostToolUseFailure", ""},
		{"Notification", "permission_prompt|idle_prompt"},
		{"Stop", ""},
		{"StopFailure", ""},
		{"PostCompact", ""},
		{"SessionEnd", ""},
	}

	for _, hd := range hookDefs {
		eventKey := hd.Event
		existing, ok := hooks[eventKey].([]interface{})
		if !ok {
			existing = []interface{}{}
		}

		if hasCcmonBlock(existing) {
			continue
		}

		block := buildHookBlock(ccmonBinary, source, configDir, hd.Matcher)
		existing = append(existing, block)
		hooks[eventKey] = existing
	}

	return writeSettingsJSON(settingsPath, settings)
}

// Uninstall removes only ccmon hook blocks from settings.json.
func Uninstall(configDir string) error {
	settingsPath := filepath.Join(configDir, "settings.json")

	settings, err := readSettingsJSON(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading settings: %w", err)
	}

	if err := backup(settingsPath); err != nil {
		return fmt.Errorf("creating backup: %w", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return nil
	}

	for eventKey, val := range hooks {
		arr, ok := val.([]interface{})
		if !ok {
			continue
		}
		filtered := removeCcmonBlocks(arr)
		if len(filtered) == 0 {
			delete(hooks, eventKey)
		} else {
			hooks[eventKey] = filtered
		}
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	}

	return writeSettingsJSON(settingsPath, settings)
}

// buildHookBlock creates a matcher-group with the correct Claude Code hooks format:
//
//	{
//	  "matcher": "...",
//	  "__ccmon__": true,
//	  "hooks": [
//	    { "type": "command", "command": "ccmon hook ..." }
//	  ]
//	}
func buildHookBlock(binary, source, configDir, matcher string) map[string]interface{} {
	cmd := fmt.Sprintf("%s hook --source %s --config-dir %s", binary, source, configDir)

	return map[string]interface{}{
		"matcher":    matcher,
		ccmonMarker: true,
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": cmd,
			},
		},
	}
}

func hasCcmonBlock(arr []interface{}) bool {
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if ok {
			if _, exists := m[ccmonMarker]; exists {
				return true
			}
		}
	}
	return false
}

func removeCcmonBlocks(arr []interface{}) []interface{} {
	var result []interface{}
	for _, item := range arr {
		m, ok := item.(map[string]interface{})
		if ok {
			if _, exists := m[ccmonMarker]; exists {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

func readSettingsJSON(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	return settings, nil
}

func writeSettingsJSON(path string, settings map[string]interface{}) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}

func backup(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path+".bak", data, 0644)
}
