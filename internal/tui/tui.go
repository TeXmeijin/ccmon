package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/TeXmeijin/ccmon/internal/db"
	"github.com/TeXmeijin/ccmon/internal/ghostty"
	"github.com/TeXmeijin/ccmon/internal/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tickMsg time.Time

type Model struct {
	store          *db.Store
	cards          []model.SessionCardVM
	cursor         int
	width          int
	height         int
	scrollY        int // scroll offset in card-rows
	tick           int
	source         string
	bindingLogPath string
	lastNotice     string
}

func NewModel(store *db.Store, source string, configDir string) Model {
	return Model{
		store:          store,
		source:         source,
		bindingLogPath: filepath.Join(configDir, "ccmon", "ghostty-focus.jsonl"),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		loadSessionsCmd(m.store),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type sessionsLoadedMsg struct {
	cards []model.SessionCardVM
}

func loadSessionsCmd(store *db.Store) tea.Cmd {
	return func() tea.Msg {
		sessions, err := store.ListSessions()
		if err != nil {
			return nil
		}
		now := time.Now()
		cards := make([]model.SessionCardVM, len(sessions))
		for i, s := range sessions {
			events, _ := store.RecentEvents(s.SourceNamespace, s.SessionID, 8)
			cards[i] = model.BuildCardVM(&s, events, now)
		}
		return sessionsLoadedMsg{cards: cards}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			return m, loadSessionsCmd(m.store)
		case "up", "k":
			m.moveCursor(-1)
		case "down", "j":
			m.moveCursor(1)
		case "left", "h":
			m.moveCursorH(-1)
		case "right", "l":
			m.moveCursorH(1)
		case "g", "home":
			m.cursor = 0
			m.scrollY = 0
		case "G", "end":
			if len(m.cards) > 0 {
				m.cursor = len(m.cards) - 1
			}
			m.ensureVisible()
		case "enter":
			m.focusGhosttyTerminal()
		}

	case tea.MouseMsg:
		if msg.Action == tea.MouseActionRelease && msg.Button == tea.MouseButtonLeft {
			if idx, ok := m.hitTestCard(msg.X, msg.Y); ok {
				m.cursor = idx
				m.focusGhosttyTerminal()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		m.tick++
		return m, tea.Batch(tickCmd(), loadSessionsCmd(m.store))

	case sessionsLoadedMsg:
		m.cards = msg.cards
		if m.cursor >= len(m.cards) && len(m.cards) > 0 {
			m.cursor = len(m.cards) - 1
		}
		m.ensureVisible()
	}

	return m, nil
}

func (m *Model) columns() int {
	switch {
	case m.width >= 240:
		return 3
	case m.width >= 160:
		return 2
	default:
		return 1
	}
}

func (m *Model) cardWidth() int {
	cols := m.columns()
	return m.width / cols
}

func (m *Model) visibleRows() int {
	if m.height <= 2 {
		return 1
	}
	// Each card is roughly cardHeight + 2 (border) lines
	rowH := cardHeight + 3
	return max((m.height-1)/rowH, 1)
}

func (m *Model) totalRows() int {
	cols := m.columns()
	rows := (len(m.cards) + cols - 1) / cols
	return rows
}

func (m *Model) cursorRow() int {
	cols := m.columns()
	if cols == 0 {
		return 0
	}
	return m.cursor / cols
}

func (m *Model) moveCursor(delta int) {
	cols := m.columns()
	newCursor := m.cursor + delta*cols
	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor >= len(m.cards) {
		newCursor = len(m.cards) - 1
	}
	if newCursor < 0 {
		newCursor = 0
	}
	m.cursor = newCursor
	m.ensureVisible()
}

func (m *Model) moveCursorH(delta int) {
	newCursor := m.cursor + delta
	if newCursor < 0 {
		newCursor = 0
	}
	if newCursor >= len(m.cards) {
		newCursor = len(m.cards) - 1
	}
	if newCursor < 0 {
		newCursor = 0
	}
	m.cursor = newCursor
	m.ensureVisible()
}

func (m *Model) ensureVisible() {
	row := m.cursorRow()
	vis := m.visibleRows()
	if row < m.scrollY {
		m.scrollY = row
	}
	if row >= m.scrollY+vis {
		m.scrollY = row - vis + 1
	}
}

func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}

	if len(m.cards) == 0 {
		return renderEmptyState(m.width, m.height)
	}

	cols := m.columns()
	cw := m.cardWidth()

	// Header
	header := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(fmt.Sprintf(" ccmon [%s] %d sessions", m.source, len(m.cards)))

	// Build card grid
	var rows []string
	for i := 0; i < len(m.cards); i += cols {
		var rowCards []string
		for j := 0; j < cols && i+j < len(m.cards); j++ {
			idx := i + j
			selected := idx == m.cursor
			card := renderPulseCard(m.cards[idx], selected, cw, m.tick)
			rowCards = append(rowCards, card)
		}
		// Pad incomplete rows
		for len(rowCards) < cols {
			rowCards = append(rowCards, strings.Repeat(" ", cw))
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, rowCards...)
		rows = append(rows, row)
	}

	// Apply scroll
	vis := m.visibleRows()
	startRow := m.scrollY
	if startRow > len(rows) {
		startRow = 0
	}
	endRow := startRow + vis
	if endRow > len(rows) {
		endRow = len(rows)
	}
	visibleRows := rows[startRow:endRow]

	// Scroll indicator
	scrollInfo := ""
	if len(rows) > vis {
		scrollInfo = mutedStyle().Render(fmt.Sprintf(" [%d/%d]", startRow+1, len(rows)))
	}

	body := lipgloss.JoinVertical(lipgloss.Left, visibleRows...)

	// Footer
	footerText := " q:quit  hjkl:move  r:reload  enter:focus" + scrollInfo
	if m.lastNotice != "" {
		footerText += "  |  " + m.lastNotice
	}
	footer := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(footerText)

	// Combine
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	return content
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// focusGhosttyTerminal activates the Ghostty pane for the currently selected card.
func (m *Model) focusGhosttyTerminal() {
	if m.cursor < 0 || m.cursor >= len(m.cards) {
		return
	}
	termID := m.cards[m.cursor].GhosttyTerminalID
	if termID == "" {
		m.lastNotice = "pane unbound"
		_ = ghostty.AppendBindingLog(m.bindingLogPath, map[string]any{
			"at":               ghostty.Timestamp(),
			"kind":             "focus",
			"source_namespace": m.cards[m.cursor].SourceNamespace,
			"session_id":       m.cards[m.cursor].SessionID,
			"cwd":              m.cards[m.cursor].Cwd,
			"target_id":        "",
			"result":           "unbound",
		})
		return
	}

	result, err := ghostty.FocusTerminalByID(termID)
	if err != nil {
		m.lastNotice = "pane lookup failed"
		_ = ghostty.AppendBindingLog(m.bindingLogPath, map[string]any{
			"at":               ghostty.Timestamp(),
			"kind":             "focus",
			"source_namespace": m.cards[m.cursor].SourceNamespace,
			"session_id":       m.cards[m.cursor].SessionID,
			"cwd":              m.cards[m.cursor].Cwd,
			"target_id":        termID,
			"result":           string(result),
			"error":            err.Error(),
		})
		return
	}

	_ = ghostty.AppendBindingLog(m.bindingLogPath, map[string]any{
		"at":               ghostty.Timestamp(),
		"kind":             "focus",
		"source_namespace": m.cards[m.cursor].SourceNamespace,
		"session_id":       m.cards[m.cursor].SessionID,
		"cwd":              m.cards[m.cursor].Cwd,
		"target_id":        termID,
		"result":           string(result),
	})

	switch result {
	case ghostty.FocusResultFocused:
		m.lastNotice = ""
	case ghostty.FocusResultMissing:
		m.lastNotice = "pane binding stale"
		_ = m.store.ClearGhosttyTerminalID(m.cards[m.cursor].SourceNamespace, m.cards[m.cursor].SessionID)
		m.cards[m.cursor].GhosttyTerminalID = ""
	case ghostty.FocusResultAmbiguous:
		m.lastNotice = "pane binding ambiguous"
	default:
		m.lastNotice = "Ghostty unavailable"
	}
}

// hitTestCard determines which card index was clicked based on mouse coordinates.
func (m *Model) hitTestCard(x, y int) (int, bool) {
	if len(m.cards) == 0 {
		return 0, false
	}
	cols := m.columns()
	cw := m.cardWidth()
	rowH := cardHeight + 3 // card content + border

	// y=0 is the header line
	cardY := y - 1
	if cardY < 0 {
		return 0, false
	}

	row := cardY/rowH + m.scrollY
	col := x / cw
	if col >= cols {
		col = cols - 1
	}

	idx := row*cols + col
	if idx < 0 || idx >= len(m.cards) {
		return 0, false
	}
	return idx, true
}
