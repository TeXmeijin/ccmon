package hook

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/TeXmeijin/ccmon/internal/config"
)

const (
	ccmonMarker    = "__ccmon__"
	commandMarker  = "CCMON_MARKER=1"
	featuresHeader = "[features]"
)

type hookDef struct {
	Event   string
	Matcher string
}

// Install adds ccmon hook blocks to the provider's hook configuration.
func Install(provider config.Provider, configDir, source, ccmonBinary string) error {
	switch provider {
	case config.ProviderCodex:
		if err := ensureCodexHooksEnabled(configDir); err != nil {
			return err
		}
		return installJSONHooks(codexHooksPath(configDir), codexHookDefs(), provider, source, configDir, ccmonBinary, false)
	default:
		return installJSONHooks(claudeSettingsPath(configDir), claudeHookDefs(), provider, source, configDir, ccmonBinary, true)
	}
}

// Uninstall removes only ccmon hook blocks from the provider's hook configuration.
func Uninstall(provider config.Provider, configDir string) error {
	switch provider {
	case config.ProviderCodex:
		return uninstallJSONHooks(codexHooksPath(configDir), false)
	default:
		return uninstallJSONHooks(claudeSettingsPath(configDir), true)
	}
}

func claudeHookDefs() []hookDef {
	return []hookDef{
		{Event: "SessionStart"},
		{Event: "UserPromptSubmit"},
		{Event: "PreToolUse"},
		{Event: "PostToolUse"},
		{Event: "PostToolUseFailure"},
		{Event: "Notification", Matcher: "permission_prompt|idle_prompt"},
		{Event: "Stop"},
		{Event: "StopFailure"},
		{Event: "PostCompact"},
		{Event: "SessionEnd"},
	}
}

func codexHookDefs() []hookDef {
	return []hookDef{
		{Event: "SessionStart", Matcher: "startup|resume"},
		{Event: "UserPromptSubmit"},
		{Event: "PreToolUse", Matcher: "Bash"},
		{Event: "PostToolUse", Matcher: "Bash"},
		{Event: "Stop"},
	}
}

func installJSONHooks(path string, defs []hookDef, provider config.Provider, source, configDir, ccmonBinary string, useSettingsEnvelope bool) error {
	doc, err := readJSONFile(path)
	if err != nil {
		return fmt.Errorf("reading %s: %w", path, err)
	}

	if err := backup(path); err != nil {
		return fmt.Errorf("creating backup for %s: %w", path, err)
	}

	hooks := extractHooksMap(doc, useSettingsEnvelope)
	for _, def := range defs {
		existing, ok := hooks[def.Event].([]interface{})
		if !ok {
			existing = []interface{}{}
		}
		if hasOwnedBlock(existing) {
			continue
		}

		existing = append(existing, buildHookBlock(provider, ccmonBinary, source, configDir, def.Matcher, useSettingsEnvelope))
		hooks[def.Event] = existing
	}

	if err := writeJSONFile(path, doc); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func uninstallJSONHooks(path string, useSettingsEnvelope bool) error {
	doc, err := readJSONFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading %s: %w", path, err)
	}

	if err := backup(path); err != nil {
		return fmt.Errorf("creating backup for %s: %w", path, err)
	}

	hooks := extractHooksMap(doc, useSettingsEnvelope)
	for eventKey, raw := range hooks {
		arr, ok := raw.([]interface{})
		if !ok {
			continue
		}

		filtered := removeOwnedBlocks(arr)
		if len(filtered) == 0 {
			delete(hooks, eventKey)
			continue
		}
		hooks[eventKey] = filtered
	}

	if useSettingsEnvelope {
		if len(hooks) == 0 {
			delete(doc, "hooks")
		}
	} else if len(hooks) == 0 {
		delete(doc, "hooks")
	}

	if err := writeJSONFile(path, doc); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

func extractHooksMap(doc map[string]interface{}, useSettingsEnvelope bool) map[string]interface{} {
	hooks, ok := doc["hooks"].(map[string]interface{})
	if ok {
		return hooks
	}

	hooks = make(map[string]interface{})
	doc["hooks"] = hooks
	return hooks
}

func buildHookBlock(provider config.Provider, binary, source, configDir, matcher string, useCompatMarker bool) map[string]interface{} {
	cmd := buildCommand(provider, binary, source, configDir)

	block := map[string]interface{}{
		"matcher": matcher,
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": cmd,
			},
		},
	}

	// Preserve backward-compatible ownership markers for Claude settings.json installs.
	if useCompatMarker {
		block[ccmonMarker] = true
	}

	return block
}

func buildCommand(provider config.Provider, binary, source, configDir string) string {
	args := []string{
		commandMarker,
		shellQuote(binary),
		"hook",
		"--provider",
		shellQuote(string(provider)),
		"--source",
		shellQuote(source),
		"--config-dir",
		shellQuote(configDir),
	}
	return strings.Join(args, " ")
}

func hasOwnedBlock(arr []interface{}) bool {
	for _, item := range arr {
		if isOwnedBlock(item) {
			return true
		}
	}
	return false
}

func removeOwnedBlocks(arr []interface{}) []interface{} {
	result := make([]interface{}, 0, len(arr))
	for _, item := range arr {
		if isOwnedBlock(item) {
			continue
		}
		result = append(result, item)
	}
	return result
}

func isOwnedBlock(item interface{}) bool {
	block, ok := item.(map[string]interface{})
	if !ok {
		return false
	}

	if _, exists := block[ccmonMarker]; exists {
		return true
	}

	rawHooks, ok := block["hooks"].([]interface{})
	if !ok {
		return false
	}

	for _, rawHook := range rawHooks {
		h, ok := rawHook.(map[string]interface{})
		if !ok {
			continue
		}

		command, _ := h["command"].(string)
		if strings.Contains(command, commandMarker) {
			return true
		}
	}

	return false
}

func readJSONFile(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]interface{}), nil
		}
		return nil, err
	}

	var doc map[string]interface{}
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	if doc == nil {
		doc = make(map[string]interface{})
	}
	return doc, nil
}

func writeJSONFile(path string, doc map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
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

func ensureCodexHooksEnabled(configDir string) error {
	path := filepath.Join(configDir, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading %s: %w", path, err)
		}

		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(featuresHeader+"\n"+"codex_hooks = true\n"), 0644)
	}

	updated, changed := setCodexHooksFlag(string(data))
	if !changed {
		return nil
	}

	if err := backup(path); err != nil {
		return fmt.Errorf("creating backup for %s: %w", path, err)
	}
	return os.WriteFile(path, []byte(updated), 0644)
}

func setCodexHooksFlag(contents string) (string, bool) {
	lines := strings.Split(contents, "\n")
	featuresStart := -1
	featuresEnd := len(lines)

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == featuresHeader {
			featuresStart = i
			for j := i + 1; j < len(lines); j++ {
				if isTableHeader(strings.TrimSpace(lines[j])) {
					featuresEnd = j
					break
				}
			}
			break
		}
	}

	if featuresStart == -1 {
		contents = strings.TrimRight(contents, "\n")
		if contents != "" {
			contents += "\n\n"
		}
		return contents + featuresHeader + "\n" + "codex_hooks = true\n", true
	}

	for i := featuresStart + 1; i < featuresEnd; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(trimmed, "codex_hooks") {
			continue
		}
		if strings.Contains(trimmed, "=") {
			prefix := leadingWhitespace(lines[i])
			next := prefix + "codex_hooks = true"
			if lines[i] == next {
				return contents, false
			}
			lines[i] = next
			return strings.Join(lines, "\n"), true
		}
	}

	insertAt := featuresStart + 1
	lines = append(lines[:insertAt], append([]string{"codex_hooks = true"}, lines[insertAt:]...)...)
	return strings.Join(lines, "\n"), true
}

func isTableHeader(line string) bool {
	return strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]")
}

func leadingWhitespace(s string) string {
	for i, r := range s {
		if r != ' ' && r != '\t' {
			return s[:i]
		}
	}
	return s
}

func claudeSettingsPath(configDir string) string {
	return filepath.Join(configDir, "settings.json")
}

func codexHooksPath(configDir string) string {
	return filepath.Join(configDir, "hooks.json")
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}
