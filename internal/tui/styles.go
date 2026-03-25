package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/TeXmeijin/ccmon/internal/model"
)

// Color palette
var (
	colorBg        = lipgloss.Color("#1a1b26")
	colorCardBg    = lipgloss.Color("#24283b")
	colorCardSelBg = lipgloss.Color("#2f3347")
	colorBorder    = lipgloss.Color("#3b4261")
	colorSelBorder = lipgloss.Color("#7aa2f7")
	colorMuted     = lipgloss.Color("#565f89")
	colorText      = lipgloss.Color("#a9b1d6")
	colorBright    = lipgloss.Color("#c0caf5")

	// Status colors
	colorRunning    = lipgloss.Color("#7dcfff")
	colorWaitInput  = lipgloss.Color("#bb9af7")
	colorWaitPerm   = lipgloss.Color("#ff9e64")
	colorCompleted  = lipgloss.Color("#9ece6a")
	colorFailed     = lipgloss.Color("#f7768e")
	colorStale      = lipgloss.Color("#565f89")
	colorEnded      = lipgloss.Color("#414868")

	// Dot colors
	colorDotTool    = lipgloss.Color("#7aa2f7")
	colorDotNotify  = lipgloss.Color("#ff9e64")
	colorDotStop    = lipgloss.Color("#9ece6a")
	colorDotFail    = lipgloss.Color("#f7768e")
	colorDotCompact = lipgloss.Color("#bb9af7")
)

func statusColor(s model.Status) lipgloss.Color {
	switch s {
	case model.StatusRunning:
		return colorRunning
	case model.StatusWaitingInput:
		return colorWaitInput
	case model.StatusWaitingPermission:
		return colorWaitPerm
	case model.StatusCompleted:
		return colorCompleted
	case model.StatusFailed:
		return colorFailed
	case model.StatusStale:
		return colorStale
	case model.StatusEnded:
		return colorEnded
	default:
		return colorText
	}
}

func dotColor(d model.DotKind) lipgloss.Color {
	switch d {
	case model.DotTool:
		return colorDotTool
	case model.DotNotify:
		return colorDotNotify
	case model.DotStop:
		return colorDotStop
	case model.DotFail:
		return colorDotFail
	case model.DotCompact:
		return colorDotCompact
	default:
		return colorMuted
	}
}

func statusBadge(s model.Status) string {
	label := ""
	switch s {
	case model.StatusRunning:
		label = " RUN "
	case model.StatusWaitingInput:
		label = " WAIT "
	case model.StatusWaitingPermission:
		label = " PERM "
	case model.StatusCompleted:
		label = " DONE "
	case model.StatusFailed:
		label = " FAIL "
	case model.StatusStale:
		label = " IDLE "
	case model.StatusEnded:
		label = " END "
	}

	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#1a1b26")).
		Background(statusColor(s))

	return style.Render(label)
}

func cardStyle(selected bool, status model.Status, width int) lipgloss.Style {
	bg := colorCardBg
	border := colorBorder
	if selected {
		bg = colorCardSelBg
		border = colorSelBorder
	}

	// Dim ended/stale cards
	if status == model.StatusEnded {
		bg = lipgloss.Color("#1e2030")
	}

	return lipgloss.NewStyle().
		Width(width - 2).
		Background(bg).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(border).
		Padding(0, 1)
}

func headerStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorBright).
		Bold(true)
}

func mutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorMuted)
}

func textStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(colorText)
}

// headlineStyle returns prefix string and style based on headline source.
func headlineStyle(src model.HeadlineSource) (string, lipgloss.Style) {
	switch src {
	case model.HeadlineUser:
		return "> ", lipgloss.NewStyle().Foreground(colorBright)
	case model.HeadlineAssistant:
		return "< ", lipgloss.NewStyle().Foreground(colorText)
	case model.HeadlineNotification:
		return "! ", lipgloss.NewStyle().Foreground(colorWaitPerm).Bold(true)
	case model.HeadlineSummary:
		return "~ ", lipgloss.NewStyle().Foreground(colorMuted)
	default:
		return "", mutedStyle()
	}
}
