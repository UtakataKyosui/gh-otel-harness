package tui

import (
	"fmt"
	"strings"

	"github.com/UtakataKyosui/gh-c2-harness/internal/classify"
	"github.com/UtakataKyosui/gh-c2-harness/internal/fingerprint"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	styleSelected   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	styleUnselected = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	styleCursor     = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	styleHeader     = lipgloss.NewStyle().Bold(true).Underline(true)
	styleCategory   = map[classify.Category]lipgloss.Style{
		classify.CategoryToolError:   lipgloss.NewStyle().Foreground(lipgloss.Color("1")),
		classify.CategoryRefusal:     lipgloss.NewStyle().Foreground(lipgloss.Color("3")),
		classify.CategoryToolAnomaly: lipgloss.NewStyle().Foreground(lipgloss.Color("4")),
	}
)

type Model struct {
	events   []*classify.Event
	cursor   int
	selected map[int]bool
	quitting bool
	Chosen   []*classify.Event // populated after confirm
}

func New(events []*classify.Event) *Model {
	return &Model{
		events:   events,
		selected: make(map[int]bool),
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
			m.quitting = true
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
			if m.cursor > 0 {
				m.cursor--
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
			if m.cursor < len(m.events)-1 {
				m.cursor++
			}
		case key.Matches(msg, key.NewBinding(key.WithKeys(" "))):
			m.selected[m.cursor] = !m.selected[m.cursor]
		case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
			for i, e := range m.events {
				if m.selected[i] {
					m.Chosen = append(m.Chosen, e)
				}
			}
			return m, tea.Quit
		case key.Matches(msg, key.NewBinding(key.WithKeys("a"))):
			// toggle all
			if len(m.selected) == len(m.events) {
				m.selected = make(map[int]bool)
			} else {
				for i := range m.events {
					m.selected[i] = true
				}
			}
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if m.quitting {
		return ""
	}
	var b strings.Builder
	b.WriteString(styleHeader.Render("Claude Code 失敗イベント候補") + "\n")
	b.WriteString("space: 選択切替  a: 全選択  enter: 起票  q: 終了\n\n")

	for i, e := range m.events {
		fp := fingerprint.Compute(e.EventName, e.ToolName, e.ErrorType, e.Body)
		check := "[ ]"
		if m.selected[i] {
			check = "[x]"
		}
		cursor := "  "
		if i == m.cursor {
			cursor = styleCursor.Render("> ")
		}

		catStyle, ok := styleCategory[e.Category]
		if !ok {
			catStyle = styleUnselected
		}

		line := fmt.Sprintf("%s %s %s %-12s  %s  %s",
			cursor,
			check,
			fp,
			catStyle.Render(string(e.Category)),
			e.Timestamp.Format("01-02 15:04"),
			truncateStr(e.Title(), 55),
		)

		if m.selected[i] {
			b.WriteString(styleSelected.Render(line))
		} else if i == m.cursor {
			b.WriteString(styleUnselected.Render(line))
		} else {
			b.WriteString(line)
		}
		b.WriteString("\n")
	}

	selected := len(m.selected)
	b.WriteString(fmt.Sprintf("\n%d 件選択中 / %d 件\n", selected, len(m.events)))
	return b.String()
}

func truncateStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
