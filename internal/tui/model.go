package tui

import (
	"Kindria/internal/core/api/books"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

type model struct {
	books    []*metadata.Package
	cursor   int
	selected map[int]struct{}
}

func InitialModel(b []*metadata.Package) model {
	return model{
		books:    b,
		selected: make(map[int]struct{}),
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {

		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}

		case "down", "j":
			if m.cursor < len(m.books)-1 {
				m.cursor++
			}

		case "enter", " ":
			_, ok := m.selected[m.cursor]
			if ok {
				delete(m.selected, m.cursor)
			} else {
				m.selected[m.cursor] = struct{}{}
			}
		}
	}
	return m, nil
}

func (m model) View() string {
	s := "Kindria, your TUI e-book library\n"

	for i, book := range m.books {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if _, ok := m.selected[i]; ok {
			checked = "x"
		}
		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, book.Metadata.Title)
	}

	s += "\nPress q to quit. \n"

	return s
}
