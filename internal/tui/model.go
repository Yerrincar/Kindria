package tui

import (
	"Kindria/internal/core/api/books"
	"strings"

	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	cardWith   = 20
	cardHeight = 12
)

var (
	normal    = lipgloss.Color("#EEEEEE")
	subtle    = lipgloss.AdaptiveColor{Light: "#D9DCCF", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}

	list = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
		BorderForeground(subtle).
		MarginRight(2).
		Height(cardHeight).
		Width(cardWith)

	listFocused = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).
			BorderForeground(highlight).
			MarginRight(2).
			Height(cardHeight).
			Width(cardWith)
)

type Model struct {
	books     []*metadata.Package
	cursor    int
	selected  map[int]struct{}
	width     int
	height    int
	paginator paginator.Model
}

func InitialModel(b []*metadata.Package) Model {

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	return Model{
		books:     b,
		selected:  make(map[int]struct{}),
		paginator: p,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	}

	return m, nil
}

func (m Model) View() string {
	cols := m.width / cardWith //160 / 20
	if cols < 1 {
		cols = 1
	}
	rows := m.height / cardHeight //36 / 12
	itemsPerPage := cols * rows
	m.paginator.PerPage = itemsPerPage

	start, end := m.paginator.GetSliceBounds(len(m.books))
	booksCards := make([]string, 0)

	for _, book := range m.books[start:end] {
		renderedBook := renderBookCard(book, false)
		booksCards = append(booksCards, renderedBook)
	}

	finalRows := make([]string, 0)
	for i := 0; i < len(booksCards); i += cols {
		end := i + cols
		if end > len(booksCards) {
			end = len(booksCards)
		}

		rowSlice := booksCards[i:end]
		finalRows = append(finalRows, lipgloss.JoinHorizontal(lipgloss.Top, rowSlice...))
		//Devolver el resultado de join vertical
	}
	finalRowsStacked := lipgloss.JoinVertical(lipgloss.Top, finalRows...)

	return finalRowsStacked
}

func renderBookCard(p *metadata.Package, isFocused bool) string {
	lipglossStyle := list
	if isFocused {
		lipglossStyle = listFocused
	}

	line010 := strings.Repeat("\n", 10)
	line11 := truncString(p.Metadata.Title, 18)
	line12 := truncString(p.Metadata.Author, 18)

	finalCard := lipglossStyle.Render(lipgloss.JoinVertical(lipgloss.Left, line010, line11, line12))
	return finalCard
}

func truncString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}
