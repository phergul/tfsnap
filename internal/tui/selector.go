package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Item struct {
	Label   string
	Content string
	Meta    any
}

type Action struct {
	Key         string
	Label       string
	Description string
}

type ActionResult struct {
	Item   *Item
	Action string
}

type SelectorModel struct {
	Items         []Item
	Actions       []Action
	SelectedIndex int
	Width         int
	Height        int
	Title         string
	Quitting      bool
	Selected      *Item
	Result        *ActionResult
}

func NewSelector(title string, items []Item) SelectorModel {
	return SelectorModel{
		Title:         title,
		Items:         items,
		SelectedIndex: 0,
		Width:         120,
		Height:        30,
	}
}

func NewActionSelector(title string, items []Item, actions []Action) SelectorModel {
	return SelectorModel{
		Title:         title,
		Items:         items,
		Actions:       actions,
		SelectedIndex: 0,
		Width:         120,
		Height:        30,
	}
}

func (m SelectorModel) Init() tea.Cmd {
	return nil
}

func (m SelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.Quitting = true
			return m, tea.Quit

		case "enter":
			if len(m.Items) > 0 {
				if len(m.Actions) > 0 {
					m.Result = &ActionResult{
						Item:   &m.Items[m.SelectedIndex],
						Action: "enter",
					}
				} else {
					m.Selected = &m.Items[m.SelectedIndex]
				}
			}
			m.Quitting = true
			return m, tea.Quit

		case "up", "k":
			if m.SelectedIndex > 0 {
				m.SelectedIndex--
			}

		case "down", "j":
			if m.SelectedIndex < len(m.Items)-1 {
				m.SelectedIndex++
			}

		default:
			if len(m.Actions) > 0 && len(m.Items) > 0 {
				keyStr := msg.String()
				for _, action := range m.Actions {
					if keyStr == action.Key {
						m.Result = &ActionResult{
							Item:   &m.Items[m.SelectedIndex],
							Action: action.Key,
						}
						m.Quitting = true
						return m, tea.Quit
					}
				}
			}
		}

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
	}

	return m, nil
}

func (m SelectorModel) View() string {
	if m.Quitting {
		return ""
	}

	if len(m.Items) == 0 {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Render("No items available.")
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("170")).
		Background(lipgloss.Color("235"))

	normalStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")).
		Padding(1, 0, 0, 2)

	leftWidth := m.Width/3 - 2
	rightWidth := m.Width - leftWidth - 6
	contentHeight := m.Height - 6
	if len(m.Actions) > 0 {
		contentHeight = m.Height - 8
	}

	var leftPane strings.Builder
	leftPane.WriteString(titleStyle.Render(m.Title) + "\n\n")

	visibleStart := 0
	visibleEnd := len(m.Items)

	maxVisible := contentHeight - 2
	if len(m.Items) > maxVisible {
		if m.SelectedIndex > maxVisible/2 {
			visibleStart = min(m.SelectedIndex-maxVisible/2, len(m.Items)-maxVisible)
			visibleEnd = min(visibleStart+maxVisible, len(m.Items))
		} else {
			visibleEnd = min(maxVisible, len(m.Items))
		}
	}

	for i := visibleStart; i < visibleEnd; i++ {
		cursor := "  "
		style := normalStyle
		if i == m.SelectedIndex {
			cursor = "▸ "
			style = selectedStyle
		}

		label := m.Items[i].Label
		if len(label) > leftWidth-4 {
			label = label[:leftWidth-7] + "..."
		}

		leftPane.WriteString(cursor + style.Render(label) + "\n")
	}

	leftPaneStyle := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1)

	var rightPane strings.Builder
	if len(m.Items) > 0 {
		content := m.Items[m.SelectedIndex].Content

		lines := strings.Split(content, "\n")
		var wrappedLines []string
		for _, line := range lines {
			if len(line) <= rightWidth-4 {
				wrappedLines = append(wrappedLines, line)
			} else {
				words := strings.Fields(line)
				currentLine := ""
				for _, word := range words {
					if len(currentLine)+len(word)+1 <= rightWidth-4 {
						if currentLine != "" {
							currentLine += " "
						}
						currentLine += word
					} else {
						if currentLine != "" {
							wrappedLines = append(wrappedLines, currentLine)
						}
						currentLine = word
					}
				}
				if currentLine != "" {
					wrappedLines = append(wrappedLines, currentLine)
				}
			}
		}

		displayLines := wrappedLines
		if len(displayLines) > contentHeight-2 {
			displayLines = displayLines[:contentHeight-2]
		}

		rightPane.WriteString(strings.Join(displayLines, "\n"))
	}

	rightPaneStyle := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Padding(0, 1)

	leftRendered := leftPaneStyle.Render(leftPane.String())
	rightRendered := rightPaneStyle.Render(rightPane.String())

	content := lipgloss.JoinHorizontal(
		lipgloss.Top,
		leftRendered,
		rightRendered,
	)

	var help string
	if len(m.Actions) > 0 {
		actionStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("86")).
			Bold(true)

		var helpParts []string
		helpParts = append(helpParts, "↑/↓: navigate")

		for _, action := range m.Actions {
			helpParts = append(helpParts, fmt.Sprintf("%s: %s", actionStyle.Render(action.Key), action.Label))
		}

		helpParts = append(helpParts, "q: quit")
		help = helpStyle.Render(strings.Join(helpParts, " • "))
	} else {
		help = helpStyle.Render("↑/↓: navigate • enter: select • q/esc: quit")
	}

	return fmt.Sprintf("%s\n%s", content, help)
}

func RunSelector(title string, items []Item) (*Item, error) {
	m := NewSelector(title, items)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(SelectorModel)
	return result.Selected, nil
}

func RunActionSelector(title string, items []Item, actions []Action) (*ActionResult, error) {
	m := NewActionSelector(title, items, actions)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(SelectorModel)
	return result.Result, nil
}
