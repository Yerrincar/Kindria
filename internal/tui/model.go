package tui

import (
	metadata "Kindria/internal/core/api/books"
	"log"
	"os"
	"strings"

	"github.com/blacktop/go-termimg"
	"github.com/charmbracelet/bubbles/paginator"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	cardWith   = 30
	cardHeight = 25
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
	cols      int
	paginator paginator.Model
	covers    map[int]string
	handler   metadata.Handler
}

type coversLoadedMsg map[int]string

var debugEnabled = os.Getenv("KINDRIA_DEBUG") != ""

func debugLog(format string, args ...any) {
	if !debugEnabled {
		return
	}
	log.Printf("tui: "+format, args...)
}

func InitialModel(b []*metadata.Package, h *metadata.Handler) *Model {

	p := paginator.New()
	p.Type = paginator.Dots
	p.ActiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "235", Dark: "252"}).Render("•")
	p.InactiveDot = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "250", Dark: "238"}).Render("•")
	debugLog("InitialModel: books=%d", len(b))
	return &Model{
		books:     b,
		selected:  make(map[int]struct{}),
		paginator: p,
		covers:    make(map[int]string),
		handler:   *h,
	}
}

func (m *Model) Init() tea.Cmd {
	debugLog("Init")
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmdSync tea.Cmd
	var cmdPaginator tea.Cmd
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		debugLog("WindowSize: w=%d h=%d", msg.Width, msg.Height)
		m.width, m.height = msg.Width, msg.Height
		m.cols = m.width / cardWith
		if m.cols < 1 {
			m.cols = 1
		}

		m.paginator.PerPage = m.cols * (m.height / cardHeight)
		m.paginator.SetTotalPages(len(m.books))
		cmdSync = m.syncVisibleWidget()

	case tea.KeyMsg:
		debugLog("Key: %s", msg.String())
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
		case "right", "l":
			m.paginator.NextPage()
			cmdSync = m.syncVisibleWidget()
		case "left", "h":
			m.paginator.PrevPage()
			cmdSync = m.syncVisibleWidget()
		}

	case coversLoadedMsg:
		debugLog("coversLoadedMsg: count=%d", len(msg))
		m.covers = msg
		return m, nil
	}

	m.paginator, cmdPaginator = m.paginator.Update(msg)

	return m, tea.Batch(cmdPaginator, cmdSync)
}

func (m Model) View() string {
	debugLog("View")
	var b strings.Builder
	b.WriteString("\n Kindria, your TUI e-book library\n")

	start, end := m.paginator.GetSliceBounds(len(m.books))
	booksCards := make([]string, 0)

	for i, book := range m.books[start:end] {
		absoluteIndex := i + start
		cover, ok := m.covers[absoluteIndex]
		if !ok || cover == "" {
			cover += strings.Repeat("\n", 10)
		}
		renderedBook := m.renderBookCard(book, absoluteIndex == m.cursor, cover)
		booksCards = append(booksCards, renderedBook)
	}

	finalRows := make([]string, 0)
	if m.cols < 1 {
		m.cols = 1
	}
	for i := 0; i < len(booksCards); i += m.cols {
		endRows := i + m.cols
		if endRows > len(booksCards) {
			endRows = len(booksCards)
		}

		rowSlice := booksCards[i:endRows]
		finalRows = append(finalRows, lipgloss.JoinHorizontal(lipgloss.Top, rowSlice...))
	}
	finalRowStacked := lipgloss.JoinVertical(lipgloss.Top, finalRows...)
	b.WriteString(finalRowStacked)
	b.WriteString("\n  " + m.paginator.View())
	b.WriteString("\n\n  h/l ←/→ page • q: quit\n")

	return b.String()
}

func (m Model) renderBookCard(p *metadata.Package, isFocused bool, imageString string) string {
	lipglossStyle := list
	if isFocused {
		lipglossStyle = listFocused
	}

	line11 := truncString(p.Metadata.Title, 18)
	line12 := truncString(p.Metadata.Author, 18)

	finalCard := lipglossStyle.Render(lipgloss.JoinVertical(lipgloss.Left, imageString, line11, line12))
	return finalCard
}

func truncString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func (m *Model) syncVisibleWidget() tea.Cmd {
	start, end := m.paginator.GetSliceBounds(len(m.books))
	localCovers := make(map[int]string)
	booksToLoad := m.books[start:end]
	debugLog("syncVisibleWidget queued: start=%d end=%d books=%d", start, end, len(booksToLoad))
	return func() tea.Msg {
		debugLog("syncVisibleWidget start: start=%d end=%d books=%d", start, end, len(booksToLoad))
		for i, book := range booksToLoad {
			path, err := m.handler.SelectBookPath(book.BookFile)
			if err != nil {
				log.Printf("Error getting book path for book: "+book.BookFile+"%v: ", err)
				continue
			}
			cover, err := termimg.NewImageWidgetFromFile(path)
			if err != nil {
				log.Printf("Error trying to get cover from file: %v ", err)
				continue
			}
			cover.SetSize(30, 20).SetProtocol(termimg.Kitty)

			finalCover := ""
			if cover != nil {
				coverRendered, err := cover.Render()
				if err != nil {
					log.Printf("Err rendering cover: %v ", err)
				}
				finalCover += coverRendered
			} else {
				finalCover += ""
			}
			localCovers[i+start] = finalCover
		}
		debugLog("syncVisibleWidget end: loaded=%d", len(localCovers))
		return coversLoadedMsg(localCovers)
	}
}
