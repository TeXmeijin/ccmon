package model

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type Status string

const (
	StatusRunning           Status = "running"
	StatusWaitingInput      Status = "waiting_input"
	StatusWaitingPermission Status = "waiting_permission"
	StatusCompleted         Status = "completed"
	StatusFailed            Status = "failed"
	StatusStale             Status = "stale"
	StatusEnded             Status = "ended"
)

const StaleThreshold = 90 * time.Second

type DotKind string

const (
	DotTool    DotKind = "tool"
	DotNotify  DotKind = "notify"
	DotStop    DotKind = "stop"
	DotFail    DotKind = "fail"
	DotCompact DotKind = "compact"
)

type HeadlineSource string

const (
	HeadlineUser         HeadlineSource = "user"
	HeadlineAssistant    HeadlineSource = "assistant"
	HeadlineNotification HeadlineSource = "notification"
	HeadlineSummary      HeadlineSource = "summary"
	HeadlineNone         HeadlineSource = "none"
)

type Session struct {
	SourceNamespace    string
	SessionID          string
	Cwd                string
	CwdLabel           string
	Status             Status
	StartedAt          time.Time
	LastEventAt        time.Time
	EndedAt            *time.Time
	CurrentAction      string
	Headline           string
	HeadlineSource     HeadlineSource
	SessionTitle       string
	ShortID            string
	GhosttyTerminalID  string
}

type Event struct {
	ID               int64
	SourceNamespace  string
	SessionID        string
	EventName        string
	EventAt          time.Time
	ToolName         *string
	NotificationType *string
	Preview          *string
}

type SessionCardVM struct {
	SourceNamespace    string
	SessionID          string
	Cwd                string
	CwdLabel           string
	Status             Status
	ElapsedLabel       string
	CurrentAction      string
	Headline           string
	HeadlineSource     HeadlineSource
	SessionTitle       string
	EventDots          []DotKind
	UpdatedAt          string
	GhosttyTerminalID  string
}

func BuildCardVM(s *Session, recentEvents []Event, now time.Time) SessionCardVM {
	status := s.Status
	if status == StatusRunning || status == StatusCompleted || status == StatusFailed {
		if now.Sub(s.LastEventAt) > StaleThreshold {
			status = StatusStale
		}
	}

	return SessionCardVM{
		SourceNamespace:    s.SourceNamespace,
		SessionID:          s.SessionID,
		Cwd:                s.Cwd,
		CwdLabel:           buildCwdLabel(s.Cwd),
		Status:             status,
		ElapsedLabel:       buildElapsedLabel(s.StartedAt, now),
		CurrentAction:      s.CurrentAction,
		Headline:           truncate(s.Headline, 80),
		HeadlineSource:     s.HeadlineSource,
		SessionTitle:       s.SessionTitle,
		EventDots:          buildEventDots(recentEvents),
		UpdatedAt:          s.LastEventAt.Format(time.RFC3339),
		GhosttyTerminalID:  s.GhosttyTerminalID,
	}
}

func buildCwdLabel(cwd string) string {
	if cwd == "" {
		return "?"
	}
	base := filepath.Base(cwd)
	if base == "." || base == "/" {
		short := cwd
		home, _ := filepath.Abs("~")
		if strings.HasPrefix(cwd, home) {
			short = "~" + cwd[len(home):]
		}
		if len(short) > 30 {
			short = "..." + short[len(short)-27:]
		}
		return short
	}
	return base
}

func buildElapsedLabel(start time.Time, now time.Time) string {
	d := now.Sub(start)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh%dm", int(d.Hours()), int(d.Minutes())%60)
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func buildEventDots(events []Event) []DotKind {
	max := 8
	if len(events) < max {
		max = len(events)
	}
	dots := make([]DotKind, 0, max)
	for i := 0; i < max; i++ {
		e := events[i]
		switch {
		case e.EventName == "PostToolUseFailure" || e.EventName == "StopFailure":
			dots = append(dots, DotFail)
		case e.EventName == "Stop":
			dots = append(dots, DotStop)
		case e.EventName == "Notification":
			dots = append(dots, DotNotify)
		case e.EventName == "PostCompact":
			dots = append(dots, DotCompact)
		case e.EventName == "PreToolUse" || e.EventName == "PostToolUse":
			dots = append(dots, DotTool)
		default:
			dots = append(dots, DotTool)
		}
	}
	return dots
}

func truncate(s string, max int) string {
	lines := strings.SplitN(s, "\n", 2)
	s = lines[0]
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}

// TransitionStatus determines the new session status based on an incoming event.
func TransitionStatus(eventName string, notificationType *string) Status {
	switch eventName {
	case "SessionStart":
		return StatusRunning
	case "UserPromptSubmit":
		return StatusRunning
	case "PreToolUse":
		return StatusRunning
	case "PostToolUse":
		return StatusRunning
	case "PostToolUseFailure":
		return StatusFailed
	case "Notification":
		if notificationType != nil {
			switch *notificationType {
			case "permission_prompt":
				return StatusWaitingPermission
			case "idle_prompt":
				return StatusWaitingInput
			}
		}
		return StatusRunning
	case "Stop":
		return StatusCompleted
	case "StopFailure":
		return StatusFailed
	case "SessionEnd":
		return StatusEnded
	default:
		return StatusRunning
	}
}

// BuildCurrentAction generates the currentAction display string from a tool event.
func BuildCurrentAction(toolName string, preview string) string {
	if toolName == "" {
		return ""
	}
	maxLen := 50
	switch toolName {
	case "Bash":
		return truncateAction("Bash: "+preview, maxLen)
	case "Edit", "Write":
		return truncateAction("Edit: "+preview, maxLen)
	case "Read":
		return truncateAction("Read: "+preview, maxLen)
	default:
		return truncateAction(toolName+": "+preview, maxLen)
	}
}

func truncateAction(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}
