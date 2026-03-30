package tui

import (
	"fmt"
	"strings"

	"github.com/TeXmeijin/ccmon/internal/model"
	"github.com/charmbracelet/lipgloss"
)

const cardHeight = 4 // fixed inner height (lines)

func renderCard(vm model.SessionCardVM, selected bool, width int, tick int) string {
	style := cardStyle(selected, vm.Status, width)
	return renderCardInner(vm, selected, width, style, tick)
}

func renderCardInner(vm model.SessionCardVM, selected bool, width int, style lipgloss.Style, tick int) string {
	innerW := width - 4 // border + padding

	// Row 1: badge + spinner + cwd + elapsed
	badge := statusBadge(vm.Status)
	elapsed := mutedStyle().Render(vm.ElapsedLabel)

	spinner := ""
	if vm.Status == model.StatusRunning {
		spinChars := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		spinner = lipgloss.NewStyle().Foreground(colorRunning).Render(spinChars[tick%len(spinChars)]) + " "
	}

	// Reserve space for badge + spinner + cwd + title + elapsed
	fixedW := lipgloss.Width(badge) + lipgloss.Width(spinner) + lipgloss.Width(elapsed) + 3 // spaces
	availW := innerW - fixedW

	cwdText := vm.CwdLabel
	titleText := ""

	if vm.SessionTitle != "" && availW > lipgloss.Width(cwdText)+4 {
		// Show title in remaining space after cwd
		cwdW := lipgloss.Width(cwdText)
		titleMaxW := availW - cwdW - 1 // 1 for space separator
		if titleMaxW > 3 {
			titleText = vm.SessionTitle
			if lipgloss.Width(titleText) > titleMaxW {
				titleText = truncateStr(titleText, titleMaxW)
			}
		}
	}

	if lipgloss.Width(cwdText) > availW {
		cwdText = truncateStr(cwdText, availW)
	}

	cwd := headerStyle().Render(cwdText)
	title := ""
	if titleText != "" {
		title = " " + mutedStyle().Render(titleText)
	}

	leftPart := badge + " " + spinner + cwd + title
	gapW := innerW - lipgloss.Width(leftPart) - lipgloss.Width(elapsed)
	if gapW < 1 {
		gapW = 1
	}
	row1 := leftPart + strings.Repeat(" ", gapW) + elapsed

	// Row 2: headline — what this session is about, with source prefix + color
	row2 := ""
	if vm.Headline != "" {
		prefix, style := headlineStyle(vm.HeadlineSource)
		hlText := prefix + vm.Headline
		if lipgloss.Width(hlText) > innerW {
			hlText = truncateStr(hlText, innerW)
		}
		row2 = style.Render(hlText)
	}

	// Row 3: current action — what tool is running right now
	row3 := ""
	if vm.CurrentAction != "" {
		actionText := vm.CurrentAction
		if lipgloss.Width(actionText) > innerW {
			actionText = truncateStr(actionText, innerW)
		}
		row3 = textStyle().Render(actionText)
	}

	// Row 4: event dots + short session id
	dots := renderDots(vm.EventDots)
	sid := mutedStyle().Render(vm.SessionID[:min(8, len(vm.SessionID))])
	dotsGap := innerW - lipgloss.Width(dots) - lipgloss.Width(sid)
	if dotsGap < 1 {
		dotsGap = 1
	}
	row4 := dots + strings.Repeat(" ", dotsGap) + sid

	lines := []string{row1, row2, row3, row4}
	content := strings.Join(lines, "\n")
	return style.Render(content)
}

func renderDots(dots []model.DotKind) string {
	if len(dots) == 0 {
		return mutedStyle().Render("···")
	}
	parts := make([]string, len(dots))
	for i, d := range dots {
		c := dotColor(d)
		parts[i] = lipgloss.NewStyle().Foreground(c).Render("●")
	}
	return strings.Join(parts, " ")
}

func renderPulseCard(vm model.SessionCardVM, selected bool, width int, tick int) string {
	// Permission waiting: pulse the border color
	if vm.Status == model.StatusWaitingPermission {
		style := cardStyle(selected, vm.Status, width)
		if tick%4 < 2 {
			style = style.BorderForeground(colorWaitPerm)
		} else {
			style = style.BorderForeground(lipgloss.Color("#ffcb6b"))
		}
		return renderCardInner(vm, selected, width, style, tick)
	}
	return renderCard(vm, selected, width, tick)
}

// truncateStr truncates s to fit within max display columns (CJK-aware).
func truncateStr(s string, maxCols int) string {
	if maxCols <= 1 {
		return "…"
	}
	if lipgloss.Width(s) <= maxCols {
		return s
	}
	runes := []rune(s)
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + "…"
		if lipgloss.Width(candidate) <= maxCols {
			return candidate
		}
	}
	return "…"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderEmptyState shows a placeholder when no sessions exist.
func renderEmptyState(width, height int) string {
	msg := fmt.Sprintf(
		"%s\n\n%s\n%s",
		headerStyle().Render("ccmon"),
		textStyle().Render("No active sessions"),
		mutedStyle().Render("Waiting for hook events..."),
	)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, msg)
}
