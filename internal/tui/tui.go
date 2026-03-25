package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/TeXmeijin/ccmon/internal/db"
	"github.com/TeXmeijin/ccmon/internal/model"
)

type tickMsg time.Time

type Model struct {
	store    *db.Store
	cards    []model.SessionCardVM
	cursor   int
	width    int
	height   int
	scrollY  int // scroll offset in card-rows
	tick     int
	source   string
}

func NewModel(store *db.Store, source string) Model {
	return Model{
		store:  store,
		source: source,
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
	case m.width >= 140:
		return 3
	case m.width >= 80:
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
	footer := lipgloss.NewStyle().
		Foreground(colorMuted).
		Render(" q:quit  hjkl:move  r:reload" + scrollInfo)

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
