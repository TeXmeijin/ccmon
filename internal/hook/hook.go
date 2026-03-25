package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TeXmeijin/ccmon/internal/db"
	"github.com/TeXmeijin/ccmon/internal/model"
)

// HookPayload represents the JSON payload from Claude Code command hooks.
// Fields vary by event type.
type HookPayload struct {
	SessionID     string `json:"session_id"`
	HookEventName string `json:"hook_event_name"`

	// Common
	Cwd            string `json:"cwd"`
	TranscriptPath string `json:"transcript_path"`

	// PreToolUse / PostToolUse / PostToolUseFailure
	ToolName  string          `json:"tool_name"`
	ToolInput json.RawMessage `json:"tool_input"`

	// Notification
	NotificationType string `json:"notification_type"`
	Message          string `json:"message"`
	Title            string `json:"title"`

	// Stop
	StopReason           string `json:"stop_reason"`
	LastAssistantMessage string `json:"last_assistant_message"`

	// PostCompact
	CompactSummary string `json:"compact_summary"`

	// UserPromptSubmit
	Prompt string `json:"prompt"`
}

// Process reads a hook payload from the given reader and updates the database.
// If debugDir is non-empty, raw payloads are appended to debugDir/payloads.jsonl.
func Process(r io.Reader, store *db.Store, sourceNamespace string, debugDir string) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	// Debug dump
	if debugDir != "" {
		os.MkdirAll(debugDir, 0755)
		f, err := os.OpenFile(filepath.Join(debugDir, "payloads.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err == nil {
			f.Write(data)
			f.Write([]byte("\n"))
			f.Close()
		}
	}

	var payload HookPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("parsing payload: %w", err)
	}

	if payload.SessionID == "" {
		return fmt.Errorf("missing session_id")
	}

	now := time.Now()

	// Determine notification type pointer
	var notifType *string
	if payload.NotificationType != "" {
		notifType = &payload.NotificationType
	}

	// Build preview
	preview := buildPreview(&payload)
	var previewPtr *string
	if preview != "" {
		previewPtr = &preview
	}

	// Determine tool name pointer
	var toolNamePtr *string
	if payload.ToolName != "" {
		toolNamePtr = &payload.ToolName
	}

	// Insert event
	evt := &model.Event{
		SourceNamespace:  sourceNamespace,
		SessionID:        payload.SessionID,
		EventName:        payload.HookEventName,
		EventAt:          now,
		ToolName:         toolNamePtr,
		NotificationType: notifType,
		Preview:          previewPtr,
	}
	if err := store.InsertEvent(evt); err != nil {
		return fmt.Errorf("inserting event: %w", err)
	}

	// Compute new status
	newStatus := model.TransitionStatus(payload.HookEventName, notifType)

	// Build current action
	currentAction := ""
	if payload.HookEventName == "PreToolUse" {
		currentAction = model.BuildCurrentAction(payload.ToolName, extractToolPreview(&payload))
	}

	// Build headline + source
	headline, headlineSource := buildHeadline(&payload)

	// Determine cwd
	cwd := payload.Cwd
	cwdLabel := ""
	if cwd != "" {
		cwdLabel = filepath.Base(cwd)
	}

	// Build short ID
	shortID := payload.SessionID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	// Session title: set from first UserPromptSubmit (DB preserves first non-empty value)
	sessionTitle := ""
	if payload.HookEventName == "UserPromptSubmit" && payload.Prompt != "" {
		sessionTitle = truncStr(firstLine(payload.Prompt), 80)
	}

	// Upsert session
	sess := &model.Session{
		SourceNamespace: sourceNamespace,
		SessionID:       payload.SessionID,
		Cwd:             cwd,
		CwdLabel:        cwdLabel,
		Status:          newStatus,
		StartedAt:       now,
		LastEventAt:     now,
		CurrentAction:   currentAction,
		Headline:        headline,
		HeadlineSource:  headlineSource,
		SessionTitle:    sessionTitle,
		ShortID:         shortID,
	}

	if payload.HookEventName == "SessionEnd" {
		sess.EndedAt = &now
	}

	return store.UpsertSession(sess)
}

func buildPreview(p *HookPayload) string {
	switch p.HookEventName {
	case "PreToolUse", "PostToolUse", "PostToolUseFailure":
		return truncStr(extractToolPreview(p), 100)
	case "Notification":
		return truncStr(p.Message, 100)
	case "Stop":
		return truncStr(firstLine(p.LastAssistantMessage), 100)
	case "PostCompact":
		return truncStr(firstLine(p.CompactSummary), 100)
	case "UserPromptSubmit":
		return truncStr(firstLine(p.Prompt), 100)
	default:
		return ""
	}
}

func buildHeadline(p *HookPayload) (string, model.HeadlineSource) {
	switch {
	case p.HookEventName == "Notification" && p.Message != "":
		return truncStr(firstLine(p.Message), 80), model.HeadlineNotification
	case p.HookEventName == "Stop" && p.LastAssistantMessage != "":
		return truncStr(firstLine(p.LastAssistantMessage), 80), model.HeadlineAssistant
	case p.HookEventName == "PostCompact" && p.CompactSummary != "":
		return truncStr(firstLine(p.CompactSummary), 80), model.HeadlineSummary
	case p.HookEventName == "UserPromptSubmit" && p.Prompt != "":
		return truncStr(firstLine(p.Prompt), 80), model.HeadlineUser
	default:
		return "", model.HeadlineNone
	}
}

func extractToolPreview(p *HookPayload) string {
	if p.ToolName == "Bash" {
		var input struct {
			Command string `json:"command"`
		}
		if json.Unmarshal(p.ToolInput, &input) == nil && input.Command != "" {
			return firstLine(input.Command)
		}
	}
	if p.ToolName == "Edit" || p.ToolName == "Write" || p.ToolName == "Read" {
		var input struct {
			FilePath string `json:"file_path"`
		}
		if json.Unmarshal(p.ToolInput, &input) == nil && input.FilePath != "" {
			return input.FilePath
		}
	}
	// Generic: try to extract a short representation
	if len(p.ToolInput) > 0 && len(p.ToolInput) < 200 {
		return string(p.ToolInput)
	}
	return ""
}

func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

func truncStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}
